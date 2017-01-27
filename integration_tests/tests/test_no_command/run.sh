#!/bin/bash

logFail() {
    echo "${1}"
    exit 1
}

docker-compose run -d app
TEST_ID=$(docker ps -a | awk -F' +' '/testnocommand/{print $1}')
docker logs $TEST_ID | grep -qv panic
result=$?
docker stop $TEST_ID || logFail 'should still be running'
docker rm $TEST_ID
exit $result
