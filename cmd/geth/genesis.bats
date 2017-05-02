#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
}

teardown() {
	rm -fr $DATA_DIR
}

# Test `init` command, which reads from a given genesis JSON file.
@test "genesis" {
	echo '{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x020000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000042",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00"
	}' > $DATA_DIR/genesis.json

	run $GETH_CMD --datadir $DATA_DIR init $DATA_DIR/genesis.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"successfully wrote genesis block and/or chain rule set"* ]]

	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[[ "$output" == *'"0x0000000000000042"'* ]]
}

@test "genesis empty chain config" {
	echo '{
	"alloc"      : {},
	"coinbase"   : "0x0000000000000000000000000000000000000000",
	"difficulty" : "0x020000",
	"extraData"  : "",
	"gasLimit"   : "0x2fefd8",
	"nonce"      : "0x0000000000000042",
	"mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
	"timestamp"  : "0x00",
	"config"     : {}
	}' > $DATA_DIR/genesis.json

	run $GETH_CMD --datadir $DATA_DIR init $DATA_DIR/genesis.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"successfully wrote genesis block and/or chain rule set"* ]]

	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[[ "$output" == *'"0x0000000000000042"'* ]]
}

@test "genesis alloc noprefixhex" {
	echo '{
    "nonce": "0x00006d6f7264656e",
    "timestamp": "",
    "parentHash": "",
    "extraData": "",
    "gasLimit": "0x2FEFD8",
    "difficulty": "0x020000",
    "mixhash": "0x00000000000000000000000000000000000000647572616c65787365646c6578",
    "coinbase": "",
    "alloc": {
      "0000000000000000000000000000000000000001": {
        "balance": "1"
      },
      "0000000000000000000000000000000000000002": {
        "balance": "1"
      },
      "0000000000000000000000000000000000000003": {
        "balance": "1"
      },
      "0000000000000000000000000000000000000004": {
        "balance": "15"
      },
      "102e61f5d8f9bc71d0ad4a084df4e65e05ce0e1c": {
        "balance": "1606938044258990275541962092341162602522202993782792835301376"
      }
    }
  }' > $DATA_DIR/genesis.json

	run $GETH_CMD --datadir $DATA_DIR init $DATA_DIR/genesis.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"successfully wrote genesis block and/or chain rule set"* ]]

	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[[ "$output" == *'"0x00006d6f7264656e"'* ]]

	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBalance("102e61f5d8f9bc71d0ad4a084df4e65e05ce0e1c").toString(10)' console
	echo "$output"
	[[ "$output" == *'"1606938044258990275541962092341162602522202993782792835301376"'* ]]

	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBalance("0000000000000000000000000000000000000004").toString(10)' console
	echo "$output"
	[[ "$output" == *'"15"'* ]]
}

@test "genesis alloc prefixhex" {
	echo '{
    "nonce": "0x00006d6f7264656e",
    "timestamp": "",
    "parentHash": "",
    "extraData": "",
    "gasLimit": "0x2FEFD8",
    "difficulty": "0x020000",
    "mixhash": "0x00000000000000000000000000000000000000647572616c65787365646c6578",
    "coinbase": "",
    "alloc": {
        	"0x3030303861636137636530353865656161303936": {"balance": "100000000000000000000000"},
            "0x3030306164613834383336326436613033393261": {"balance": "22100000000000000000000"},
            "0x3030306433393066623866386536353865616565": {"balance": "1000000000000000000000"},
            "0x3030313464396162393061303264373863326130": {"balance": "2000000000000000000000"},
            "0x3030313831323730616364323762386666363166": {"balance": "5348000000000000000000"}
    }
  }' > $DATA_DIR/genesis.json

	run $GETH_CMD --datadir $DATA_DIR init $DATA_DIR/genesis.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"successfully wrote genesis block and/or chain rule set"* ]]

	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[[ "$output" == *'"0x00006d6f7264656e"'* ]]

	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBalance("0x3030313831323730616364323762386666363166").toString(10)' console
	echo "$output"
	[[ "$output" == *'"5348000000000000000000"'* ]]

	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBalance("0x3030303861636137636530353865656161303936").toString(10)' console
	echo "$output"
	[[ "$output" == *'"100000000000000000000000"'* ]]
}

