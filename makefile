MAKEFLAGS += --warn-undefined-variables
SHELL := bash
.SHELLFLAGS := -o pipefail -euc
.DEFAULT_GOAL := build

.PHONY: clean test integration consul ship dockerfile docker cover lint local dep-* tools kirby

IMPORT_PATH := github.com/joyent/containerpilot
VERSION ?= dev-build-not-for-release
LDFLAGS := -X ${IMPORT_PATH}/version.GitHash=$(shell git rev-parse --short HEAD) -X ${IMPORT_PATH}/version.Version=${VERSION}

ROOT := $(shell pwd)
RUNNER := -v ${ROOT}:/go/src/${IMPORT_PATH} -w /go/src/${IMPORT_PATH} containerpilot_build
docker := docker run --rm -e LDFLAGS="${LDFLAGS}" $(RUNNER)
export PATH :=$(PATH):$(GOPATH)/bin

# flags for local development
GOPATH ?= $(shell go env GOPATH)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED := 0
GOEXPERIMENT := framepointer

CONSUL_VERSION := 1.13.3

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
build/containerpilot:  build/containerpilot_build */*/*.go */*.go */*/*.go *.go
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

## remove build/test artifacts, test fixtures
clean:
	rm -rf build release cover
	docker rmi -f containerpilot_build > /dev/null 2>&1 || true
	docker rm -f containerpilot_consul > /dev/null 2>&1 || true
	./scripts/test.sh clean

# ----------------------------------------------
# dependencies

## install all packages in go.mod and go.sum
deps: build/dep-install
build/dep-install: build/containerpilot_build go.mod
	$(docker) go get ./...
	@echo date > build/update

## update all dependencies (minor versions)
dep-update:
	$(docker) go get -u ./...
	@echo date > build/update

# run 'GOOS=darwin make tools' if you're installing on MacOS
## set up local dev environment
tools:
	@go version | grep 1.19 || (echo 'WARNING: go1.19 should be installed!')
	@$(if $(value GOPATH),, $(error 'GOPATH not set'))
	go get golang.org/x/lint/golint
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

## run `go vet`, `staticcheck` and other code quality tools
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
