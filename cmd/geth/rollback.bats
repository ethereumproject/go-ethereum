#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
}

teardown() {
	rm -fr $DATA_DIR
}

@test "rollback 123" {
	run $GETH_CMD --datadir $DATA_DIR --exec="eth.getBlock(eth.defaultBlock).number" console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"0"* ]]

	echo "sleeping 1m..."
	run $GETH_CMD --datadir $DATA_DIR > /dev/null 2>&1 &
	sleep 1m; kill $!
	
	run $GETH_CMD --datadir $DATA_DIR --exec="eth.getBlock(eth.defaultBlock).number" console
	echo "$output"
	[ "$status" -eq 0 ]

	run $GETH_CMD rollback 42
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"SUCCESS"* ]]

	# run $GETH_CMD --datadir $DATA_DIR --exec="debug.setHead(123)" console
	run $GETH_CMD --datadir $DATA_DIR --exec="eth.getBlock(eth.defaultBlock).number" console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"42"* ]]	
}
