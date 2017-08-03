#!/bin/bash
set -e

function finish {
    result=$?
    if [[ "$result" -ne 0 ]]; then docker logs "$app" | tee app.log; fi
    exit $result
}
trap finish EXIT


# start up app & consul and wait for leader election
docker-compose up -d consul app
docker exec -it "$(docker-compose ps -q consul)" assert ready

app=$(docker-compose ps -q app)
sleep 1 # need time for 1st health checks to fire
logs=$(docker logs "$app")

# ensure that correct log level, log format, and passthru/nonpassthru
# settings are respected for ContainerPilot and each of the 4 jobs
echo "$logs" | grep -q 'loaded config\:' || (echo "did not show loading config" && exit 1)
echo "$logs" | grep -q 'msg=\"job1 exec' || (echo "bad job1 exec log" && exit 1)
echo "$logs" | grep -q '^job2 exec' || (echo "bad job2 exec log" && exit 1)
echo "$logs" | grep -q '^job3 health' || (echo "bad job3 health log" && exit 1)
echo "$logs" | grep -q 'msg=\"job4 health' || (echo "bad job4 health log" && exit 1)
