#!/bin/bash
set -e

function finish {
    result=$?
    docker-compose rm -f
    exit $result
}
trap finish EXIT


# start up consul and app
docker-compose up -d consul

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi

docker-compose up -d app

APP_ID="$(docker-compose ps -q app)"
IP=$(docker inspect -f '{{ .NetworkSettings.Networks.testtelemetry_default.IPAddress }}' "${APP_ID}")

# This interface takes a while to converge
set +e
for i in {20..1}; do
    sleep 1 # sleep has to be before b/c we want exit code
    docker exec -it "${APP_ID}" curl -s "${IP}:9090/metrics" | grep 'containerpilot_app_some_counter 42' && break
done
result=$?
set -e
if [ $result -ne 0 ]; then exit $result; fi

# Make we register and tear down telemetry service in Consul
CONSUL_ID="$(docker-compose ps -q consul)"
set +e
for i in {5..1}; do
    sleep 1
    docker exec -it "${CONSUL_ID}" curl -s --fail localhost:8500/v1/catalog/service/containerpilot | grep 'containerpilot' && break
done
result=$?
set -e
if [ $result -ne 0 ]; then exit $result; fi

docker-compose stop app
docker exec -it "${CONSUL_ID}" curl -s --fail localhost:8500/v1/catalog/service/containerpilot | grep -v 'containerpilot'
