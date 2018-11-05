#!/usr/bin/env bash

# For local installation, see: https://github.com/golangci/golangci-lint#local-installation

# To process a specific folder recursively, run:
# ./lint.sh services/consensusalgo/leanhelixconsensus/...

which golangci-lint
rc=$?
if [[ $rc -ne 0 ]] ; then
    echo
    echo "Error running linter, perhaps it is not installed."
    echo "On MacOS, run: brew install golangci/tap/golangci-lint"
    echo
    echo "For additional installation instructions, go to:"
    echo "https://github.com/golangci/golangci-lint#local-installation"
    echo

    echo
    exit 1
fi

golangci-lint run $1


echo
echo "Done."
echo