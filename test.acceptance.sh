#!/bin/sh -x

source ./test.common.sh

go test ./test/acceptance -count 100 -timeout 10m -failfast > test.out
check_exit_code_and_report
