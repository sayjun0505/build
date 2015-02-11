// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// The demo command shows and tests usage of the gerrit package.
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/build/gerrit"
)

func main() {
	gobotPass, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), "keys", "gobot-golang-org.cookie"))
	if err != nil {
		log.Fatal(err)
	}
	c := gerrit.NewClient("https://go-review.googlesource.com/a",
		gerrit.BasicAuth("git-gobot.golang.org", strings.TrimSpace(string(gobotPass))))
	cl, err := c.QueryChanges("label:TryBot=1 label:TryBot-Result=0 project:go status:open", gerrit.QueryChangesOpt{
		Fields: []string{"CURRENT_REVISION"},
	})
	if err != nil {
		log.Fatal(err)
	}
	v, _ := json.MarshalIndent(cl, "", "  ")
	os.Stdout.Write(v)
}
