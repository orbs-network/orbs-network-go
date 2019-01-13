#!/bin/bash -x

ulimit -S -n 20000

source ./test.common.sh

go test -timeout 7m ./... -failfast -v 2>&1 > _out/test.out
go-junit-report -set-exit-code < _out/test.out > _out/results.xml
check_exit_code_and_report

# this test must run separately since zero parallel package tests are allowed concurrently
source ./test.goroutine-leaks.sh

# this test must run separately since zero parallel package tests are allowed concurrently
source ./test.memory-leaks.sh

# uncomment to run component tests
# ./test.components.sh
