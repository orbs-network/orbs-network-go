#!/bin/bash -x

rm -rf *.so
go clean -cache

time go build -gcflags '-N -l' -buildmode=plugin -o counter100.so counter100.go

time go build -gcflags '-N -l' -buildmode=plugin -o counter200.so counter200.go

time go build -gcflags '-N -l' -buildmode=plugin -o counter100.so counter100.go

time go build -gcflags '-N -l' -buildmode=plugin -o counter200.so counter200.go

ls -al *.so