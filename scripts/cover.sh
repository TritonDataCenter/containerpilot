#!/usr/bin/env bash
OUT=${OUT:-cover/cover.out}
TMP=${TMP:-cover/temp.out}
echo "mode: set" > $OUT
for pkg in $(go list ./... | grep -v '/vendor/\|_test' | sed 's+_/'$(pwd)'+github.com/tritondatacenter/containerpilot+');
do
  go test -v -coverprofile=$TMP $pkg
  if [ -f $TMP  ]; then
      cat $TMP | grep -v "mode: set" >> $OUT
  fi
done
rm -rf $TMP
go tool cover -html=cover/cover.out -o cover/cover.html
