#!/bin/bash

set -e

function finish {
    local result=$?
    if [ $result -ne 0 ]; then
        CONSUL_ID="$(docker-compose ps -q consul)"
        TEST_ID=$(docker ps -l -f "ancestor=cpfix_test_probe" --format="{{.ID}}")

        echo '----- TEST LOGS ------'
        #docker logs "$TEST_ID" | tee test.log
        #echo '----- APP LOGS ------'
        docker logs "$APP_ID" | tee app.log
        #echo '----- CONSUL LOGS ------'
        #docker logs "$CONSUL_ID" | tee consul.log
        echo '---------------------'
    fi
    exit $result
}

trap finish EXIT

docker-compose up -d consul app

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul

APP_ID="$(docker-compose ps -q app)"
docker-compose run --no-deps test /go/bin/test_probe test_sigterm "$APP_ID"

# verify preStop fired
docker logs "$APP_ID" | grep "msg=\"'preStop fired on app stopping"

# # verify postStop fired
docker logs "$APP_ID" | grep "msg=\"'postStop fired on app stopped"
