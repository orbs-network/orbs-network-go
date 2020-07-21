#!/bin/bash

multilog_err=1
multilog_cmd="multilog s16777215 n2 '!tai64nlocal' /opt/orbs/logs"

while [[ "${multilog_err}" -ne "0" ]]; do
    sleep 1
    echo "orbs-network-go logging pre checks.." | $multilog_cmd
    multilog_err=$?
done

echo "Running orbs-network-go.."

/opt/orbs/orbs-node $@ | $multilog_cmd 2>&1