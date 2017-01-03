#!/bin/bash

docker-compose up -d consul app

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi

APP_ID="$(docker-compose ps -q app)"
logs=$(docker logs $APP_ID)
result=1
if [[ $logs == *"Loaded config:"* ]]; then
    result=0
fi

if [ $result -ne 0 ]; then
  echo "==== APP LOGS ===="
  docker logs $APP_ID
fi
exit $result
