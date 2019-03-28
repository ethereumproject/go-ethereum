#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/. $DATA_DIR/
}

teardown() {
	rm -fr $DATA_DIR
}

@test "rollback 42 | sets head from 384 -> 42" {
	run $GETH_CMD --datadir $DATA_DIR rollback 42
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"Local head full block"* ]] # original head
	[[ "$output" == *"384"* ]] # original head
	[[ "$output" == *"Success. Head block set to: 42"* ]]


	# Check that 'latest' block is 42.
	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec="eth.getBlock('latest').number" console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"42"* ]]
}

@test "rollback <noarg> | fails" {
	run $GETH_CMD --datadir $DATA_DIR rollback
	echo "$output"
	[ "$status" -gt 0 ]
	[[ "$output" == *'missing argument: use `rollback 12345` to specify required block number to roll back to'* ]] # original head
}

@test "rollback 420 | fails (420 > 384; block not yet in database)" {
	run $GETH_CMD --datadir $DATA_DIR rollback 420
	echo "$output"
	[ "$status" -gt 0 ]
	[[ "$output" == *'ERROR: Wanted rollback to set head to: 420, instead current head is: 384'* ]] # original head
}
