MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail
.DEFAULT_GOAL := build

.PHONY: clean test

ROOT := $(shell pwd)
GO := docker run --rm -e CGO_ENABLED=0 -e GOPATH=/root/.godeps:/src -v ${ROOT}:/root -w /root/src/containerbot golang go

clean:
	rm -rf build # .godeps

# fetch dependencies
.godeps:
	mkdir -p .godeps/
	GOPATH=${ROOT}/.godeps:${ROOT} go get github.com/hashicorp/consul/api

# build our binary in a container
build: .godeps
	mkdir -p build
	${GO} build -a -o /root/build/containerbot
	chmod +x ${ROOT}/build/containerbot

# run unit tests and exec test
test: .godeps
	${GO} vet
	${GO} test -v -coverprofile=/root/coverage.out

# run main
run: .godeps
	@docker rm containerbot || true
	docker run -d --name containerbot -e CGO_ENABLED=0 -e GOPATH=/root/.godeps:/src -v ${ROOT}:/root -w /root/src/containerbot golang go run main.go /root/examples/test.sh sleepStuff -debug
