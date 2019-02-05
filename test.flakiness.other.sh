#!/bin/bash -x

. ./test.common.sh

go_test_junit_report blockstorage ./services/blockstorage/test -count 100 -timeout 10m -failfast -tags "cpunoise norecover"

go_test_junit_report internodesync ./services/blockstorage/internodesync -count 100 -timeout 7m -failfast -tags "cpunoise norecover"

go_test_junit_report servicesync ./services/blockstorage/servicesync -count 100 -timeout 7m -failfast -tags -tags "cpunoise norecover"

go_test_junit_report transactionpool ./services/transactionpool/test -count 100 -timeout 7m -failfast -tags -tags "cpunoise norecover"
