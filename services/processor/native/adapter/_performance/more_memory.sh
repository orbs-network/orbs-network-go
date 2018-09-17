#!/bin/bash -x

rm -rf *.so
go clean -cache

time GOGC=off go build -buildmode=plugin -o counter100.so counter100.go

time GOGC=off go build -buildmode=plugin -o counter200.so counter200.go

time GOGC=off go build -buildmode=plugin -o counter100.so counter100.go

time GOGC=off go build -buildmode=plugin -o counter200.so counter200.go

ls -al *.so