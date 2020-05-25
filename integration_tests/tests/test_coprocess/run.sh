#!/bin/bash
set -e

function finish {
    result=$?
    if [[ "$result" -ne 0 ]]; then
        docker exec -i "$app" ps -ef
        docker logs "$app" | tee app.log
    fi
    exit $result
}
trap finish EXIT

# stand up app and consul, and wait for consul to elect a leader
docker-compose up -d
docker exec -i "$(docker-compose ps -q consul)" assert ready

# verify the coprocess is running
app=$(docker-compose ps -q app)
docker exec -i "$app" ps -ef | grep coprocess

# kill the coprocess and verify it restarts
docker exec -i "$app" pkill coprocess
sleep 1
docker exec -i "$app" ps -ef | grep coprocess

# kill the coprocess and verify it doesn't restart again
docker exec -i "$app" pkill coprocess
sleep 1
docker exec -i "$app" ps -ef | grep coprocess && exit 1

# update the ContainerPilot config and verify the coprocess is running
# with the new flags (this resets the restart limit)
docker exec -i "$app" sed -i 's/arg1/arg2/' /etc/containerpilot-with-coprocess.json5
docker exec -i "$app" /reload-containerpilot.sh single
sleep 1
docker exec -i "$app" ps -ef | grep coprocess | grep arg2

# kill the coprocess and verify it restarts
docker exec -i "$app" pkill coprocess
sleep 1
docker exec -i "$app" ps -ef | grep coprocess
