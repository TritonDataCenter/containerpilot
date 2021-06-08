#!/bin/bash

docker-compose up -d app
TEST_ID="$(docker-compose ps -q app)"
docker logs "$TEST_ID" | grep dev-build-not-for-release
result=$?
docker-compose down
exit $result
