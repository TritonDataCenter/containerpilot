#!/bin/bash
result=0
for pkg in $(go list ./... | grep -v '/vendor/\|_test'); do
  if golint $pkg; then
    result=1
  fi
done
exit $result
