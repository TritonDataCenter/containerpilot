#!/bin/bash
# test_config_reload: runs an application container and send it commands
# to reload config

docker-compose up -d consul app
APP_ID=$(docker-compose ps -q app)

# single reload and verify config has reloaded
docker exec "$APP_ID" /reload-containerpilot.sh single
sleep 1
docker logs "$APP_ID" > app.log

reloads=$(grep -c "control: reloaded app via control plane" app.log)
serves=$(grep -c "control: serving at /var/run/containerpilot.socket" app.log)
if [[ "$reloads" -ne 1 ]] || [[ "$serves" -ne 2 ]]; then
    echo '--------------------'
    echo 'single reload failed'
    echo '----- APP LOGS -----'
    exit 1
fi

# slam reload endpoint to verify we don't deadlock
docker exec "$APP_ID" /reload-containerpilot.sh multi
sleep 10
docker exec "$APP_ID" /reload-containerpilot.sh single
if [[ $? -ne 0 ]]; then
    echo '--------------------'
    echo 'multi reload failed'
    echo '----- APP LOGS -----'
    exit 1
fi

exit 0
