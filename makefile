MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -euc
.DEFAULT_GOAL := build

.PHONY: clean test integration consul etcd run-consul run-etcd example example-consul example-etcd ship dockerfile docker cover lint vendor

IMPORT_PATH := github.com/joyent/containerbuddy
VERSION ?= dev-build-not-for-release
LDFLAGS := -X ${IMPORT_PATH}/config.GitHash='$(shell git rev-parse --short HEAD)' -X ${IMPORT_PATH}/config.Version='${VERSION}'

ROOT := $(shell pwd)

COMPOSE_PREFIX_ETCD := exetcd
COMPOSE_PREFIX_CONSUL := exconsul

DOCKERRUN := docker run --rm \
	--link containerbuddy_consul:consul \
	--link containerbuddy_etcd:etcd \
	-v ${ROOT}/vendor:/go/src \
	-v ${ROOT}:/cb/src/${IMPORT_PATH} \
	-w /cb/src/${IMPORT_PATH} \
	containerbuddy_build

DOCKERBUILD := docker run --rm \
	-e LDFLAGS="${LDFLAGS}" \
	-v ${ROOT}/vendor:/go/src \
	-v ${ROOT}:/cb/src/${IMPORT_PATH} \
	-w /cb/src/${IMPORT_PATH} \
	containerbuddy_build

clean:
	rm -rf build release cover vendor
	docker rmi -f containerbuddy_build > /dev/null 2>&1 || true
	docker rm -f containerbuddy_consul > /dev/null 2>&1 || true
	docker rm -f containerbuddy_etcd > /dev/null 2>&1 || true
	./test.sh clean

# ----------------------------------------------
# docker build

# default top-level target
build: build/containerbuddy

build/containerbuddy:  build/containerbuddy_build */*.go vendor
	${DOCKERBUILD} go build -o build/containerbuddy -ldflags "$(LDFLAGS)"
	@rm -rf src || true

# builds the builder container
build/containerbuddy_build:
	mkdir -p ${ROOT}/build
	docker rmi -f containerbuddy_build > /dev/null 2>&1 || true
	docker build -t containerbuddy_build ${ROOT}
	docker inspect -f "{{ .ID }}" containerbuddy_build > build/containerbuddy_build

# shortcut target for other targets: asserts a
# working test environment
docker: build/containerbuddy_build consul etcd

# top-level target for vendoring our packages: godep restore requires
# being in the package directory so we have to run this for each package
vendor: build/containerbuddy_build
	${DOCKERBUILD} godep restore

# fetch a dependency via go get, vendor it, and then save into the parent
# package's Godeps
# usage DEP=github.com/owner/package make add-dep
add-dep: build/containerbuddy_build
	docker run --rm \
		-e LDFLAGS="${LDFLAGS}" \
		-v ${ROOT}:/cb/src/${IMPORT_PATH} \
		-w /cb/src/${IMPORT_PATH} \
		containerbuddy_build \
		bash ./scripts/add_dep.sh DEP=$(DEP)

# ----------------------------------------------
# develop and test

lint: vendor
	-${DOCKERBUILD} bash ./scripts/lint.sh

# run unit tests and write out test coverage
test: docker vendor
	${DOCKERRUN} bash ./scripts/unit_test.sh

cover: docker
	mkdir -p cover
	${DOCKERRUN} bash ./scripts/cover.sh

# run integration tests
integration: build
	./scripts/test.sh

# ------ Backends

# Consul Backend
consul:
	docker rm -f containerbuddy_consul > /dev/null 2>&1 || true
	docker run -d -m 256m --name containerbuddy_consul \
		progrium/consul:latest -server -bootstrap-expect 1 -ui-dir /ui

# Etcd Backend
etcd:
	docker rm -f containerbuddy_etcd > /dev/null 2>&1 || true
	docker run -d -m 256m --name containerbuddy_etcd -h etcd quay.io/coreos/etcd:v2.0.8 \
		-name etcd0 \
		-advertise-client-urls http://etcd:2379,http://etcd:4001 \
		-listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
		-initial-advertise-peer-urls http://etcd:2380 \
		-listen-peer-urls http://0.0.0.0:2380 \
		-initial-cluster-token etcd-cluster-1 \
		-initial-cluster etcd0=http://etcd:2380 \
		-initial-cluster-state new

release: build
	mkdir -p release
	git tag $(VERSION)
	git push joyent --tags
	cd build && tar -czf ../release/containerbuddy-$(VERSION).tar.gz containerbuddy
	@echo
	@echo Upload this file to Github release:
	@sha1sum release/containerbuddy-$(VERSION).tar.gz
