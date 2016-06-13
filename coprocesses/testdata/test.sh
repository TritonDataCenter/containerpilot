#!/bin/bash

trap 'exit 2' SIGTERM

usage() {
    cat <<EOF
usage: $0 [COMMAND]

Does nothing.

EOF
}

echoOut() {
  echo -n "$1" >> $2
}

printDots() {
  for i in {1..10}; do
    echo -n "." >> "$1"
    sleep 0.1
  done
}

cmd="${1:-usage}"
shift
$cmd "$@"
