#!/bin/bash

function finish {
    result=$?
    if [[ "$result" -ne 0 ]]; then docker logs "$app" | tee app.log; fi
    exit $result
}
trap finish EXIT

# start up app
docker-compose up -d app
app=$(docker-compose ps -q app)

docker exec -i $app /tmp/test_reopen.sh

