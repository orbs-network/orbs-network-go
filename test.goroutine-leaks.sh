#!/bin/sh

mkdir -p _out
NO_LOG_STDOUT=true go test ./test/acceptance -tags goroutineleak -run TestGoroutineLeaks -count 1 > _out/test.out

export EXIT_CODE=$?

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

  cat _out/test.out

  exit $EXIT_CODE
fi
