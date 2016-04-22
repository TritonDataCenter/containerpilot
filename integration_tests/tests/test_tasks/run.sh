#!/bin/bash

# start up consul and app
docker-compose up -d consul
sleep 2
docker-compose up -d app

APP_ID="$(docker-compose ps -q app)"

sleep 3.5
docker logs ${APP_ID} > ${APP_ID}.log
docker exec ${APP_ID} cat /task1.txt > ${APP_ID}.task1
docker exec ${APP_ID} cat /task2.txt > ${APP_ID}.task2
docker exec ${APP_ID} cat /task3.txt > ${APP_ID}.task3

PASS=0

## TASK 1
TASK1_TOS=$(cat ${APP_ID}.log | grep "task1 timeout" | wc -l | tr -d '[[:space:]]')
TASK1_RUNS=$(cat ${APP_ID}.task1 | wc -l | tr -d '[[:space:]]')
rm ${APP_ID}.task1

if [ $TASK1_RUNS -ne 7 ]; then
  echo "Expected task1 to have 7 executions: got $TASK1_RUNS"
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

if [ $TASK2_RUNS -ne 2 ]; then
  echo "Expected task2 to have 2 executions: got $TASK2_RUNS"
  PASS=1
fi
if [ $TASK2_TOS -ne 1 ]; then
  echo "Expected task2 to have 1 timeouts after 1500ms: got $TASK2_TOS"
  PASS=1
fi

## TASK 3
TASK3_TOS=$(cat ${APP_ID}.log | grep "task3 timeout after 100ms" | wc -l | tr -d '[[:space:]]')
TASK3_RUNS=$(cat ${APP_ID}.task3 | wc -l | tr -d '[[:space:]]')

if [ $TASK3_RUNS -ne 2 ]; then
  echo "Expected task3 to have 2 executions: got $TASK3_RUNS"
  PASS=1
fi
if [ $TASK3_TOS -ne 2 ]; then
  echo "Expected task3 to have 2 timeouts after 100ms: got $TASK3_TOS"
  PASS=1
fi

rm ${APP_ID}.task3

rm ${APP_ID}.log

result=$PASS

# cleanup
docker-compose rm -f
exit $result
