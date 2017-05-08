#!/bin/bash

echo -e 'POST /v3/metric HTTP/1.1\r\nHost: control\r\nContent-Type: application/json\r\nContent-Length: 39\r\n\r\n{"containerpilot_app_some_counter": 42}' | \
    nc -U /var/run/containerpilot.socket
