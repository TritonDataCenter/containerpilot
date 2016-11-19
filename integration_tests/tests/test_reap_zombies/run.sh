#!/bin/bash
# Test to verify that we're correctly reaping zombies.
# At any given time we may have up to 1 zombie parented to PID1 (it has been
# reparented but not yet reaped) and 1 zombie not yet parented to PID1.
# We can't test any more precisely than this without racing the kernel
# reparenting mechanism.

docker-compose up -d consul app > /dev/null 2>&1

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi

APP_ID="$(docker-compose ps -q app)"
sleep 6

PTREE=$(docker exec $APP_ID ps -o stat,ppid,pid,comm)
REPARENTED_ZOMBIES=$(echo $PTREE | awk '
BEGIN { count=0 }
$1 ~ /^Z/ && $2 ~ /^1$/ { count++ }
END { print count }
')

TOTAL_ZOMBIES=$(echo $PTREE | awk '
BEGIN { count=0 }
$1 ~ /^Z/ { count++ }
END { print count }
')

if [ $REPARENTED_ZOMBIES -gt 1 ] || [ $TOTAL_ZOMBIES -gt 2 ]; then
  echo "More than permitted number of zombies." >&2
  echo $PTREE >&2
  docker logs $APP_ID
  exit 1
fi
exit 0
