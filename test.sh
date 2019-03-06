#!/bin/bash -x

ulimit -S -n 20000

. ./test.common.sh

go_test_junit_report standard -tags "unsafetests" -timeout 7m ./... -failfast

# this test must run separately since zero parallel package tests are allowed concurrently
. ./test.goroutine-leaks.sh

# this test must run separately since zero parallel package tests are allowed concurrently
. ./test.memory-leaks.sh

# uncomment to run component tests
# ./test.components.sh
