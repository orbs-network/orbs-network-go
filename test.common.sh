#!/bin/sh

if [ "$SKIP_TESTS" != "" ]; then
    exit 0
fi

check_exit_code_and_report () {
    export EXIT_CODE=$?

    if [ $EXIT_CODE != 0 ]; then
        grep -B 150 -A 15 -- "FAIL:" test.out > fail.out
        cat fail.out

        grep -B 150 -A 150 -- "test timed out" test.out > timed.out
        cat timed.out

        if [ ! -s fail.out ] && [ ! -s timed.out ]; then
            cat test.out
        fi

    fi

    # copy full log for further investigation
    mkdir -p logs
    cp *.out logs

    exit $EXIT_CODE
}
