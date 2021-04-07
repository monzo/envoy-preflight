#!/bin/bash
# Helper script that self destructs scuttle by finding it's process id and killing it
# Helpful when testing signal handling
# use:
#       ./scuttle /bin/bash scripts/self_destruct.sh
scuttle_pid=$(pidof scuttle)
echo "------ Sending SIGURG, should be ignored ------ "
kill -URG $scuttle_pid
echo "------ Sending SIGINT, Scuttle should pass to child, child should exit, scuttle should exit ------ "
kill -INT $scuttle_pid
# Print "." until process ends
while 1>0; do echo -n "." && sleep 1; done;