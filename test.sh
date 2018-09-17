#!/bin/sh -x

COUNT=200
# IMPORTANT: The timeout properly is for OVERALL running time of go test, not per test instance
# This means that if a single test should time out after 10s, then 100 instances should timeout after 1000s
SINGLE_RUN_TIMEOUT=10s
MULTIPLE_RUNS_TIMEOUT=120s
ACCEPTANCE_TESTS_TIMEOUT=200s
OUT=test.out
#SERIAL="-parallel 1"

if [ "$SKIP_TESTS" != "" ]; then
    exit 0
fi

check_exit_code_and_report () {
    export EXIT_CODE=$?

    if [ $EXIT_CODE != 0 ]; then
        cat ${OUT} | grep -B 300 -A 30 -- "FAIL"
        cat ${OUT} | grep -B 150 -A 150 -- "timed out"

        exit $EXIT_CODE
    fi
}

get_date() {
    date +"%Y-%m-%d %H:%M:%S"
}
run_all_tests () {

    echo "=====> $(get_date) Starting to run all tests once <=====\n" | tee -a ${OUT}
    # Don't change the next line to tee -a ${OUT} because it will always return rc=0 even on error
    time go test ./... -v ${SERIAL} -timeout ${SINGLE_RUN_TIMEOUT} -failfast 2&>1 >> ${OUT}
    rc=$?
    echo "=====> $(get_date) Finished running all tests once (rc=${rc}) <=====\n" | tee -a ${OUT}
    return $rc
}

run_specific_test() {
    time go test ./services/blockstorage/test -v -run TestSyncCompletePetitionerSyncFlow ${SERIAL} -count ${COUNT} -timeout ${MULTIPLE_RUNS_TIMEOUT} -failfast 2&>1 >> ${OUT}

}
run_component_tests () {

    # "blockstorage/test" "consensusalgo/benchmarkconsensus/test" "consensuscontext/test" "gossip/adapter" "processor/native/test" "publicapi/test" "statestorage/test" "transactionpool/test" "virtualmachine/test"
#    for comp in "blockstorage/test" "consensusalgo/benchmarkconsensus/test" "consensuscontext/test" "gossip/adapter" "processor/native/test" "publicapi/test" "statestorage/test" "transactionpool/test" "virtualmachine/test"
    for comp in "blockstorage/test"
    do
        echo "=====> $(get_date) Starting to run component tests under ${comp} ${COUNT} times <=====" | tee -a ${OUT}
        time go test ./services/${comp} -v ${SERIAL} -count ${COUNT} -timeout ${MULTIPLE_RUNS_TIMEOUT} -failfast 2&>1 >> ${OUT}
        rc=$?
        if [ $rc != 0 ] ; then
            echo "!!! Error in component test ${comp} rc=${rc}" | tee -a ${OUT}
            return ${rc}
        fi

       echo "=====> $(get_date) Finished running component tests for ${comp} ${COUNT} times <=====" | tee -a ${OUT}
    done

}

run_acceptance_tests () {

    echo "=====> $(get_date) Starting to run acceptance tests ${COUNT} times <=====\n" | tee -a ${OUT}
    # Don't change the next line to tee -a ${OUT} because it will always return rc=0 even on error
    time go test ./test/acceptance -v ${SERIAL} -count ${COUNT} -timeout ${ACCEPTANCE_TESTS_TIMEOUT} -failfast 2&>1 >> ${OUT}
    rc=$?
    echo "=====> $(get_date) Finished running acceptance tests ${COUNT} times (rc=${rc}) <=====\n" | tee -a ${OUT}
    return $rc

}

# Omitted -a from "tee" command on purpose, so that output file will be truncated
echo "@@@@@ STARTING TO RUN TESTS (output: ${OUT}) @@@@@\n" | tee ${OUT}

source ./test.common.sh

run_all_tests
check_exit_code_and_report

#run_specific_test
run_component_tests
check_exit_code_and_report

run_acceptance_tests
check_exit_code_and_report

echo "\n@@@@@ FINISHED RUNNING ALL TESTS (output: ${OUT}) @@@@@" | tee -a ${OUT}
echo "\n\n@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n" | tee -a ${OUT}
