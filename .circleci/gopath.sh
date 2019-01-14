#!/bin/sh -x

mkdir -p /go/src/github.com/orbs-network/
mv ~/project/ /go/src/github.com/orbs-network/orbs-network-go

mkdir -p /go/src/github.com/orbs-network/orbs-network-go/_out

go get -u github.com/jstemmer/go-junit-report
