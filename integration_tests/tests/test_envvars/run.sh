#!/bin/bash
set -e

# stand up Consul
docker-compose up -d consul

# give the instance enough time to self-elect
n=0
while true
do
    if [ $n == 15 ]; then
        echo 'Timed out waiting for Consul.'
        exit 1;
    fi
    curl -Ls --fail "http://${DOCKER_HOST:-localhost}:8500/v1/status/leader" | grep 8300 >/dev/null 2>&1 && break
    n=$((n+1))
    sleep 1
done
