#!/bin/bash -x

. ./test.common.sh
LAST_COMMIT_MESSAGE=`git --no-pager log --decorate=short --pretty=oneline -n1 $CIRCLE_SHA1`
EXTRA_FLAKY_FAILFAST="-failfast"
EXTRA_FLAKY_COUNT=35

if [[ "${LAST_COMMIT_MESSAGE}" == *"#extraflaky"* ]]; then
    EXTRA_FLAKY_FAILFAST=""
    EXTRA_FLAKY_COUNT=350
fi

if [ "$CIRCLE_NODE_INDEX" == 0 ] || [ "$CIRCLE_NODE_INDEX" == 1 ] || [ "$CIRCLE_NODE_INDEX" == 2 ] || [ "$CIRCLE_NODE_INDEX" == 3 ] || [ -z "$CIRCLE_NODE_INDEX" ]; then
    if [[ "${LAST_COMMIT_MESSAGE}" == *"#extraflaky"* ]]; then
        echo "Running in extra flakiness mode (x10 count) and no fail fast, (take a seat this might take a while)"
    fi
    go_test_junit_report acceptance ./test/acceptance -count $EXTRA_FLAKY_COUNT -timeout 20m $EXTRA_FLAKY_FAILFAST -tags "norecover"
fi

if [ "$CIRCLE_NODE_INDEX" == 4 ] || [ "$CIRCLE_NODE_INDEX" == 5 ] || [ -z "$CIRCLE_NODE_INDEX" ]; then
    go_test_junit_report blockstorage ./services/blockstorage/test -count 70 -timeout 10m -failfast -tags "norecover"

    go_test_junit_report internodesync ./services/blockstorage/internodesync -count 70 -timeout 7m -failfast -tags "norecover"

    go_test_junit_report servicesync ./services/blockstorage/servicesync -count 70 -timeout 7m -failfast -tags -tags "norecover"

    go_test_junit_report transactionpool ./services/transactionpool/test -count 70 -timeout 7m -failfast -tags -tags "norecover"
fi