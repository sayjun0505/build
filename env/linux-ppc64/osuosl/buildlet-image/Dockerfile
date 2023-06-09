# Copyright 2019 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# build locally from debian-sid.sh
FROM murp/debian:sid-ppc64

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && \
    apt-get install --yes \
          gcc curl strace \
          ca-certificates netbase \
          procps lsof psmisc \
          libc6-dev gdb \
          openssh-server

RUN mkdir /usr/local/go-bootstrap && \
    curl --silent https://storage.googleapis.com/go-builder-data/gobootstrap-linux-ppc64.tar.gz | \
    tar -C /usr/local/go-bootstrap -zxv

ENV GOROOT_BOOTSTRAP /usr/local/go-bootstrap
RUN curl -o  /usr/local/bin/stage0 https://storage.googleapis.com/go-builder-data/buildlet-stage0.linux-ppc64 && \
    chmod +x /usr/local/bin/stage0

ENV GO_BUILDER_ENV host-linux-ppc64-osu

ENV GO_BUILD_KEY_DELETE_AFTER_READ true
ENV GO_BUILD_KEY_PATH /buildkey/gobuildkey

CMD ["/usr/local/bin/stage0"]
