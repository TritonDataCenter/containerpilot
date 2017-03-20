#!/bin/bash

docker-compose up -d consul app > /dev/null 2>&1

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi

APP_ID="$(docker-compose ps -q app)"
docker-compose run --no-deps test /go/bin/test_probe test_sigterm "$APP_ID" > /dev/null 2>&1
result=$?

CONSUL_ID="$(docker-compose ps -q consul)"
TEST_ID=$(docker ps -l -f "ancestor=cpfix_test_probe" --format="{{.ID}}")

if [ $result -ne 0 ]; then
    docker logs "$TEST_ID" | tee test.log
    docker logs "$APP_ID" | tee app.log
    docker logs "$CONSUL_ID" | tee consul.log
fi
exit $result
