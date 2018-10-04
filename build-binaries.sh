#!/bin/bash -x

time go build -o orbs-node -a main.go

time go test -a -c ./test/e2e

if [ "$SKIP_DEVTOOLS" == "" ]; then
    time go build -o gamma-cli -a devtools/gammacli/main/main.go

    time go build -o gamma-server -a devtools/gamma-server/main/main.go
fi
