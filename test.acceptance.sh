#!/bin/sh -x

source ./test.common.sh

go test ./test/acceptance -count 100 -timeout 15m -failfast > test.out
check_exit_code_and_report
