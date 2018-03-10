#!/usr/bin/env bash

GETH_ARCHIVE_NAME="geth-ellaism-$TRAVIS_OS_NAME"
zip "$GETH_ARCHIVE_NAME.zip" geth

shasum -a 256 $GETH_ARCHIVE_NAME.zip
shasum -a 256 $GETH_ARCHIVE_NAME.zip > $GETH_ARCHIVE_NAME.zip.sha256
