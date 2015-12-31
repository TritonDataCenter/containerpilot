#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BACKEND="consul"

case $1 in
  consul|etcd)
    BACKEND="$1"
    shift
  ;;
esac

BACKEND_DIR="${DIR}/${BACKEND}"

if [ ! -d "$BACKEND_DIR" ]; then
  echo "Unable to find backend directory: $BACKEND_DIR" >&2
  exit 1
fi
cd $BACKEND_DIR

./start.sh "$@"
