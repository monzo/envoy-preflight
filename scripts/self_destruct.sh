#!/bin/bash
# Helper script that self destructs scuttle by finding it's process id and killing it
# Helpful when testing signal handling
# use:
#       ./scuttle /bin/bash scripts/self_destruct.sh
kill -INT $(pidof scuttle)
while 1>0; do echo -n "." && sleep 1; done;