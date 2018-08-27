FROM golang:1.10-alpine

WORKDIR /src/github.com/orbs-network/orbs-network-go/
ENV GOPATH=/

ADD . /src/github.com/orbs-network/orbs-network-go/
RUN go build -o /usr/bin/sambusac sambusac/main.go

USER nobody
CMD ["sambusac"]