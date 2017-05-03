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

@test "aliases for directory-flags" {
	old_command_names=(datadir keystore docroot ipcpath)
	new_command_names=(data-dir keystore doc-root ipc-path)

	for var in "${old_command_names[@]}"
	do
		# hardcode --datadir/--data-dir
		run $GETH_CMD --$var $DATA_DIR/abc$var console
		echo "$output"

		[ "$status" -eq 0 ]
		[[ "$output" == *"Starting"* ]]
		[[ "$output" == *"Blockchain DB Version: "* ]]
		[[ "$output" == *"Starting Server"* ]]
	done

	for var in "${new_command_names[@]}"
	do
		run $GETH_CMD --$var $DATA_DIR/abc$var console
		[ "$status" -eq 0 ]
		[[ "$output" == *"Starting"* ]]
		[[ "$output" == *"Blockchain DB Version: "* ]]
		[[ "$output" == *"Starting Server"* ]]
	done
}

@test "alias for hyphenated-commands" {
	old_command_names=(nodiscover ipcdisable) # ... assuming that if two work, the rest will work.
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

# TODO
# This doesn't pass, and that's an issue.
# @test "displays help with valid command and invalid subcommand" {
# 	# lisr
# 	run $GETH_CMD account lisr
# 	echo "$output"

# 	[ "$status" -eq 3 ]
# 	[[ "$output" == *"SUBCOMMANDS"* ]]
# }