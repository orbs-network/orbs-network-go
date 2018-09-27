#!/bin/sh -x

bzip2 logs/*.out

tar jcvf logs/acceptance.tar.bz2 logs/acceptance/ && rm -rf logs/acceptance
