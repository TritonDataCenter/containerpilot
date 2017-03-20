#!/bin/bash

# start up consul and app
docker-compose up -d consul

# Wait for consul to elect a leader
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi

docker-compose up -d app

APP_ID="$(docker-compose ps -q app)"

sleep 3.5
docker logs ${APP_ID} > ${APP_ID}.log
docker exec ${APP_ID} cat /task1.txt > ${APP_ID}.task1
docker exec ${APP_ID} cat /task2.txt > ${APP_ID}.task2
docker exec ${APP_ID} cat /task3.txt > ${APP_ID}.task3

PASS=0

## TASK 1
TASK1_TOS=$(cat ${APP_ID}.log | grep "\[task1\] timeout" | wc -l | tr -d '[[:space:]]')
TASK1_RUNS=$(cat ${APP_ID}.task1 | wc -l | tr -d '[[:space:]]')
rm ${APP_ID}.task1

if [[ $TASK1_RUNS -lt 7 || $TASK1_RUNS -gt 8 ]]; then
  echo "Expected task1 to have 7 or 8 executions: got $TASK1_RUNS"
  PASS=1
fi
if [ $TASK1_TOS -gt 0 ]; then
  echo "Expected task1 to never time out: got $TASK1_TOS"
  PASS=1
fi

## TASK 2
TASK2_TOS=$(cat ${APP_ID}.log | grep "task2 timeout after 1500ms" | wc -l | tr -d '[[:space:]]')
TASK2_RUNS=$(cat ${APP_ID}.task2 | wc -l | tr -d '[[:space:]]')
rm ${APP_ID}.task2

if [[ $TASK2_RUNS -lt 2 || $TASK2_RUNS -gt 3 ]]; then
  echo "Expected task2 to have 2 or 3 executions: got $TASK2_RUNS"
  PASS=1
fi
if [[ $TASK2_TOS -lt 1 || $TASK2_TOS -gt 2 ]]; then
  echo "Expected task2 to have 1 or 2 timeouts after 1500ms: got $TASK2_TOS"
  PASS=1
fi

## TASK 3
TASK3_TOS=$(cat ${APP_ID}.log | grep "task3 timeout after 100ms" | wc -l | tr -d '[[:space:]]')
TASK3_RUNS=$(cat ${APP_ID}.task3 | wc -l | tr -d '[[:space:]]')

if [[ $TASK3_RUNS -lt 2 || $TASK3_RUNS -gt 3 ]]; then
  echo "Expected task3 to have 2 or 3 executions: got $TASK3_RUNS"
  PASS=1
fi
if [ $TASK3_TOS -ne $TASK3_RUNS ]; then
  echo "Expected task3 to have $TASK3_RUNS timeouts after 100ms: got $TASK3_TOS"
  PASS=1
fi

rm ${APP_ID}.task3

#rm ${APP_ID}.log

result=$PASS

exit $result
