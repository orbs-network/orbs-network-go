#!/bin/sh -xe

if [ -z "$(which addlicense)" ]; then
    go get -u github.com/google/addlicense
fi

find . -type f -name '*.go' ! -path './vendor/*' ! -path './.idea/*' ! -path './.git/*' \
    -exec addlicense -f LICENSE-HEADER {} \;
