#!/usr/bin/env bats

: ${GETH_CMD:=$GOPATH/bin/geth}

setup() {
	DATA_DIR=`mktemp -d`
}

teardown() {
	rm -fr $DATA_DIR
}

# Invalid flags exit with exit code 1.
# Invalid commands and subcommands exit with exit code 3.

@test "runs with valid command" {
	run $GETH_CMD version
	[ "$status" -eq 0 ]
	[[ "$output" == *"Geth"* ]]
	[[ "$output" == *"Version: "* ]]
	[[ "$output" == *"Go Version: "* ]]
	[[ "$output" == *"OS: "* ]]
	[[ "$output" == *"GOPATH="* ]]
	[[ "$output" == *"GOROOT="* ]]
}

@test "displays help with invalid command" {
	run $GETH_CMD verison
	[ "$status" -eq 3 ]
	[[ "$output" == *"Invalid command"* ]]
	[[ "$output" == *"USAGE"* ]]
}

@test "displays help with invalid flag" {
	run $GETH_CMD --fat
	[ "$status" -eq 1 ]
	[[ "$output" == *"flag provided but not defined"* ]]
}

@test "runs with valid flag and valid command" {
	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'exit' console
	echo "$output"

	[ "$status" -eq 0 ]
	[[ "$output" == *"Starting"* ]]
	[[ "$output" == *"Blockchain DB Version: "* ]]
	[[ "$output" == *"Starting Server"* ]]
}

@test "displays help with invalid flag and valid command" {
	# --nodisco
	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodisco --nat none --ipcdisable --exec 'exit' console
	echo "$output"

	[ "$status" -eq 1 ]
	[[ "$output" == *"flag provided but not defined"* ]]
	[[ "$output" == *"USAGE"* ]]
}

@test "displays help with valid flag and invalid command" {
	# conso
	run $GETH_CMD --datadir $DATA_DIR --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'exit' conso
	echo "$output"

	[ "$status" -eq 3 ]
	[[ "$output" == *"Invalid command"* ]]
	[[ "$output" == *"USAGE"* ]]
}

# TODO
# This doesn't pass, and that's an issue.
# @test "displays help with valid command and invalid subcommand" {
# 	# lisr
# 	run $GETH_CMD account lisr
# 	echo "$output"

# 	[ "$status" -eq 3 ]
# 	[[ "$output" == *"SUBCOMMANDS"* ]]
# }

@test "aliasing directory flags: --data-dir==--datadir, --ipc-path==--ipcpath" {

	# keystore not hyphenated
	run $GETH_CMD --datadir $DATA_DIR --keystore $DATA_DIR/keyhere console
	[ "$status" -eq 0 ]
	[ -d $DATA_DIR/keyhere ]

	# data-dir/datadir
	run $GETH_CMD --datadir $DATA_DIR console
	[ "$status" -eq 0 ]
	[ -d $DATA_DIR ]

	run $GETH_CMD --data-dir $DATA_DIR console
	[ "$status" -eq 0 ]
	[ -d $DATA_DIR ]

	# # ipc-path/ipcpath
	run $GETH_CMD --data-dir $DATA_DIR --ipc-path abc.ipc console
	[ "$status" -eq 0 ]
	[[ "$output" == *"IPC endpoint opened: $DATA_DIR/mainnet/abc.ipc"* ]]

	run $GETH_CMD --data-dir $DATA_DIR --ipcpath $DATA_DIR/mainnet/abc.ipc console
	[ "$status" -eq 0 ]
	[[ "$output" == *"IPC endpoint opened: $DATA_DIR/mainnet/abc.ipc"* ]]

	run $GETH_CMD --data-dir $DATA_DIR --ipc-path $DATA_DIR/mainnet/abc console
	[ "$status" -eq 0 ]
	[[ "$output" == *"IPC endpoint opened: $DATA_DIR/mainnet/abc"* ]]

	run $GETH_CMD --data-dir $DATA_DIR --ipcpath $DATA_DIR/mainnet/abc console
	[ "$status" -eq 0 ]
	[[ "$output" == *"IPC endpoint opened: $DATA_DIR/mainnet/abc"* ]]
	
}

# ... assuming that if two work, the rest will work.
@test "aliasing hyphenated flags: --no-discover==--nodiscover, --ipc-disable==--ipcdisable | exit 0" {
	old_command_names=(nodiscover ipcdisable) 
	new_command_names=(no-discover ipc-disable)
	
	for var in "${old_command_names[@]}"
	do
		# hardcode --datadir/--data-dir
		run $GETH_CMD --datadir $DATA_DIR --$var --exec 'exit' console
		[ "$status" -eq 0 ]
		[[ "$output" == *"Starting"* ]]
		[[ "$output" == *"Blockchain DB Version: "* ]]
		[[ "$output" == *"Starting Server"* ]]
	done

	for var in "${new_command_names[@]}"
	do
		run $GETH_CMD --data-dir $DATA_DIR --$var --exec 'exit' console
		[ "$status" -eq 0 ]
		[[ "$output" == *"Starting"* ]]
		[[ "$output" == *"Blockchain DB Version: "* ]]
		[[ "$output" == *"Starting Server"* ]]
	done
}

@test "--cache 16 | exit 0" {
	run $GETH_CMD --data-dir $DATA_DIR --cache 17 console
	[ "$status" -eq 0 ]
	[[ "$output" == *"Alloted 17MB cache"* ]]
}


# Test `dump` command.
# All tests copy testdata/testdatadir to $DATA_DIR to ensure we're consistently testing against a not-only-init'ed chaindb.
@test "dump [noargs] | exit >0" {
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/ $DATA_DIR/ 
	run $GETH_CMD --data-dir $DATA_DIR dump
	echo "$output"
	[ "$status" -gt 0 ]
	[[ "$output" == *"invalid"* ]] # invalid use
}
@test "dump 0 [noaddress] | exit 0" {
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/ $DATA_DIR/
	run $GETH_CMD --data-dir $DATA_DIR dump 0
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"root"* ]] # block state root
	[[ "$output" == *"balance"* ]]
	[[ "$output" == *"accounts"* ]]
	[[ "$output" == *"d7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544"* ]] # block state root
	[[ "$output" == *"ffec0913c635baca2f5e57a37aa9fb7b6c9b6e26"* ]] # random hex address existing in genesis 
	[[ "$output" == *"253319000000000000000"* ]] # random address balance existing in genesis 
}
@test "dump 0 fff7ac99c8e4feb60c9750054bdc14ce1857f181 | exit 0" {
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/ $DATA_DIR/
	run $GETH_CMD --data-dir $DATA_DIR dump 0 fff7ac99c8e4feb60c9750054bdc14ce1857f181
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"root"* ]] # block state root
	[[ "$output" == *"balance"* ]]
	[[ "$output" == *"accounts"* ]]
	[[ "$output" == *"d7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544"* ]] # block state root
	[[ "$output" == *"fff7ac99c8e4feb60c9750054bdc14ce1857f181"* ]] # hex address
	[[ "$output" == *"1000000000000000000000"* ]] # address balance
}
@test "dump 0 fff7ac99c8e4feb60c9750054bdc14ce1857f181,0xffe8cbc1681e5e9db74a0f93f8ed25897519120f | exit 0" {
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/ $DATA_DIR/
	run $GETH_CMD --data-dir $DATA_DIR dump 0 fff7ac99c8e4feb60c9750054bdc14ce1857f181,0xffe8cbc1681e5e9db74a0f93f8ed25897519120f
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"root"* ]] # block state root
	[[ "$output" == *"balance"* ]]
	[[ "$output" == *"accounts"* ]]
	[[ "$output" == *"d7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544"* ]] # block 0  state root
	
	[[ "$output" == *"fff7ac99c8e4feb60c9750054bdc14ce1857f181"* ]] # hex address
	[[ "$output" == *"1000000000000000000000"* ]] # address balance

	[[ "$output" == *"ffe8cbc1681e5e9db74a0f93f8ed25897519120f"* ]] # hex address
	[[ "$output" == *"1507000000000000000000"* ]] # address balance
}
@test "dump 0,1 fff7ac99c8e4feb60c9750054bdc14ce1857f181,0xffe8cbc1681e5e9db74a0f93f8ed25897519120f | exit 0" {
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/ $DATA_DIR/
	run $GETH_CMD --data-dir $DATA_DIR dump 0,1 fff7ac99c8e4feb60c9750054bdc14ce1857f181,0xffe8cbc1681e5e9db74a0f93f8ed25897519120f
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"root"* ]] # block state root
	[[ "$output" == *"balance"* ]]
	[[ "$output" == *"accounts"* ]]
	
	[[ "$output" == *"d7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544"* ]] # block 0  state root
	[[ "$output" == *"d67e4d450343046425ae4271474353857ab860dbc0a1dde64b41b5cd3a532bf3"* ]] # block 1  state root
	
	[[ "$output" == *"fff7ac99c8e4feb60c9750054bdc14ce1857f181"* ]] # hex address
	[[ "$output" == *"1000000000000000000000"* ]] # address balance

	[[ "$output" == *"ffe8cbc1681e5e9db74a0f93f8ed25897519120f"* ]] # hex address
	[[ "$output" == *"1507000000000000000000"* ]] # address balance
}
@test "dump 0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3 fff7ac99c8e4feb60c9750054bdc14ce1857f181,0xffe8cbc1681e5e9db74a0f93f8ed25897519120f | exit 0" {
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/ $DATA_DIR/
	run $GETH_CMD --data-dir $DATA_DIR dump 0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3 fff7ac99c8e4feb60c9750054bdc14ce1857f181,0xffe8cbc1681e5e9db74a0f93f8ed25897519120f
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" == *"root"* ]] # block state root
	[[ "$output" == *"balance"* ]]
	[[ "$output" == *"accounts"* ]]
	
	[[ "$output" == *"d7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544"* ]] # block 0  state root
	
	[[ "$output" == *"fff7ac99c8e4feb60c9750054bdc14ce1857f181"* ]] # hex address
	[[ "$output" == *"1000000000000000000000"* ]] # address balance

	[[ "$output" == *"ffe8cbc1681e5e9db74a0f93f8ed25897519120f"* ]] # hex address
	[[ "$output" == *"1507000000000000000000"* ]] # address balance
}











