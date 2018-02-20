#!/usr/bin/env bash

set -e

KEY_FILE="./gcloud-travis.json.enc"
if [ $1 ]; then
	KEY_FILE=$1
fi

OS_NAME=`echo $(uname) | tr "[:upper:]" "[:lower:]"`
if [ $OS_NAME == "darwin" ]; then
	OS_NAME="osx"
fi

GETH_ARCHIVE_NAME="geth-classic-$OS_NAME-$(janus version -format='TAG_OR_NIGHTLY')"
zip "$GETH_ARCHIVE_NAME.zip" geth
tar -zcf "$GETH_ARCHIVE_NAME.tar.gz" geth

mkdir deploy
mv *.zip *.tar.gz deploy/
ls -l deploy/

GPG=""
if [ $CIRCLECI ]; then
	GPG=-gpg
fi
janus deploy $GPG -to="builds.etcdevteam.com/go-ethereum/$(janus version -format='v%M.%m.x')/" -files="./deploy/*" -key="$KEY_FILE"
