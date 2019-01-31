#!/bin/bash

mkdir -p _out

check_exit_code_and_report () {
    export EXIT_CODE=$?

    if [ $EXIT_CODE != 0 ]; then
        grep -B 150 -A 15 -- "FAIL:" ./_out/test.out > ./_out/fail.out
        cat ./_out/fail.out

        grep -B 150 -A 150 -- "test timed out" ./_out/test.out > ./_out/timed.out
        cat ./_out/timed.out

        if [ ! -s ./_out/fail.out ] && [ ! -s ./_out/timed.out ]; then
            cat ./_out/test.out
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
    shift

    mkdir -p $OUT_DIR
    go test -v $@ &> ${OUT_DIR}/test.out || true # so that we always go to the junit report step
    go-junit-report -set-exit-code < ${OUT_DIR}/test.out > ${OUT_DIR}/results.xml
}
