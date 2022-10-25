#!/usr/bin/env bash
# test_config_reload: runs an application container and send it commands
# to reload config

docker-compose up -d consul app
app=$(docker-compose ps -q app)

# single reload and verify config has reloaded
docker exec "$app" /reload-containerpilot.sh single
for i in $(seq 0 10); do
    sleep 1
    docker logs "$app" > app.log
    reloads=$(grep -c "control: reloaded app via control plane" app.log)
    serves=$(grep -c "control: serving at /var/run/containerpilot.socket" app.log)
    if [[ "$reloads" -eq 1 ]] && [[ "$serves" -eq 2 ]]; then
        echo "reloaded within $i seconds"
        break
    fi
done
if [[ "$reloads" -ne 1 ]] || [[ "$serves" -ne 2 ]]; then
    echo '--------------------'
    echo 'single reload failed'
    echo '----- APP LOGS -----'
    cat app.log
    exit 1
fi

# slam reload endpoint to verify we don't deadlock
docker exec "$app" /reload-containerpilot.sh multi
for _ in $(seq 0 20)
do
    # might take a little while for the control server to settle
    docker exec "$app" /reload-containerpilot.sh single && break
    sleep 1
done
if [[ $? -ne 0 ]]; then
    echo '--------------------'
    echo 'multi reload failed'
    echo '----- APP LOGS -----'
    cat app.log
    exit 1
fi
