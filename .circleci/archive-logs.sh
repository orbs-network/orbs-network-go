#!/bin/sh -x

bzip2 _logs/*.out

tar jcvf _logs/acceptance.tar.bz2 _logs/acceptance/ && rm -rf _logs/acceptance
