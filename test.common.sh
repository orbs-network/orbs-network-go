#!/bin/bash

if [ "$SKIP_TESTS" == "true" ]; then
    exit 0
fi

check_exit_code_and_report () {
    export EXIT_CODE="${PIPESTATUS[0]}"

    if [ $EXIT_CODE != 0 ]; then
        echo "***** some tests have failed *****"
        grep -B 150 -A 15 -- "FAIL:" ./test.out > ./fail.out
        cat ./fail.out

        grep -B 150 -A 150 -- "test timed out" ./test.out > ./timed.out
        cat ./timed.out

        if [ ! -s ./fail.out ] && [ ! -s ./timed.out ]; then
            cat ./test.out
        fi

    fi

    # copy full log for further investigation
    mkdir -p ./_logs
    cp ./*.out ./_logs

    if [ $EXIT_CODE != 0 ]; then
        exit $EXIT_CODE
    fi
}

