#!/bin/bash -xe
set -o pipefail

go test ./...
go test ./test/acceptance -count 100 -failfast | grep -A 15 -- "--- FAIL:"