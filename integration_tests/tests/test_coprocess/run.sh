#!/bin/bash
set -e

function finish {
    result=$?
    if [ $result -ne 0 ]; then
        APP_ID=$(docker ps -l -f "ancestor=cpfix_app" --format="{{.ID}}")
        echo '----- APP LOGS ------'
        docker logs "${APP_ID}" | tee app.log
        echo '---------------------'
    fi
    exit $result
}
trap finish EXIT

docker-compose up -d

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi

ID=$(docker ps -l -f "ancestor=cpfix_app" --format="{{.ID}}")

# verify the coprocess is running
docker exec -it "${ID}" ps -ef | grep coprocess

# kill the coprocess and verify it restarts
docker exec -it "${ID}" pkill coprocess
sleep 1
docker exec -it "${ID}" ps -ef | grep coprocess

# kill the coprocess and verify it doesn't restart again
docker exec -it "${ID}" pkill coprocess
sleep 1

set +e
docker exec -it "${ID}" ps -ef | grep coprocess && exit 1
set -e

# update the ContainerPilot config and verify the coprocess is running
# with the new flags (this resets the restart limit)
docker exec -it "${ID}" sed -i 's/arg1/arg2/' /etc/containerpilot-with-coprocess.json
docker exec -it "${ID}" kill -SIGHUP 1
sleep 1
docker exec -it "${ID}" ps -ef | grep coprocess | grep arg2

# kill the coprocess and verify it restarts
docker exec -it "${ID}" pkill coprocess
sleep 1
docker exec -it "${ID}" ps -ef | grep coprocess
