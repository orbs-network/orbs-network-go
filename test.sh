#!/bin/sh -x

ulimit -S -n 20000

. ./test.common.sh

go test -timeout 7m ./... -failfast > test.out
check_exit_code_and_report

# this test must run separately since zero parallel package tests are allowed concurrently
. ./test.goroutine-leaks.sh

# this test must run separately since zero parallel package tests are allowed concurrently
. ./test.memory-leaks.sh

# uncomment to run component tests
# ./test.components.sh
