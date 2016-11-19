#!/bin/bash

docker-compose up -d consul app

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi

APP_ID="$(docker-compose ps -q app)"
docker-compose run --no-deps test /go/bin/test_probe test_sighup_deadlock $APP_ID > /dev/null 2>&1
result=$?
TEST_ID=$(docker ps -l -f "ancestor=cpfix_test_probe" --format="{{.ID}}")
if [ $result -ne 0 ]; then
  echo "==== TEST LOGS ===="
  docker logs $TEST_ID
  echo "==== APP LOGS ===="
  docker logs $APP_ID
fi
docker rm -f $TEST_ID > /dev/null 2>&1
exit $result
