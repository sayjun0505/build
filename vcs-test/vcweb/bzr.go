// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux
// +build linux

package main

import (
	"net/http"
	"path/filepath"
)

func bzrHandler() http.Handler {
	return http.StripPrefix("/bzr/", http.FileServer(http.Dir(filepath.Join(*dir, "bzr"))))
}
