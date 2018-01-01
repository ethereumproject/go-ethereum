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
	[[ "$output" == *"Blockchain"* ]]
	[[ "$output" == *"Local head"* ]]
	[[ "$output" == *"Starting server"* ]]
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

@test "exactly overlapping flags not allowed: --chain=morden --testnet" {
	run $GETH_CMD --data-dir $DATA_DIR --testnet --chain=morden --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'exit' console
	echo "$output"
	[ "$status" -ne 0 ]
}

@test "custom testnet subdir --testnet --chain=morden2 | exit !=0" {
	run $GETH_CMD --data-dir $DATA_DIR --testnet --chain=morden2 --maxpeers 0 --nodiscover --nat none --ipcdisable --exec 'exit' console
	echo "$output"
	[ "$status" -ne 0 ]
	[[ "$output" == *"invalid flag "* ]]
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
	run $GETH_CMD --datadir $DATA_DIR --keystore $DATA_DIR/keyhere --exec 'exit;' console
	[ "$status" -eq 0 ]
	[ -d $DATA_DIR/keyhere ]

	# data-dir/datadir
	run $GETH_CMD --datadir $DATA_DIR --exec 'exit;' console
	[ "$status" -eq 0 ]
	[ -d $DATA_DIR ]

	run $GETH_CMD --data-dir $DATA_DIR --exec 'exit;' console
	[ "$status" -eq 0 ]
	[ -d $DATA_DIR ]

	# # ipc-path/ipcpath
	run $GETH_CMD --data-dir $DATA_DIR --ipc-path abc.ipc --exec 'exit;' console
	[ "$status" -eq 0 ]
    echo "$output"
	[[ "$output" == *"IPC endpoint opened"* ]]
	[[ "$output" == *"$DATA_DIR/mainnet/abc.ipc"* ]]

	run $GETH_CMD --data-dir $DATA_DIR --ipcpath $DATA_DIR/mainnet/abc.ipc --exec 'exit;' console
	[ "$status" -eq 0 ]
    echo "$output"
	[[ "$output" == *"IPC endpoint opened"* ]]
	[[ "$output" == *"$DATA_DIR/mainnet/abc.ipc"* ]]

	run $GETH_CMD --data-dir $DATA_DIR --ipc-path $DATA_DIR/mainnet/abc --exec 'exit;' console
	[ "$status" -eq 0 ]
    echo "$output"
	[[ "$output" == *"IPC endpoint opened"* ]]
	[[ "$output" == *"$DATA_DIR/mainnet/abc"* ]]

	run $GETH_CMD --data-dir $DATA_DIR --ipcpath $DATA_DIR/mainnet/abc --exec 'exit;' console
	[ "$status" -eq 0 ]
    echo "$output"
	[[ "$output" == *"IPC endpoint opened"* ]]
	[[ "$output" == *"$DATA_DIR/mainnet/abc"* ]]

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
	    [[ "$output" == *"Blockchain"* ]]
    	[[ "$output" == *"Local head"* ]]
    	[[ "$output" == *"Starting server"* ]]
	done

	for var in "${new_command_names[@]}"
	do
		run $GETH_CMD --data-dir $DATA_DIR --$var --exec 'exit' console
		[ "$status" -eq 0 ]
	    [[ "$output" == *"Blockchain"* ]]
    	[[ "$output" == *"Local head"* ]]
    	[[ "$output" == *"Starting server"* ]]
	done
}

@test "--cache 16 | exit 0" {
	run $GETH_CMD --data-dir $DATA_DIR --cache 17 --exec 'exit;' console
	[ "$status" -eq 0 ]
	[[ "$output" == *"Allotted"* ]]
	[[ "$output" == *"17MB"* ]]
	[[ "$output" == *"cache"* ]]
}

# Test `dump` command.
# All tests copy testdata/testdatadir to $DATA_DIR to ensure we're consistently testing against a not-only-init'ed chaindb.
@test "dump [noargs] | exit >0" {
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/. $DATA_DIR/
	run $GETH_CMD --data-dir $DATA_DIR dump
	echo "$output"
	[ "$status" -gt 0 ]
	[[ "$output" == *"invalid"* ]] # invalid use
}
@test "dump 0 [noaddress] | exit 0" {
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/. $DATA_DIR/
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
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/. $DATA_DIR/
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
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/. $DATA_DIR/
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
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/. $DATA_DIR/
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
	cp -a $BATS_TEST_DIRNAME/../../cmd/geth/testdata/testdatadir/. $DATA_DIR/
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

@test "--chain=morden creates /morden subdir, activating testnet genesis" {
	run $GETH_CMD --data-dir $DATA_DIR --chain=morden --exec 'eth.getBlock(0).hash' console
	[ "$status" -eq 0 ]
	[[ "$output" == *"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"* ]]

	[ -d $DATA_DIR/morden ]
}

# Command: status
@test "status command present and true for mainnet" {
	run $GETH_CMD --data-dir $DATA_DIR status
	[ "$status" -eq 0 ]

	# bug(whilei): warning: command substitution: ignored null byte in input
	# When I tried using heredoc and other more elegant solutions than line-by-line checking.

	# Chain Configuration Genesis
    echo "$output"
	[[ "$output" == *"mainnet"* ]]
	[[ "$output" == *"Ethereum Classic Mainnet"* ]]
	[[ "$output" == *"Genesis"* ]]
	[[ "$output" == *"0x0000000000000042"* ]]
	[[ "$output" == *"0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa"* ]]
	[[ "$output" == *"0x1388"* ]]
	[[ "$output" == *"0x0400000000"* ]]
	[[ "$output" == *"8893"* ]]

	# Run twice, because the second time will have set up database.
	run $GETH_CMD --data-dir $DATA_DIR --exec 'exit' console # set up db
	[ "$status" -eq 0 ]
	run $GETH_CMD --data-dir $DATA_DIR status
	[ "$status" -eq 0 ]

	# Chain database Genesis
    echo "$output"
	[[ "$output" == *"Genesis"* ]]
	[[ "$output" == *"0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"* ]]
	[[ "$output" == *"0x0000000000000000000000000000000000000000000000000000000000000000"* ]]
	[[ "$output" == *"66"* ]]
	[[ "$output" == *"17179869184"* ]]
	[[ "$output" == *"0x0000000000000000000000000000000000000000"* ]]
	[[ "$output" == *"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"* ]]
	[[ "$output" == *"5000"* ]]
	[[ "$output" == *"0"* ]]
	[[ "$output" == *"[17 187 232 219 78 52 123 78 140 147 124 28 131 112 228 181 237 51 173 179 219 105 203 219 122 56 225 229 11 27 130 250]"* ]]
}

@test "status command present and true for morden" {
	run $GETH_CMD --data-dir $DATA_DIR --chain morden status
	[ "$status" -eq 0 ]

	# bug(whilei): warning: command substitution: ignored null byte in input
	# When I tried using heredoc and other more elegant solutions than line-by-line checking.

	# Chain Configuration Genesis
	[[ "$output" == *"Genesis"* ]]
	[[ "$output" == *"morden"* ]]
	[[ "$output" == *"Morden Testnet"* ]]
	[[ "$output" == *"0x2FEFD8"* ]]
	[[ "$output" == *"0x020000"* ]]
	[[ "$output" == *"5"* ]]

	# Run twice, because the second time will have set up database.
	run $GETH_CMD --data-dir $DATA_DIR --chain morden --exec 'exit' console # set up db
	[ "$status" -eq 0 ]
	run $GETH_CMD --data-dir $DATA_DIR --chain morden status
	[ "$status" -eq 0 ]

	# Chain database Genesis
	[[ "$output" == *"Genesis"* ]]
	[[ "$output" == *"0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303"* ]]
	[[ "$output" == *"0x0000000000000000000000000000000000000000000000000000000000000000"* ]]
	[[ "$output" == *"120325427979630"* ]]
	[[ "$output" == *"131072"* ]]
	[[ "$output" == *"0x0000000000000000000000000000000000000000"* ]]
	[[ "$output" == *"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"* ]]
	[[ "$output" == *"3141592"* ]]
	[[ "$output" == *"0"* ]]
	[[ "$output" == *"[]"* ]]
}
