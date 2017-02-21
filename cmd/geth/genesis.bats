#!/usr/bin/env bats

setup() {
	DATA_DIR=`mktemp -d`
}

teardown() {
	rm -fr $DATA_DIR
}

@test "genesis" {
	echo '{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x20000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000042",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00"
}' > $DATA_DIR/genesis.json

	run ./geth --datadir $DATA_DIR init $DATA_DIR/genesis.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"successfully wrote genesis block and/or chain rule set"* ]]

	run ./geth --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[[ "$output" == *'"0x0000000000000042"'* ]]
}

@test "genesis empty chain config" {
	echo '{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x20000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000042",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00",
	"config"     : {}
}' > $DATA_DIR/genesis.json

	run ./geth --datadir $DATA_DIR init $DATA_DIR/genesis.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"successfully wrote genesis block and/or chain rule set"* ]]

	run ./geth --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[[ "$output" == *'"0x0000000000000042"'* ]]
}

@test "genesis chain config" {
	echo '{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x20000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000042",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00",
	"config"     : {}
}' > $DATA_DIR/genesis.json

	run ./geth --datadir $DATA_DIR init $DATA_DIR/genesis.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"successfully wrote genesis block and/or chain rule set"* ]]

	run ./geth --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[[ "$output" == *'"0x0000000000000042"'* ]]
}
