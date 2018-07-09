#!/bin/bash -x

if [ "$SKIP_BUILD_DEPS" == "true" ]; then
    echo "Skipping dependencies' build"
    exit 0
fi

TYPES_PATH=`pwd`/vendor/github.com/orbs-network/orbs-spec/types/
HAS_MEMBUFS=$(which membufc)

if [ "$HAS_MEMBUFS" == "" ]; then
    go get -u github.com/orbs-network/pbparser

    go get -u github.com/gobuffalo/packr/...
    cd `pwd`/vendor/github.com/orbs-network/membuffers/go/membufc/

    packr build

    export PATH=$PATH:`pwd`
    ls
fi

cd $TYPES_PATH
./build.sh
