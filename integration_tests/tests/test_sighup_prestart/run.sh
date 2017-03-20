#!/bin/bash
# test_sighup_prestart: runs an application container with a prestart
# that SIGHUPs ContainerPilot to force it to reload the config

docker-compose up -d consul app

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi

APP_ID=$(docker-compose ps -q app)

docker-compose run --no-deps test /go/bin/test_probe test_sighup_prestart "${APP_ID}" > /dev/null 2>&1
result=$?

CONSUL_ID=$(docker-compose ps -q consul)
TEST_ID=$(docker ps -l -f "ancestor=cpfix_test_probe" --format="{{.ID}}")

if [ $result -ne 0 ]; then
    echo '----- TEST LOGS ------'
    docker logs "${TEST_ID}" | tee test.log
    echo '----- APP LOGS ------'
    docker logs "${APP_ID}" | tee app.log
    echo '---------------------'
    docker logs "${CONSUL_ID}" | tee consul.log
    echo "test probe logs in ./test.log"
    echo "test target logs in ./app.log"
    echo "test consul logs in ./consul.log"
fi

exit $result
