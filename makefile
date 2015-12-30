MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail
.DEFAULT_GOAL := build

.PHONY: clean test consul run example ship dockerfile docker cover lint

VERSION ?= dev-build-not-for-release
LDFLAGS := '-X main.GitHash=$(shell git rev-parse --short HEAD) -X main.Version=${VERSION}'

ROOT := $(shell pwd)

DOCKERMAKE := docker run --rm --link containerbuddy_consul:consul \
	-v ${ROOT}/src/containerbuddy:/go/src/containerbuddy \
	-v ${ROOT}/build:/build \
	-v ${ROOT}/cover:/cover \
	-v ${ROOT}/examples:/root/examples:ro \
	-v ${ROOT}/Makefile.docker:/go/makefile:ro \
	-e LDFLAGS=${LDFLAGS} \
	containerbuddy_build

clean:
	rm -rf build release cover
	docker rmi -f containerbuddy_build > /dev/null 2>&1 || true
	docker rm -f containerbuddy_consul > /dev/null 2>&1 || true

# ----------------------------------------------
# docker build

build/containerbuddy_build:
	mkdir -p ${ROOT}/build
	docker rmi -f containerbuddy_build > /dev/null 2>&1 || true
	docker build -t containerbuddy_build ${ROOT}
	docker inspect -f "{{ .ID }}" containerbuddy_build > build/containerbuddy_build

docker: build/containerbuddy_build consul

build: docker
	${DOCKERMAKE} build

# ----------------------------------------------
# develop and test

lint: docker
	${DOCKERMAKE} lint

# run unit tests and exec test
test: docker
	${DOCKERMAKE} test

cover: docker
	${DOCKERMAKE} cover

# run consul
consul:
	docker rm -f containerbuddy_consul > /dev/null 2>&1 || true
	docker run -d -m 256m --name containerbuddy_consul \
		progrium/consul:latest -server -bootstrap-expect 1 -ui-dir /ui

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
	cd examples && docker-compose -p example -f docker-compose-local.yml build

# run example application locally for testing
run: example
	cd examples && ./start.sh -p example -f docker-compose-local.yml

# tag and ship example to Docker Hub registry
ship: example
	docker tag example_nginx 0x74696d/containerbuddy-demo-nginx
	docker tag example_app 0x74696d/containerbuddy-demo-app
	docker push 0x74696d/containerbuddy-demo-nginx
	docker push 0x74696d/containerbuddy-demo-app
