#!/bin/bash

docker-compose run app
TEST_ID=$(docker ps -a | awk -F' +' '/testversionflag/{print $1}')
docker logs $TEST_ID | grep dev-build-not-for-release
result=$?
docker rm $TEST_ID
exit $result
