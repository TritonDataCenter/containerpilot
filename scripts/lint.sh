#!/bin/bash
result=0
for pkg in $(go list ./... | grep -v '/vendor/\|_test' | sed 's+_/'$(pwd)'+github.com/joyent/containerbuddy+'); do
  if golint $pkg; then
    result=1
  fi
done
exit $result
