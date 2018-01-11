#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
}

teardown() {
	rm -fr $DATA_DIR
}

@test "version sets default from git" {

    local example_out_place=`mktemp -d`
    local another_place=`mktemp -d`

    # Ensure building from arbitrary CWD to arbitrary DIR/geth yields expected
    # version value.
    cd # cwd=$home
    go build -o "$example_out_place/geth" github.com/ethereumproject/go-ethereum/cmd/geth

    run "$example_out_place/geth" version
    [ "$status" -eq 0 ]
    [[ "$output" == *"Version: source_v"* ]]

    # Ensure moving existing binary does not impact version value.
    mv "$example_out_place/geth" "$another_place"/
    run "$another_place/geth" version
    [ "$status" -eq 0 ]
    [[ "$output" == *"Version: source_v"* ]]

    # Ensure messing around with source doesn't impact version value for
    # existing binary.
    mv "$GOPATH/src/github.com/ethereumproject/go-ethereum/cmd/geth" "$GOPATH/src/github.com/ethereumproject/go-ethereum/geth"
    run "$another_place/geth" version
    [ "$status" -eq 0 ]
	[[ "$output" == *"Version: source_v"* ]]
    # put it back
    mv "$GOPATH/src/github.com/ethereumproject/go-ethereum/geth" "$GOPATH/src/github.com/ethereumproject/go-ethereum/cmd/geth"

    # Ensure building from project WD yields expected version value.
    cd "$GOPATH/src/github.com/ethereumproject/go-ethereum"
    go build ./cmd/geth
    [ -f ./geth ]
    run ./geth version
    [ "$status" -eq 0 ]
	[[ "$output" == *"Version: source_v"* ]]
    rm ./geth

    # Ensure building from project package 'geth' yields expected version value.
    cd ./cmd/geth
    go build .
    [ -f ./geth ]
    run ./geth version
    [ "$status" -eq 0 ]
	[[ "$output" == *"Version: source_v"* ]]
    rm ./geth

    # Cleanup
    rm -rf "$example_out_place"
    rm -rf "$another_place"
}

