#!/bin/bash

NIGHTLY=0
mkdir -p _out

check_exit_code_and_report () {
    if [ $EXIT_CODE != 0 ]; then
        grep -B 150 -A 15 -- "FAIL:" ./$OUT_DUR/test.out > ./$OUT_DUR/fail.out
        cat ./$OUT_DUR/fail.out

        grep -B 150 -A 150 -- "test timed out" ./$OUT_DUR/test.out > ./$OUT_DUR/timed.out
        cat ./$OUT_DUR/timed.out

        if [ ! -s ./$OUT_DUR/fail.out ] && [ ! -s ./$OUT_DUR/timed.out ]; then
            cat ./$OUT_DUR/test.out
        fi

    fi

    # copy full log for further investigation
    mkdir -p ./_logs
    cp ./_out/*.out ./_logs

    if [ $EXIT_CODE != 0 ]; then
        exit $EXIT_CODE
    fi
}

go_test_junit_report () {
    OUT_DIR="_out/$1"
    REPORTS_DIR="_reports/$1"
    shift

    mkdir -p $OUT_DIR
    mkdir -p $REPORTS_DIR
    GODEBUG=gctrace=1 go test -v $@ &> ${OUT_DIR}/test.out || true # so that we always go to the junit report step
    go-junit-report -set-exit-code < ${OUT_DIR}/test.out > ${OUT_DIR}/results.xml
    EXIT_CODE=$?

    cp ${OUT_DIR}/results.xml ${REPORTS_DIR}/results.xml # so that we have it in _out for uploading as artifact, and separately in _reports since CircleCI doesn't like test summary dir to contain huge files

    if [ $EXIT_CODE != 0 ]; then
        # junit-xml-stats is a globally installed npm package specifically for the purpose
        # of grouping together logs of failing tests ONLY.
        if [ $NIGHTLY == 1 ]; then
            junit-xml-stats ${OUT_DIR}/results.xml
        else
            # find the last RUN line number in the log file
            LOG_START_LINE=$(grep -n "^=== RUN" ${OUT_DIR}/test.out | grep -Eo '^[^:]+' | tail -n 1)
            # find the last line number in the log file
            LOG_END_LINE=$(cat ${OUT_DIR}/test.out | wc -l | awk '{$1=$1};1')
            # print the lines in between these line numbers to get the required failed log
            # and nothing else
            sed -n "${LOG_START_LINE},${LOG_END_LINE}p" ${OUT_DIR}/test.out
        fi

        exit $EXIT_CODE
    fi
}
