#!/bin/bash -x

CGO_ENABLED=0 go build -o orbs-node -a -ldflags '-extldflags "-static"' main.go

CGO_ENABLED=0 go test -a -ldflags '-extldflags "-static"' -c ./test/e2e

if [ "$SKIP_DEVTOOLS" == "" ]; then
    CGO_ENABLED=0 go build -o orbs-json-client -a -ldflags '-extldflags "-static"' devtools/jsonapi/main/main.go

    CGO_ENABLED=0 go build -o sambusac -a -ldflags '-extldflags "-static"' devtools/sambusac/main/main.go
fi
