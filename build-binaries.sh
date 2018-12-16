#!/bin/bash -x

rm -rf _bin

time go build -o _bin/orbs-node -a main.go

if [ "$SKIP_DEVTOOLS" == "" ]; then
    time go build -o _bin/gamma-server -a bootstrap/gamma/main/main.go
fi
