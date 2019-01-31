#!/bin/bash -x

OUT_DIR=_out/goroutine_leaks

mkdir -p $OUT_DIR
go test ./test/acceptance -tags goroutineleak -run TestGoroutineLeaks -count 1 -v &> ${OUT_DIR}/test.out || true # so that we always go to the junit report step
go-junit-report -set-exit-code < ${OUT_DIR}/test.out > ${OUT_DIR}/results.xml

EXIT_CODE=$?

if [ $EXIT_CODE != 0 ]; then
  echo "Test failed! Found leaking goroutines"

  echo ""
  echo ""
  echo "****** Goroutines before test:"
  echo ""
  cat /tmp/gorou-shutdown-before.out

  echo ""
  echo ""
  echo "****** Goroutines after test:"
  echo ""
  cat /tmp/gorou-shutdown-after.out

  cat ${OUT_DIR}/test.out

  exit $EXIT_CODE
fi
