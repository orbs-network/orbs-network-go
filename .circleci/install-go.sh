#!/bin/bash -e

GO_VERSION="1.12.7"

sudo rm -rf /usr/local/go
wget "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz"
sudo tar -xf "go${GO_VERSION}.linux-amd64.tar.gz"
sudo mv go /usr/local/
go version