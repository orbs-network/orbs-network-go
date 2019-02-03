#!/bin/bash -x

. ./test.common.sh

go_test_junit_report goroutine_leaks ./test/acceptance -tags goroutineleak -run TestGoroutineLeaks -count 1
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
