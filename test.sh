#!/bin/bash -x

if [ "$SKIP_TESTS" != "" ]; then
    exit 0
fi

check_exit_code_and_report () {
    export EXIT_CODE=$?

    if [ $EXIT_CODE != 0 ]; then
        cat test.out | grep -B 150 -A 15 -- "FAIL:"
        cat test.out | grep -B 150 -A 150 -- "timed out"

        exit $EXIT_CODE
    fi
}

go test -timeout 20s ./... -failfast > test.out
check_exit_code_and_report

go test ./test/acceptance -count 100 -timeout 30s -failfast > test.out
check_exit_code_and_report
