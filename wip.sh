#!/usr/bin/env bash

outfile=tests.out

go test -v ./tests |& tee "$outfile"

declare -a keys=(PASS SKIP FAIL PANIC)
for KEY in "${keys[@]}"
do
	count=$(cat "$outfile" | grep "$KEY" | wc -l)
	echo "$KEY: $count"
done

