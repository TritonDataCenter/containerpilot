#!/bin/bash

docker-compose up -d consul app > /dev/null 2>&1
APP_ID="$(docker-compose ps -q app)"
sleep 6
NUM_ZOMBIES=$(docker exec $APP_ID ps -o stat,ppid,pid,comm | awk '
BEGIN { count=0 }
$1 ~ /^Z/ && $2 ~ /1/ { count++ }
END { print count }
')
if [ $NUM_ZOMBIES -gt 1 ]; then
  echo "Number of zombies > 1: $NUM_ZOMBIES" >&2
  docker exec $APP_ID ps -o stat,ppid,pid,args
  docker logs $APP_ID
  exit 1
fi
exit 0
