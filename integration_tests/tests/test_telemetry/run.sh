#!/bin/bash

# start up consul and app
docker-compose up -d consul
sleep 2
docker-compose up -d app

APP_ID="$(docker-compose ps -q app)"
IP=$(docker inspect -f '{{ .NetworkSettings.IPAddress }}' ${APP_ID})

# this interface takes a while to converge
for i in {20..1}; do
    sleep 1
    docker exec -it ${APP_ID} curl -v -s --fail ${IP}:9090/metrics | grep 'containerbuddy_app_some_counter 42' && break
done

result=$?

# cleanup
docker-compose rm -f
exit $result
