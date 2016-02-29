#!/bin/bash

# start up consul, app, nginx
docker-compose up -d consul > /dev/null 2>&1
sleep 2
docker-compose up -d > /dev/null 2>&1

# run the test_demo code against stack to make sure that App and Nginx
# both show in Consul and that Nginx has a working route to App
docker-compose run --no-deps test /go/bin/test_probe test_demo > /dev/null 2>&1
result=$?

# cleanup
docker-compose rm -f
exit $result
