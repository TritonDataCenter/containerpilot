#!/bin/bash

SLEEP=${1:-"1"}
FILE=${2:-"/tmp/test"}

trap "echo INTERRUPTED >> $FILE" SIGINT SIGTERM SIGQUIT

date +%s.%N >> "$FILE"
sleep "$SLEEP" &
wait
