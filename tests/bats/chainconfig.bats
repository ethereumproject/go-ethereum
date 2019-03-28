#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
	default_mainnet_genesis_hash='"0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"'
	customnet_genesis_hash='"0x76bc07fbdfe084b9aff37425c24453f774d1945b28412a3b4b8c25d8d3c81df2"'
	testnet_genesis_hash='"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"'
	GENESIS_TESTNET=0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303
}

teardown() {
	rm -fr $DATA_DIR
	unset default_mainnet_genesis_hash
	unset customnet_genesis_hash
	unset GENESIS_TESTNET
}

## dump-chain-config JSON

# Test dumping chain configuration to JSON file.
@test "dump-chain-config | exit 0" {
	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 dump-chain-config $DATA_DIR/dump.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	[ -f $DATA_DIR/dump.json ]

	run grep -R "mainnet" $DATA_DIR/dump.json
	[ "$status" -eq 0 ]
	[[ "$output" == *"\"identity\": \"mainnet\","* ]]
}

@test "--testnet dump-chain-config | exit 0" {
	run $GETH_CMD --datadir $DATA_DIR --testnet dump-chain-config $DATA_DIR/dump.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	[ -f $DATA_DIR/dump.json ]

	run grep -R "morden" $DATA_DIR/dump.json
	echo "$output"
	[[ "$output" == *"\"identity\": \"morden\"," ]]
}

@test "--chain morden dump-chain-config | exit 0" {
	run $GETH_CMD --datadir $DATA_DIR --chain morden dump-chain-config $DATA_DIR/dump.json
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]
	[ -f $DATA_DIR/dump.json ]

	run grep -R "morden" $DATA_DIR/dump.json
	[[ "$output" == *"\"identity\": \"morden\"," ]]
}

@test "--chain kittyCoin dump-chain-config | exit !=0" {
	run $GETH_CMD --datadir $DATA_DIR --chain kittyCoin dump-chain-config $DATA_DIR/dump.json
	echo "$output"
	[ "$status" -ne 0 ]
}

@test "dump-chain-config | --chain | exit 0" {

	# Same as 'chainconfig customnet dump'... higher complexity::more confidence
	customnet="$DATA_DIR"/kitty
	mkdir -p "$customnet"

	run $GETH_CMD --datadir $DATA_DIR dump-chain-config "$customnet"/chain.json
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]

	[ -f "$customnet"/chain.json ]

	run grep -R "mainnet" "$customnet"/chain.json
	[ "$status" -eq 0 ]
	echo "$output"
	[[ "$output" == *"\"identity\": \"mainnet\"," ]]

	# Ensure JSON file dump is loadable as external config
	sed -i.bak s/mainnet/kitty/ "$customnet"/chain.json
	run $GETH_CMD --datadir $DATA_DIR --chain kitty --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
	[ "$status" -eq 0 ]
	echo "$output"
	[[ "$output" == *"$default_mainnet_genesis_hash"* ]]
}

@test "dump-chain-config | --chain privatenet.json | exit 0" {

	run $GETH_CMD --data-dir $DATA_DIR --chain morden dump-chain-config $DATA_DIR/privatenet.json
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]

	[ -f "$DATA_DIR"/privatenet.json ]

	run grep -R "morden" "$DATA_DIR"/privatenet.json
	[ "$status" -eq 0 ]
	echo "$output"
	[[ "$output" == *"\"identity\": \"morden\"," ]]

	# Ensure JSON file dump is loadable as external config
	sed -i.bak s/morden/kitty/ "$DATA_DIR"/privatenet.json
	run $GETH_CMD --datadir $DATA_DIR --chain "$DATA_DIR"/privatenet.json --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
	[ "$status" -eq 0 ]
	echo "$output"
	[[ "$output" == *"$testnet_genesis_hash"* ]]
	[[ "$output" == *"kitty"* ]]

	# ensure chain config file is copied to chaindata dir
	[ -f "$DATA_DIR"/kitty/chain.json ]
}

@test "--chain morden dump-chain-config | --chain == morden | exit 0" {

	# Same as 'chainconfig customnet dump'... higher complexity::more confidence
	customnet="$DATA_DIR"/kitty
	mkdir -p "$customnet"

	run $GETH_CMD --datadir $DATA_DIR --chain=morden dump-chain-config "$customnet"/chain.json
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]

	[ -f "$customnet"/chain.json ]

	run grep -R "morden" "$customnet"/chain.json
	[ "$status" -eq 0 ]
	echo "$output"
	[[ "$output" == *"\"identity\": \"morden\"," ]]

	# Ensure JSON file dump is loadable as external config
	sed -i.bak s/morden/kitty/ "$customnet"/chain.json
	run $GETH_CMD --datadir $DATA_DIR --chain kitty --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
	[ "$status" -eq 0 ]
	echo "$output"
	[[ "$output" == *"$testnet_genesis_hash"* ]]
}

# Dump morden and make customization and test that customizations are installed.
@test "--chain morden dump-chain-config | --chain -> kitty | exit 0" {

	# Same as 'chainconfig customnet dump'... higher complexity::more confidence
	customnet="$DATA_DIR"/kitty
	mkdir -p "$customnet"

	run $GETH_CMD --datadir $DATA_DIR --chain=morden dump-chain-config "$customnet"/chain.json
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"Wrote chain config file"* ]]

	[ -f "$customnet"/chain.json ]

	run grep -R "morden" "$customnet"/chain.json
	[ "$status" -eq 0 ]
	echo "$output"
	[[ "$output" == *"\"identity\": \"morden\"," ]]

	# Ensure JSON file dump is loadable as external config
	sed -i.bak s/morden/kitty/ "$customnet"/chain.json
	sed -i.bak s/identity/id/ "$customnet"/chain.json	# ensure identity is aliases to identity
	# remove starting nonce from external config
	# config file should still be valid, but genesis should have different hash than default morden genesis
	grep -v 'startingNonce' "$customnet"/chain.json > "$DATA_DIR"/stripped.json
	[ "$status" -eq 0 ]
	mv "$DATA_DIR"/stripped.json "$customnet"/chain.json

	sed -e '0,/2/ s/2/666/' "$customnet"/chain.json > "$DATA_DIR"/net_id.json
	mv "$DATA_DIR"/net_id.json "$customnet"/chain.json

	run $GETH_CMD --datadir $DATA_DIR --chain kitty --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
	[ "$status" -eq 0 ]
	echo "$output"
	[[ "$output" != *"$testnet_genesis_hash"* ]] # different genesis (stateRoot is diff)

	run $GETH_CMD --datadir $DATA_DIR --chain kitty status
	[ "$status" -eq 0 ]
	echo "$output"
	[[ "$output" != *"Network: 666"* ]] # new network id
}

## load /data

# Test loading mainnet chain configuration from data/ JSON file.
# Test ensures
# - can load default external JSON config
# - use datadir/subdir schema (/mainnet)
# - configured nonce matches external nonce (soft check since 42 is default, too)
@test "--chain=main | exit 0" {
	run $GETH_CMD --datadir $DATA_DIR --chain=main --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
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
@test "--chain morden | exit 0" {
	run $GETH_CMD --datadir $DATA_DIR --chain=morden --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"* ]]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR.
	[ -d $DATA_DIR/morden ]
	[ -d $DATA_DIR/morden/chaindata ]
	[ -f $DATA_DIR/morden/chaindata/CURRENT ]
	[ -f $DATA_DIR/morden/chaindata/LOCK ]
	[ -f $DATA_DIR/morden/chaindata/LOG ]
	[ -d $DATA_DIR/morden/keystore ]
}

# prove that trailing slashes in chain=val/ get removed harmlessly
@test "--chain='morden/' | exit 0" {

    # *nix path separator == /
    run $GETH_CMD --data-dir $DATA_DIR --chain=morden/ --exec 'eth.getBlock(0).hash' console
    echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"* ]]

	# Ensure we're using the --chain named subdirectory under main $DATA_DIR.
	[ -d $DATA_DIR/morden ]
	[ -d $DATA_DIR/morden/chaindata ]
}

## load /testdata

# Test loading customnet chain configuration from testdata/ JSON file.
# Test ensures
# - chain is loaded from custom external file and determines datadir/subdir scheme
@test "--chain customnet @ chain_config_dump-ok-custom.json | exit 0" {
	# Ensure non-default nonce 43 (42 is default).
	# Ensure chain subdir is determined by config `id`
	mkdir -p "$DATA_DIR"/customnet
	cp $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-ok-custom.json $DATA_DIR/customnet/chain.json
	sed -i.bak s/mainnet/customnet/ $DATA_DIR/customnet/chain.json
	run $GETH_CMD --datadir $DATA_DIR --chain=customnet  --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).hash' console
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

@test "--chain kitty --testnet | exit !=0" {
	mkdir -p $DATA_DIR/kitty
	cp $BATS_TEST_DIRNAME/../../core/config/mainnet.json $DATA_DIR/kitty/chain.json
	sed -i.bak s/mainnet/kitty/ $DATA_DIR/kitty/chain.json
	run $GETH_CMD --data-dir $DATA_DIR --chain kitty  --testnet console
	echo "$output"
	[ "$status" -ne 0 ]
	[[ "$output" == *"invalid flag or context value: used redundant/conflicting flags"* ]]
}

@test "--chain kitty --bootnodes=enode://e809c4a2fec7daed400e5e28564e23693b23b2cc5a019b612505631bbe7b9ccf709c1796d2a3d29ef2b045f210caf51e3c4f5b6d3587d43ad5d6397526fa6179@174.112.32.157:30303 | exit 0" {
	mkdir -p $DATA_DIR/kitty
	cp $BATS_TEST_DIRNAME/../../core/config/mainnet.json $DATA_DIR/kitty/chain.json
	cp $BATS_TEST_DIRNAME/../../core/config/mainnet_genesis.json $DATA_DIR/kitty/kitty_genesis.json
	cp $BATS_TEST_DIRNAME/../../core/config/mainnet_bootnodes.json $DATA_DIR/kitty/kitty_bootnodes.json
	cp $BATS_TEST_DIRNAME/../../core/config/mainnet_genesis_alloc.csv $DATA_DIR/kitty/kitty_genesis_alloc.csv
	sed -i.bak s/mainnet/kitty/ $DATA_DIR/kitty/chain.json
	sed -i.bak s/mainnet/kitty/ $DATA_DIR/kitty/kitty_genesis.json
	run $GETH_CMD --data-dir $DATA_DIR --chain kitty --bootnodes=enode://e809c4a2fec7daed400e5e28564e23693b23b2cc5a019b612505631bbe7b9ccf709c1796d2a3d29ef2b045f210caf51e3c4f5b6d3587d43ad5d6397526fa6179@174.112.32.157:30303 --exec 'exit;' console
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"Overwriting external bootnodes"* ]]
}

# Test fails to load invalid chain configuration from testdata/ JSON file.
# Test ensures
# - external chain configuration should require JSON to parse
@test "--chain kitty @ chain_config_dump-invalid-comment.json | exit !=0" {
	mkdir -p $DATA_DIR/kitty
	cp $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-invalid-comment.json $DATA_DIR/kitty/chain.json
	sed -i.bak s/mainnet/kitty/ $DATA_DIR/kitty/chain.json
	run $GETH_CMD --datadir $DATA_DIR --chain=kitty --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -ne 0 ]
}

# Test fails to load invalid chain configuration from testdata/ JSON file.
# Test ensures
# - external chain configuration should require JSON to parse
@test "--chain kitty testdata/chain_config_dump-invalid-coinbase.json | exit !=0" {
	mkdir -p $DATA_DIR/kitty
	cp $BATS_TEST_DIRNAME/../../cmd/geth/testdata/chain_config_dump-invalid-coinbase.json $DATA_DIR/kitty/chain.json
	sed -i.bak s/mainnet/kitty/ $DATA_DIR/kitty/chain.json
	run $GETH_CMD --datadir $DATA_DIR --chain kitty  --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'eth.getBlock(0).nonce' console
	echo "$output"
	[ "$status" -ne 0 ]
}

freshconfig() {
	rm -fr $DATA_DIR
	DATA_DIR=`mktemp -d`
	mkdir -p $DATA_DIR/kitty
	cp "$BATS_TEST_DIRNAME/../../core/config/mainnet.json" "$DATA_DIR/kitty/chain.json"
	sed -i.bak s/mainnet/kitty/ $DATA_DIR/kitty/chain.json
}

@test "--chain kitty @ sed -i s/key/badkey/ kitty/chain.json | exit !=0" {
	declare -a OK_VARS=(id genesis chainConfig bootstrap) # 'name' can be blank... it's only for human consumption
	declare -a NOTOK_VARS=(did genes chainconfig bootsrap)

	mkdir -p $DATA_DIR/kitty
	cp $BATS_TEST_DIRNAME/../../core/config/mainnet.json $DATA_DIR/kitty/chain.json
	sed -i.bak s/mainnet/kitty/ $DATA_DIR/kitty/chain.json

	counter=0
	for var in "${OK_VARS[@]}"
	do
		sed -i.bu "s/${var}/${NOTOK_VARS[counter]}/" "$DATA_DIR/kitty/chain.json"

		run $GETH_CMD --datadir $DATA_DIR --chain kitty console
		echo "$output"
		[ "$status" -ne 0 ]
		if [ ! "$status" -ne 0 ]; then
			echo "allowed invalid attribute: ${var}"
		fi

		freshconfig()
		((counter=counter+1))
	done
}

@test "--chain kitty @ sed -i s/subkey/badsubkey/ kitty/chain.json | exit !=0" {
	declare -a OK_VARS=(nonce gasLimit difficulty forks alloc balance Block Hash) # 'name' can be blank... it's only for human consumption
	declare -a NOTOK_VARS=(noneonce gasLim dificile knives allok bills Clock Cash)

	mkdir -p $DATA_DIR/kitty
	cp $BATS_TEST_DIRNAME/../../core/config/mainnet.json $DATA_DIR/kitty/chain.json
	sed -i.bak s/mainnet/kitty/ $DATA_DIR/kitty/chain.json

	counter=0
	for var in "${OK_VARS[@]}"
	do
		sed -i.bu "s/${var}/${NOTOK_VARS[counter]}/" "$DATA_DIR/kitty/chain.json"

		run $GETH_CMD --datadir $DATA_DIR --chain kitty console
		echo "$output"
		[ "$status" -ne 0 ]
		if [ ! "$status" -ne 0 ]; then
			echo "allowed invalid attribute: ${var}"
		fi

		freshconfig()
		((counter=counter+1))
	done
}

@test "--chain kitty @ sed -i s/value/badvalue/ kitty/chain.json | exit !=0" {
	declare -a    OK_VARS=(0x0000000000000042 0x0000000000000000000000000000000000000000000000000000000000001388 0x0000000000000000000000000000000000000000 enode homestead) # 'name' can be blank... it's only for human consumption
	declare -a NOTOK_VARS=(Ox0000000000000042 Ox0000000000000000000000000000000000000000000000000000000000001388 0x000000000000000000000000000000000000000  ewok  homeinbed)

	mkdir -p $DATA_DIR/kitty
	cp $BATS_TEST_DIRNAME/../../core/config/mainnet.json $DATA_DIR/kitty/chain.json
	sed -i.bak s/mainnet/kitty/ $DATA_DIR/kitty/chain.json

	counter=0
	for var in "${OK_VARS[@]}"
	do
		sed -i.bu "s/${var}/${NOTOK_VARS[counter]}/" "$DATA_DIR/kitty/chain.json"

		run $GETH_CMD --datadir $DATA_DIR --chain kitty console
		echo "$output"
		[ "$status" -ne 0 ]
		if [ ! "$status" -ne 0 ]; then
			echo "allowed invalid attribute: ${var}"
		fi

		freshconfig()
		((counter=counter+1))
	done
}
