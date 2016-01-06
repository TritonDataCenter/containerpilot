#!/bin/bash

if [ -z "$VIRTUALHOST" ]; then
    # fetch latest virtualhost template from Consul k/v
    curl -s --fail consul:8500/v1/kv/nginx/template?raw > /tmp/virtualhost.ctmpl
else
    # dump the $VIRTUALHOST environment variable as a file
    echo $VIRTUALHOST > /tmp/virtualhost.ctmpl
fi

# render virtualhost template using values from Consul and reload Nginx
consul-template \
    -once \
    -consul consul:8500 \
    -template "/tmp/virtualhost.ctmpl:/etc/nginx/conf.d/default.conf:nginx -s reload"
