# Copyright 2022 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

IMAGE_NAME=$(shell basename $(CURDIR))
PROD_REPO=gcr.io/symbolic-datum-552

usage:
	echo "Use prod or dev targets. For dev, specify your Docker repository with the REPO=foo argument." ; exit 1

prod: Dockerfile
	sed 's|{{REPO}}|'"$(PROD_REPO)"'|g' Dockerfile > Dockerfile.make
	docker build -t $(PROD_REPO)/$(IMAGE_NAME):latest -f Dockerfile.make .

pushprod: prod
	docker push $(PROD_REPO)/$(IMAGE_NAME):latest
	rm Dockerfile.make

# You must provide a REPO=your-repo-name arg when you make
# this target. REPO is the name of the Docker repository
# that will be prefixed to the name of the image being built.
dev: Dockerfile
	sed 's|{{REPO}}|'"$(REPO)"'|g' Dockerfile > Dockerfile.make
	docker build -t $(REPO)/$(IMAGE_NAME):latest -f Dockerfile.make .
	docker push $(REPO)/$(IMAGE_NAME):latest
	rm Dockerfile.make
