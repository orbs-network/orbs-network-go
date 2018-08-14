#!/bin/bash -xe -o pipefail

go test ./...
go test ./test/acceptance -count 100 | grep -A 15 -- "--- FAIL:"