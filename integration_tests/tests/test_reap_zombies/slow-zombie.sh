#!/bin/sh

/slow-child.sh &
tail -F # need to make sure job is long-lived
