MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail
.DEFAULT_GOAL := build

.PHONY: build run

ROOT := $(shell pwd)

# build Nginx example
build: .base
	cp ${ROOT}/../../build/containerbuddy ${ROOT}/containerbuddy
	docker-compose -p example build --no-cache

# ship Nginx example to registry
ship:
	docker tag -f example_nginx 0x74696d/containerbuddy-demo-nginx
	docker push 0x74696d/containerbuddy-demo-nginx

# run Nginx example
run: build
	docker-compose -p example up -d

# build a base so that we don't have to rebuild the whole thing
# during debugging.
.base:
	docker build -f ./Dockerfile-base -t="0x74696d/example-base" .
	@touch .base
