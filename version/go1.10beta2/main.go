// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The go1.10beta2 command runs the go command from go1.10beta2.
//
// To install, run:
//
//     $ go get golang.org/x/build/version/go1.10beta2
//     $ go1.10beta2 download
//
// And then use the go1.10beta2 command as if it were your normal go
// command.
//
// See the release notes at https://beta.golang.org/doc/go1.10
//
// File bugs at https://golang.org/issues/new
package main

import "golang.org/x/build/version"

func main() {
	version.Run("go1.10beta2")
}
