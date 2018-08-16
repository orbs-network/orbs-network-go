#!/bin/bash -xe

go test -timeout 5s ./...
go test ./test/acceptance -count 100 | grep -A 15 -- "--- FAIL:"
test ${PIPESTATUS[0]} -eq 0