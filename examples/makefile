MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail
.DEFAULT_GOAL := build

.PHONY: build ship

ROOT := $(shell pwd)

# build Nginx and App examples
build:
	cp ${ROOT}/../build/containerbuddy ${ROOT}/nginx/opt/containerbuddy/containerbuddy
	cp ${ROOT}/../build/containerbuddy ${ROOT}/app/opt/containerbuddy/containerbuddy
	docker-compose -f docker-compose-local.yml -p example build --no-cache

# ship Nginx and App example to registry
ship:
	docker tag -f example_nginx 0x74696d/containerbuddy-demo-nginx
	docker tag -f example_app 0x74696d/containerbuddy-demo-app
	docker push 0x74696d/containerbuddy-demo-nginx
	docker push 0x74696d/containerbuddy-demo-app
