#!/bin/bash

set -e

function finish {
    local result=$?
    if [[ "$result" -ne 0 ]]; then docker logs "$app" | tee app.log; fi
    exit $result
}

trap finish EXIT


# start up consul and wait for leader election
docker-compose up -d consul
consul=$(docker-compose ps -q consul)
docker exec -it "$consul" assert ready

# start up app and wait for it to register
docker-compose up -d app
app=$(docker-compose ps -q app)
docker exec -it "$consul" assert service app 1

# send SIGHUP into the app container
docker kill -s HUP "$app"
sleep 2
# send SIGUSR2 into the app container
docker kill -s USR2 "$app"

# verify that the app is still running and that both `on-sighup` and
# `on-sigusr2` signal event based jobs have executed
docker exec -it "$consul" assert service app 1
docker logs "$app" | grep "msg=\"'on-sighup job fired on SIGHUP"
docker logs "$app" | grep "msg=\"'on-sigusr2 job fired on SIGUSR2"
