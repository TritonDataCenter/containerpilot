#!/bin/bash

COMPOSE_CFG=
PREFIX=example

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

echo 'Starting Consul.'
${COMPOSE} up -d consul

# get network info from consul and poll it for liveness
if [ -z "${COMPOSE_CFG}" ]; then
    CONSUL_IP=$(sdc-listmachines --name ${PREFIX}_consul_1 | json -a ips.1)
else
    CONSUL_IP=${CONSUL_IP:-$(docker-machine ip default)}
fi

echo "Writing template values to Consul at ${CONSUL_IP}"
while :
do
    # we'll sometimes get an HTTP500 here if consul hasn't completed
    # it's leader election on boot yet, so poll till we get a good response.
    sleep 1
    curl --fail -s -X PUT --data-binary @./nginx/default.ctmpl \
         http://${CONSUL_IP}:8500/v1/kv/nginx/template && break
    echo -ne .
done
echo
echo 'Opening consul console'
open http://${CONSUL_IP}:8500/ui

echo 'Starting application servers and Nginx'
${COMPOSE} up -d

# get network info from Nginx and poll it for liveness
if [ -z "${COMPOSE_CFG}" ]; then
    NGINX_IP=$(sdc-listmachines --name ${PREFIX}_nginx_1 | json -a ips.1)
else
    NGINX_IP=${NGINX_IP:-$(docker-machine ip default)}
fi
NGINX_PORT=$(docker inspect ${PREFIX}_nginx_1 | json -a NetworkSettings.Ports."80/tcp".0.HostPort)
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
