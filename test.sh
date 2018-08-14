#!/bin/bash -xe

go test ./...
go test ./test/acceptance -count 100 | grep -A 15 -- "--- FAIL:"