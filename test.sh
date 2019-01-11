#!/bin/bash -x

BASEDIR=$(dirname "$0")

ulimit -S -n 20000

source $BASEDIR/test.common.sh

go test -timeout 7m "$BASEDIR/..." -failfast -v 2>&1 | tee >(go-junit-report > "$BASEDIR/report.xml") > "$BASEDIR/test.out"
check_exit_code_and_report

# this test must run separately since zero parallel package tests are allowed concurrently
source $BASEDIR/test.goroutine-leaks.sh

# this test must run separately since zero parallel package tests are allowed concurrently
source $BASEDIR/test.memory-leaks.sh

# uncomment to run component tests
# $BASEDIR/test.components.sh
