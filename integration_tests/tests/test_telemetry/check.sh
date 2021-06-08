#!/bin/bash
set -eo pipefail
set -x


IP="$1"
curl -s "${IP}:9090/status" | json -a .Services.0.Status | grep "healthy"
curl -s "${IP}:9090/status" | json -a .Services.1.Status | grep "healthy"
