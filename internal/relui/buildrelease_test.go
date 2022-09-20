// Copyright 2022 Go Authors All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package relui

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-github/github"
	"github.com/google/uuid"
	"golang.org/x/build/gerrit"
	"golang.org/x/build/internal"
	"golang.org/x/build/internal/gcsfs"
	"golang.org/x/build/internal/task"
	"golang.org/x/build/internal/workflow"
)

func TestRelease(t *testing.T) {
	t.Run("beta", func(t *testing.T) {
		testRelease(t, "go1.18beta1", task.KindBeta)
	})
	t.Run("rc", func(t *testing.T) {
		testRelease(t, "go1.18rc1", task.KindRC)
	})
}

func TestSecurity(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		testSecurity(t, true)
	})
	t.Run("failure", func(t *testing.T) {
		testSecurity(t, false)
	})
}

type releaseTestDeps struct {
	ctx            context.Context
	buildlets      *task.FakeBuildlets
	goRepo         *task.FakeRepo
	gerrit         *reviewerCheckGerrit
	versionTasks   *task.VersionTasks
	buildTasks     *BuildReleaseTasks
	milestoneTasks *task.MilestoneTasks
	publishedFiles map[string]*task.WebsiteFile
	outputListener func(taskName string, output interface{})
}

func newReleaseTestDeps(t *testing.T, wantVersion string) *releaseTestDeps {
	task.AwaitDivisor = 100
	t.Cleanup(func() { task.AwaitDivisor = 1 })
	ctx, cancel := context.WithCancel(context.Background())
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Requires bash shell scripting support.")
	}

	// Set up a server that will be used to serve inputs to the build.
	bootstrapServer := httptest.NewServer(http.HandlerFunc(serveBootstrap))
	t.Cleanup(bootstrapServer.Close)
	fakeBuildlets := task.NewFakeBuildlets(t, bootstrapServer.URL)

	// Set up the fake signing process.
	scratchDir := t.TempDir()
	argRe := regexp.MustCompile(`--relui_staging="(.*?)"`)
	outputListener := func(taskName string, output interface{}) {
		if taskName != "Start signing command" {
			return
		}
		matches := argRe.FindStringSubmatch(output.(string))
		if matches == nil {
			return
		}
		u, err := url.Parse(matches[1])
		if err != nil {
			t.Fatal(err)
		}
		go fakeSign(ctx, t, u.Path)
	}

	// Set up the fake CDN publishing process.
	servingDir := t.TempDir()
	dlDir := t.TempDir()
	dlServer := httptest.NewServer(http.FileServer(http.FS(os.DirFS(dlDir))))
	t.Cleanup(dlServer.Close)
	go fakeCDNLoad(ctx, t, servingDir, dlDir)

	// Set up the fake website to publish to.
	var filesMu sync.Mutex
	files := map[string]*task.WebsiteFile{}
	publishFile := func(f *task.WebsiteFile) error {
		filesMu.Lock()
		defer filesMu.Unlock()
		files[strings.TrimPrefix(f.Filename, wantVersion+".")] = f
		return nil
	}

	goRepo := task.NewFakeRepo(t, "go")
	base := goRepo.Commit(goFiles)
	goRepo.Tag("go1.17", base)
	dlRepo := task.NewFakeRepo(t, "dl")
	fakeGerrit := task.NewFakeGerrit(t, goRepo, dlRepo)

	gerrit := &reviewerCheckGerrit{FakeGerrit: fakeGerrit}
	versionTasks := &task.VersionTasks{
		Gerrit:    gerrit,
		GoProject: "go",
	}
	milestoneTasks := &task.MilestoneTasks{
		Client:    &fakeGitHub{},
		RepoOwner: "golang",
		RepoName:  "go",
		ApproveAction: func(ctx *workflow.TaskContext) error {
			return fmt.Errorf("unexpected approval request for %q", ctx.TaskName)
		},
	}

	buildTasks := &BuildReleaseTasks{
		GerritClient:     gerrit,
		GerritHTTPClient: http.DefaultClient,
		GerritURL:        fakeGerrit.GerritURL() + "/go",
		GCSClient:        nil,
		ScratchURL:       "file://" + filepath.ToSlash(scratchDir),
		ServingURL:       "file://" + filepath.ToSlash(servingDir),
		CreateBuildlet:   fakeBuildlets.CreateBuildlet,
		DownloadURL:      dlServer.URL,
		PublishFile:      publishFile,
		ApproveAction: func(ctx *workflow.TaskContext) error {
			if strings.Contains(ctx.TaskName, "Release Coordinator Approval") {
				return nil
			}
			return fmt.Errorf("unexpected approval request for %q", ctx.TaskName)
		},
	}
	// Cleanups are called in reverse order, and we need to cancel the context
	// before the temp dirs are deleted.
	t.Cleanup(cancel)
	return &releaseTestDeps{
		ctx:            ctx,
		buildlets:      fakeBuildlets,
		goRepo:         goRepo,
		gerrit:         gerrit,
		versionTasks:   versionTasks,
		buildTasks:     buildTasks,
		milestoneTasks: milestoneTasks,
		publishedFiles: files,
		outputListener: outputListener,
	}
}

func testRelease(t *testing.T, wantVersion string, kind task.ReleaseKind) {
	deps := newReleaseTestDeps(t, wantVersion)
	wd := workflow.New()

	deps.gerrit.wantReviewers = []string{"heschi", "dmitshur"}
	v := addSingleReleaseWorkflow(deps.buildTasks, deps.milestoneTasks, deps.versionTasks, wd, 18, kind, workflow.Const(deps.gerrit.wantReviewers))
	workflow.Output(wd, "Published Go version", v)

	w, err := workflow.Start(wd, map[string]interface{}{
		"Targets to skip testing (or 'all') (optional)":            []string{"js-wasm"},
		"Ref from the private repository to build from (optional)": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Run(deps.ctx, &verboseListener{t, deps.outputListener})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range deps.publishedFiles {
		if f.ChecksumSHA256 == "" || f.Size < 1 || f.Filename == "" || f.Kind == "" {
			t.Errorf("release process produced an invalid artifact: %#v", f)
		}
	}

	dlURL, files := deps.buildTasks.DownloadURL, deps.publishedFiles
	checkTGZ(t, dlURL, files, "src.tar.gz", &task.WebsiteFile{
		OS:   "",
		Arch: "",
		Kind: "source",
	}, map[string]string{
		"go/VERSION":       wantVersion,
		"go/src/make.bash": makeScript,
	})
	checkContents(t, dlURL, files, "windows-amd64.msi", &task.WebsiteFile{
		OS:   "windows",
		Arch: "amd64",
		Kind: "installer",
	}, "I'm an MSI!\n")
	checkTGZ(t, dlURL, files, "linux-amd64.tar.gz", &task.WebsiteFile{
		OS:   "linux",
		Arch: "amd64",
		Kind: "archive",
	}, map[string]string{
		"go/VERSION":                        wantVersion,
		"go/tool/something_orother/compile": "",
		"go/pkg/something_orother/race.a":   "",
	})
	checkZip(t, dlURL, files, "windows-arm64.zip", &task.WebsiteFile{
		OS:   "windows",
		Arch: "arm64",
		Kind: "archive",
	}, map[string]string{
		"go/VERSION":                        wantVersion,
		"go/tool/something_orother/compile": "",
	})
	checkTGZ(t, dlURL, files, "linux-armv6l.tar.gz", &task.WebsiteFile{
		OS:   "linux",
		Arch: "armv6l",
		Kind: "archive",
	}, map[string]string{
		"go/VERSION":                        wantVersion,
		"go/tool/something_orother/compile": "",
	})
	checkContents(t, dlURL, files, "darwin-amd64.pkg", &task.WebsiteFile{
		OS:   "darwin",
		Arch: "amd64",
		Kind: "installer",
	}, "I'm a .pkg!\n")

	head, err := deps.gerrit.ReadBranchHead(deps.ctx, "dl", "master")
	if err != nil {
		t.Fatal(err)
	}
	content, err := deps.gerrit.ReadFile(deps.ctx, "dl", head, wantVersion+"/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), fmt.Sprintf("version.Run(%q)", wantVersion)) {
		t.Errorf("unexpected dl content: %v", content)
	}

	tag, err := deps.gerrit.GetTag(deps.ctx, "go", wantVersion)
	if err != nil {
		t.Fatal(err)
	}

	if kind != task.KindBeta {
		version, err := deps.gerrit.ReadFile(deps.ctx, "go", tag.Revision, "VERSION")
		if err != nil {
			t.Fatal(err)
		}
		if string(version) != wantVersion {
			t.Errorf("VERSION file is %q, expected %q", version, wantVersion)
		}
	}
}

func testSecurity(t *testing.T, mergeFixes bool) {
	deps := newReleaseTestDeps(t, "go1.18rc1")

	// Set up the fake merge process. Once we stop to ask for approval, commit
	// the fix to the public server.
	privateRepo := task.NewFakeRepo(t, "go-private")
	privateRepo.Commit(goFiles)
	securityFix := map[string]string{"security.txt": "This file makes us secure"}
	privateRef := privateRepo.Commit(securityFix)
	privateGerrit := task.NewFakeGerrit(t, privateRepo)
	deps.buildTasks.PrivateGerritURL = privateGerrit.GerritURL() + "/go-private"

	defaultApprove := deps.buildTasks.ApproveAction
	deps.buildTasks.ApproveAction = func(tc *workflow.TaskContext) error {
		if mergeFixes {
			deps.goRepo.Commit(securityFix)
		}
		return defaultApprove(tc)
	}

	// Run the release.
	wd := workflow.New()
	v := addSingleReleaseWorkflow(deps.buildTasks, deps.milestoneTasks, deps.versionTasks, wd, 18, task.KindRC, workflow.Slice[string]())
	workflow.Output(wd, "Published Go version", v)

	w, err := workflow.Start(wd, map[string]interface{}{
		"Targets to skip testing (or 'all') (optional)":            []string{"js-wasm"},
		"Ref from the private repository to build from (optional)": privateRef,
	})
	if err != nil {
		t.Fatal(err)
	}

	if mergeFixes {
		_, err = w.Run(deps.ctx, &verboseListener{t, deps.outputListener})
		if err != nil {
			t.Fatal(err)
		}
	} else {
		runToFailure(t, deps.ctx, w, "Check branch state matches source archive", &verboseListener{t, deps.outputListener})
		return
	}
	checkTGZ(t, deps.buildTasks.DownloadURL, deps.publishedFiles, "src.tar.gz", &task.WebsiteFile{
		OS:   "",
		Arch: "",
		Kind: "source",
	}, map[string]string{
		"go/security.txt": "This file makes us secure",
	})
}

func TestAdvisoryTrybotFail(t *testing.T) {
	deps := newReleaseTestDeps(t, "go1.18rc1")
	defaultApprove := deps.buildTasks.ApproveAction
	approvedTrybots := false
	deps.buildTasks.ApproveAction = func(ctx *workflow.TaskContext) error {
		if strings.Contains(ctx.TaskName, "TryBot failures") {
			approvedTrybots = true
			return nil
		}
		return defaultApprove(ctx)
	}

	// Run the release.
	wd := workflow.New()
	v := addSingleReleaseWorkflow(deps.buildTasks, deps.milestoneTasks, deps.versionTasks, wd, 18, task.KindRC, workflow.Slice[string]())
	workflow.Output(wd, "Published Go version", v)

	w, err := workflow.Start(wd, map[string]interface{}{
		"Targets to skip testing (or 'all') (optional)":            []string(nil),
		"Ref from the private repository to build from (optional)": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Run(deps.ctx, &verboseListener{t, deps.outputListener}); err != nil {
		t.Fatal(err)
	}
	if !approvedTrybots {
		t.Errorf("advisory trybots didn't need approval")
	}

}

// makeScript pretends to be make.bash. It creates a fake go command that
// knows how to fake the commands the release process runs.
const makeScript = `#!/bin/bash

GO=../
mkdir -p $GO/bin

cat <<'EOF' >$GO/bin/go
#!/bin/bash -eu
case "$1 $2" in
"run releaselet.go")
    # We're building an MSI. The command should be run in the gomote work dir.
	ls go/src/make.bash >/dev/null
	mkdir msi
	echo "I'm an MSI!" > msi/thisisanmsi.msi
	;;
"install -race")
	# Installing the race mode stdlib. Doesn't matter where it's run.
	mkdir -p $(dirname $0)/../pkg/something_orother/
	touch $(dirname $0)/../pkg/something_orother/race.a
	;;
*)
	echo "unexpected command $@"
	exit 1
	;;
esac
EOF
chmod 0755 $GO/bin/go

cp $GO/bin/go $GO/bin/go.exe
# We don't know what GOOS_GOARCH we're "building" for, write some junk for
# versimilitude.
mkdir -p $GO/tool/something_orother/
touch $GO/tool/something_orother/compile
`

// allScript pretends to be all.bash. It is hardcoded to pass.
const allScript = `#!/bin/bash -eu

echo "I'm a test! :D"

if [[ $GO_BUILDER_NAME =~ "js-wasm" ]]; then
  echo "Oh no, WASM is broken"
  exit 1
fi

exit 0
`

var goFiles = map[string]string{
	"src/make.bash": makeScript,
	"src/make.bat":  makeScript,
	"src/all.bash":  allScript,
	"src/all.bat":   allScript,
	"src/race.bash": allScript,
	"src/race.bat":  allScript,
}

func serveBootstrap(w http.ResponseWriter, r *http.Request) {
	task.ServeTarball("go-builder-data/go", map[string]string{
		"bin/go": "I'm a dummy bootstrap go command!",
	}, w, r)
}

func checkFile(t *testing.T, dlURL string, files map[string]*task.WebsiteFile, filename string, meta *task.WebsiteFile, check func(*testing.T, []byte)) {
	t.Run(filename, func(t *testing.T) {
		f, ok := files[filename]
		if !ok {
			t.Fatalf("file %q not published", filename)
		}
		if diff := cmp.Diff(meta, f, cmpopts.IgnoreFields(task.WebsiteFile{}, "Filename", "Version", "ChecksumSHA256", "Size")); diff != "" {
			t.Errorf("file metadata mismatch (-want +got):\n%v", diff)
		}
		resp, err := http.Get(dlURL + "/" + f.Filename)
		if err != nil {
			t.Fatalf("getting %v: %v", f.Filename, err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("reading %v: %v", f.Filename, err)
		}
		check(t, body)
	})
}

func checkContents(t *testing.T, dlURL string, files map[string]*task.WebsiteFile, filename string, meta *task.WebsiteFile, contents string) {
	checkFile(t, dlURL, files, filename, meta, func(t *testing.T, b []byte) {
		if got, want := string(b), contents; got != want {
			t.Errorf("%v contains %q, want %q", filename, got, want)
		}
	})
}

func checkTGZ(t *testing.T, dlURL string, files map[string]*task.WebsiteFile, filename string, meta *task.WebsiteFile, contents map[string]string) {
	checkFile(t, dlURL, files, filename, meta, func(t *testing.T, b []byte) {
		gzr, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		tr := tar.NewReader(gzr)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			want, ok := contents[h.Name]
			if !ok {
				continue
			}
			b, err := ioutil.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			delete(contents, h.Name)
			if string(b) != want {
				t.Errorf("contents of %v were %q, want %q", h.Name, string(b), want)
			}
		}
		if len(contents) != 0 {
			t.Errorf("not all files were found: missing %v", contents)
		}
	})
}

func checkZip(t *testing.T, dlURL string, files map[string]*task.WebsiteFile, filename string, meta *task.WebsiteFile, contents map[string]string) {
	checkFile(t, dlURL, files, filename, meta, func(t *testing.T, b []byte) {
		zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range zr.File {
			want, ok := contents[f.Name]
			if !ok {
				continue
			}
			r, err := zr.Open(f.Name)
			if err != nil {
				t.Fatal(err)
			}
			b, err := ioutil.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			delete(contents, f.Name)
			if string(b) != want {
				t.Errorf("contents of %v were %q, want %q", f.Name, string(b), want)
			}
		}
		if len(contents) != 0 {
			t.Errorf("not all files were found: missing %v", contents)
		}
	})
}

type reviewerCheckGerrit struct {
	wantReviewers []string
	*task.FakeGerrit
}

func (g *reviewerCheckGerrit) CreateAutoSubmitChange(ctx context.Context, input gerrit.ChangeInput, reviewers []string, contents map[string]string) (string, error) {
	if diff := cmp.Diff(g.wantReviewers, reviewers, cmpopts.EquateEmpty()); diff != "" {
		return "", fmt.Errorf("unexpected reviewers for CL: %v", diff)
	}
	return g.FakeGerrit.CreateAutoSubmitChange(ctx, input, reviewers, contents)
}

type fakeGitHub struct {
}

func (g *fakeGitHub) FetchMilestone(ctx context.Context, owner, repo, name string, create bool) (int, error) {
	return 0, nil
}

func (g *fakeGitHub) Query(ctx context.Context, q interface{}, variables map[string]interface{}) error {
	return nil
}

func (g *fakeGitHub) EditIssue(ctx context.Context, owner string, repo string, number int, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
	return nil, nil, nil
}

func (g *fakeGitHub) EditMilestone(ctx context.Context, owner string, repo string, number int, milestone *github.Milestone) (*github.Milestone, *github.Response, error) {
	return nil, nil, nil
}

type verboseListener struct {
	t              *testing.T
	outputListener func(string, interface{})
}

func (l *verboseListener) TaskStateChanged(_ uuid.UUID, _ string, st *workflow.TaskState) error {
	switch {
	case !st.Finished:
		l.t.Logf("task %-10v: started", st.Name)
	case st.Error != "":
		l.t.Logf("task %-10v: error: %v", st.Name, st.Error)
	default:
		l.t.Logf("task %-10v: done: %v", st.Name, st.Result)
		if l.outputListener != nil {
			l.outputListener(st.Name, st.Result)
		}
	}
	return nil
}

func (l *verboseListener) Logger(_ uuid.UUID, task string) workflow.Logger {
	return &testLogger{t: l.t, task: task}
}

type testLogger struct {
	t    *testing.T
	task string
}

func (l *testLogger) Printf(format string, v ...interface{}) {
	l.t.Logf("task %-10v: LOG: %s", l.task, fmt.Sprintf(format, v...))
}

func runToFailure(t *testing.T, ctx context.Context, w *workflow.Workflow, task string, wrap workflow.Listener) string {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	t.Helper()
	var message string
	listener := &errorListener{
		taskName: task,
		callback: func(m string) {
			message = m
			cancel()
		},
		Listener: wrap,
	}
	_, err := w.Run(ctx, listener)
	if err == nil {
		t.Fatalf("workflow unexpectedly succeeded")
	}
	return message
}

type errorListener struct {
	taskName string
	callback func(string)
	workflow.Listener
}

func (l *errorListener) TaskStateChanged(id uuid.UUID, taskID string, st *workflow.TaskState) error {
	if st.Name == l.taskName && st.Finished && st.Error != "" {
		l.callback(st.Error)
	}
	l.Listener.TaskStateChanged(id, taskID, st)
	return nil
}

// fakeSign acts like a human running the signbinaries job periodically.
func fakeSign(ctx context.Context, t *testing.T, dir string) {
	seen := map[string]bool{}
	periodicallyDo(ctx, t, 100*time.Millisecond, func() error {
		return fakeSignOnce(t, dir, seen)
	})
}

func fakeSignOnce(t *testing.T, dir string, seen map[string]bool) error {
	dirFS := gcsfs.DirFS(dir)
	_, err := fs.Stat(dirFS, "ready")
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	contents, err := fs.ReadDir(dirFS, ".")
	if err != nil {
		return err
	}
	for _, fi := range contents {
		fn := fi.Name()
		if fn == "signed" || seen[fn] {
			continue
		}
		var copy, gpgSign, makePkg bool
		hasSuffix := func(suffix string) bool { return strings.HasSuffix(fn, suffix) }
		switch {
		case strings.Contains(fn, "darwin") && hasSuffix(".tar.gz"):
			copy = true
			gpgSign = true
			makePkg = true
		case strings.Contains(fn, "darwin") && hasSuffix(".pkg"):
			copy = true
		case hasSuffix(".tar.gz"):
			gpgSign = true
		case hasSuffix("msi"):
			copy = true
		}

		writeSignedWithHash := func(filename string, contents []byte) error {
			if err := gcsfs.WriteFile(dirFS, "signed/"+filename, contents); err != nil {
				return err
			}
			hash := fmt.Sprintf("%x", sha256.Sum256(contents))
			return gcsfs.WriteFile(dirFS, "signed/"+filename+".sha256", []byte(hash))
		}

		if copy {
			bytes, err := fs.ReadFile(dirFS, fn)
			if err != nil {
				return err
			}
			if err := writeSignedWithHash(fn, bytes); err != nil {
				return err
			}
		}
		if makePkg {
			if err := writeSignedWithHash(strings.ReplaceAll(fn, ".tar.gz", ".pkg"), []byte("I'm a .pkg!\n")); err != nil {
				return err
			}
		}
		if gpgSign {
			if err := writeSignedWithHash(fn+".asc", []byte("gpg signature")); err != nil {
				return err
			}
		}
		seen[fn] = true
	}
	return nil
}

// These are the files created by the Go 1.18 release.
const inputs = `
go1.18.darwin-amd64.tar.gz
go1.18.darwin-arm64.tar.gz
go1.18.freebsd-386.tar.gz
go1.18.freebsd-amd64.tar.gz
go1.18.linux-386.tar.gz
go1.18.linux-amd64.tar.gz
go1.18.linux-arm64.tar.gz
go1.18.linux-armv6l.tar.gz
go1.18.linux-ppc64le.tar.gz
go1.18.linux-s390x.tar.gz
go1.18.src.tar.gz
go1.18.windows-386.msi
go1.18.windows-386.zip
go1.18.windows-amd64.msi
go1.18.windows-amd64.zip
go1.18.windows-arm64.msi
go1.18.windows-arm64.zip
`

// These are the files created in the "signed" folder by the signing run for Go 1.18.
const outputs = `
go1.18.darwin-amd64.pkg
go1.18.darwin-amd64.pkg.sha256
go1.18.darwin-amd64.tar.gz
go1.18.darwin-amd64.tar.gz.asc
go1.18.darwin-amd64.tar.gz.asc.sha256
go1.18.darwin-amd64.tar.gz.sha256
go1.18.darwin-arm64.pkg
go1.18.darwin-arm64.pkg.sha256
go1.18.darwin-arm64.tar.gz
go1.18.darwin-arm64.tar.gz.asc
go1.18.darwin-arm64.tar.gz.asc.sha256
go1.18.darwin-arm64.tar.gz.sha256
go1.18.freebsd-386.tar.gz.asc
go1.18.freebsd-386.tar.gz.asc.sha256
go1.18.freebsd-amd64.tar.gz.asc
go1.18.freebsd-amd64.tar.gz.asc.sha256
go1.18.linux-386.tar.gz.asc
go1.18.linux-386.tar.gz.asc.sha256
go1.18.linux-amd64.tar.gz.asc
go1.18.linux-amd64.tar.gz.asc.sha256
go1.18.linux-arm64.tar.gz.asc
go1.18.linux-arm64.tar.gz.asc.sha256
go1.18.linux-armv6l.tar.gz.asc
go1.18.linux-armv6l.tar.gz.asc.sha256
go1.18.linux-ppc64le.tar.gz.asc
go1.18.linux-ppc64le.tar.gz.asc.sha256
go1.18.linux-s390x.tar.gz.asc
go1.18.linux-s390x.tar.gz.asc.sha256
go1.18.src.tar.gz.asc
go1.18.src.tar.gz.asc.sha256
go1.18.windows-386.msi
go1.18.windows-386.msi.sha256
go1.18.windows-amd64.msi
go1.18.windows-amd64.msi.sha256
go1.18.windows-arm64.msi
go1.18.windows-arm64.msi.sha256
`

func TestFakeSign(t *testing.T) {
	dir := t.TempDir()
	for _, f := range strings.Split(strings.TrimSpace(inputs), "\n") {
		if err := ioutil.WriteFile(filepath.Join(dir, f), []byte("hi"), 0777); err != nil {
			t.Fatal(err)
		}
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "ready"), nil, 0777); err != nil {
		t.Fatal(err)
	}
	fakeSignOnce(t, dir, map[string]bool{})
	want := map[string]bool{}
	for _, f := range strings.Split(strings.TrimSpace(outputs), "\n") {
		want[f] = true
	}
	got := map[string]bool{}
	files, err := ioutil.ReadDir(filepath.Join(dir, "signed"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		got[f.Name()] = true
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("signed outputs mismatch (-want +got):\n%v", diff)
	}
}

func fakeCDNLoad(ctx context.Context, t *testing.T, from, to string) {
	fromFS, toFS := gcsfs.DirFS(from), gcsfs.DirFS(to)
	seen := map[string]bool{}
	periodicallyDo(ctx, t, 100*time.Millisecond, func() error {
		files, err := fs.ReadDir(fromFS, ".")
		if err != nil {
			return err
		}
		for _, f := range files {
			if seen[f.Name()] {
				continue
			}
			seen[f.Name()] = true
			contents, err := fs.ReadFile(fromFS, f.Name())
			if err != nil {
				return err
			}
			if err := gcsfs.WriteFile(toFS, f.Name(), contents); err != nil {
				return err
			}
		}
		return nil
	})
}

func periodicallyDo(ctx context.Context, t *testing.T, period time.Duration, f func() error) {
	var err error
	childCtx, cancel := context.WithCancel(ctx)
	internal.PeriodicallyDo(childCtx, period, func(_ context.Context, _ time.Time) {
		err = f()
		if err != nil {
			cancel()
		}
	})
	// Suppress errors caused by the test finishing before we notice.
	if err != nil && ctx.Err() == nil {
		t.Fatal(err)
	}
}
