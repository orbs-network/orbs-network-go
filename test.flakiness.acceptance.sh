#!/bin/bash -x

. ./test.common.sh

go_test_junit_report acceptance ./test/acceptance -count 30 -timeout 20m -failfast -tags "cpunoise norecover"
