#!/bin/bash -x

OUT_DIR=_out/memory_leaks

mkdir -p $OUT_DIR
go test ./test/acceptance -tags memoryleak -run TestMemoryLeaks -count 1 -v &> ${OUT_DIR}/test.out || true # so that we always go to the junit report step
go-junit-report -set-exit-code < ${OUT_DIR}/test.out > ${OUT_DIR}/results.xml

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
