#!/bin/bash

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

# get all the healthy application servers and write the json to file
curl -s consul:8500/v1/health/service/app?passing | json > /tmp/lastQuery.json

cat <<EOF > /srv/index.html
<html>
<head>
<title>ContainerPilot Demo</title>
<script>
function timedRefresh(timeoutPeriod) {
    setTimeout("location.reload(true);",timeoutPeriod);
}
</script>
</head>
<body onload="JavaScript:timedRefresh(5000);">
<h1>ContainerPilot Demo</h1>
<h2>This page served by app server: $(hostname)</h2>
Last service health check changed at $(date):
<pre>
$(cat /tmp/lastQuery.json)
</pre>
<script>
</body><html>
EOF
