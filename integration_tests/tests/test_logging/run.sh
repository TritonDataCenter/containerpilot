#!/bin/bash

function finish {
    result=$?
    if [[ "$result" -ne 0 ]]; then docker logs "$app" | tee app.log; fi
    exit $result
}
trap finish EXIT


# start up app & consul and wait for leader election
docker-compose up -d consul app
docker exec -it "$(docker-compose ps -q consul)" assert ready

app=$(docker-compose ps -q app)
logs=$(docker logs "$app")
result=1
if [[ $logs == *"loaded config:"* ]]; then
    result=0
fi
