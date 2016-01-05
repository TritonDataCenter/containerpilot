#!/bin/bash

COMPOSE_CFG=
PREFIX=exetcd

while getopts "f:p:" optchar; do
    case "${optchar}" in
        f) COMPOSE_CFG=" -f ${OPTARG}" ;;
        p) PREFIX=${OPTARG} ;;
    esac
done
shift $(expr $OPTIND - 1 )

COMPOSE="docker-compose -p ${PREFIX}${COMPOSE_CFG:-}"
CONFIG_FILE=${COMPOSE_CFG:-docker-compose.yml}

echo "Starting example application"
echo "project prefix:      $PREFIX"
echo "docker-compose file: $CONFIG_FILE"

echo 'Pulling latest container versions'
${COMPOSE} pull

echo 'Starting Etcd.'
${COMPOSE} up -d etcd

# get network info from etcd and poll it for liveness
if [ -z "${COMPOSE_CFG}" ]; then
    ETCD_IP=$(sdc-listmachines --name ${PREFIX}_etcd_1 | json -a ips.1)
else
    ETCD_IP=${ETCD_IP:-$(docker-machine ip default)}
fi

echo 'Starting application servers and Nginx'
${COMPOSE} up -d

# get network info from Nginx and poll it for liveness
if [ -z "${COMPOSE_CFG}" ]; then
    NGINX_IP=$(sdc-listmachines --name ${PREFIX}_nginx_1 | json -a ips.1)
else
    NGINX_IP=${NGINX_IP:-$(docker-machine ip default)}
fi
NGINX_PORT=$(docker inspect --format='{{(index (index .NetworkSettings.Ports "80/tcp") 0).HostPort}}' ${PREFIX}_nginx_1)
echo "Waiting for Nginx at ${NGINX_IP}:${NGINX_PORT} to pick up initial configuration."
while :
do
    sleep 1
    curl -s --fail -o /dev/null "http://${NGINX_IP}:${NGINX_PORT}/app/" && break
    echo -ne .
done
echo
echo 'Opening web page... the page will reload every 5 seconds with any updates.'
open http://${NGINX_IP}:${NGINX_PORT}/app/

echo 'Try scaling up the app!'
echo "${COMPOSE} scale app=3"
