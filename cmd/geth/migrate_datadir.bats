#!/usr/bin/env bats

# Current build.
: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {

	TMP_DIR=`mktemp -d`
	CMD_DIR=`mktemp -d`
	DATA_DIR=`mktemp -d`
	# Fake it.
	HOME_DEF="$HOME"
	HOME="$DATA_DIR"

	# Decide OS var for release download links.
	TEST_OS_HF=placeholder
	TEST_OS_C=placeholder
	DATA_DIR_PARENT=placeholder
	DATA_DIR_NAME=placeholder
	DATA_DIR_NAME_EX=placeholder
	# http://stackoverflow.com/questions/3466166/how-to-check-if-running-in-cygwin-mac-or-linux
	if [ "$(uname)" == "Darwin" ]; then
	    # Do something under Mac OS X platform
	    TEST_OS_HF=darwin
	    TEST_OS_C=osx
	    DATA_DIR_PARENT="$HOME/Library"
	    DATA_DIR_NAME_EX="Ethereum"
	    DATA_DIR_NAME="EthereumClassic"
	elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
	    # Do something under GNU/Linux platform
	    TEST_OS_HF=linux
	    TEST_OS_C=linux
	    DATA_DIR_PARENT="$HOME"
	    DATA_DIR_NAME_EX=".ethereum"
	    DATA_DIR_NAME=".ethereum-classic"
	elif [ "$(expr substr $(uname -s) 1 10)" == "MINGW32_NT" ]; then
	    # Do something under 32 bits Windows NT platform
	    echo "Win32 not supported."
	    exit 1 
	elif [ "$(expr substr $(uname -s) 1 10)" == "MINGW64_NT" ]; then
	    # Do something under 64 bits Windows NT platform
	    TEST_OS_HF=windows
	    TEST_OS_C=win64
	    DATA_DIR_PARENT="$HOME/AppData/Roaming"
	    DATA_DIR_NAME_EX="Ethereum"
	    DATA_DIR_NAME="EthereumClassic"
	fi

	# Install 1.6 and 1.5 of Ethereum Geth
	# Travis Linux+Mac, AppVeyor Windows all use amd64.
	curl -o "$TMP_DIR/gethf1.6.tar.gz" https://gethstore.blob.core.windows.net/builds/geth-"$TEST_OS_HF"-amd64-1.6.0-facc47cb.tar.gz
	curl -o "$TMP_DIR/gethf1.5.tar.gz" https://gethstore.blob.core.windows.net/builds/geth-"$TEST_OS_HF"-amd64-1.5.0-c3c58eb6.tar.gz
	tar xf "$TMP_DIR/gethf1.6.tar.gz" -C "$TMP_DIR"
	tar xf "$TMP_DIR/gethf1.5.tar.gz" -C "$TMP_DIR"
	mv "$TMP_DIR/geth-$TEST_OS_HF-amd64-1.6.0-facc47cb/geth" "$CMD_DIR/gethf1.6"
	mv "$TMP_DIR/geth-$TEST_OS_HF-amd64-1.5.0-c3c58eb6/geth" "$CMD_DIR/gethf1.5"

	# Install 3.3 of EthereumClassic Geth
	curl -L -o "$TMP_DIR/gethc3.3.zip" https://github.com/ethereumproject/go-ethereum/releases/download/v3.3.0/geth-classic-"$TEST_OS_C"-v3.3.0-1-gdd95f05.zip
	unzip "$TMP_DIR/gethc3.3.zip" -d "$TMP_DIR"
	mv "$TMP_DIR/geth" "$CMD_DIR/gethc3.3"
}

teardown() {
	rm -rf $TMP_DIR
	rm -rf $CMD_DIR
	rm -rf $DATA_DIR

	# Put back original.
	HOME="$HOME_DEF"
}

# Migrate ETC.
# mainnet
@test "should migrate datadir /Ethereum/ -> /EthereumClassic/ from ETC3.3 schema" {
	# Should create $HOME/Ethereum/chaindata,keystore,nodes,...
	run "$CMD_DIR/gethc3.3" --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ] # 3.3 schema

	run $GETH_CMD --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	# Ensure datadir is renamed.
	! [ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	! [ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/chaindata ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet/chaindata ]
}

# testnet
@test "should migrate datadir /Ethereum/ -> /EthereumClassic/ from ETC3.3 schema | --testnet" {
	# Should create $HOME/Ethereum/testnet/chaindata,keystore,nodes,...
	run "$CMD_DIR/gethc3.3" --fast --testnet console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/testnet ] # 3.3 schema
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/testnet/chaindata ]

	run $GETH_CMD --fast --testnet console
	echo "$output"
	[ "$status" -eq 0 ]

	# Ensure datadir is renamed.
	! [ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	! [ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/testnet ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/morden ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/morden/chaindata ]
}

# chainconfig VALID default
@test "should migrate datadir /Ethereum/ -> /EthereumClassic/ from ETC3.3 schema | --chainconfig testdata/chain_config_dump-ok.json (mainnet)" {
	run "$CMD_DIR/gethc3.3" --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]

	run $GETH_CMD --fast --chainconfig "$BATS_TEST_DIRNAME/testdata/chain_config_dump-ok.json"  console
	echo "$output"
	[ "$status" -eq 0 ]

	! [ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet ]
}

# customnet
@test "shouldnot migrate datadir /Ethereum/ -> /EthereumClassic/ from ETC3.3 schema | --chain customnet" {
	# Should create $HOME/Ethereum/chaindata,keystore,nodes,...
	run "$CMD_DIR/gethc3.3" --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ] # 3.3 schema

	run $GETH_CMD --fast --chain customnet --ipcdisable console
	echo "$output"
	[ "$status" -eq 0 ]

	# Ensure datadir is not renamed.
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	! [ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/chaindata ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/customnet ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/customnet/chaindata ]
}

# datadir
@test "shouldnot migrate datadir /Ethereum/ -> /EthereumClassic/ from ETC3.3 schema | --datadir data" {
	run "$CMD_DIR/gethc3.3" --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]

	run $GETH_CMD --fast --datadir "$DATA_DIR/data" console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]

	[ -d "$DATA_DIR"/data ]
	[ -d "$DATA_DIR"/data/mainnet ]
	[ -d "$DATA_DIR"/data/mainnet/chaindata ]
}

# chainconfig VALID custom
@test "shouldnot migrate datadir /Ethereum/ -> /EthereumClassic/ from ETC3.3 schema | --chainconfig testdata/chain_config_dump-ok-custom.json (customnet)" {
	run "$CMD_DIR/gethc3.3" --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]

	run $GETH_CMD --fast --chainconfig "$BATS_TEST_DIRNAME/testdata/chain_config_dump-ok-custom.json" --ipcdisable  console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/customnet ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/customnet/chaindata ]
}

# chainconfig INVALID
@test "shouldnot migrate datadir /Ethereum/ -> /EthereumClassic/ from ETC3.3 schema | --chainconfig testdata/chain_config_dump-invalid-coinbase.json" {
	run "$CMD_DIR/gethc3.3" --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]

	run $GETH_CMD --fast --chainconfig "$BATS_TEST_DIRNAME/testdata/chain_config_dump-invalid-coinbase.json" console
	echo "$output"
	[ "$status" -gt 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]

	! [ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	! [ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet ]
	! [ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet/chaindata ]
}

# Don't migrate ETHF.
@test "shouldnot migrate datadir /Ethereum/ -> /EthereumClassic/ from ETHF1.5 schema" {
	run "$CMD_DIR/gethf1.5" --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/geth ]

	run $GETH_CMD --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/geth ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet ]
}

@test "shouldnot migrate datadir /Ethereum/ -> /EthereumClassic/ from ETHF1.6 schema" {
	run "$CMD_DIR/gethf1.6" --fast console
	echo "$output"
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/geth ]

	run $GETH_CMD --fast console
	echo "$output"	
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/geth ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet ]
}

@test "shouldnot migrate datadir /Ethereum/ -> /EthereumClassic/ from ETC3.3 schema without any ETC data in it" {
	
	mkdir -p "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/{chaindata,nodes,dapp,keystore} # We're on Bash 4.0, right?
	echo "bellow word" > "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/nodekey

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]

	run $GETH_CMD --fast console
	echo "$output"	
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/chaindata ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/nodes ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/dapp ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/keystore ]
	[ -f "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/nodekey ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet/chaindata ]
}

@test "shouldnot migrate datadir /Ethereum/ -> /EthereumClassic/ from ETHF schema without any ETH data in it" {
	
	mkdir -p "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/geth # We're on Bash 4.0, right?
	echo "bellow word" > "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/geth.ipc

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/geth ]

	run $GETH_CMD --fast console
	echo "$output"	
	[ "$status" -eq 0 ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/geth ]
	[ -f "$DATA_DIR_PARENT"/"$DATA_DIR_NAME_EX"/geth.ipc ]

	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME" ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet ]
	[ -d "$DATA_DIR_PARENT"/"$DATA_DIR_NAME"/mainnet/chaindata ]
}


