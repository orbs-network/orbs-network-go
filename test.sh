#!/bin/bash -x

check_exit_code_and_report () {
    export EXIT_CODE=$?

    if [ $EXIT_CODE != 0 ]; then
        cat test.out | grep -A 15 -- "FAIL"
        cat test.out | grep -A 15 -- "timed out"

        exit $EXIT_CODE
    fi
}

go test -timeout 5s ./... > test.out
check_exit_code_and_report

go test ./services/consensusalgo/... -count 100 -timeout 5s -failfast > test.out
check_exit_code_and_report

go test ./test/acceptance -count 100 -timeout 10s -failfast > test.out
check_exit_code_and_report
