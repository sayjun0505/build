# Copyright 2014 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

MUTABLE_VERSION ?= latest
VERSION ?= $(shell git rev-parse --short HEAD)

IMAGE_STAGING := gcr.io/go-dashboard-dev/gitmirror
IMAGE_PROD := gcr.io/symbolic-datum-552/gitmirror

docker-prod:
	docker build -f Dockerfile --force-rm --tag=$(IMAGE_PROD):$(VERSION) ../..
	docker tag $(IMAGE_PROD):$(VERSION) $(IMAGE_PROD):$(MUTABLE_VERSION)
docker-staging:
	docker build -f Dockerfile --force-rm --tag=$(IMAGE_STAGING):$(VERSION) ../..
	docker tag $(IMAGE_STAGING):$(VERSION) $(IMAGE_STAGING):$(MUTABLE_VERSION)

push-prod: docker-prod
	docker push $(IMAGE_PROD):$(MUTABLE_VERSION)
	docker push $(IMAGE_PROD):$(VERSION)
push-staging: docker-staging
	docker push $(IMAGE_STAGING):$(MUTABLE_VERSION)
	docker push $(IMAGE_STAGING):$(VERSION)

deploy-prod: push-prod
	kubectl set image --namespace=prod deployment/gitmirror-serving-deployment gitmirror=$(IMAGE_PROD):$(VERSION)
	kubectl set image --namespace=prod deployment/gitmirror-mirroring-deployment gitmirror=$(IMAGE_PROD):$(VERSION)

deploy-staging: push-staging
	echo "no staging configuration"
	exit 1
