#!/bin/bash

mkdir -p _out

check_exit_code_and_report () {
    export EXIT_CODE=$?

    # copy full log for further investigation
    mkdir -p ./_logs
    cp ./_out/*.out ./_logs

    if [ $EXIT_CODE != 0 ]; then
        exit $EXIT_CODE
    fi
}

