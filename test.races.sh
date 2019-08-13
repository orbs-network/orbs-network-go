#!/bin/bash -x

ulimit -S -n 20000

. ./test.common.sh

time go_test_junit_report races -timeout 20m ./... -failfast -race
