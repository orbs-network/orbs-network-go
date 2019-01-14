#!/bin/bash -x

source ./test.common.sh
COUNT=20
# IMPORTANT: The timeout properly is for OVERALL running time of go test, not per test instance
# This means that if a single test should time out after 10s, then 100 instances should timeout after 1000s
SINGLE_RUN_TIMEOUT=10s
MULTIPLE_RUNS_TIMEOUT=120s
ACCEPTANCE_TESTS_TIMEOUT=200s
#SERIAL="-parallel 1"

run_specific_test() {
    time go test ./... -v -run $1 ${SERIAL} -count ${COUNT} -timeout ${MULTIPLE_RUNS_TIMEOUT} -failfast 2&>1 >> test.out
}

run_component_tests() {
    for dir in `find services -type d -name "test"` ; do
        time go test ./${dir} -v ${SERIAL} -count ${COUNT} -timeout ${MULTIPLE_RUNS_TIMEOUT} -failfast 2&>1 >> test.out
        check_exit_code_and_report
    done
}

# Uncomment to run component tests
run_component_tests

# Uncomment to run a single specific test
# run_specific_test TestSyncCompletePetitionerSyncFlow
