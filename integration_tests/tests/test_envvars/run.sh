#!/bin/bash
set -e

docker-compose up -d etcd

# give the instance enough time to self-elect
sleep 2
curl --fail -s http://${DOCKER_HOST:-localhost}:2379/health
