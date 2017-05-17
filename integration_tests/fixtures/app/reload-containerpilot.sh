#!/bin/bash
set -e

single() {
    /bin/containerpilot -reload > /dev/null 2>&1
    exit 0
}

multi() {
    # we mask the output here because we expect many many lines that
    # say "dial unix /var/run/containerpilot.sock: connect: no such
    # file or directory" while the config is being reloaded
    for i in {1..200}; do
        /bin/containerpilot -reload > /dev/null 2>&1
    done
    exit 0
}

cmd="$1"
"$cmd"
