#!/bin/bash

# start up consul and wait for consul to elect a leader
docker-compose up -d consul
docker-compose run --no-deps test /go/bin/test_probe test_consul
if [ ! $? -eq 0 ] ; then exit 1 ; fi

# start app and nginx, then wait a bit for them to converge
docker-compose up -d app nginx
sleep 5 # TODO: this is awful

# run the test_probe against stack to make sure that App and Nginx
# both show in Consul and that Nginx has a working route to App
docker-compose run --no-deps test /go/bin/test_probe test_discovery
result=$?

if [ ! $result -eq 0 ]; then
    APP_ID=$(docker ps -l -f "ancestor=cpfix_app" --format="{{.ID}}")
    CONSUL_ID=$(docker ps -l -f "ancestor=consul" --format="{{.ID}}")
    NGINX_ID=$(docker ps -l -f "ancestor=cpfix_nginx" --format="{{.ID}}")
    echo '----- CONSUL LOGS ------'
    docker logs "${CONSUL_ID}" | tee consul.log
    echo '----- NGINX LOGS ------'
    docker logs "${NGINX_ID}" | tee nginx.log
    echo '----- APP LOGS ------'
    docker logs "${APP_ID}" | tee app.log
    echo '---------------------'
fi

exit $result
