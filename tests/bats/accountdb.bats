#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
	mkdir "$DATA_DIR/mainnet"
}

teardown() {
	rm -fr $DATA_DIR
}

@test "account <no command> yields help/usage" {
	run $GETH_CMD --datadir $DATA_DIR account
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"USAGE"* ]]
}

@test "account list yields <blank> (no accounts)" {
	run $GETH_CMD --datadir $DATA_DIR account list
	echo "$output"

	[ "$status" -eq 0 ]
	[ "$output" = "" ]
}

@test "account list testdata keystore (db)" {
	cp -R $BATS_TEST_DIRNAME/../../accounts/testdata/keystore $DATA_DIR/mainnet

	# Create index.
	run $GETH_CMD --data-dir $DATA_DIR --index-accounts account index
	[ "$status" -eq 0 ]

	run $GETH_CMD --datadir $DATA_DIR --index-accounts account list
	echo "$output"

	[ "$status" -eq 0 ]
	# Note: cachedb stores files relative to keystore dir, while mem stores absolute paths.
	# (whilei): I prefer relative to dir in case user wants to move keystore then paths will remain stable and not need to be reindexed.
	# Also, both mem and cache implementations only scan 1 level (not recursively) and thus absolute path is unnecessarily verbose.
	[ "${lines[0]}" == "Account #0: {7ef5a6135f1fd6a02593eedc869c6d41d934aef8} UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8" ]
	[ "${lines[1]}" == "Account #1: {f466859ead1932d743d622cb74fc058882e8648a} aaa" ]
	[ "${lines[2]}" == "Account #2: {289d485d9771714cce91d3393d764e1311907acc} zzz" ]
}

@test "account create (db)" {
	run $GETH_CMD --datadir $DATA_DIR --lightkdf --index-accounts account new <<< $'secret\nsecret\n'
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" =~ Address:.\{[0-9a-f]{40}\}$ ]]
}

@test "account create pass mismatch (db)" {
	run $GETH_CMD --datadir $DATA_DIR --lightkdf --index-accounts account new <<< $'secret\nother\n'
	echo "$output"

	[ "$status" -ne 0 ]
	[[ "$output" == *"Passphrases do not match" ]]
}

@test "account update pass (db)" {
	cp -R $BATS_TEST_DIRNAME/../../accounts/testdata/keystore $DATA_DIR/mainnet
	run $GETH_CMD --keystore $DATA_DIR/mainnet/keystore --index-accounts account index
	[ "$status" -eq 0 ]

	# Create index.
	run $GETH_CMD --data-dir $DATA_DIR --index-accounts account index
	[ "$status" -eq 0 ]

	run $GETH_CMD --datadir $DATA_DIR --lightkdf --index-accounts account update f466859ead1932d743d622cb74fc058882e8648a <<< $'foobar\nother\nother\n'
	echo "$output"

	[ "$status" -eq 0 ]
}

@test "account import (db)" {
	run $GETH_CMD --datadir $DATA_DIR --lightkdf --index-accounts wallet import $GOPATH/src/github.com/ethereumproject/go-ethereum/cmd/geth/testdata/guswallet.json <<< $'foo\n'
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Address: {d4584b5f6229b7be90727b0fc8c6b91bb427821f}" ]]

	echo "=== data dir files:"
	ls $DATA_DIR/mainnet/keystore
	[ $(ls $DATA_DIR/mainnet/keystore | wc -l) -eq 2 ] # keyfile + accounts.db
}

@test "account import pass mismatch (db)" {
	run $GETH_CMD --datadir $DATA_DIR --lightkdf --index-accounts wallet import $GOPATH/src/github.com/ethereumproject/go-ethereum/cmd/geth/testdata/guswallet.json <<< $'wrong\n'
	echo "$output"

	[ "$status" -ne 0 ]
	[[ "$output" == *"could not decrypt key with given passphrase" ]]
}

@test "account unlock (db)" {
	cp -R $BATS_TEST_DIRNAME/../../accounts/testdata/keystore $DATA_DIR/mainnet
	run $GETH_CMD --keystore $DATA_DIR/mainnet/keystore --index-accounts account index
	[ "$status" -eq 0 ]
	touch $DATA_DIR/empty.js

	# Create index.
	run $GETH_CMD --data-dir $DATA_DIR --index-accounts account index
	[ "$status" -eq 0 ]

	run $GETH_CMD --datadir $DATA_DIR --index-accounts --keystore="$DATA_DIR"/mainnet/keystore\
	 --nat none --nodiscover --dev --unlock f466859ead1932d743d622cb74fc058882e8648a js $DATA_DIR/empty.js <<< $'foobar\n'
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Unlocked account f466859ead1932d743d622cb74fc058882e8648a"* ]]
}

@test "account unlock pass mismatch (db)" {
	cp -R $BATS_TEST_DIRNAME/../../accounts/testdata/keystore $DATA_DIR/mainnet
	run $GETH_CMD --keystore $DATA_DIR/mainnet/keystore --index-accounts account index
	[ "$status" -eq 0 ]
	touch $DATA_DIR/empty.js

	# Create index.
	run $GETH_CMD --data-dir $DATA_DIR --index-accounts account index
	[ "$status" -eq 0 ]

	run $GETH_CMD --datadir $DATA_DIR --index-accounts --keystore="$DATA_DIR"/mainnet/keystore\
	 --nat none --nodiscover --dev --unlock f466859ead1932d743d622cb74fc058882e8648a js $DATA_DIR/empty.js <<< $'wrong1\nwrong2\nwrong3\n'
	echo "$output"

	[ "$status" -ne 0 ]
	[[ "$output" == *"Failed to unlock account f466859ead1932d743d622cb74fc058882e8648a (could not decrypt key with given passphrase)" ]]
}

@test "account unlock multiple (db)" {
	cp -R $BATS_TEST_DIRNAME/../../accounts/testdata/keystore $DATA_DIR/mainnet
	run $GETH_CMD --keystore $DATA_DIR/mainnet/keystore --index-accounts account index
	[ "$status" -eq 0 ]
	touch $DATA_DIR/empty.js

	# Create index.
	run $GETH_CMD --data-dir $DATA_DIR --index-accounts account index
	[ "$status" -eq 0 ]

	run $GETH_CMD --datadir $DATA_DIR --index-accounts --keystore="$DATA_DIR"/mainnet/keystore\
	 --nat none --nodiscover --dev --unlock 0,2 js $DATA_DIR/empty.js <<< $'foobar\nfoobar\n'
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Unlocked account 7ef5a6135f1fd6a02593eedc869c6d41d934aef8"* ]]
	[[ "$output" == *"Unlocked account 289d485d9771714cce91d3393d764e1311907acc"* ]]
}

@test "account unlock multiple with pass file (db)" {
	cp -R $BATS_TEST_DIRNAME/../../accounts/testdata/keystore $DATA_DIR/mainnet
	run $GETH_CMD --keystore $DATA_DIR/mainnet/keystore --index-accounts account index
	[ "$status" -eq 0 ]
	touch $DATA_DIR/empty.js

	# Create index.
	run $GETH_CMD --data-dir $DATA_DIR --index-accounts account index
	[ "$status" -eq 0 ]

	echo $'foobar\nfoobar\nfoobar\n' > $DATA_DIR/pass.txt

	run $GETH_CMD --datadir $DATA_DIR --index-accounts --keystore="$DATA_DIR"/mainnet/keystore\
	 --nat none --nodiscover --dev --password $DATA_DIR/pass.txt --unlock 0,2 js $DATA_DIR/empty.js
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Unlocked account 7ef5a6135f1fd6a02593eedc869c6d41d934aef8"* ]]
	[[ "$output" == *"Unlocked account 289d485d9771714cce91d3393d764e1311907acc"* ]]
}

@test "account unlock multiple with wrong pass file (db)" {
	cp -R $BATS_TEST_DIRNAME/../../accounts/testdata/keystore $DATA_DIR/mainnet
	run $GETH_CMD --keystore $DATA_DIR/mainnet/keystore --index-accounts account index
	[ "$status" -eq 0 ]
	touch $DATA_DIR/empty.js

	# Create index.
	run $GETH_CMD --data-dir $DATA_DIR --index-accounts account index
	[ "$status" -eq 0 ]

	echo $'wrong\nwrong\nwrong\n' > $DATA_DIR/pass.txt

	run $GETH_CMD --datadir $DATA_DIR --nat none --index-accounts --keystore="$DATA_DIR"/mainnet/keystore\
	 --nat none --nodiscover --dev --password $DATA_DIR/pass.txt --unlock 0,2 js $DATA_DIR/empty.js
	echo "$output"

	[ "$status" -ne 0 ]
	[[ "$output" == *"Failed to unlock account 0 (could not decrypt key with given passphrase)" ]]
}

@test "account unlock ambiguous (db)" {
	cp -R $BATS_TEST_DIRNAME/../../accounts/testdata/dupes $DATA_DIR/mainnet/store
	touch $DATA_DIR/empty.js

	run $GETH_CMD --keystore $DATA_DIR/mainnet/store --index-accounts account index
	[ "$status" -eq 0 ]

	run $GETH_CMD --datadir $DATA_DIR --keystore $DATA_DIR/mainnet/store --index-accounts --keystore="$DATA_DIR"/mainnet/store\
	 --nat none --nodiscover --dev --unlock f466859ead1932d743d622cb74fc058882e8648a js $DATA_DIR/empty.js <<< $'foobar\n'$DATA_DIR/store/1
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Multiple key files exist for address f466859ead1932d743d622cb74fc058882e8648a:"* ]]
	[[ "$output" == *"Your passphrase unlocked 1"* ]]
	[[ "$output" == *"Unlocked account f466859ead1932d743d622cb74fc058882e8648a"* ]]
}

@test "account unlock ambiguous pass mismatch (db)" {
	cp -R $BATS_TEST_DIRNAME/../../accounts/testdata/dupes $DATA_DIR/mainnet/store
	touch $DATA_DIR/empty.js

	run $GETH_CMD --keystore $DATA_DIR/mainnet/store --index-accounts account index
	[ "$status" -eq 0 ]

	run $GETH_CMD --datadir $DATA_DIR --keystore $DATA_DIR/mainnet/store --index-accounts --keystore="$DATA_DIR"/mainnet/store\
	 --nat none --nodiscover --dev --unlock f466859ead1932d743d622cb74fc058882e8648a js $DATA_DIR/empty.js <<< $'wrong\n'$DATA_DIR/store/1
	echo "$output"

	[ "$status" -ne 0 ]
	[[ "$output" == *"Multiple key files exist for address f466859ead1932d743d622cb74fc058882e8648a:"* ]]
	[[ "$output" == *"None of the listed files could be unlocked."* ]]
}
