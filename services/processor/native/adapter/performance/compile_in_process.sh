#!/bin/bash -x

rm -rf *.so
rm -rf *.a

cp compile_in_process.go.ignore compile_in_process.go
go tool compile compile_in_process.go
rm -rf compile_in_process.go

go tool link -o compile_in_process_test compile_in_process.o
rm -rf *.o

time ./compile_in_process_test
rm -rf compile_in_process_test