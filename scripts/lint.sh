#!/bin/bash
result=0
for pkg in $(go list ./... | grep -v '/vendor/\|_test' | sed 's+_/'$(pwd)'+github.com/joyent/containerpilot+'); do
  if ! golint -set_exit_status $pkg; then
    result=1
  fi
done
exit $result
