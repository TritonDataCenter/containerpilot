MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -euc
.DEFAULT_GOAL := build

.PHONY: clean test integration consul ship dockerfile docker cover lint local vendor dep-* tools

IMPORT_PATH := github.com/joyent/containerpilot
VERSION ?= dev-build-not-for-release
LDFLAGS := -X ${IMPORT_PATH}/core.GitHash='$(shell git rev-parse --short HEAD)' -X ${IMPORT_PATH}/core.Version='${VERSION}'

ROOT := $(shell pwd)
RUNNER := -v ${ROOT}:/go/src/${IMPORT_PATH} -w /go/src/${IMPORT_PATH} containerpilot_build
docker := docker run --rm -e LDFLAGS="${LDFLAGS}" $(RUNNER)
export PATH :=$(PATH):$(GOPATH)/bin

# flags for local development
GOOS := $(shell uname -s | tr A-Z a-iz)
GOARCH := amd64
CGO_ENABLED := 0
GOEXPERIMENT := framepointer

## display this help message
help:
	@echo -e "\033[32m"
	@echo "Targets in this Makefile build and test ContainerPilot in a build container in"
	@echo "Docker. For testing (only), use the 'local' prefix target to run targets directly"
	@echo "on your workstation (ex. 'make local test'). You will need to have its GOPATH set"
	@echo "and have already run 'make tools'. Set GOOS=linux to build binaries for Docker."
	@echo "Do not use 'make local' for building binaries for public release!"
	@echo
	@awk '/^##.*$$/,/[a-zA-Z_-]+:/' $(MAKEFILE_LIST) | awk '!(NR%2){print $$0p}{p=$$0}' | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}' | sort


# ----------------------------------------------
# building

## build the ContainerPilot binary
build: build/containerpilot
build/containerpilot:  build/containerpilot_build build/glide-installed */*.go *.go
	$(docker) go build -o build/containerpilot -ldflags "$(LDFLAGS)"
	@rm -rf src || true

# builds the builder container
build/containerpilot_build:
	mkdir -p ${ROOT}/build
	docker rmi -f containerpilot_build > /dev/null 2>&1 || true
	docker build -t containerpilot_build ${ROOT}
	docker inspect -f "{{ .ID }}" containerpilot_build > build/containerpilot_build

# Before packaging always `make clean build test integration`!
## tag and package ContainerPilot for release; `VERSION=make release`
release: build
	mkdir -p release
	git tag $(VERSION)
	git push joyent --tags
	cd build && tar -czf ../release/containerpilot-$(VERSION).tar.gz containerpilot
	@echo
	@cd release && sha1sum containerpilot-$(VERSION).tar.gz
	@cd release && sha1sum containerpilot-$(VERSION).tar.gz > containerpilot-$(VERSION).sha1.txt
	@echo Upload files in release/ directory to GitHub release.

## remove build/test artifacts, test fixtures, and vendor directories
clean:
	rm -rf build release cover vendor
	docker rmi -f containerpilot_build > /dev/null 2>&1 || true
	docker rm -f containerpilot_consul > /dev/null 2>&1 || true
	./scripts/test.sh clean

# ----------------------------------------------
# dependencies
# NOTE: glide will be replaced with `dep` when its production-ready
# ref https://github.com/golang/dep

## install any changed packages in the glide.yaml
vendor: build/glide-installed
build/glide-installed: build/containerpilot_build glide.yaml
	$(docker) glide install
	mkdir -p vendor
	@echo date > build/glide-installed

## install all vendored packages in the glide.yaml
dep-install:
	mkdir -p vendor
	$(docker) glide install
	@echo date > build/glide-installed

# usage DEP=github.com/owner/package make dep-add
## fetch a dependency and vendor it via `glide`
dep-add: build/containerpilot_build
	$(docker) bash -c "DEP=$(DEP) ./scripts/add_dep.sh"

# run 'GOOS=darwin make tools' if you're installing on MacOS
## set up local dev environment
tools:
	@go version | grep 1.8 || (echo 'go1.8 not installed'; exit 1)
	@$(if $(value GOPATH),, $(error 'GOPATH not set'))
	go get github.com/golang/lint/golint
	curl --fail -Lso glide.tgz "https://github.com/Masterminds/glide/releases/download/v0.12.3/glide-v0.12.3-$(GOOS)-$(GOARCH).tar.gz"
	tar -C "$(GOPATH)/bin" -xzf glide.tgz --strip=1 $(GOOS)-$(GOARCH)/glide
	rm glide.tgz
	curl --fail -Lso consul.zip "https://releases.hashicorp.com/consul/0.7.5/consul_0.7.5_$(GOOS)_$(GOARCH).zip"
	unzip consul.zip -d "$(GOPATH)/bin"
	rm consul.zip


# ----------------------------------------------
# develop and test

## print environment info about this dev environment
debug:
	@$(if $(value DOCKER_HOST), echo "DOCKER_HOST=$(DOCKER_HOST)", echo 'DOCKER_HOST not set')
	@echo IMPORT_PATH=$(IMPORT_PATH)
	@echo VERSION=$(VERSION)
	@echo ROOT=$(ROOT)
	@echo GOPATH=$(GOPATH)
	@echo PATH=$(PATH)
	@echo GOOS=$(GOOS)
	@echo GOARCH=$(GOARCH)
	@echo CGO_ENABLED=$(CGO_ENABLED)
	@echo GO15VENDOREXPERIMENT=$(GO15VENDOREXPERIMENT)
	@echo GOEXPERIMENT=$(GOEXPERIMENT)
	@echo LDFLAGS="$(LDFLAGS)"
	@echo
	@echo docker commands run as:
	@echo $(docker)

## prefix before other make targets to run in your local dev environment
local: | quiet
	@$(eval docker= )
quiet: # this is silly but shuts up 'Nothing to be done for `local`'
	@:

## run `go lint` and other code quality tools
lint:
	$(docker) bash ./scripts/lint.sh

## run unit tests
test: build/containerpilot_build
	$(docker) bash ./scripts/unit_test.sh

## run unit tests and write out HTML file of test coverage
cover: build/containerpilot_build
	mkdir -p cover
	$(docker) bash ./scripts/cover.sh


## generate stringer code
generate:
	go install github.com/joyent/containerpilot/jobs
	go install github.com/joyent/containerpilot/events
	cd jobs && stringer -type jobStatus
	cd events && stringer -type EventCode
	# fix this up for making it pass linting
	sed -i 's/_jobStatus_/jobStatus/g' ./jobs/jobstatus_string.go
	sed -i 's/_EventCode_/eventCode/g' ./events/eventcode_string.go

TEST ?= "all"
## run integration tests; filter with `TEST=testname make integration`
integration: build
	./scripts/test.sh test $(TEST)

## stand up a Consul server in development mode in Docker
consul:
	docker rm -f containerpilot_consul > /dev/null 2>&1 || true
	docker run -d -m 256m -p 8500:8500 --name containerpilot_consul \
		consul:latest agent -dev -client 0.0.0.0 -bind=0.0.0.0
