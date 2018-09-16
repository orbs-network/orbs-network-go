#!/bin/sh -x

source ./test.common.sh

go test -timeout 3m ./... -failfast > test.out
check_exit_code_and_report
