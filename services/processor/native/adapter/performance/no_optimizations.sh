#!/bin/bash -x

rm -rf *.so
go clean -cache

cp counter100.go.ignore counter100.go
cp counter200.go.ignore counter200.go

time go build -gcflags '-N -l' -buildmode=plugin -o counter100.so counter100.go

time go build -gcflags '-N -l' -buildmode=plugin -o counter200.so counter200.go

time go build -gcflags '-N -l' -buildmode=plugin -o counter100.so counter100.go

time go build -gcflags '-N -l' -buildmode=plugin -o counter200.so counter200.go

rm -rf counter100.go
rm -rf counter200.go

ls -al *.so