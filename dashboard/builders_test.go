// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dashboard

import (
	"strings"
	"testing"
	"time"
)

func TestOSARCHAccessors(t *testing.T) {
	valid := func(s string) bool { return s != "" && !strings.Contains(s, "-") }
	for _, conf := range Builders {
		os := conf.GOOS()
		arch := conf.GOARCH()
		osArch := os + "-" + arch
		if !valid(os) || !valid(arch) || !(conf.Name == osArch || strings.HasPrefix(conf.Name, osArch+"-")) {
			t.Errorf("OS+ARCH(%q) = %q, %q; invalid", conf.Name, os, arch)
		}
	}
}

func TestDistTestsExecTimeout(t *testing.T) {
	tests := []struct {
		c    *BuildConfig
		want time.Duration
	}{
		{
			&BuildConfig{
				env:          []string{},
				testHostConf: &HostConfig{},
			},
			20 * time.Minute,
		},
		{
			&BuildConfig{
				env:          []string{"GO_TEST_TIMEOUT_SCALE=2"},
				testHostConf: &HostConfig{},
			},
			40 * time.Minute,
		},
		{
			&BuildConfig{
				env: []string{},
				testHostConf: &HostConfig{
					env: []string{"GO_TEST_TIMEOUT_SCALE=3"},
				},
			},
			60 * time.Minute,
		},
		// BuildConfig's env takes precedence:
		{
			&BuildConfig{
				env: []string{"GO_TEST_TIMEOUT_SCALE=2"},
				testHostConf: &HostConfig{
					env: []string{"GO_TEST_TIMEOUT_SCALE=3"},
				},
			},
			40 * time.Minute,
		},
	}
	for i, tt := range tests {
		got := tt.c.DistTestsExecTimeout(nil)
		if got != tt.want {
			t.Errorf("%d. got %v; want %v", i, got, tt.want)
		}
	}
}

func TestListTrybots(t *testing.T) {
	forProj := func(proj string) {
		t.Run(proj, func(t *testing.T) {
			tryBots := TryBuildersForProject(proj)
			t.Logf("Builders:")
			for _, conf := range tryBots {
				t.Logf("  - %s", conf.Name)
			}
		})
	}
	forProj("go")
	forProj("net")
	forProj("sys")
}

func TestHostConfigsAllUsed(t *testing.T) {
	used := map[string]bool{}
	for _, conf := range Builders {
		used[conf.HostType] = true
	}
	for hostType := range Hosts {
		if !used[hostType] {
			// Currently host-linux-armhf-cross and host-linux-armel-cross aren't
			// referenced, but the coordinator hard-codes them, so don't make
			// this an error for now.
			t.Logf("warning: host type %q is not referenced from any build config", hostType)
		}
	}
}

func TestSubrepoTrybots(t *testing.T) {
	bc := Builders["linux-amd64"]

	for _, tt := range []struct {
		repo, branch, goBranch string
		want                   bool
	}{
		{"mobile", "master", "release-branch.go1.10", false},
		{"mobile", "master", "release-branch.go1.11", false},
		{"mobile", "master", "release-branch.go1.12", false}, // requires Go 1.13+
		{"mobile", "master", "release-branch.go1.13", true},
		{"mobile", "master", "master", true},

		{"net", "master", "release-branch.go1.10", false}, // too old
		{"net", "master", "release-branch.go1.11", true},
		{"net", "master", "release-branch.go1.12", true},
		{"net", "master", "release-branch.go1.13", true},
	} {
		got := bc.BuildBranch(tt.repo, tt.branch, tt.goBranch)
		if got != tt.want {
			t.Errorf("BuildBranch(%q, %q, %q) = %v; want %v", tt.repo, tt.branch, tt.goBranch, got, tt.want)
		}
	}
}

func TestBuildConfigBuildRepo(t *testing.T) {
	tests := []struct {
		builder string
		repo    string
		want    bool
	}{
		// The physical ARM Androids only run x/mobile (and "go"):
		{"android-arm-wiko-fever", "go", true},
		{"android-arm-wiko-fever", "mobile", true},
		{"android-arm64-wiko-fever", "mobile", true},
		{"android-arm64-wiko-fever", "net", false},

		// But the emulators run all:
		{"android-amd64-emu", "mobile", true},
		{"android-386-emu", "mobile", true},
		{"android-amd64-emu", "net", true},
		{"android-386-emu", "net", true},
		{"android-amd64-emu", "go", true},
		{"android-386-emu", "go", true},
	}
	for _, tt := range tests {
		bc, ok := Builders[tt.builder]
		if !ok {
			t.Fatalf("unknown builder %q", tt.builder)
		}
		got := bc.BuildRepo(tt.repo)
		if got != tt.want {
			t.Errorf("%s: BuildRepo(%q) = %v; want %v", tt.builder, tt.repo, got, tt.want)
		}
	}
}
