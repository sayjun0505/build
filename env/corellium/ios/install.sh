#!/bin/sh
# Copyright 2022 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.


# install.sh sets up a newly created Corellium iPhone device.
# Set HOST to root@<ip> where <ip> is the device ssh
# address.
#
# Put a builder key in `buildkey`.
#
# Use `bootstrap.bash` from the Go standard distribution and build a
# ios/arm64 bootstrap toolchain with cgo enabled and the compiler set
# to the clang wrapper from $GOROOT/misc/ios:
#
# 	GOOS=ios GOARCH=arm64 CGO_ENABLED=1 CC_FOR_TARGET=$(pwd)/../misc/ios/clangwrap.sh ./bootstrap.bash
#
# Put it in `go-ios-arm64-bootstrap.tbz`.
#
# Finally, install.sh assumes an iPhone SDK in `iPhoneOS.sdk`.

ios() {
	ssh "$HOST" "$@"
}

ios apt-get update
ios apt install -y --allow-unauthenticated git tmux rsync org.coolstar.iostoolchain ld64 com.linusyang.localeutf8

# Run builder at boot.
scp files/org.golang.builder.plist $HOST:/Library/LaunchDaemons/
ios launchctl load -w /Library/LaunchDaemons/org.golang.builder.plist
scp files/builder.sh $HOST:/var/root

scp go-ios-arm64-bootstrap.tbz $HOST:/var/root
ios tar xjf go-ios-arm64-bootstrap.tbz
scp buildkey $HOST:/var/root/.gobuildkey-host-ios-arm64-corellium-ios
scp files/profile $HOST:/var/root/.profile
rsync -va iPhoneOS.sdk $HOST:/var/root/

# Dummy sign Go bootstrap toolchain.
ios find go-ios-arm64-bootstrap -executable -type f| ios xargs -n1 ldid -S

ios mkdir -p /var/root/bin

# Build wrappers on the host.
CGO_ENABLED=1 GOOS=ios CC=$(go env GOROOT)/misc/ios/clangwrap.sh GOARCH=arm64 go build -o clangwrap -ldflags="-X main.sdkpath=/var/root/iPhoneOS.sdk" files/clangwrap.go
CGO_ENABLED=1 GOOS=ios CC=$(go env GOROOT)/misc/ios/clangwrap.sh GOARCH=arm64 go build -o arwrap files/arwrap.go
scp arwrap $HOST:/var/root/bin/ar
scp clangwrap $HOST:/var/root/bin/clangwrap
# Dummy sign them.
ios ldid -S /var/root/bin/clangwrap
ios ldid -S /var/root/bin/ar
ios reboot
