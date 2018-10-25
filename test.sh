#!/bin/sh -x

ulimit -S -n 20000

source ./test.common.sh

go test -timeout 5m ./... -failfast > test.out
check_exit_code_and_report

# this test must run separately since zero parallel package tests are allowed concurrently
source ./test.goroutine-leaks.sh

# this test must run separately since zero parallel package tests are allowed concurrently
source ./test.memory-leaks.sh

# uncomment to run component tests
# ./test.components.sh