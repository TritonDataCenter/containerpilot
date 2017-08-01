#!/bin/bash
set -e

function finish {
    local result=$?
    if [ $result -ne 0 ]; then
        app=$(docker-compose ps -q app)
        echo '----- CONSUL LOGS ------'
        docker logs "$consul" | tee consul.log
        echo '----- NGINX LOGS ------'
        docker logs "$nginx" | tee nginx.log
        echo '----- APP LOGS ------'
        docker logs "$app" | tee app.log
        echo '---------------------'
    fi
    exit $result
}

trap finish EXIT


# start up consul and wait for leader election
docker-compose up -d consul
consul=$(docker-compose ps -q consul)
docker exec -it "$consul" assert ready

# start app and nginx, wait for them to register
docker-compose up -d app nginx
docker exec -it "$consul" assert service app 1
docker exec -it "$consul" assert service nginx 1

# test that nginx config has been updated
nginx=$(docker-compose ps -q nginx)
for _ in $(seq 0 10); do
    docker exec -it "$nginx" curl -s -o /dev/null --fail "http://localhost:80/app/"
    [ $? -eq 0 ] && exit 0
    sleep 1
done || (echo "no route for /app/" && exit 1)
