#!/bin/bash -x

ulimit -S -n 20000

. ./test.common.sh

go test -timeout 7m ./... -failfast -v &> _out/test.out
go-junit-report -set-exit-code < _out/test.out > _out/results.xml
EXIT_CODE=$?
if [ $EXIT_CODE != 0 ]; then
    exit $EXIT_CODE
fi

# this test must run separately since zero parallel package tests are allowed concurrently
. ./test.goroutine-leaks.sh

# this test must run separately since zero parallel package tests are allowed concurrently
. ./test.memory-leaks.sh

# uncomment to run component tests
# ./test.components.sh
