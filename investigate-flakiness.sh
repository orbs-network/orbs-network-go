#!/bin/bash

check_exit_code_and_report () {
    export EXIT_CODE=$?

    if [ $EXIT_CODE != 0 ]; then
        cat test.out | grep -A 15 -- "FAIL"
        cat test.out | grep -A 15 -- "timed out"

        exit $EXIT_CODE
    fi
}

# single test
# go test -p 1 -parallel 1 ./services/consensusalgo/benchmarkconsensus/test -run TestLeaderRetriesCommitOnErrorGeneratingBlock -count 1000 -failfast > test.out

# entire package
go test -p 1 -parallel 1 ./services/consensusalgo/benchmarkconsensus/test -count 1000 -failfast > test.out

# using -timeout 10s is not recommended since it's suspected to give false positives on timeout failures

check_exit_code_and_report
