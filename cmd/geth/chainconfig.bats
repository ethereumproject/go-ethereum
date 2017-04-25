#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
}

teardown() {
	rm -fr $DATA_DIR
}

# Test dumping chain configuration to JSON file.
@test "chainconfig default dump" {
	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 dumpChainConfig $DATA_DIR/dump.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	[ -f $DATA_DIR/dump.json ]
	[ -d $DATA_DIR/mainnet ]

	run grep -R "mainnet" $DATA_DIR/dump.json
	[ "$status" -eq 0 ]
	[[ "$output" == *"\"id\": \"mainnet\","* ]]
}

@test "chainconfig testnet dump" {
	run $GETH_CMD --datadir $DATA_DIR --testnet dumpChainConfig $DATA_DIR/dump.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	[ -f $DATA_DIR/dump.json ]
	[ -d $DATA_DIR/morden ]

	run grep -R "morden" $DATA_DIR/dump.json
	[[ "$output" == *"\"id\": \"morden\"," ]]
}

@test "chainconfig customnet dump" {
	run $GETH_CMD --datadir $DATA_DIR --chain kittyCoin dumpChainConfig $DATA_DIR/dump.json
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	
	# Ensure JSON dump file and named subdirectory (conainting chaindata) exists.
	[ -f $DATA_DIR/dump.json ]
	[ -d $DATA_DIR/kittyCoin ]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR
	run grep -R "kittyCoin" $DATA_DIR/dump.json
	[ "$status" -eq 0 ]
	[[ "$output" == *"\"name\": \"kittyCoin\"," ]]
}

# Test loading chain configuration from JSON file.
@test "chainconfig configurable from file" {
	# Ensure non-default nonce 43 (42 is default).
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-ok.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[[ "$output" == *"0x0000000000000043"* ]]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR.
	[ -d $DATA_DIR/mainnet ]
}


















