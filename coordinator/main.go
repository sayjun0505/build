// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build build_coordinator

// The coordinator runs on GCE and coordinates builds in Docker containers.
package main // import "golang.org/x/tools/dashboard/coordinator"

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/cloud/compute/metadata"
)

var (
	masterKeyFile = flag.String("masterkey", "", "Path to builder master key. Else fetched using GCE project attribute 'builder-master-key'.")
	maxBuilds     = flag.Int("maxbuilds", 6, "Max concurrent builds")

	cleanZones = flag.String("zones", "us-central1-a,us-central1-b,us-central1-f", "Comma-separated list of zones to periodically clean of stale build VMs (ones that failed to shut themselves down)")

	// Debug flags:
	addTemp = flag.Bool("temp", false, "Append -temp to all builders.")
	just    = flag.String("just", "", "If non-empty, run single build in the foreground. Requires rev.")
	rev     = flag.String("rev", "", "Revision to build.")
)

var (
	startTime = time.Now()
	builders  = map[string]buildConfig{} // populated once at startup
	watchers  = map[string]watchConfig{} // populated once at startup
	donec     = make(chan builderRev)    // reports of finished builders

	statusMu   sync.Mutex // guards both status (ongoing ones) and statusDone (just finished)
	status     = map[builderRev]*buildStatus{}
	statusDone []*buildStatus // finished recently, capped to maxStatusDone
)

const maxStatusDone = 30

type imageInfo struct {
	url string // of tar file

	mu      sync.Mutex
	lastMod string
}

var images = map[string]*imageInfo{
	"go-commit-watcher":          {url: "https://storage.googleapis.com/go-builder-data/docker-commit-watcher.tar.gz"},
	"gobuilders/linux-x86-base":  {url: "https://storage.googleapis.com/go-builder-data/docker-linux.base.tar.gz"},
	"gobuilders/linux-x86-clang": {url: "https://storage.googleapis.com/go-builder-data/docker-linux.clang.tar.gz"},
	"gobuilders/linux-x86-gccgo": {url: "https://storage.googleapis.com/go-builder-data/docker-linux.gccgo.tar.gz"},
	"gobuilders/linux-x86-nacl":  {url: "https://storage.googleapis.com/go-builder-data/docker-linux.nacl.tar.gz"},
	"gobuilders/linux-x86-sid":   {url: "https://storage.googleapis.com/go-builder-data/docker-linux.sid.tar.gz"},
}

type buildConfig struct {
	name    string   // "linux-amd64-race"
	image   string   // Docker image to use to build
	cmd     string   // optional -cmd flag (relative to go/src/)
	env     []string // extra environment ("key=value") pairs
	dashURL string   // url of the build dashboard
	tool    string   // the tool this configuration is for
}

type watchConfig struct {
	repo     string        // "https://go.googlesource.com/go"
	dash     string        // "https://build.golang.org/" (must end in /)
	interval time.Duration // Polling interval
}

func main() {
	flag.Parse()
	addBuilder(buildConfig{name: "linux-386"})
	addBuilder(buildConfig{name: "linux-386-387", env: []string{"GO386=387"}})
	addBuilder(buildConfig{name: "linux-amd64"})
	addBuilder(buildConfig{name: "linux-amd64-nocgo", env: []string{"CGO_ENABLED=0", "USER=root"}})
	addBuilder(buildConfig{name: "linux-amd64-noopt", env: []string{"GO_GCFLAGS=-N -l"}})
	addBuilder(buildConfig{name: "linux-amd64-race"})
	addBuilder(buildConfig{name: "nacl-386"})
	addBuilder(buildConfig{name: "nacl-amd64p32"})
	addBuilder(buildConfig{
		name:    "linux-amd64-gccgo",
		image:   "gobuilders/linux-x86-gccgo",
		cmd:     "make RUNTESTFLAGS=\"--target_board=unix/-m64\" check-go -j16",
		dashURL: "https://build.golang.org/gccgo",
		tool:    "gccgo",
	})
	addBuilder(buildConfig{
		name:    "linux-386-gccgo",
		image:   "gobuilders/linux-x86-gccgo",
		cmd:     "make RUNTESTFLAGS=\"--target_board=unix/-m32\" check-go -j16",
		dashURL: "https://build.golang.org/gccgo",
		tool:    "gccgo",
	})
	addBuilder(buildConfig{name: "linux-386-sid", image: "gobuilders/linux-x86-sid"})
	addBuilder(buildConfig{name: "linux-amd64-sid", image: "gobuilders/linux-x86-sid"})
	addBuilder(buildConfig{name: "linux-386-clang", image: "gobuilders/linux-x86-clang"})
	addBuilder(buildConfig{name: "linux-amd64-clang", image: "gobuilders/linux-x86-clang"})

	addWatcher(watchConfig{repo: "https://go.googlesource.com/go", dash: "https://build.golang.org/"})
	// TODO(adg,cmang): fix gccgo watcher
	// addWatcher(watchConfig{repo: "https://code.google.com/p/gofrontend", dash: "https://build.golang.org/gccgo/"})

	if (*just != "") != (*rev != "") {
		log.Fatalf("--just and --rev must be used together")
	}
	if *just != "" {
		conf, ok := builders[*just]
		if !ok {
			log.Fatalf("unknown builder %q", *just)
		}
		cmd := exec.Command("docker", append([]string{"run"}, conf.dockerRunArgs(*rev)...)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Build failed: %v", err)
		}
		return
	}

	http.HandleFunc("/", handleStatus)
	http.HandleFunc("/logs", handleLogs)
	go http.ListenAndServe(":80", nil)

	go cleanUpOldContainers()
	go cleanUpOldVMs()

	for _, watcher := range watchers {
		if err := startWatching(watchers[watcher.repo]); err != nil {
			log.Printf("Error starting watcher for %s: %v", watcher.repo, err)
		}
	}

	workc := make(chan builderRev)
	for name, builder := range builders {
		go findWorkLoop(name, builder.dashURL, workc)
	}

	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case work := <-workc:
			log.Printf("workc received %+v; len(status) = %v, maxBuilds = %v; cur = %p", work, len(status), *maxBuilds, status[work])
			mayBuild := mayBuildRev(work)
			if mayBuild {
				if numBuilds() > *maxBuilds {
					mayBuild = false
				}
			}
			if mayBuild {
				if st, err := startBuilding(builders[work.name], work.rev); err == nil {
					setStatus(work, st)
					log.Printf("%v now building in %v", work, st.container)
				} else {
					log.Printf("Error starting to build %v: %v", work, err)
				}
			}
		case done := <-donec:
			log.Printf("%v done", done)
			markDone(done)
		case <-ticker.C:
			if numCurrentBuilds() == 0 && time.Now().After(startTime.Add(10*time.Minute)) {
				// TODO: halt the whole machine to kill the VM or something
			}
		}
	}
}

func numCurrentBuilds() int {
	statusMu.Lock()
	defer statusMu.Unlock()
	return len(status)
}

func mayBuildRev(work builderRev) bool {
	statusMu.Lock()
	defer statusMu.Unlock()
	return len(status) < *maxBuilds && status[work] == nil
}

func setStatus(work builderRev, st *buildStatus) {
	statusMu.Lock()
	defer statusMu.Unlock()
	status[work] = st
}

func markDone(work builderRev) {
	statusMu.Lock()
	defer statusMu.Unlock()
	st, ok := status[work]
	if !ok {
		return
	}
	delete(status, work)
	if len(statusDone) == maxStatusDone {
		copy(statusDone, statusDone[1:])
		statusDone = statusDone[:len(statusDone)-1]
	}
	statusDone = append(statusDone, st)
}

func getStatus(work builderRev) *buildStatus {
	statusMu.Lock()
	defer statusMu.Unlock()
	if st, ok := status[work]; ok {
		return st
	}
	for _, st := range statusDone {
		if st.builderRev == work {
			return st
		}
	}
	return nil
}

type byAge []*buildStatus

func (s byAge) Len() int           { return len(s) }
func (s byAge) Less(i, j int) bool { return s[i].start.Before(s[j].start) }
func (s byAge) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func handleStatus(w http.ResponseWriter, r *http.Request) {
	var active []*buildStatus
	var recent []*buildStatus
	statusMu.Lock()
	for _, st := range status {
		active = append(active, st)
	}
	recent = append(recent, statusDone...)
	statusMu.Unlock()

	sort.Sort(byAge(active))
	sort.Sort(byAge(recent))

	io.WriteString(w, "<html><body><h1>Go build coordinator</h1>")

	fmt.Fprintf(w, "<h2>running</h2>%d of max %d builds running:<p><pre>", len(status), *maxBuilds)
	for _, st := range active {
		io.WriteString(w, st.htmlStatusLine())
	}
	io.WriteString(w, "</pre>")

	io.WriteString(w, "<h2>recently completed</h2><pre>")
	for _, st := range recent {
		io.WriteString(w, st.htmlStatusLine())
	}
	io.WriteString(w, "</pre>")

	fmt.Fprintf(w, "<h2>disk space</h2><pre>%s</pre></body></html>", html.EscapeString(diskFree()))
}

func diskFree() string {
	out, _ := exec.Command("df", "-h").Output()
	return string(out)
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	st := getStatus(builderRev{r.FormValue("name"), r.FormValue("rev")})
	if st == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, st.logs())
	// TODO: if st is still building, stream them to the user with
	// http.Flusher.Flush and CloseNotifier and registering interest
	// of new writes with the buildStatus. Will require moving the
	// BUILDERKEY scrubbing into the Write method.
}

func findWorkLoop(builderName, dashURL string, work chan<- builderRev) {
	// TODO: make this better
	for {
		rev, err := findWork(builderName, dashURL)
		if err != nil {
			log.Printf("Finding work for %s: %v", builderName, err)
		} else if rev != "" {
			work <- builderRev{builderName, rev}
		}
		time.Sleep(60 * time.Second)
	}
}

func findWork(builderName, dashURL string) (rev string, err error) {
	var jres struct {
		Response struct {
			Kind string
			Data struct {
				Hash        string
				PerfResults []string
			}
		}
	}
	res, err := http.Get(dashURL + "/todo?builder=" + builderName + "&kind=build-go-commit")
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("unexpected http status %d", res.StatusCode)
	}
	err = json.NewDecoder(res.Body).Decode(&jres)
	if jres.Response.Kind == "build-go-commit" {
		rev = jres.Response.Data.Hash
	}
	return rev, err
}

// builderRev is a build configuration type and a revision.
type builderRev struct {
	name string // e.g. "linux-amd64-race"
	rev  string // lowercase hex git hash
}

// returns the part after "docker run"
func (conf buildConfig) dockerRunArgs(rev string) (args []string) {
	if key := builderKey(conf.name); key != "" {
		tmpKey := "/tmp/" + conf.name + ".buildkey"
		if _, err := os.Stat(tmpKey); err != nil {
			if err := ioutil.WriteFile(tmpKey, []byte(key), 0600); err != nil {
				log.Fatal(err)
			}
		}
		// Images may look for .gobuildkey in / or /root, so provide both.
		// TODO(adg): fix images that look in the wrong place.
		args = append(args, "-v", tmpKey+":/.gobuildkey")
		args = append(args, "-v", tmpKey+":/root/.gobuildkey")
	}
	for _, pair := range conf.env {
		args = append(args, "-e", pair)
	}
	args = append(args,
		conf.image,
		"/usr/local/bin/builder",
		"-rev="+rev,
		"-dashboard="+conf.dashURL,
		"-tool="+conf.tool,
		"-buildroot=/",
		"-v",
	)
	if conf.cmd != "" {
		args = append(args, "-cmd", conf.cmd)
	}
	args = append(args, conf.name)
	return
}

func addBuilder(c buildConfig) {
	if c.name == "" {
		panic("empty name")
	}
	if *addTemp {
		c.name += "-temp"
	}
	if _, dup := builders[c.name]; dup {
		panic("dup name")
	}
	if c.dashURL == "" {
		c.dashURL = "https://build.golang.org"
	}
	if c.tool == "" {
		c.tool = "go"
	}

	if strings.HasPrefix(c.name, "nacl-") {
		if c.image == "" {
			c.image = "gobuilders/linux-x86-nacl"
		}
		if c.cmd == "" {
			c.cmd = "/usr/local/bin/build-command.pl"
		}
	}
	if strings.HasPrefix(c.name, "linux-") && c.image == "" {
		c.image = "gobuilders/linux-x86-base"
	}
	if c.image == "" {
		panic("empty image")
	}
	builders[c.name] = c
}

// returns the part after "docker run"
func (conf watchConfig) dockerRunArgs() (args []string) {
	log.Printf("Running watcher with master key %q", masterKey())
	if key := masterKey(); len(key) > 0 {
		tmpKey := "/tmp/watcher.buildkey"
		if _, err := os.Stat(tmpKey); err != nil {
			if err := ioutil.WriteFile(tmpKey, key, 0600); err != nil {
				log.Fatal(err)
			}
		}
		// Images may look for .gobuildkey in / or /root, so provide both.
		// TODO(adg): fix images that look in the wrong place.
		args = append(args, "-v", tmpKey+":/.gobuildkey")
		args = append(args, "-v", tmpKey+":/root/.gobuildkey")
	}
	args = append(args,
		"go-commit-watcher",
		"/usr/local/bin/watcher",
		"-repo="+conf.repo,
		"-dash="+conf.dash,
		"-poll="+conf.interval.String(),
	)
	return
}

func addWatcher(c watchConfig) {
	if c.repo == "" {
		c.repo = "https://go.googlesource.com/go"
	}
	if c.dash == "" {
		c.dash = "https://build.golang.org/"
	}
	if c.interval == 0 {
		c.interval = 10 * time.Second
	}
	watchers[c.repo] = c
}

func condUpdateImage(img string) error {
	ii := images[img]
	if ii == nil {
		log.Fatalf("Image %q not described.", img)
	}
	ii.mu.Lock()
	defer ii.mu.Unlock()
	res, err := http.Head(ii.url)
	if err != nil {
		return fmt.Errorf("Error checking %s: %v", ii.url, err)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Error checking %s: %v", ii.url, res.Status)
	}
	if res.Header.Get("Last-Modified") == ii.lastMod {
		return nil
	}

	res, err = http.Get(ii.url)
	if err != nil || res.StatusCode != 200 {
		return fmt.Errorf("Get after Head failed for %s: %v, %v", ii.url, err, res)
	}
	defer res.Body.Close()

	log.Printf("Running: docker load of %s\n", ii.url)
	cmd := exec.Command("docker", "load")
	cmd.Stdin = res.Body

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if cmd.Run(); err != nil {
		log.Printf("Failed to pull latest %s from %s and pipe into docker load: %v, %s", img, ii.url, err, out.Bytes())
		return err
	}
	ii.lastMod = res.Header.Get("Last-Modified")
	return nil
}

// numBuilds finds the number of go builder instances currently running.
func numBuilds() int {
	out, _ := exec.Command("docker", "ps").Output()
	numBuilds := 0
	ps := bytes.Split(out, []byte("\n"))
	for _, p := range ps {
		if bytes.HasPrefix(p, []byte("gobuilders/")) {
			numBuilds++
		}
	}
	log.Printf("num current docker builds: %d", numBuilds)
	return numBuilds
}

func startBuilding(conf buildConfig, rev string) (*buildStatus, error) {
	if err := condUpdateImage(conf.image); err != nil {
		log.Printf("Failed to setup container for %v %v: %v", conf.name, rev, err)
		return nil, err
	}

	cmd := exec.Command("docker", append([]string{"run", "-d"}, conf.dockerRunArgs(rev)...)...)
	all, err := cmd.CombinedOutput()
	log.Printf("Docker run for %v %v = err:%v, output:%s", conf.name, rev, err, all)
	if err != nil {
		return nil, err
	}
	container := strings.TrimSpace(string(all))
	bs := &buildStatus{
		builderRev: builderRev{
			name: conf.name,
			rev:  rev,
		},
		container: container,
		start:     time.Now(),
	}
	go func() {
		all, err := exec.Command("docker", "wait", container).CombinedOutput()
		output := strings.TrimSpace(string(all))
		var ok bool
		if err == nil {
			exit, err := strconv.Atoi(output)
			ok = (err == nil && exit == 0)
		}
		bs.setDone(ok)
		log.Printf("docker wait %s/%s: %v, %s", container, rev, err, output)
		donec <- builderRev{conf.name, rev}
		exec.Command("docker", "rm", container).Run()
	}()
	go func() {
		cmd := exec.Command("docker", "logs", "-f", container)
		cmd.Stdout = bs
		cmd.Stderr = bs
		if err := cmd.Run(); err != nil {
			// The docker logs subcommand always returns
			// success, even if the underlying process
			// fails.
			log.Printf("failed to follow docker logs of %s: %v", container, err)
		}
	}()
	return bs, nil
}

// buildStatus is the status of a build.
type buildStatus struct {
	// Immutable:
	builderRev
	start     time.Time
	container string // container ID for docker, else it's a VM

	mu        sync.Mutex   // guards following
	done      time.Time    // finished running
	succeeded bool         // set when done
	output    bytes.Buffer // stdout and stderr
}

func (st *buildStatus) setDone(succeeded bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.succeeded = succeeded
	st.done = time.Now()
}

// htmlStatusLine returns the HTML to show within the <pre> block on
// the main page's list of active builds.
func (st *buildStatus) htmlStatusLine() string {
	st.mu.Lock()
	defer st.mu.Unlock()

	urlPrefix := "https://go-review.googlesource.com/#/q/"
	if strings.Contains(st.name, "gccgo") {
		urlPrefix = "https://code.google.com/p/gofrontend/source/detail?r="
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "<a href='https://github.com/golang/go/wiki/DashboardBuilders'>%s</a> rev <a href='%s%s'>%s</a>",
		st.name, urlPrefix, st.rev, st.rev)

	if st.done.IsZero() {
		buf.WriteString(", running")
	} else if st.succeeded {
		buf.WriteString(", succeeded")
	} else {
		buf.WriteString(", failed")
	}

	if st.container != "" {
		fmt.Fprintf(&buf, " in container <a href='/logs?name=%s&rev=%s'>%s</a>", st.name, st.rev, st.container)
	}

	t := st.done
	if t.IsZero() {
		t = st.start
	}
	fmt.Fprintf(&buf, ", %v ago\n", time.Since(t))
	return buf.String()
}

func (st *buildStatus) logs() string {
	st.mu.Lock()
	logs := st.output.String()
	st.mu.Unlock()
	key := builderKey(st.name)
	return strings.Replace(string(logs), key, "BUILDERKEY", -1)
}

func (st *buildStatus) Write(p []byte) (n int, err error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	const maxBufferSize = 2 << 20 // 2MB of output is way more than we expect.
	if st.output.Len()+len(p) > maxBufferSize {
		p = p[:maxBufferSize-st.output.Len()]
	}
	return st.output.Write(p)
}

func startWatching(conf watchConfig) (err error) {
	defer func() {
		if err != nil {
			restartWatcherSoon(conf)
		}
	}()
	log.Printf("Starting watcher for %v", conf.repo)
	if err := condUpdateImage("go-commit-watcher"); err != nil {
		log.Printf("Failed to setup container for commit watcher: %v", err)
		return err
	}

	cmd := exec.Command("docker", append([]string{"run", "-d"}, conf.dockerRunArgs()...)...)
	all, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Docker run for commit watcher = err:%v, output: %s", err, all)
		return err
	}
	container := strings.TrimSpace(string(all))
	// Start a goroutine to wait for the watcher to die.
	go func() {
		exec.Command("docker", "wait", container).Run()
		exec.Command("docker", "rm", "-v", container).Run()
		log.Printf("Watcher crashed. Restarting soon.")
		restartWatcherSoon(conf)
	}()
	return nil
}

func restartWatcherSoon(conf watchConfig) {
	time.AfterFunc(30*time.Second, func() {
		startWatching(conf)
	})
}

func builderKey(builder string) string {
	master := masterKey()
	if len(master) == 0 {
		return ""
	}
	h := hmac.New(md5.New, master)
	io.WriteString(h, builder)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func masterKey() []byte {
	keyOnce.Do(loadKey)
	return masterKeyCache
}

var (
	keyOnce        sync.Once
	masterKeyCache []byte
)

func loadKey() {
	if *masterKeyFile != "" {
		b, err := ioutil.ReadFile(*masterKeyFile)
		if err != nil {
			log.Fatal(err)
		}
		masterKeyCache = bytes.TrimSpace(b)
		return
	}
	req, _ := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/project/attributes/builder-master-key", nil)
	req.Header.Set("Metadata-Flavor", "Google")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("No builder master key available")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("No builder-master-key project attribute available.")
	}
	slurp, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	masterKeyCache = bytes.TrimSpace(slurp)
}

func cleanUpOldContainers() {
	for {
		for _, cid := range oldContainers() {
			log.Printf("Cleaning old container %v", cid)
			exec.Command("docker", "rm", "-v", cid).Run()
		}
		time.Sleep(30 * time.Second)
	}
}

func oldContainers() []string {
	out, _ := exec.Command("docker", "ps", "-a", "--filter=status=exited", "--no-trunc", "-q").Output()
	return strings.Fields(string(out))
}

// cleanUpOldVMs loops forever and periodically enumerates virtual
// machines and deletes those which have expired.
//
// A VM is considered expired if it has a "delete-at" metadata
// attribute having a unix timestamp before the current time.
//
// This is the safety mechanism to delete VMs which stray from the
// normal deleting process. VMs are created to run a single build and
// should be shut down by a controlling process. Due to various types
// of failures, they might get stranded. To prevent them from getting
// stranded and wasting resources forever, we instead set the
// "delete-at" metadata attribute on them when created to some time
// that's well beyond their expected lifetime.
func cleanUpOldVMs() {
	if !hasComputeScope() {
		log.Printf("The coordinator is not running with access to read and write Compute resources. Background VM cleaning disabled.")
		return
	}
	ts := google.ComputeTokenSource("default")
	computeService, _ := compute.New(oauth2.NewClient(oauth2.NoContext, ts))
	for {
		for _, zone := range strings.Split(*cleanZones, ",") {
			zone = strings.TrimSpace(zone)
			if err := cleanZoneVMs(computeService, zone); err != nil {
				log.Printf("Error cleaning VMs in zone %q: %v", zone, err)
			}
		}
		time.Sleep(time.Minute)
	}
}

// cleanZoneVMs is part of cleanUpOldVMs, operating on a single zone.
func cleanZoneVMs(svc *compute.Service, zone string) error {
	proj, err := metadata.ProjectID()
	if err != nil {
		return fmt.Errorf("failed to get current GCE ProjectID: %v", err)
	}
	// Fetch the first 500 (default) running instances and clean
	// thoes. We expect that we'll be running many fewer than
	// that. Even if we have more, eventually the first 500 will
	// either end or be cleaned, and then the next call will get a
	// partially-different 500.
	// TODO(bradfitz): revist this code if we ever start running
	// thousands of VMs.
	list, err := svc.Instances.List(proj, zone).Do()
	if err != nil {
		return fmt.Errorf("listing instances: %v", err)
	}
	for _, inst := range list.Items {
		if inst.Metadata == nil {
			// Defensive. Not seen in practice.
			continue
		}
		for _, it := range inst.Metadata.Items {
			if it.Key == "delete-at" {
				unixDeadline, err := strconv.ParseInt(it.Value, 10, 64)
				if err != nil {
					log.Printf("invalid delete-at value %q seen; ignoring", it.Value)
				}
				if err == nil && time.Now().Unix() > unixDeadline {
					log.Printf("Deleting expired VM %q in zone %q ...", inst.Name, zone)
					deleteVM(svc, zone, inst.Name)
				}
			}
		}
	}
	return nil
}

func deleteVM(svc *compute.Service, zone, instName string) {
	proj, err := metadata.ProjectID()
	if err != nil {
		log.Printf("failed to get project id to delete instace: %v", err)
		return
	}
	op, err := svc.Instances.Delete(proj, zone, instName).Do()
	if err != nil {
		log.Printf("Failed to delete instance %q in zone %q: %v", instName, zone, err)
		return
	}
	log.Printf("Sent request to delete instance %q in zone %q. Operation ID == %v", instName, zone, op.Id)
}

func hasComputeScope() bool {
	if !metadata.OnGCE() {
		return false
	}
	scopes, err := metadata.Scopes("default")
	if err != nil {
		log.Printf("failed to query metadata default scopes: %v", err)
		return false
	}
	for _, v := range scopes {
		if v == compute.DevstorageFull_controlScope {
			return true
		}
	}
	return false
}
