#!/bin/bash

# start up consul, app, nginx and then wait
# this can take a while to converge
docker-compose up -d consul

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi

docker-compose up -d app nginx > /dev/null 2>&1
sleep 5

# run the test_demo code against stack to make sure that App and Nginx
# both show in Consul and that Nginx has a working route to App
docker-compose run --no-deps test /go/bin/test_probe test_discovery > /dev/null 2>&1
result=$?

# cleanup
docker-compose rm -f
exit $result
