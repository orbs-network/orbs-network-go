#!/bin/bash -x

. ./test.common.sh
go_test_junit_report memory_leaks ./test/acceptance -tags memoryleak -run TestMemoryLeaks -count 1

EXIT_CODE=$?

if [ $EXIT_CODE != 0 ]; then
  echo "Test failed! Found leaking memory"

  echo ""
  echo ""
  echo "****** Memory delta:"
  echo ""
  go tool pprof --inuse_space -nodecount 10 -top --base /tmp/mem-shutdown-before.prof /tmp/mem-shutdown-after.prof

  cat test.out

  exit $EXIT_CODE
fi
