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

@test "personal_sign1" {
    run $GETH_CMD $GETH_OPTS \
				--password=<(echo $tesetacc_pass) \
        --exec="personal.sign('0xdeadbeef', '"$testacc"', '"$tesetacc_pass"');" console 2> /dev/null
		echo "$output"
		[ "$status" -eq 0 ]
    [[ "$output" =~ $regex_signature_success ]]
}

@test "personal_sign2" {
    run $GETH_CMD $GETH_OPTS \
				--password=<(echo $tesetacc_pass) \
        --exec="personal.sign(web3.fromAscii('Schoolbus'), '"$testacc"', '"$tesetacc_pass"');" console 2> /dev/null
		echo "$output"
		[ "$status" -eq 0 ]
    [[ "$output" =~ $regex_signature_success ]]
}

@test "personal_listAccounts" {
    run $GETH_CMD $GETH_OPTS \
				--password=<(echo $tesetacc_pass) \
        --exec="personal.listAccounts;" console 2> /dev/null
		echo "$output"
		[ "$status" -eq 0 ]
    [[ "$output" =~ $testacc ]]
}

@test "personal_lockAccount" {
    run $GETH_CMD $GETH_OPTS \
				--password=<(echo $tesetacc_pass) \
        --exec="personal.lockAccount('"$testacc"');" console 2> /dev/null
		echo "$output"
		[ "$status" -eq 0 ]
    [[ "$output" =~ 'true' ]]
}

@test "personal_unlockAccount" {
    run $GETH_CMD $GETH_OPTS \
				--password=<(echo $tesetacc_pass) \
        --exec="personal.lockAccount('"$testacc"') && personal.unlockAccount('"$testacc"', '"$tesetacc_pass"', 0);" console 2> /dev/null
		echo "$output"
		[ "$status" -eq 0 ]
    [[ "$output" =~ 'true' ]]
}
