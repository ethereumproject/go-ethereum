#!/usr/bin/env bash

# Unencrypt JSON key file.
openssl aes-256-cbc -k "$GCP_PASSWD" -in gcloud-travis.json.enc -out .gcloud.json -d

# Run golang upload script to update GCP Storage bucket.
go run gcs-deploy.go -bucket builds.etcdevteam.com -object go-ethereum/$(cat version-base.txt)/geth-classic-$TRAVIS_OS_NAME-$(cat version-app.txt).zip -file geth-classic-$TRAVIS_OS_NAME-$(cat version-app.txt).zip -key .gcloud.json

# Clean up.
rm .gcloud.json
