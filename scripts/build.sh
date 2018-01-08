#!/usr/bin/env bash

set -e

LDFLAGS="$1" "$2"
BINARY="$3"
FOLDERS=$(ls cmd)

for CMD in $FOLDERS;
do
    echo "Building $BINARY/$CMD..."
    go build $LDFLAGS -o $BINARY/$CMD ./cmd/$CMD
done
