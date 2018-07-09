#!/bin/bash

git rev-parse --abbrev-ref HEAD | sed -e 's/\//-/g'
