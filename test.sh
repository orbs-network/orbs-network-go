#!/bin/sh -x

source ./test.common.sh

go test -timeout 5m ./... -failfast > test.out
check_exit_code_and_report

# Uncomment to run component tests
# ./test.components.sh