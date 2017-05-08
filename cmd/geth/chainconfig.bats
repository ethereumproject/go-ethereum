#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
	default_mainnet_genesis_hash='"0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"'
	customnet_genesis_hash='"0x76bc07fbdfe084b9aff37425c24453f774d1945b28412a3b4b8c25d8d3c81df2"'
}

teardown() {
	rm -fr $DATA_DIR
	unset default_mainnet_genesis_hash
	unset customnet_genesis_hash
}

## dump-chain-config JSON 

# Test dumping chain configuration to JSON file.
@test "dump-chain-config | exit 0" {
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

@test "--testnet dump-chain-config | exit 0" {
	run $GETH_CMD --datadir $DATA_DIR --testnet dump-chain-config $DATA_DIR/dump.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	[ -f $DATA_DIR/dump.json ]
	[ -d $DATA_DIR/morden ]

	run grep -R "morden" $DATA_DIR/dump.json
	[[ "$output" == *"\"id\": \"morden\"," ]]
}

@test "--chain kittyCoin dump-chain-config | exit 0" {
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

@test "dump-chain-config | --chain-config | exit 0" {
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
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $DATA_DIR/dump.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"$default_mainnet_genesis_hash"* ]]

	# Ensure we can specify this chaindata subdir with --chain comand.
	run $GETH_CMD --datadir $DATA_DIR --chain kittyCoin --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"$default_mainnet_genesis_hash"* ]]	

}

## load /data

# Test loading mainnet chain configuration from data/ JSON file.
# Test ensures 
# - can load default external JSON config
# - use datadir/subdir schema (/mainnet)
# - configured nonce matches external nonce (soft check since 42 is default, too)
@test "--chain-config config/mainnet.json | exit 0" {
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"$default_mainnet_genesis_hash"* ]]

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
@test "--chain-config config/testnet.json | exit 0" {
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

# Test loading customnet chain configuration from testdata/ JSON file.
# Test ensures
# - chain is loaded from custom external file and determines datadir/subdir scheme
@test "--chain-config testdata/chain_config_dump-ok-custom.json | exit 0" {
	# Ensure non-default nonce 43 (42 is default).
	# Ensure chain subdir is determined by config `id`
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-ok-custom.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"$customnet_genesis_hash"* ]]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR.
	[ -d $DATA_DIR/customnet ]
	[ -d $DATA_DIR/customnet/chaindata ]
	[ -f $DATA_DIR/customnet/chaindata/CURRENT ]
	[ -f $DATA_DIR/customnet/chaindata/LOCK ]
	[ -f $DATA_DIR/customnet/chaindata/LOG ]
	[ -d $DATA_DIR/customnet/keystore ]
}

@test "--chain-config config/testnet.json --testnet | exit >1" {
	run $GETH_CMD --data-dir $DATA_DIR --chain-config $BATS_TEST_DIRNAME/../../cmd/geth/config/testnet.json --testnet console
	echo "$output"
	[ "$status" -gt 0 ]
	[[ "$output" == *"invalid flag or context value: invalid chainID: external and flag configurations are conflicting, please use only one"* ]]
}

@test "--chain-config config/mainnet.json --chain mainnet | exit >1" {
	run $GETH_CMD --data-dir $DATA_DIR --chain-config $BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json --chain mainnet console
	echo "$output"
	[ "$status" -gt 0 ]
	[[ "$output" == *"invalid flag or context value: invalid chainID: external and flag configurations are conflicting, please use only one"* ]]
}

@test "--chain-config config/mainnet.json --chain kittyCoin | exit >1" {
	run $GETH_CMD --data-dir $DATA_DIR --chain-config $BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json --chain kittyCoin console
	echo "$output"
	[ "$status" -gt 0 ]
	[[ "$output" == *"invalid flag or context value: invalid chainID: external and flag configurations are conflicting, please use only one"* ]]
}

@test "--chain-config config/mainnet.json --bootnodes=enode://e809c4a2fec7daed400e5e28564e23693b23b2cc5a019b612505631bbe7b9ccf709c1796d2a3d29ef2b045f210caf51e3c4f5b6d3587d43ad5d6397526fa6179@174.112.32.157:30303 | exit >1" {
	run $GETH_CMD --data-dir $DATA_DIR --chain-config $BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json --bootnodes=enode://e809c4a2fec7daed400e5e28564e23693b23b2cc5a019b612505631bbe7b9ccf709c1796d2a3d29ef2b045f210caf51e3c4f5b6d3587d43ad5d6397526fa6179@174.112.32.157:30303 console
	echo "$output"
	[ "$status" -gt 0 ]
	[[ "$output" == *"Conflicting --chain-config and --bootnodes flags."* ]]
}

# Test fails to load invalid chain configuration from testdata/ JSON file.
# Test ensures
# - external chain configuration should require JSON to parse
@test "--chain-config testdata/chain_config_dump-invalid-comment.json | exit >0" {
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-invalid-comment.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -gt 0 ]
}

# Test fails to load invalid chain configuration from testdata/ JSON file.
# Test ensures
# - external chain configuration should require JSON to parse
@test "--chain-config testdata/chain_config_dump-invalid-coinbase.json | exit >0" {
	run $GETH_CMD --datadir $DATA_DIR --chainconfig $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-invalid-coinbase.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -gt 0 ]
}

freshconfig() {
	rm -fr $DATA_DIR
	DATA_DIR=`mktemp -d`
	cp "$BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json" "$DATA_DIR/"
}

@test "--chain-config @ sed -i s/key/badkey/ config/mainnet.json | exit >0" {
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

@test "--chain-config @ sed -i s/subkey/badsubkey/ config/mainnet.json | exit >0" {
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

@test "--chain-config @ sed -i s/value/badvalue/ config/mainnet.json | exit >0" {
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

@test "--chain-config config/mainnet.json && --chain-config testdata/chain_config_dump-ok-custom.json | does overwrite genesis" {

	# establish default mainnet chaindata
	run $GETH_CMD --data-dir $DATA_DIR --chain-config $BATS_TEST_DIRNAME/../../cmd/geth/config/mainnet.json --exec="eth.getBlock(0).hash" console
	echo "$output"
	[ "$status" -eq 0 ]
	# ensure genesis block is mainnet default (by hash)
	[[ "$output" == *"$default_mainnet_genesis_hash"* ]]

	# start with --chain-config customnet data
	run $GETH_CMD --data-dir $DATA_DIR --chain-config $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-ok-custom.json --exec="eth.getBlock(0).hash" console
	echo "$output"
	[ "$status" -eq 0 ]
	# ensure genesis block is different	(by hash)
	[[ "$output" == *"$customnet_genesis_hash"* ]]
}











