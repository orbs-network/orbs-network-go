#!/bin/bash -x

. ./test.common.sh

if [ "$CIRCLE_NODE_INDEX" == 0 ] || [ "$CIRCLE_NODE_INDEX" == 1 ] || [ "$CIRCLE_NODE_INDEX" == 2 ] || [ "$CIRCLE_NODE_INDEX" == 3 ] || [ -z "$CIRCLE_NODE_INDEX" ]; then
    go_test_junit_report acceptance ./test/acceptance -count 50 -timeout 20m -failfast -tags "norecover"
fi

if [ "$CIRCLE_NODE_INDEX" == 4 ] || [ "$CIRCLE_NODE_INDEX" == 5 ] || [ -z "$CIRCLE_NODE_INDEX" ]; then
    go_test_junit_report blockstorage ./services/blockstorage/test -count 100 -timeout 10m -failfast -tags "norecover"

    go_test_junit_report internodesync ./services/blockstorage/internodesync -count 100 -timeout 7m -failfast -tags "norecover"

    go_test_junit_report servicesync ./services/blockstorage/servicesync -count 100 -timeout 7m -failfast -tags -tags "norecover"

    go_test_junit_report transactionpool ./services/transactionpool/test -count 100 -timeout 7m -failfast -tags -tags "norecover"
fi