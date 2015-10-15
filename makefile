MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail
.DEFAULT_GOAL := build

.PHONY: clean test

ROOT := $(shell pwd)
GO := docker run --rm --link containerbuddy_consul:consul -e CGO_ENABLED=0 -e GOPATH=/root/.godeps:/src -v ${ROOT}:/root -w /root/src/containerbuddy golang go

clean:
	rm -rf build # .godeps

# fetch dependencies
.godeps:
	mkdir -p .godeps/
	GOPATH=${ROOT}/.godeps:${ROOT} go get github.com/hashicorp/consul/api

# build our binary in a container
build: .godeps
	mkdir -p build
	${GO} build -a -o /root/build/containerbuddy
	chmod +x ${ROOT}/build/containerbuddy

# run unit tests and exec test
test: .godeps consul
	${GO} vet
	${GO} test -v -coverprofile=/root/coverage.out
	docker rm -f containerbuddy_consul || true

# run main
run: .godeps consul
	@docker rm containerbuddy || true
	docker run -d --name containerbuddy -e CGO_ENABLED=0 -e GOPATH=/root/.godeps:/src -v ${ROOT}:/root -w /root/src/containerbuddy golang go run main.go /root/examples/test.sh sleepStuff -debug

# run consul
consul:
	docker rm -f containerbuddy_consul || true
	docker run -d -m 256m --name containerbuddy_consul \
		progrium/consul:latest -server -bootstrap-expect 1 -ui-dir /ui
