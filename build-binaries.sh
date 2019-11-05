#!/bin/bash -x

rm -rf _bin
export CONFIG_PKG="github.com/orbs-network/orbs-network-go/config"

time go build -o _bin/orbs-node -ldflags "-X $CONFIG_PKG.SemanticVersion=$SEMVER -X $CONFIG_PKG.CommitVersion=$GIT_COMMIT" -tags "$BUILD_FLAG" -a main.go

time go build -o _bin/orbs-signer -ldflags "-w -extldflags '-static' -X $CONFIG_PKG.SemanticVersion=$SEMVER -X $CONFIG_PKG.CommitVersion=$GIT_COMMIT" -tags "$BUILD_FLAG" -a bootstrap/signer/main/main.go

time go build -o _bin/gamma-server -ldflags "-X $CONFIG_PKG.SemanticVersion=$SEMVER -X $CONFIG_PKG.CommitVersion=$GIT_COMMIT" -tags "$BUILD_FLAG" -a bootstrap/gamma/main/main.go

# Experimental feature
# time go build -o _bin/orbs-dummy-plugin -buildmode plugin -tags javascript ./services/processor/plugins/dummy/
