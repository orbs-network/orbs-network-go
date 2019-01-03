FROM golang:1.11.4-alpine

RUN apk add --no-cache gcc musl-dev

ADD ./vendor/github.com/orbs-network/orbs-contract-sdk/go/ /go/src/github.com/orbs-network/orbs-network-go/vendor/github.com/orbs-network/orbs-contract-sdk/go/

ADD ./_bin/gamma-server /opt/orbs/

WORKDIR /opt/orbs

CMD ./gamma-server
