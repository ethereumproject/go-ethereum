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
	if [ "$(uname)" == "Darwin" ]; then
	    # Do something under Mac OS X platform
	    TEST_OS_HF=darwin
	    TEST_OS_C=osx
	    DATA_DIR_PARENT="$HOME/Library"
	elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
	    # Do something under GNU/Linux platform
	    TEST_OS_HF=linux
	    TEST_OS_C=linux
	    DATA_DIR_PARENT="$HOME"
	elif [ "$(expr substr $(uname -s) 1 10)" == "MINGW32_NT" ]; then
	    # Do something under 32 bits Windows NT platform
	    echo "Win32 bit not supported."
	    exit 1 
	elif [ "$(expr substr $(uname -s) 1 10)" == "MINGW64_NT" ]; then
	    # Do something under 64 bits Windows NT platform
	    TEST_OS_HF=windows
	    TEST_OS_C=win64
	    DATA_DIR_PARENT="$HOME/AppData/Roaming"
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

@test "migrate datadir Ethereum/ -> EthereumClassic/ with valid ETC3.3 config" {
	# Should create $HOME/Ethereum/chaindata,keystore,nodes,...
	run "$CMD_DIR/gethc3.3" --fast console
	echo "$output"

	[ -d "$DATA_DIR_PARENT"/Ethereum ]

	run $GETH_CMD --fast console
	echo "$output"

	[ -d "$DATA_DIR_PARENT"/EthereumClassic ]
	! [ -d "$DATA_DIR_PARENT"/Ethereum ]
}


