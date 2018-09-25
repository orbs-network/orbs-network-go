#!/bin/sh -x

source ./test.common.sh

go test -timeout 3m ./... -failfast > test.out
check_exit_code_and_report

# this test must run separately since zero parallel package tests are allowed concurrently
source ./test.goroutine-leaks.sh

# uncomment to run component tests
# ./test.components.sh