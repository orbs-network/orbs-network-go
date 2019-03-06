#!/bin/bash -x

LAST_COMMIT_MESSAGE=`git --no-pager log --decorate=short --pretty=oneline -n1 $CIRCLE_SHA1`

BUILD_FLAGS=""
if [[ "${LAST_COMMIT_MESSAGE}" == *"#unsafetests"* ]]; then
    BUILD_FLAGS="unsafetests"
fi

rm -rf _bin
export CONFIG_PKG="github.com/orbs-network/orbs-network-go/config"

time go build -o _bin/orbs-node -ldflags "-X $CONFIG_PKG.SemanticVersion=$SEMVER -X $CONFIG_PKG.CommitVersion=$GIT_COMMIT" -tags $BUILD_FLAGS -a main.go

if [ "$SKIP_DEVTOOLS" == "" ]; then
    time go build -o _bin/gamma-server -ldflags "-X $CONFIG_PKG.SemanticVersion=$SEMVER -X $CONFIG_PKG.CommitVersion=$GIT_COMMIT" -a bootstrap/gamma/main/main.go
fi
