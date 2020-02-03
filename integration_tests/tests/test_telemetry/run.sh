#!/bin/bash
set -e


# start up consul and wait for leader election
docker-compose up -d consul
consul=$(docker-compose ps -q consul)
docker exec -it "$consul" assert ready

# start up app
docker-compose up -d app

app="$(docker-compose ps -q app)"
IP=$(docker inspect -f '{{ .NetworkSettings.Networks.test_telemetry_default.IPAddress }}' "$app")

# This interface takes a while to converge
for _ in $(seq 0 20); do
    sleep 1
    metrics=$(docker exec -it "$app" curl -s "${IP}:9090/metrics")
    echo "$metrics" | grep 'containerpilot_app_some_counter 42' && break
done || (echo "did not get expected metrics output" && exit 1)

# check last /metrics scrape for the rest of the events
echo "$metrics" | grep 'containerpilot_events' || \
    ( echo 'no containerpilot_events metrics' && exit 1 )
echo "$metrics" | grep 'containerpilot_control_http_requests' || \
    ( echo 'no containerpilot_control_http_requests metrics' && exit 1 )
echo "$metrics" | grep 'containerpilot_watch_instances' || \
    ( echo 'no containerpilot_watch_instances metrics' && exit 1 )

# Check the status endpoint too
docker exec -it "$app" /check.sh "${IP}"

# Make we register and tear down telemetry service in Consul
docker exec -it "$consul" assert service containerpilot 1
docker-compose stop app
docker exec -it "$consul" curl -s --fail localhost:8500/v1/catalog/service/containerpilot | grep -v 'containerpilot'
