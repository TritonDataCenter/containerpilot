#!/bin/bash
# Sends a signal to ContainerPilot during the preStart

# wait a few seconds for the Consul container to become available
n=0
while true
do
    if [ n == 10 ]; then
        echo "Timed out waiting for Consul"
        exit 1;
    fi
    curl -Ls --fail http://consul:8500/v1/status/leader | grep 8300 && break
    n=$((n+1))
    sleep 1
done

if [[ ${1} == "HUP" ]]; then
    # Change our config to actually pass the healthcheck
    sed -i s/8888/8000/ /app-with-consul-prestart-sighup.json
fi

kill -${1} 1
