#!/bin/bash

usage() {
    cat <<EOF
usage: $0 [COMMAND]

Does nothing.

EOF
}

doStuff() {
    echo "Running doStuff with args: $@"
}

failStuff() {
    echo "Running failStuff with args: $@"
    exit -1
}

sleepStuff() {
    echo "Sleeping 10 seconds..."
    sleep 5
}

cmd="${1:-usage}"
shift
$cmd "$@"
