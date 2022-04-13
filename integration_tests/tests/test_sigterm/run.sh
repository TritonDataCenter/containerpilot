#!/usr/bin/env bash

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
docker exec -i "$consul" assert ready

# start up app and wait for it to register
docker-compose up -d app
app=$(docker-compose ps -q app)
docker exec -i "$consul" assert service app 1

# sigterm app container
docker stop "$app"

# and verify it's exited gracefully from consul, and that
# both preStop and postStop jobs have executed
docker exec -i "$consul" assert service app 1
docker logs "$app" | grep "msg=\"'preStop fired on app stopping"
docker logs "$app" | grep "msg=\"'postStop fired on app stopped"
