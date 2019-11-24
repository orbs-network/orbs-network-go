#!/bin/bash

NIGHTLY=0
mkdir -p _out

sudo chmod +x ./.circleci/dockerized-go-junit-report.sh

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

    go test -v $@ &> ${OUT_DIR}/test.out || true # so that we always go to the junit report step
    ./.circleci/dockerized-go-junit-report.sh -set-exit-code < ${OUT_DIR}/test.out > ${OUT_DIR}/results.xml
#    go-junit-report -set-exit-code < ${OUT_DIR}/test.out > ${OUT_DIR}/results.xml
    EXIT_CODE=$?

    cp ${OUT_DIR}/results.xml ${REPORTS_DIR}/results.xml # so that we have it in _out for uploading as artifact, and separately in _reports since CircleCI doesn't like test summary dir to contain huge files

    if [ $EXIT_CODE != 0 ]; then
        echo "xxxxxxxxxxxxxxxxxxxxx"
        echo "Tests failed!"
        echo ""

        # junit-xml-stats is a globally installed npm package specifically for the purpose
        # of grouping together logs of failing tests ONLY.
        if [ $NIGHTLY == 1 ]; then
            junit-xml-stats ${OUT_DIR}/results.xml
        else
            # Let's look for the Go package that failed:
            GOLANG_PKG_ERR_LINE=$(grep -n "^FAIL" ${OUT_DIR}/test.out | grep -Eo '^[^:]+' | tail -n 1)

            # Reduce the test.out file only to section that we focus on and reverse it using tac
            sed -n "1,${GOLANG_PKG_ERR_LINE}p" ${OUT_DIR}/test.out | tac > ${OUT_DIR}/truncated.tac.test.out

            # Look for the last test that ran before this package failed 
            # (It's called FIRST in the variable here since we look at the log upside down)
            FIRST_RUN_LINE=$(grep -n "^=== RUN" -m 1 ${OUT_DIR}/truncated.tac.test.out | grep -Eo '^[^:]+' | tail -n 1)

            echo "The following test failed: "
            sed -n "1,${FIRST_RUN_LINE}p" ${OUT_DIR}/truncated.tac.test.out | tac
        fi

        exit $EXIT_CODE
    else
        echo "**************************"
        echo "Tests passed!"
        echo "JUnit-style XML for this run generated at: ${OUT_DIR}/results.xml"
        echo "Go test output saved for reference at: ${OUT_DIR}/test.out"
        echo "**************************"
    fi
}
