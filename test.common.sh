#!/bin/sh

if [ "$SKIP_TESTS" != "" ]; then
    exit 0
fi

check_exit_code_and_report () {
    export EXIT_CODE=$?

    if [ $EXIT_CODE != 0 ]; then
        grep -B 150 -A 15 -- "FAIL:" test.out
        grep -B 150 -A 150 -- "test timed out" test.out

        exit $EXIT_CODE
    fi
}
