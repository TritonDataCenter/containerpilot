#!/bin/bash
set -e

single() {
    echo -e "POST /v3/reload HTTP/1.1\r\nHost: control\r\n\r\n" | \
        nc -U /var/run/containerpilot.socket > /dev/null 2>&1
    exit 0
}

multi() {
    # we mask the output here because we expect many many lines that
    # say "nc: unix connect failed: No such file or directory" while
    # the config is being reloaded
    for i in {1..200}; do
        echo -e "POST /v3/reload HTTP/1.1\r\nHost: control\r\n\r\n" | \
            nc -U /var/run/containerpilot.socket > /dev/null 2>&1
    done
    exit 0
}

cmd="$1"
"$cmd"
