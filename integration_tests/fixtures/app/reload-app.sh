#!/bin/bash

SIGNAL=${1:-false}

if [[ $SIGNAL != false ]]; then
  if [[ $SIGNAL == "HUP" ]]; then
    # Change our config to actually pass the healthcheck
    sed -i s/8888/8000/ /app-with-consul-prestart-sighup.json
  fi

  kill -${SIGNAL} 1
fi

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
