#!/bin/bash -x

ulimit -S -n 20000

. ./test.common.sh

time go_test_junit_report standard -tags "unsafetests" -timeout 10m ./... -failfast

# uncomment to run component tests
# ./test.components.sh
