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

