// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dashboard

import (
	"fmt"
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

// TestTrybots tests that a given repo & its branch yields the provided
// complete set of builders. See also: TestBuilders, which tests both trybots
// and post-submit builders, both at arbitrary branches.
func TestTrybots(t *testing.T) {
	tests := []struct {
		repo   string // "go", "net", etc
		branch string // of repo
		want   []string
	}{
		{
			repo:   "go",
			branch: "master",
			want: []string{
				"freebsd-amd64-12_0",
				"js-wasm",
				"linux-386",
				"linux-amd64",
				"linux-amd64-race",
				"misc-compile",
				"misc-compile-freebsd",
				"misc-compile-mips",
				"misc-compile-nacl",
				"misc-compile-netbsd",
				"misc-compile-openbsd",
				"misc-compile-plan9",
				"misc-compile-ppc",
				"misc-vet-vetall",
				"nacl-386",
				"nacl-amd64p32",
				"openbsd-amd64-64",
				"windows-386-2008",
				"windows-amd64-2016",
			},
		},
		{
			repo:   "go",
			branch: "release-branch.go1.12",
			want: []string{
				"freebsd-amd64-10_3",
				"freebsd-amd64-12_0",
				"js-wasm",
				"linux-386",
				"linux-amd64",
				"linux-amd64-race",
				"misc-compile",
				"misc-compile-freebsd",
				"misc-compile-mips",
				"misc-compile-nacl",
				"misc-compile-netbsd",
				"misc-compile-openbsd",
				"misc-compile-plan9",
				"misc-compile-ppc",
				"misc-vet-vetall",
				"nacl-386",
				"nacl-amd64p32",
				"openbsd-amd64-64",
				"windows-386-2008",
				"windows-amd64-2016",
			},
		},
		{
			repo:   "mobile",
			branch: "master",
			want: []string{
				"android-amd64-emu",
				"linux-amd64-androidemu",
			},
		},
		{
			repo:   "sys",
			branch: "master",
			want: []string{
				"freebsd-386-11_2",
				"freebsd-amd64-11_2",
				"freebsd-amd64-12_0",
				"linux-386",
				"linux-amd64",
				"netbsd-amd64-8_0",
				"openbsd-386-64",
				"openbsd-amd64-64",
				"windows-386-2008",
				"windows-amd64-2016",
			},
		},
	}
	for i, tt := range tests {
		if tt.branch == "" || tt.repo == "" {
			t.Errorf("incomplete test entry %d", i)
			return
		}
		t.Run(fmt.Sprintf("%s/%s", tt.repo, tt.branch), func(t *testing.T) {
			var got []string
			goBranch := tt.branch // hard-code the common case for now
			for _, bc := range TryBuildersForProject(tt.repo, tt.branch, goBranch) {
				got = append(got, bc.Name)
			}
			m := map[string]bool{}
			for _, b := range tt.want {
				m[b] = true
			}
			for _, b := range got {
				if _, ok := m[b]; !ok {
					t.Errorf("got unexpected %q", b)
				}
				delete(m, b)
			}
			for b := range m {
				t.Errorf("missing expected %q", b)
			}
		})
	}
}

// TestBuilderConfig whether a given builder and repo at different
// branches is either a post-submit builder, trybot, neither, or both.
func TestBuilderConfig(t *testing.T) {
	// builderConfigWant is bitmask of 4 different things to assert are wanted:
	// - being a post-submit builder
	// - NOT being a post-submit builder
	// - being a trybot builder
	// - NOT being a post-submit builder
	type want uint8
	const (
		isTrybot want = 1 << iota
		notTrybot
		isBuilder  // post-submit
		notBuilder // not post-submit

		none     = notTrybot + notBuilder
		both     = isTrybot + isBuilder
		onlyPost = notTrybot + isBuilder
	)

	type builderAndRepo struct {
		testName string
		builder  string
		repo     string
		branch   string
		goBranch string
	}
	// builder may end in "@go1.N" (as alias for "@release-branch.go1.N") or "@branch-name".
	// repo may end in "@1.N" (as alias for "@release-branch.go1.N")
	b := func(builder, repo string) builderAndRepo {
		br := builderAndRepo{
			testName: builder + "," + repo,
			builder:  builder,
			goBranch: "master",
			repo:     repo,
			branch:   "master",
		}
		if strings.Contains(builder, "@") {
			f := strings.SplitN(builder, "@", 2)
			br.builder = f[0]
			br.goBranch = f[1]
		}
		if strings.Contains(repo, "@") {
			f := strings.SplitN(repo, "@", 2)
			br.repo = f[0]
			br.branch = f[1]
		}
		expandBranch := func(s *string) {
			if strings.HasPrefix(*s, "go1.") {
				*s = "release-branch." + *s
			} else if strings.HasPrefix(*s, "1.") {
				*s = "release-branch.go" + *s
			}
		}
		expandBranch(&br.branch)
		expandBranch(&br.goBranch)
		if br.repo == "go" {
			br.branch = br.goBranch
		}
		return br
	}
	tests := []struct {
		br   builderAndRepo
		want want
	}{
		{b("linux-amd64", "go"), both},
		{b("linux-amd64", "net"), both},
		{b("linux-amd64", "sys"), both},

		// Don't test all subrepos on all the builders.
		{b("linux-amd64-ssacheck", "net"), none},
		{b("linux-amd64-ssacheck@go1.10", "net"), none},
		{b("linux-amd64-noopt@go1.11", "net"), none},
		{b("linux-386-387@go1.11", "net"), none},
		{b("linux-386-387@go1.11", "go"), onlyPost},
		{b("linux-386-387", "crypto"), onlyPost},
		{b("linux-arm-arm5spacemonkey@go1.11", "net"), none},
		{b("linux-arm-arm5spacemonkey@go1.12", "net"), none},

		// The mobile repo requires Go 1.13+.
		{b("android-amd64-emu", "go"), onlyPost},
		{b("android-amd64-emu", "mobile"), both},
		{b("android-amd64-emu", "mobile@1.10"), none},
		{b("android-amd64-emu", "mobile@1.11"), none},
		{b("android-amd64-emu@go1.10", "mobile"), none},
		{b("android-amd64-emu@go1.11", "mobile"), none},
		{b("android-amd64-emu@go1.12", "mobile"), none},
		{b("android-amd64-emu@go1.13", "mobile"), both},
		{b("android-amd64-emu", "mobile@1.13"), both},

		{b("android-386-emu", "go"), onlyPost},
		{b("android-386-emu", "mobile"), onlyPost},
		{b("android-386-emu", "mobile@1.10"), none},
		{b("android-386-emu", "mobile@1.11"), none},
		{b("android-386-emu@go1.10", "mobile"), none},
		{b("android-386-emu@go1.11", "mobile"), none},
		{b("android-386-emu@go1.12", "mobile"), none},
		{b("android-386-emu@go1.13", "mobile"), onlyPost},
		{b("android-386-emu", "mobile@1.13"), onlyPost},

		{b("linux-amd64", "net"), both},
		{b("linux-amd64", "net@1.12"), both},
		{b("linux-amd64@go1.12", "net@1.12"), both},
		{b("linux-amd64", "net@1.11"), both},
		{b("linux-amd64", "net@1.11"), both},
		{b("linux-amd64", "net@1.10"), none},   // too old
		{b("linux-amd64@go1.10", "net"), none}, // too old
		{b("linux-amd64@go1.11", "net"), both},
		{b("linux-amd64@go1.11", "net@1.11"), both},
		{b("linux-amd64@go1.12", "net@1.12"), both},

		// go1.12.html: "Go 1.12 is the last release that is
		// supported on FreeBSD 10.x [... and 11.1]"
		{b("freebsd-386-10_3", "go"), none},
		{b("freebsd-386-10_3", "net"), none},
		{b("freebsd-amd64-10_3", "go"), none},
		{b("freebsd-amd64-10_3", "net"), none},
		{b("freebsd-amd64-11_1", "go"), none},
		{b("freebsd-amd64-11_1", "net"), none},
		{b("freebsd-amd64-10_3@go1.12", "go"), both},
		{b("freebsd-amd64-10_3@go1.12", "net@1.12"), both},
		{b("freebsd-amd64-10_3@go1.11", "go"), both},
		{b("freebsd-amd64-10_3@go1.11", "net@1.11"), both},
		{b("freebsd-amd64-11_1@go1.13", "go"), none},
		{b("freebsd-amd64-11_1@go1.13", "net@1.12"), none},
		{b("freebsd-amd64-11_1@go1.12", "go"), isBuilder},
		{b("freebsd-amd64-11_1@go1.12", "net@1.12"), isBuilder},
		{b("freebsd-amd64-11_1@go1.11", "go"), isBuilder},
		{b("freebsd-amd64-11_1@go1.11", "net@1.11"), isBuilder},

		{b("linux-amd64-nocgo", "mobile"), none},

		// The physical ARM Androids only runs "go":
		// They run on GOOS=android mode which is not
		// interesting for x/mobile. The interesting tests run
		// on the darwin-amd64-wikofever below where
		// GOOS=darwin.
		{b("android-arm-wikofever", "go"), isBuilder},
		{b("android-arm-wikofever", "mobile"), notBuilder},
		{b("android-arm64-wikofever", "go"), isBuilder},
		{b("android-arm64-wikofever", "mobile"), notBuilder},
		{b("android-arm64-wikofever", "net"), notBuilder},

		// A GOOS=darwin variant of the physical ARM Androids
		// runs x/mobile and nothing else:
		{b("darwin-amd64-wikofever", "mobile"), isBuilder},
		{b("darwin-amd64-wikofever", "go"), notBuilder},
		{b("darwin-amd64-wikofever", "net"), notBuilder},

		// But the emulators run all:
		{b("android-amd64-emu", "mobile"), isBuilder},
		{b("android-386-emu", "mobile"), isBuilder},
		{b("android-amd64-emu", "net"), isBuilder},
		{b("android-386-emu", "net"), isBuilder},
		{b("android-amd64-emu", "go"), isBuilder},
		{b("android-386-emu", "go"), isBuilder},

		{b("nacl-386", "go"), both},
		{b("nacl-386", "net"), none},
		{b("nacl-amd64p32", "go"), both},
		{b("nacl-amd64p32", "net"), none},

		// Only test tip for js/wasm:
		{b("js-wasm", "go"), both},
		{b("js-wasm", "net"), onlyPost},
		{b("js-wasm@go1.12", "net"), none},
	}
	for _, tt := range tests {
		t.Run(tt.br.testName, func(t *testing.T) {
			bc, ok := Builders[tt.br.builder]
			if !ok {
				t.Fatalf("unknown builder %q", tt.br.builder)
			}
			gotPost := bc.BuildsRepoPostSubmit(tt.br.repo, tt.br.branch, tt.br.goBranch)
			if tt.want&isBuilder != 0 && !gotPost {
				t.Errorf("not a post-submit builder, but expected")
			}
			if tt.want&notBuilder != 0 && gotPost {
				t.Errorf("unexpectedly a post-submit builder")
			}

			gotTry := bc.BuildsRepoTryBot(tt.br.repo, tt.br.branch, tt.br.goBranch)
			if tt.want&isTrybot != 0 && !gotTry {
				t.Errorf("not trybot, but expected")
			}
			if tt.want&notTrybot != 0 && gotTry {
				t.Errorf("unexpectedly a trybot")
			}

			if t.Failed() {
				t.Logf("For: %+v", tt.br)
			}
		})
	}
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
