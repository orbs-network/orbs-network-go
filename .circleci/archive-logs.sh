#!/bin/sh -x

find logs/ -type f -exec bzip2 {} \;
