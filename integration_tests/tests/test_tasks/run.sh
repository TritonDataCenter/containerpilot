#!/bin/bash
# test_tasks: runs multiple tasks with timeouts for a period of time and
# then parses their logs to verify that they ran the expected number of times.


# start up and wait for Consul to elect a leader
docker-compose up -d consul
docker-compose run --no-deps test /go/bin/test_probe test_consul > /dev/null 2>&1
if [ ! $? -eq 0 ] ; then exit 1 ; fi
docker-compose up -d app
APP_ID="$(docker-compose ps -q app)"


# shutdown of ContainerPilot and its tasks happens asynchronously, so we can't
# rely on precise counts of task executions. instead we're going to calculate
# a generous permitted range based on the elapsed clock time
start=$(date +%s)
sleep 3.5
docker-compose stop app
end=$(date +%s)
elapsed=$(($end-$start+1)) # round up to the full second

docker logs "${APP_ID}" > "${APP_ID}.log"
docker cp "${APP_ID}:/task1.txt" "${APP_ID}.task1"
docker cp "${APP_ID}:/task2.txt" "${APP_ID}.task2"
docker cp "${APP_ID}:/task3.txt" "${APP_ID}.task3"

PASS=0

## TASK 1
TASK1_TOS=$(grep -c "task1 timeout" "${APP_ID}.log")
TASK1_RUNS=$(wc -l < "${APP_ID}.task1" | tr -d '[:space:]')
max=$(echo $elapsed | awk '{print int($1/.5)}')
min=$(echo 3.5 | awk '{print int($1/.5 + 1)}')
rm "${APP_ID}.task1"

if [[ $TASK1_RUNS -lt $min || $TASK1_RUNS -gt $max ]]; then
  echo "Expected task1 to have between $min and $max executions: got $TASK1_RUNS"
  PASS=1
fi
if [ "$TASK1_TOS" -gt 0 ]; then
  echo "Expected task1 to never time out: got $TASK1_TOS"
  PASS=1
fi

## TASK 2
TASK2_TOS=$(grep -c "task2 timeout after 1.5s" "${APP_ID}.log")
TASK2_RUNS=$(wc -l < "${APP_ID}.task2" | tr -d '[:space:]')
max=$(echo $elapsed | awk '{print int($1/1.5 + 1)}')
min=$(echo 3.5 | awk '{print int($1/1.5 + 1)}')
min_to=$(echo "$min" | awk '{print int($1-1)}')
rm "${APP_ID}.task2"

if [[ $TASK2_RUNS -lt $min || $TASK2_RUNS -gt $max ]]; then
  echo "Expected task2 to have between $min and $max executions: got $TASK2_RUNS"
  PASS=1
fi
if [[ $TASK2_TOS -lt $min_to || $TASK2_TOS -gt $max ]]; then
  echo "Expected task2 to have between $min_to and $max timeouts after 1500ms: got $TASK2_TOS"
  PASS=1
fi

## TASK 3
TASK3_TOS=$(grep -c "task3 timeout after 100ms" "${APP_ID}.log")
TASK3_RUNS=$(wc -l < "${APP_ID}.task3" | tr -d '[:space:]')
max=$(echo $elapsed | awk '{print int($1/1.5 + 1)}')
min=$(echo 3.5 | awk '{print int($1/1.5 + 1)}')
rm "${APP_ID}.task3"

if [[ $TASK3_RUNS -lt $min || $TASK3_RUNS -gt $max ]]; then
  echo "Expected task3 to have between $min and $max executions: got $TASK3_RUNS"
  PASS=1
fi
if [ "$TASK3_TOS" -ne "$TASK3_RUNS" ]; then
  echo "Expected task3 to have $TASK3_RUNS timeouts after 100ms: got $TASK3_TOS"
  PASS=1
fi

result=$PASS

if [ $result -ne 0 ]; then
    mv "${APP_ID}.log" task.log
    echo "test target logs in ./task.log"
else
    rm "${APP_ID}.log"
fi


exit $result
