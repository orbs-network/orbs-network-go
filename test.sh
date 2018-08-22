#!/bin/bash -x

go test -timeout 5s ./...
export EXIT_CODE=$?

if [ $EXIT_CODE != 0 ]; then
  exit $EXIT_CODE
fi

go test ./test/acceptance -count 100 -timeout 10s > test.out
export EXIT_CODE=$?

cat test.out | grep -A 15 -- "--- FAIL:"

exit $EXIT_CODE