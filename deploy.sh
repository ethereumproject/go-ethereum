#!/usr/bin/env bash

GETH_ARCHIVE_NAME="geth-classic-$TRAVIS_OS_NAME-$(janus version -format='TAG_OR_NIGHTLY')"
zip "$GETH_ARCHIVE_NAME.zip" geth
tar -zcf "$GETH_ARCHIVE_NAME.tar.gz" geth

mkdir deploy
mv *.zip *.tar.gz deploy/
ls -l deploy/

janus deploy -to="builds.etcdevteam.com/go-ethereum/$(janus version -format='v%M.%m.x')/" -files="./deploy/*" -key="./gcloud-travis.json.enc"
