#!/usr/bin/env bats

# Current build.
: ${GETH_CMD:=$GOPATH/bin/geth}
: ${GETH_OPTS:=--datadir $BATS_TMPDIR \
               		--lightkdf \
               		--verbosity 0 \
               		--display 0 \
               		--port 33333 \
               		--no-discover \
               		--keystore $GOPATH/src/github.com/ethereumproject/go-ethereum/accounts/testdata/keystore \
               		--unlock "f466859ead1932d743d622cb74fc058882e8648a" \
    }

setup() {
	# GETH_TMP_DATA_DIR=`mktemp -d`
	# mkdir "$BATS_TMPDIR/mainnet"
    testacc=f466859ead1932d743d622cb74fc058882e8648a
	tesetacc_pass=foobar
	regex_signature_success='0x[0-9a-f]{130}'
}

# teardown() {
# }

@test "eth_sign1" {
		run $GETH_CMD $GETH_OPTS \
				--password=<(echo $tesetacc_pass) \
        --exec="eth.sign('"$testacc"', '"$d"');" console 2> /dev/null
		echo "$output"
		[ "$status" -eq 0 ]
    [[ "$output" =~ $regex_signature_success ]]
}

@test "eth_sign2" {
    run $GETH_CMD $GETH_OPTS \
				--password=<(echo $tesetacc_pass) \
        --exec="eth.sign('"$testacc"', web3.fromAscii('Schoolbus'));" console 2> /dev/null
		echo "$output"
		[ "$status" -eq 0 ]
    [[ "$output" =~ $regex_signature_success ]]
}
