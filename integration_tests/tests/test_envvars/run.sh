#!/bin/bash
set -e

# run and check that we didn't get the default environment value of "FAIL"
docker-compose run --rm test | grep -v FAIL
