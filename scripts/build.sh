#!/usr/bin/env bash

# Handles building all packages from ./cmd/* EXCEPT for geth.

set -e

BINARY="$1"
FOLDERS=$(ls cmd)

for CMD in $FOLDERS;
do
	if [ ! "$CMD" == "geth" ]; then
		echo "Building $BINARY/$CMD ..."
		go build -o $BINARY/$CMD ./cmd/$CMD
	fi
done
