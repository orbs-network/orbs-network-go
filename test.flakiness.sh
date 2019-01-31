#!/bin/bash -xe

. ./test.common.sh

go_test_junit_report flakiness/acceptance ./test/acceptance -count 100 -timeout 20m -failfast -tags "cpunoise norecover"

go_test_junit_report flakiness/block_storage ./services/blockstorage/test -count 100 -timeout 7m -failfast -tags "cpunoise norecover"

go_test_junit_report flakiness/internodesync ./services/blockstorage/internodesync -count 100 -timeout 7m -failfast -tags "cpunoise norecover"

go_test_junit_report flakiness/servicesync ./services/blockstorage/servicesync -count 100 -timeout 7m -failfast -tags -tags "cpunoise norecover"

go_test_junit_report flakiness/transaction_pool ./services/transactionpool/test -count 100 -timeout 7m -failfast -tags -tags "cpunoise norecover"
