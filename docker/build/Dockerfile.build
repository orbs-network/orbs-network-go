FROM golang:1.11.4-alpine

RUN apk add --no-cache gcc musl-dev git bash

WORKDIR /go/src/github.com/orbs-network/orbs-network-go/

ADD . /go/src/github.com/orbs-network/orbs-network-go/

RUN env

RUN go env

ARG SKIP_DEVTOOLS

ARG GIT_COMMIT

ARG BUILD_FLAG

ARG SEMVER

RUN sh -x build-binaries.sh

RUN go get -u github.com/orbs-network/go-junit-report
