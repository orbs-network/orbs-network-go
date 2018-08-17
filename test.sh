#!/bin/bash -x

go test -timeout 5s ./...
go test ./test/acceptance -count 100 > test.out

export EXIT_CODE=$?

cat test.out | grep -A 15 -- "--- FAIL:"

exit $EXIT_CODE