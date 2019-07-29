#!/bin/bash -e

sudo rm -rf /usr/local/go
wget https://dl.google.com/go/go1.12.6.linux-amd64.tar.gz
sudo tar -xf go1.12.6.linux-amd64.tar.gz
sudo mv go /usr/local/
go version