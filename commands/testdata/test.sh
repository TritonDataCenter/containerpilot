#!/bin/bash

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

failStuff() {
    echo "Running failStuff with args: $@"
    exit -1
}

doNothing() {
  exit 0
}

sleepStuff() {
    echo "Sleeping 10 seconds..."
    sleep 10
}

interruptSleep() {
  for i in {1..10}; do
    echo -n "."
    sleep 1
  done
}

cmd="${1:-usage}"
shift
$cmd "$@"
