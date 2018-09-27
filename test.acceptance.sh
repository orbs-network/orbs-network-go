#!/bin/sh -x

source ./test.common.sh

STANDALONE=true go test ./test/acceptance -count 100 -timeout 6m -failfast > test.out
check_exit_code_and_report
