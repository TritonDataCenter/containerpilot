#!/bin/bash

go test -v $(go list ./... | grep -v '/vendor\|_test' | sed 's+_/'$(pwd)'+github.com/joyent/containerpilot+') -bench .
