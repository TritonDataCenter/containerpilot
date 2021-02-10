MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -euc
.DEFAULT_GOAL := build

.PHONY: clean test integration consul ship dockerfile docker cover lint local vendor dep-* tools kirby

IMPORT_PATH := github.com/joyent/containerpilot
VERSION ?= dev-build-not-for-release
LDFLAGS := -X ${IMPORT_PATH}/version.GitHash=$(shell git rev-parse --short HEAD) -X ${IMPORT_PATH}/version.Version=${VERSION}

ROOT := $(shell pwd)
RUNNER := -v ${ROOT}:/go/src/${IMPORT_PATH} -w /go/src/${IMPORT_PATH} containerpilot_build
docker := docker run --disable-content-trust --rm -e LDFLAGS="${LDFLAGS}" $(RUNNER)
export PATH :=$(PATH):$(GOPATH)/bin

# flags for local development
GOPATH ?= $(shell go env GOPATH)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED := 0
GOEXPERIMENT := framepointer

CONSUL_VERSION := 1.0.0
GLIDE_VERSION := 0.12.3

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
build/containerpilot:  build/containerpilot_build build/glide-installed */*/*.go */*.go */*/*.go *.go
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
	rm -rf build release cover vendor .glide
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
	@go version | grep 1.9 || (echo 'WARNING: go1.9 should be installed!')
	@$(if $(value GOPATH),, $(error 'GOPATH not set'))
	go get github.com/golang/lint/golint
	curl --fail -Lso glide.tgz "https://github.com/Masterminds/glide/releases/download/v$(GLIDE_VERSION)/glide-v$(GLIDE_VERSION)-$(GOOS)-$(GOARCH).tar.gz"
	tar -C "$(GOPATH)/bin" -xzf glide.tgz --strip=1 $(GOOS)-$(GOARCH)/glide
	rm glide.tgz
	curl --fail -Lso consul.zip "https://releases.hashicorp.com/consul/$(CONSUL_VERSION)/consul_$(CONSUL_VERSION)_$(GOOS)_$(GOARCH).zip"
	unzip consul.zip -d "$(GOPATH)/bin"
	rm consul.zip


# ----------------------------------------------
# develop and test

## print environment info about this dev environment
debug:
	@$(if $(value DOCKER_HOST), echo "DOCKER_HOST=$(DOCKER_HOST)", echo 'DOCKER_HOST not set')
	@echo CGO_ENABLED=$(CGO_ENABLED)
	@echo GO15VENDOREXPERIMENT=$(GO15VENDOREXPERIMENT)
	@echo GOARCH=$(GOARCH)
	@echo GOEXPERIMENT=$(GOEXPERIMENT)
	@echo GOOS=$(GOOS)
	@echo GOPATH=$(GOPATH)
	@echo IMPORT_PATH=$(IMPORT_PATH)
	@echo LDFLAGS="$(LDFLAGS)"
	@echo PATH=$(PATH)
	@echo ROOT=$(ROOT)
	@echo VERSION=$(VERSION)
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
	go install github.com/joyent/containerpilot/events
	cd events && stringer -type EventCode
	# fix this up for making it pass linting
	sed -i '.bak' 's/_EventCode_/eventCode/g' ./events/eventcode_string.go
	@rm -f ./events/eventcode_string.go.bak

TEST ?= "all"
## run integration tests; filter with `TEST=testname make integration`
integration: build
	./scripts/test.sh test $(TEST)

## stand up a Consul server in development mode in Docker
consul:
	docker rm -f containerpilot_consul > /dev/null 2>&1 || true
	docker run -d -m 256m -p 8500:8500 --name containerpilot_consul \
		consul:latest agent -dev -client 0.0.0.0 -bind=0.0.0.0

## build documentation for Kirby
kirby: build/docs

## preview the Kirby documentation
kirby-preview: build/docs
	docker run --rm -it -p 80:80 \
		-v ${ROOT}/build/docs:/var/www/html/content/1-containerpilot/1-docs/ \
		joyent/kirby-preview-base:latest

build/docs: docs/* scripts/docs.py
	rm -rf build/docs
	./scripts/docs.py
