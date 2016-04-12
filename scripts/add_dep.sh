#!/bin/bash
set -x
if [ -z "$DEP" ]; then
  echo "No dependency provided. Expected: DEP=<go import path>"
  exit 1
fi
godep restore
go get -u ${DEP}
godep save
