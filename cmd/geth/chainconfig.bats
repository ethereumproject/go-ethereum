#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
}

teardown() {
	rm -fr $DATA_DIR
}

## dump-chain-config JSON 

# Test dumping chain configuration to JSON file.
@test "chainconfig default dump" {
	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 dump-chain-config $DATA_DIR/dump.json
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
	run $GETH_CMD --datadir $DATA_DIR --testnet dump-chain-config $DATA_DIR/dump.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	[ -f $DATA_DIR/dump.json ]
	[ -d $DATA_DIR/morden ]

	run grep -R "morden" $DATA_DIR/dump.json
	[[ "$output" == *"\"id\": \"morden\"," ]]
}

@test "chainconfig customnet dump" {
	run $GETH_CMD --datadir $DATA_DIR --chain kittyCoin dump-chain-config $DATA_DIR/dump.json
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	
	# Ensure JSON dump file and named subdirectory (conainting chaindata) exists.
	[ -f $DATA_DIR/dump.json ]
	[ -d $DATA_DIR/kittyCoin ]
	[ -d $DATA_DIR/kittyCoin/chaindata ]
	[ -f $DATA_DIR/kittyCoin/chaindata/CURRENT ]
	[ -f $DATA_DIR/kittyCoin/chaindata/LOCK ]
	[ -f $DATA_DIR/kittyCoin/chaindata/LOG ]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR
	run grep -R "kittyCoin" $DATA_DIR/dump.json
	[ "$status" -eq 0 ]
	[[ "$output" == *"\"name\": \"kittyCoin\"," ]]
}

@test "chainconfig dump-chain-config JSON dump is usable as external chainconfig" {
# Same as 'chainconfig customnet dump'... higher complexity::more confidence
	run $GETH_CMD --datadir $DATA_DIR --chain kittyCoin dump-chain-config $DATA_DIR/dump.json
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	
	# Ensure JSON dump file and named subdirectory (conainting chaindata) exists.
	[ -f $DATA_DIR/dump.json ]
	[ -d $DATA_DIR/kittyCoin ]
	[ -d $DATA_DIR/kittyCoin/chaindata ]
	[ -f $DATA_DIR/kittyCoin/chaindata/CURRENT ]
	[ -f $DATA_DIR/kittyCoin/chaindata/LOCK ]
	[ -f $DATA_DIR/kittyCoin/chaindata/LOG ]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR
	run grep -R "kittyCoin" $DATA_DIR/dump.json
	[ "$status" -eq 0 ]
	[[ "$output" == *"\"name\": \"kittyCoin\"," ]]

# Ensure JSON file dump is loadable as external config
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $DATA_DIR/dump.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0000000000000042"* ]]
}

## load /data

# Test loading mainnet chain configuration from data/ JSON file.
# Test ensures 
# - can load default external JSON config
# - use datadir/subdir schema (/mainnet)
# - configured nonce matches external nonce (soft check since 42 is default, too)
@test "chainconfig configurable from default mainnet json file" {
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0000000000000042"* ]]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR.
	[ -d $DATA_DIR/mainnet ]
	[ -d $DATA_DIR/mainnet/chaindata ]
	[ -f $DATA_DIR/mainnet/chaindata/CURRENT ]
	[ -f $DATA_DIR/mainnet/chaindata/LOCK ]
	[ -f $DATA_DIR/mainnet/chaindata/LOG ]
	[ -d $DATA_DIR/mainnet/keystore ]
}

# Test loading testnet chain configuration from data/ JSON file.
# Test ensures
# - external chain config can determine chain configuration
# - use datadir/subdir schema (/morden)
@test "chainconfig configurable from default testnet json file" {
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/config/testnet.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x00006d6f7264656e"* ]]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR.
	[ -d $DATA_DIR/morden ]
	[ -d $DATA_DIR/morden ]
	[ -d $DATA_DIR/morden/chaindata ]
	[ -f $DATA_DIR/morden/chaindata/CURRENT ]
	[ -f $DATA_DIR/morden/chaindata/LOCK ]
	[ -f $DATA_DIR/morden/chaindata/LOG ]
	[ -d $DATA_DIR/morden/keystore ]
}

## load /testdata

# Test loading mainnet chain configuration from testdata/ JSON file.
# Test ensures
# - nonce is loaded from custom external rather than default (hard check)
@test "chainconfig configurable from testdata mainnet json file" {
	# Ensure non-default nonce 43 (42 is default).
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-ok.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0000000000000043"* ]]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR.
	[ -d $DATA_DIR/mainnet ]
	[ -d $DATA_DIR/mainnet ]
	[ -d $DATA_DIR/mainnet/chaindata ]
	[ -f $DATA_DIR/mainnet/chaindata/CURRENT ]
	[ -f $DATA_DIR/mainnet/chaindata/LOCK ]
	[ -f $DATA_DIR/mainnet/chaindata/LOG ]
	[ -d $DATA_DIR/mainnet/keystore ]
}

# Test loading customnet chain configuration from testdata/ JSON file.
# Test ensures
# - chain is loaded from custom external file and determines datadir/subdir scheme
@test "chainconfig configurable from testdata customnet json file" {
	# Ensure non-default nonce 43 (42 is default).
	# Ensure chain subdir is determined by config `id`
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-ok-custom.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0000000000000043"* ]]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR.
	[ -d $DATA_DIR/customnet ]
	[ -d $DATA_DIR/customnet/chaindata ]
	[ -f $DATA_DIR/customnet/chaindata/CURRENT ]
	[ -f $DATA_DIR/customnet/chaindata/LOCK ]
	[ -f $DATA_DIR/customnet/chaindata/LOG ]
	[ -d $DATA_DIR/customnet/keystore ]
}

# Test fails to load invalid chain configuration from testdata/ JSON file.
# Test ensures
# - external chain configuration should require JSON to parse
@test "chainconfig configuration fails with invalid-comment testdata mainnet json file" {
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-invalid-comment.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -eq 1 ]
}

# Test fails to load invalid chain configuration from testdata/ JSON file.
# Test ensures
# - external chain configuration should require JSON to parse
@test "chainconfig configuration fails with invalid-coinbase testdata mainnet json file" {
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-invalid-coinbase.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -gt 0 ]
}

freshconfig() {
	rm -fr $DATA_DIR
	DATA_DIR=`mktemp -d`
	cp "$BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json" "$DATA_DIR/"
}

@test "chainconfig configuration fails with any single invalid attribute key in otherwise valid json file" {
	declare -a OK_VARS=(id genesis chainConfig bootstrap) # 'name' can be blank... it's only for human consumption
	declare -a NOTOK_VARS=(did genes chainconfig bootsrap)
	
	cp $BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json $DATA_DIR/
	
	counter=0
	for var in "${OK_VARS[@]}"
	do
		sed -i.bu "s/${var}/${NOTOK_VARS[counter]}/" "$DATA_DIR/mainnet.json"
		
		run $GETH_CMD --datadir $DATA_DIR --chainconfig "$DATA_DIR/mainnet.json" console
		echo "$output"
		[ "$status" -gt 0 ]
		if [ ! "$status" -gt 0 ]; then
			echo "allowed invalid attribute: ${var}"
		fi		

		freshconfig()
		((counter=counter+1))
	done
}

@test "chainconfig configuration fails with any single invalid required attribute subkey in otherwise valid json file" {
	declare -a OK_VARS=(nonce gasLimit difficulty forks alloc balance Block Hash) # 'name' can be blank... it's only for human consumption
	declare -a NOTOK_VARS=(noneonce gasLim dificile knives allok bills Clock Cash)
	
	cp $BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json $DATA_DIR/
	
	counter=0
	for var in "${OK_VARS[@]}"
	do
		sed -i.bu "s/${var}/${NOTOK_VARS[counter]}/" "$DATA_DIR/mainnet.json"
		
		run $GETH_CMD --datadir $DATA_DIR --chainconfig "$DATA_DIR/mainnet.json" console
		echo "$output"
		[ "$status" -gt 0 ]
		if [ ! "$status" -gt 0 ]; then
			echo "allowed invalid attribute: ${var}"
		fi		

		freshconfig()
		((counter=counter+1))
	done
}

@test "chainconfig configuration fails with any single invalid required attribute value in otherwise valid json file" {
	declare -a    OK_VARS=(0x0000000000000042 0x0000000000000000000000000000000000000000000000000000000000001388 0x0000000000000000000000000000000000000000 enode homestead) # 'name' can be blank... it's only for human consumption
	declare -a NOTOK_VARS=(Ox0000000000000042 Ox0000000000000000000000000000000000000000000000000000000000001388 0x000000000000000000000000000000000000000  ewok  homeinbed)
	
	cp $BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json $DATA_DIR/
	
	counter=0
	for var in "${OK_VARS[@]}"
	do
		sed -i.bu "s/${var}/${NOTOK_VARS[counter]}/" "$DATA_DIR/mainnet.json"
		
		run $GETH_CMD --datadir $DATA_DIR --chainconfig "$DATA_DIR/mainnet.json" console
		echo "$output"
		[ "$status" -gt 0 ]
		if [ ! "$status" -gt 0 ]; then
			echo "allowed invalid attribute: ${var}"
		fi		

		freshconfig()
		((counter=counter+1))
	done
}











