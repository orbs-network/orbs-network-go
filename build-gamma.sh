#!/bin/bash -x

rm -rf _bin
export CONFIG_PKG="github.com/orbs-network/orbs-network-go/config"

echo "Building gamma binary"
time go build -o _bin/gamma-server -ldflags "-X $CONFIG_PKG.SemanticVersion=$SEMVER -X $CONFIG_PKG.CommitVersion=$GIT_COMMIT" -tags "$BUILD_FLAG" -a bootstrap/gamma/main/main.go

echo "Exporting artifacts go.mod"
time go run ./bootstrap/build/artifacts_go_mod.go _bin/
