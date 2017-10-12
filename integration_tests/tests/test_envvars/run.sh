#!/bin/bash
set -e

## run and make sure we get the env var out but print
## the full result if it fails, for debugging
docker-compose up -d
sleep 5
app="$(docker-compose ps -q test)"

docker logs "$app" | grep CONTAINERPILOT_TESTENVVAR_IP || (docker logs "$app" && exit 1)
docker logs "$app" | grep CONTAINERPILOT_TESTPID_PID || (docker logs "$app" && exit 1)

docker-compose stop test
