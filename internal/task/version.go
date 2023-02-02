package task

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/build/buildlet"
	"golang.org/x/build/gerrit"
	"golang.org/x/build/internal/workflow"
	"golang.org/x/net/context"
)

// VersionTasks contains tasks related to versioning the release.
type VersionTasks struct {
	Gerrit           GerritClient
	GerritURL        string
	GoProject        string
	CreateBuildlet   func(context.Context, string) (buildlet.RemoteClient, error)
	LatestGoBinaries func(context.Context) (string, error)
}

func (t *VersionTasks) GetCurrentMajor(ctx context.Context) (int, error) {
	_, currentMajor, err := t.tagInfo(ctx)
	return currentMajor, err
}

func (t *VersionTasks) tagInfo(ctx context.Context) (tags map[string]bool, currentMajor int, _ error) {
	tagList, err := t.Gerrit.ListTags(ctx, t.GoProject)
	if err != nil {
		return nil, 0, err
	}
	tags = map[string]bool{}
	for _, tag := range tagList {
		tags[tag] = true
	}
	// Find the most recently released major version.
	// Going down from a high number is convenient for testing.
	currentMajor = 100
	for ; ; currentMajor-- {
		if tags[fmt.Sprintf("go1.%d", currentMajor)] {
			break
		}
	}
	return tags, currentMajor, nil
}

// GetNextVersions returns the next for each of the given types of release.
func (t *VersionTasks) GetNextVersions(ctx context.Context, kinds []ReleaseKind) ([]string, error) {
	var next []string
	for _, k := range kinds {
		n, err := t.GetNextVersion(ctx, k)
		if err != nil {
			return nil, err
		}
		next = append(next, n)
	}
	return next, nil
}

// GetNextVersion returns the next for the given type of release.
func (t *VersionTasks) GetNextVersion(ctx context.Context, kind ReleaseKind) (string, error) {
	tags, currentMajor, err := t.tagInfo(ctx)
	if err != nil {
		return "", err
	}
	findUnused := func(v string) (string, error) {
		for {
			if !tags[v] {
				return v, nil
			}
			v, err = nextVersion(v)
			if err != nil {
				return "", err
			}
		}
	}
	switch kind {
	case KindCurrentMinor:
		return findUnused(fmt.Sprintf("go1.%d.1", currentMajor))
	case KindPrevMinor:
		return findUnused(fmt.Sprintf("go1.%d.1", currentMajor-1))
	case KindBeta:
		return findUnused(fmt.Sprintf("go1.%dbeta1", currentMajor+1))
	case KindRC:
		return findUnused(fmt.Sprintf("go1.%drc1", currentMajor+1))
	case KindMajor:
		return fmt.Sprintf("go1.%d", currentMajor+1), nil
	}
	return "", fmt.Errorf("unknown release kind %v", kind)
}

func nextVersion(version string) (string, error) {
	lastNonDigit := strings.LastIndexFunc(version, func(r rune) bool {
		return r < '0' || r > '9'
	})
	if lastNonDigit == -1 || len(version) == lastNonDigit {
		return "", fmt.Errorf("malformatted Go version %q", version)
	}
	n, err := strconv.Atoi(version[lastNonDigit+1:])
	if err != nil {
		return "", fmt.Errorf("malformatted Go version %q (%v)", version, err)
	}
	return fmt.Sprintf("%s%d", version[:lastNonDigit+1], n+1), nil
}

// CreateAutoSubmitVersionCL mails an auto-submit change to update VERSION on branch.
func (t *VersionTasks) CreateAutoSubmitVersionCL(ctx *workflow.TaskContext, branch string, reviewers []string, version string) (string, error) {
	return t.Gerrit.CreateAutoSubmitChange(ctx, gerrit.ChangeInput{
		Project: t.GoProject,
		Branch:  branch,
		Subject: fmt.Sprintf("[%v] %v", branch, version),
	}, reviewers, map[string]string{
		"VERSION": version,
	})
}

// AwaitCL waits for the specified CL to be submitted, and returns the new
// branch head. Callers can pass baseCommit, the current branch head, to verify
// that no CLs were submitted between when the CL was created and when it was
// merged. If changeID is blank because the intended CL was a no-op, baseCommit
// is returned immediately.
func (t *VersionTasks) AwaitCL(ctx *workflow.TaskContext, changeID, baseCommit string) (string, error) {
	if changeID == "" {
		ctx.Printf("No CL was necessary")
		return baseCommit, nil
	}

	ctx.Printf("Awaiting review/submit of %v", ChangeLink(changeID))
	return AwaitCondition(ctx, 10*time.Second, func() (string, bool, error) {
		return t.Gerrit.Submitted(ctx, changeID, baseCommit)
	})
}

// ReadBranchHead returns the current HEAD revision of branch.
func (t *VersionTasks) ReadBranchHead(ctx *workflow.TaskContext, branch string) (string, error) {
	return t.Gerrit.ReadBranchHead(ctx, t.GoProject, branch)
}

// TagRelease tags commit as version.
func (t *VersionTasks) TagRelease(ctx *workflow.TaskContext, version, commit string) error {
	return t.Gerrit.Tag(ctx, t.GoProject, version, commit)
}

func (t *VersionTasks) CreateUpdateStdlibIndexCL(ctx *workflow.TaskContext, reviewers []string, version string) (string, error) {
	var files = make(map[string]string) // Map key is relative path, and map value is file content.

	binaries, err := t.LatestGoBinaries(ctx)
	if err != nil {
		return "", err
	}
	bc, err := t.CreateBuildlet(ctx, "linux-amd64-longtest")
	if err != nil {
		return "", err
	}
	defer bc.Close()
	if err := bc.PutTarFromURL(ctx, binaries, ""); err != nil {
		return "", err
	}
	toolsTarURL := fmt.Sprintf("%s/%s/+archive/%s.tar.gz", t.GerritURL, "tools", "master")
	if err := bc.PutTarFromURL(ctx, toolsTarURL, "tools"); err != nil {
		return "", err
	}
	writer := &LogWriter{Logger: ctx}
	go writer.Run(ctx)
	remoteErr, execErr := bc.Exec(ctx, "go/bin/go", buildlet.ExecOpts{
		Dir:    "tools",
		Args:   []string{"generate", "./internal/imports"},
		Output: writer,
	})
	if execErr != nil {
		return "", fmt.Errorf("Exec failed: %v", execErr)
	}
	if remoteErr != nil {
		return "", fmt.Errorf("Command failed: %v", remoteErr)
	}
	tgz, err := bc.GetTar(context.Background(), "tools/internal/imports")
	if err != nil {
		return "", err
	}
	defer tgz.Close()
	tools, err := tgzToMap(tgz)
	if err != nil {
		return "", err
	}
	files["internal/imports/zstdlib.go"] = tools["zstdlib.go"]

	changeInput := gerrit.ChangeInput{
		Project: "tools",
		Subject: fmt.Sprintf("internal/imports: update stdlib index for %s\n\nFor golang/go#38706.", strings.Replace(version, "go", "Go ", 1)),
		Branch:  "master",
	}
	return t.Gerrit.CreateAutoSubmitChange(ctx, changeInput, reviewers, files)
}
