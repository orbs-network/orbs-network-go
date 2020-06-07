#!/bin/bash -x

rm -rf _bin
export CONFIG_PKG="github.com/orbs-network/orbs-network-go/config"

echo "Building the node binary"
time go build -o _bin/orbs-node -ldflags "-X $CONFIG_PKG.SemanticVersion=$SEMVER -X $CONFIG_PKG.CommitVersion=$GIT_COMMIT" -tags "$BUILD_FLAG" -a main.go

echo "Exporting artifacts go.mod"
time go run ./bootstrap/build/artifacts_go_mod.go _bin/

echo "Build healthckeck binary"
export BUILD_FLAG="$BUILD_FLAG netgo osusergo" # allows static linking, further reading https://github.com/golang/go/issues/30419
time go build -o _bin/healthcheck -ldflags "-w -extldflags '-static' -X $CONFIG_PKG.SemanticVersion=$SEMVER -X $CONFIG_PKG.CommitVersion=$GIT_COMMIT" -tags "$BUILD_FLAG" -a bootstrap/healthcheck/main/main.go
