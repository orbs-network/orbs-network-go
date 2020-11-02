#!/bin/bash

multilog_err=1
multilog_cmd="multilog s16777215 n32 /opt/orbs/logs"

while [[ "${multilog_err}" -ne "0" ]]; do
    sleep 1
    echo "orbs-network-go logging pre checks.." | $multilog_cmd
    multilog_err=$?
done

echo "Running orbs-network-go.."

/opt/orbs/orbs-node $@ 2>&1 | $multilog_cmd
