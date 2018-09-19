#!/bin/bash -x

time go build -o orbs-node -a main.go

time go test -a -c ./test/e2e

if [ "$SKIP_DEVTOOLS" == "" ]; then
    time go build -o orbs-json-client -a devtools/jsonapi/main/main.go

    time go build -o sambusac -a devtools/sambusac/main/main.go
fi
