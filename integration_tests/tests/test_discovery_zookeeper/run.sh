#!/bin/bash

# start up zookeeper, app, nginx and then wait
# this can take a while to converge
docker-compose up -d zookeeper
sleep 2
docker-compose up -d app nginx > /dev/null 2>&1
sleep 5

# run the test_demo code against stack to make sure that App and Nginx
# both show in Zookeeper and that Nginx has a working route to App
docker-compose run --no-deps test /go/bin/test_probe test_discovery zookeeper > /dev/null 2>&1
result=$?

# cleanup
docker-compose rm -f
exit $result
