#!/bin/sh -x

source ./test.common.sh

time go test -timeout 3m ./... -failfast > test.out
check_exit_code_and_report
