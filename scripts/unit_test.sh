#!/usr/bin/env bash

go test -v $(go list ./... | grep -v '/vendor\|_test' | sed 's+_/'$(pwd)'+github.com/tritondatacenter/containerpilot+') -bench .
