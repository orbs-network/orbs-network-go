#!/bin/bash -xe

ulimit -S -n 20000

OUT_DIR=_out/standard

mkdir -p $OUT_DIR
go test -timeout 7m ./... -failfast -v &> ${OUT_DIR}/test.out || true # so that we always go to the junit report step
go-junit-report -set-exit-code < ${OUT_DIR}/test.out > ${OUT_DIR}/results.xml

# this test must run separately since zero parallel package tests are allowed concurrently
. ./test.goroutine-leaks.sh

# this test must run separately since zero parallel package tests are allowed concurrently
. ./test.memory-leaks.sh

# uncomment to run component tests
# ./test.components.sh
