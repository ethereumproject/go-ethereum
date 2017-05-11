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

@test "exactly overlapping flags allowed: --chain=morden --testnet" {
	run $GETH_CMD --data-dir $DATA_DIR --testnet --chain=morden --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'exit' console
	echo "$output"
	[ "$status" -eq 0 ]
}

@test "overlapping flags not allowed: --chain=morden --dev" {
	run $GETH_CMD --data-dir $DATA_DIR --dev --chain=morden --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'exit' console
	echo "$output"
	[ "$status" -gt 0 ]
	[[ "$output" == *"invalid flag "* ]]
}

@test "custom testnet subdir --testnet --chain=morden2 | exit 0" {
	run $GETH_CMD --data-dir $DATA_DIR --testnet --chain=morden2 --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'exit' console
	echo "$output"
	[ "$status" -eq 0 ]	

	[ -d $DATA_DIR/morden2 ]
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

# Ensure --testnet and --chain=morden/testnet set up respective subdirs with default 'morden'
@test "--chain=testnet creates /morden subdir, activating testnet genesis" { # This is kind of weird, but it is expected.
	run $GETH_CMD --data-dir $DATA_DIR --chain=testnet --exec 'eth.getBlock(0).hash' console
	[ "$status" -eq 0 ]

	[[ "$output" == *"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"* ]]

	[ -d $DATA_DIR/morden ]
}

@test "--testnet creates /morden subdir, activating testnet genesis" {
	run $GETH_CMD --data-dir $DATA_DIR --testnet --exec 'eth.getBlock(0).hash' console
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"* ]]

	[ -d $DATA_DIR/morden ]
}

@test "--testnet --chain=faketestnet creates /faketestnet subdir, activating testnet genesis" {
	run $GETH_CMD --data-dir $DATA_DIR --testnet --chain=faketestnet --exec 'eth.getBlock(0).hash' console
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"* ]]

	[ -d $DATA_DIR/faketestnet ]
}

@test "--chain=morden creates /morden subdir, activating testnet genesis" {
	run $GETH_CMD --data-dir $DATA_DIR --chain=morden --exec 'eth.getBlock(0).hash' console
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"* ]]

	[ -d $DATA_DIR/morden ]
}

