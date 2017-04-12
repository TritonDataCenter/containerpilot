#!/bin/bash
set -e

# run and make sure we get the env var out but print
# the full result if it fails, for debugging
result=$(docker-compose run test)
echo "$result" | grep CONTAINERPILOT_TESTENVVAR_IP || (echo "$result" && exit 1)
