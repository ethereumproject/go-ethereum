#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
	# Default constants.
	GENESIS_MAINNET=0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3
	GENESIS_TESTNET=0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303
}

teardown() {
	rm -fr $DATA_DIR
	unset GENESIS_MAINNET
	unset GENESIS_TESTNET
}

# Mainnet.
@test "defaults: genesis block hash mainnet constant @ _" {
	run $GETH_CMD --data-dir $DATA_DIR --exec 'eth.getBlock(0).hash' console
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *'"0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"'* ]]
}

# Testnet.
@test "defaults: genesis block hash constant @ --chain=morden" {
	run $GETH_CMD --chain=morden --data-dir $DATA_DIR --exec 'eth.getBlock(0).hash' console
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *'"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"'* ]]
}

@test "defaults: genesis block hash constant @ --chain=testnet" {
	run $GETH_CMD --chain=testnet --data-dir $DATA_DIR --exec 'eth.getBlock(0).hash' console
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *'"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"'* ]]
}

@test "defaults: genesis block hash constant @ --testnet" {
	run $GETH_CMD --testnet --data-dir $DATA_DIR --exec 'eth.getBlock(0).hash' console
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *'"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"'* ]]
}
