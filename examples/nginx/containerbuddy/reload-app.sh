#!/bin/bash

# get all the health application servers and write the json to file
curl -s consul:8500/v1/health/service/app?passing | jq . > /tmp/lastQuery.json

cat <<EOF > /usr/share/nginx/html/index.html
<html><head><title>Application Server: $(hostname)</title></head>
<body>
Last service health check changed at $(date):
<pre>
$(cat /tmp/lastQuery.json)
</pre></body><html>
EOF
