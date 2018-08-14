#!/bin/bash -xe

go test ./...
go test ./test/acceptance -count 300 | grep -A 15 -- "--- FAIL:"