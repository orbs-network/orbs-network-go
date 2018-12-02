#!/bin/bash -x

rm -rf _bin

time go build -o _bin/orbs-node -a main.go

time go test -o _bin/e2e.test -a -c ./test/e2e

time go test -o _bin/external.test -a -c ./test/external

if [ "$SKIP_DEVTOOLS" == "" ]; then
    time go build -o _bin/gamma-server -a bootstrap/gamma/main/main.go
fi
