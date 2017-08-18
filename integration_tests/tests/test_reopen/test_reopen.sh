#!/bin/bash

CONTAINERPILOT_LOGFILE="/tmp/containerpilot.log"
CONTAINERPILOT_ROTATEDLOGFILE="/tmp/containerpilot.log.1"

sleep 2
logs=`cat $CONTAINERPILOT_LOGFILE`
nb_lines=`cat $CONTAINERPILOT_LOGFILE | wc -l`
if [ ! -f $CONTAINERPILOT_LOGFILE ] || [[ $logs != *"hello world"* ]]; then
    exit 1
fi

#rotate logs
mv $CONTAINERPILOT_LOGFILE $CONTAINERPILOT_ROTATEDLOGFILE

sleep 2
nb_lines_rotated=`cat $CONTAINERPILOT_ROTATEDLOGFILE | wc -l`
if (( $nb_lines_rotated <= $nb_lines )); then
    exit 1
fi

#signal containerpilot to reopen file
kill -SIGUSR1 1

sleep 2
logs=`cat $CONTAINERPILOT_LOGFILE`
if [ ! -f $CONTAINERPILOT_LOGFILE ] || [[ $logs != *"hello world"* ]]; then
    exit 1
fi

exit 0