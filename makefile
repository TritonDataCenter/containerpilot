MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail
.DEFAULT_GOAL := build

.PHONY: clean test consul etcd run-consul run-etcd example example-consul example-etcd ship dockerfile docker cover lint

VERSION ?= dev-build-not-for-release
LDFLAGS := '-X main.GitHash=$(shell git rev-parse --short HEAD) -X main.Version=${VERSION}'

ROOT := $(shell pwd)

COMPOSE_PREFIX_ETCD := exetcd
COMPOSE_PREFIX_CONSUL := exconsul

DOCKERMAKE := docker run --rm \
	--link containerbuddy_consul:consul \
	--link containerbuddy_etcd:etcd \
	-v ${ROOT}/src/containerbuddy:/go/src/containerbuddy \
	-v ${ROOT}/.godeps:/go/src \
	-v ${ROOT}/build:/build \
	-v ${ROOT}/cover:/cover \
	-v ${ROOT}/examples:/root/examples:ro \
	-v ${ROOT}/Makefile.docker:/go/makefile:ro \
	-e LDFLAGS=${LDFLAGS} \
	containerbuddy_build

clean:
	rm -rf build release cover .godeps
	docker rmi -f containerbuddy_build > /dev/null 2>&1 || true
	docker rm -f containerbuddy_consul > /dev/null 2>&1 || true
	docker rm -f containerbuddy_etcd > /dev/null 2>&1 || true

# ----------------------------------------------
# docker build

build/containerbuddy_build:
	mkdir -p ${ROOT}/build
	docker rmi -f containerbuddy_build > /dev/null 2>&1 || true
	docker build -t containerbuddy_build ${ROOT}
	docker inspect -f "{{ .ID }}" containerbuddy_build > build/containerbuddy_build

docker: build/containerbuddy_build consul etcd

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
example: example-consul example-etcd

# build Nginx and App examples
example-consul: build
	cp build/containerbuddy ${ROOT}/examples/consul/nginx/opt/containerbuddy/containerbuddy
	cp build/containerbuddy ${ROOT}/examples/consul/app/opt/containerbuddy/containerbuddy
	cd examples/consul && docker-compose -p exconsul -f docker-compose-local.yml build

example-etcd: build
	cp build/containerbuddy ${ROOT}/examples/etcd/nginx/opt/containerbuddy/containerbuddy
	cp build/containerbuddy ${ROOT}/examples/etcd/app/opt/containerbuddy/containerbuddy
	cd examples/etcd && docker-compose -p exetcd -f docker-compose-local.yml build

# run example application locally for testing
run-consul: example-consul
	examples/run.sh consul -p ${COMPOSE_PREFIX_CONSUL} -f docker-compose-local.yml

run-etcd: example-etcd
	examples/run.sh etcd -p ${COMPOSE_PREFIX_ETCD} -f docker-compose-local.yml

clean-consul:
	cd examples/consul && docker-compose -p ${COMPOSE_PREFIX_CONSUL} -f docker-compose-local.yml kill
	cd examples/consul && docker-compose -p ${COMPOSE_PREFIX_CONSUL} -f docker-compose-local.yml rm -f
	docker rmi -f ${COMPOSE_PREFIX_CONSUL}_app ${COMPOSE_PREFIX_CONSUL}_nginx > /dev/null 2>&1 || true

clean-etcd:
	cd examples/etcd && docker-compose -p ${COMPOSE_PREFIX_ETCD} -f docker-compose-local.yml kill
	cd examples/etcd && docker-compose -p ${COMPOSE_PREFIX_ETCD} -f docker-compose-local.yml rm -f
	docker rmi -f ${COMPOSE_PREFIX_ETCD}_app ${COMPOSE_PREFIX_ETCD}_nginx > /dev/null 2>&1 || true

# tag and ship example to Docker Hub registry
ship: example
	docker tag ${COMPOSE_PREFIX_CONSUL}_nginx 0x74696d/containerbuddy-demo-nginx
	docker tag ${COMPOSE_PREFIX_CONSUL}_app 0x74696d/containerbuddy-demo-app
	docker tag ${COMPOSE_PREFIX_ETCD}_nginx 0x74696d/containerbuddy-etcd-demo-nginx
	docker tag ${COMPOSE_PREFIX_ETCD}_app 0x74696d/containerbuddy-etcd-demo-app
	docker push 0x74696d/containerbuddy-demo-nginx
	docker push 0x74696d/containerbuddy-demo-app
	docker push 0x74696d/containerbuddy-etcd-demo-nginx
	docker push 0x74696d/containerbuddy-etcd-demo-app
