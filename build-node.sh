#!/bin/bash -x

rm -rf _bin
export CONFIG_PKG="github.com/orbs-network/orbs-network-go/config"

echo "Building the node binary"
time go build -o _bin/orbs-node -ldflags "-X $CONFIG_PKG.SemanticVersion=$SEMVER -X $CONFIG_PKG.CommitVersion=$GIT_COMMIT" -tags "$BUILD_FLAG" -a main.go
