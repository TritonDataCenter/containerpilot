#!/bin/bash

# hacky way of writing an upstream stanza for Nginx
echo "upstream backend {" > /tmp/backends
for i in $(curl -s consul:8500/v1/health/service/app?passing | jq -r '.[].Service.Address')
do
    echo server $i";" >> /tmp/backends
done
echo "}" >> /tmp/backends

# remove the old upstream block and generate our new config file
sed -ri '/upstream /,/.*\}/d' /etc/nginx/conf.d/default.conf
cat /tmp/backends /etc/nginx/conf.d/default.conf > /tmp/default.conf
mv /tmp/default.conf /etc/nginx/conf.d/default.conf

# HUP Nginx so it reloads its configuration
kill -HUP $(ps -ef | grep master | grep -v grep | awk -F' +' '{print $2}')
