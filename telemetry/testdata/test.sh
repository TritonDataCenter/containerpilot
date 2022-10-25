#!/usr/bin/env bash

trap 'exit 2' SIGTERM

usage() {
    cat <<EOF
usage: $0 [COMMAND]

Does nothing.

EOF
}

doStuff() {
    echo "Running doStuff with args: $@"
}

measureStuff() {
    echo "42"
}

cmd="${1:-usage}"
shift
$cmd "$@"
