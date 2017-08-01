#!/bin/bash

logFail() {
    echo "${1}"
    exit 1
}

docker-compose run -d app
id=$(docker-compose ps -q app)
docker logs "$id" | grep -qv panic
result=$?
docker stop "$id" || logFail 'should still be running'

exit $result
