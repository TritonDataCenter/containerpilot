#!/usr/bin/env bash
result=0
for pkg in $(go list ./...)
do
    staticcheck "$pkg" || result=1
    go vet "$pkg" || result=1
done
exit $result
