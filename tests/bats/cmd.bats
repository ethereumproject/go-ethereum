#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
}

teardown() {
	rm -fr $DATA_DIR
}

@test "reset command" {
    cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/. $DATA_DIR/

    # Ensure chaindata dir exists before proof of removal.
    [ -d $DATA_DIR/mainnet/chaindata ]

    # Test with negative user response: SHOULD NOT remove chaindata/
    run $GETH_CMD --data-dir $DATA_DIR reset <<< $'no'
    [ "$status" -eq 0 ]
    [ -d $DATA_DIR/mainnet/chaindata ]

    # Test with affirmative user response: SHOULD remove chaindata/
    run $GETH_CMD --data-dir $DATA_DIR reset <<< $'y'
    [ "$status" -eq 0 ]
    ! [ -d $DATA_DIR/mainnet/chaindata ]
}
