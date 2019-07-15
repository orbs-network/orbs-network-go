#!/bin/sh -xe

for pkg in $(find . -type d -name e2e | grep -v vendor); do
    time go test $pkg -v
done

for pkg in $(find . -type d -name adapter | grep -v vendor); do
    time go test $pkg -v
done

for pkg in $(find . -type d -name ethereum | grep -v vendor); do
    time go test "$pkg/..." -v
done
