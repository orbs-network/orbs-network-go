#!/bin/sh -x

source ./test.common.sh

NO_LOG_STDOUT=true go test ./test/acceptance -count 100 -timeout 20m -failfast > test.out
check_exit_code_and_report
