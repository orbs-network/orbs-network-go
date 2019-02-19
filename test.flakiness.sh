#!/bin/bash -x

. ./test.common.sh

LAST_COMMIT_MESSAGE=`git --no-pager log --decorate=short --pretty=oneline -n1 $CIRCLE_SHA1`
EXTRA_FLAKY_FAILFAST="-failfast"
EXTRA_FLAKY_COUNT_ACCEPTANCE=35
EXTRA_FLAKY_COUNT_REST=70

if [[ "${LAST_COMMIT_MESSAGE}" == *"#extraflaky"* ]]; then
    EXTRA_FLAKY_FAILFAST=""
    EXTRA_FLAKY_COUNT_ACCEPTANCE=350
    EXTRA_FLAKY_COUNT_REST=700
fi

if [[ $1 == "NIGHTLY" ]]; then
    echo "performing nightly build (count 1000/2000 , no failfast)"
    EXTRA_FLAKY_FAILFAST=""
    EXTRA_FLAKY_COUNT_ACCEPTANCE=1000
    EXTRA_FLAKY_COUNT_REST=2000
fi

if [ "$CIRCLE_NODE_INDEX" == 0 ] || [ "$CIRCLE_NODE_INDEX" == 1 ] || [ "$CIRCLE_NODE_INDEX" == 2 ] || [ "$CIRCLE_NODE_INDEX" == 3 ] || [ -z "$CIRCLE_NODE_INDEX" ]; then
    go_test_junit_report acceptance ./test/acceptance -count $EXTRA_FLAKY_COUNT_ACCEPTANCE -timeout 20m $EXTRA_FLAKY_FAILFAST -tags "norecover"
fi

if [ "$CIRCLE_NODE_INDEX" == 4 ] || [ "$CIRCLE_NODE_INDEX" == 5 ] || [ -z "$CIRCLE_NODE_INDEX" ]; then
    go_test_junit_report blockstorage ./services/blockstorage/test -count $EXTRA_FLAKY_COUNT_ACCEPTANCE -timeout 10m $EXTRA_FLAKY_FAILFAST -tags "norecover"

    go_test_junit_report internodesync ./services/blockstorage/internodesync -count $EXTRA_FLAKY_COUNT_ACCEPTANCE -timeout 7m $EXTRA_FLAKY_FAILFAST -tags "norecover"

    go_test_junit_report servicesync ./services/blockstorage/servicesync -count $EXTRA_FLAKY_COUNT_ACCEPTANCE -timeout 7m $EXTRA_FLAKY_FAILFAST -tags -tags "norecover"

    go_test_junit_report transactionpool ./services/transactionpool/test -count $EXTRA_FLAKY_COUNT_ACCEPTANCE -timeout 7m $EXTRA_FLAKY_FAILFAST -tags -tags "norecover"
fi