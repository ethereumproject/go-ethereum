#!/usr/bin/env bash

mkdir ./deploy
mv ./dist/*.tar.gz ./dist/*.zip ./deploy/
janus deploy -to builds.etcdevteam.com/go-ethereum/$(janus version -format %M.%m.x) -files ./deploy/* -key ./gcloud-travis.json.enc