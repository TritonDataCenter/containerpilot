MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail
.DEFAULT_GOAL := build

.PHONY: clean test consul run example ship

VERSION ?= dev-build-not-for-release
LDFLAGS := '-X main.GitHash=$(shell git rev-parse --short HEAD) -X main.Version=${VERSION}'

ROOT := $(shell pwd)

GO := docker run --rm --link containerbuddy_consul:consul -e CGO_ENABLED=0 -e GOPATH=/root/.godeps:/src -v ${ROOT}:/root -w /root/src/containerbuddy golang go


clean:
	rm -rf build release .godeps


# ----------------------------------------------
# develop and test

# run unit tests and exec test
test: .godeps consul
	${GO} vet
	${GO} fmt
	${GO} test -v -coverprofile=/root/coverage.out
	docker rm -f containerbuddy_consul || true

cover: test
	@sed -i 's/_\/root\/src\///' coverage.out
	go tool cover -html=coverage.out


# fetch dependencies
.godeps:
	mkdir -p .godeps/src/github.com/hashicorp
	git clone https://github.com/hashicorp/consul.git .godeps/src/github.com/hashicorp/consul
	cd .godeps/src/github.com/hashicorp/consul && git checkout 158eabdd6f2408067c1d7656fa10e49434f96480

# run consul
consul:
	docker rm -f containerbuddy_consul || true
	docker run -d -m 256m --name containerbuddy_consul \
		progrium/consul:latest -server -bootstrap-expect 1 -ui-dir /ui


# ----------------------------------------------
# build and release

# build our binary in a container

ifeq "$(TRAVIS)" "true"
build: .godeps
	mkdir -p ${ROOT}/build
	export GOPATH=${ROOT}/.godeps:${ROOT}/src && \
	export CGO_ENABLED=0 && \
		cd ${ROOT}/src/containerbuddy && \
		go build -a -o ${ROOT}/build/containerbuddy -ldflags ${LDFLAGS}
	chmod +x ${ROOT}/build/containerbuddy
else
build: .godeps
	mkdir -p build
	docker run --rm -e CGO_ENABLED=0 \
			-e GOPATH=/root/.godeps:/src \
			-v ${ROOT}:/root \
			-w /root/src/containerbuddy \
			golang \
			go build -a -o /root/build/containerbuddy -ldflags ${LDFLAGS}
	chmod +x ${ROOT}/build/containerbuddy
endif

# create the files we need for an official release on Github
# run this target with the VERSION environment variable set
release: build ship
	mkdir -p release
	git tag $(VERSION)
	git push joyent --tags
	cd build && tar -czf ../release/containerbuddy-$(VERSION).tar.gz containerbuddy
	@echo
	@echo Upload this file to Github release:
	@sha1sum release/containerbuddy-$(VERSION).tar.gz

# ----------------------------------------------
# example application

# build Nginx and App examples
example: build
	cp build/containerbuddy ${ROOT}/examples/nginx/opt/containerbuddy/containerbuddy
	cp build/containerbuddy ${ROOT}/examples/app/opt/containerbuddy/containerbuddy
	cd examples/app && docker build -t 0x74696d/containerbuddy-demo-app .
	cd examples/nginx && docker build -t 0x74696d/containerbuddy-demo-nginx .

# build example and ship to Docker Hub registry
ship: example
	docker push 0x74696d/containerbuddy-demo-nginx
	docker push 0x74696d/containerbuddy-demo-app
