#!/bin/sh

LOG_FILE=$1
TXID=$2

grep flow=checkpoint $LOG_FILE | grep txHash=$TXID | sed -e 's/source=.*//g' | sed -e 's/function=.*//g'