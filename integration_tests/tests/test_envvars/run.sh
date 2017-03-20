#!/bin/bash
set -e

# run and make sure we get the env var out
docker-compose run test | grep CONTAINERPILOT_TESTENVVAR_IP
