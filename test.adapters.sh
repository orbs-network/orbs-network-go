#!/bin/sh -xe

for pkg in $(find . -type d -name e2e | grep -v vendor | grep -v \.git); do
    time go test $pkg -v
done

for pkg in $(find . -type d -name adapter | grep -v vendor | grep -v \.git); do
    time go test $pkg -v
done

for pkg in $(find . -type d -name ethereum | grep -v vendor | grep -v \.git); do
    time go test "$pkg/..." -v
done
