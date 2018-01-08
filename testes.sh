#!/usr/bin/env bash

set -e

OS='Unknown OS'
case "$(uname -s)" in
    Darwin)
        OS="Mac";;

    Linux)
        OS="Linux";;

    CYGWIN*|MINGW32*|MSYS*)
        OS="Windows";;

    *)
        echo 'Unknown OS'
        exit;;
esac

if [ "$OS" == "Windows" ]; then
    cd %GOPATH%\src\github.com\ethereumproject
    git clone https://github.com/ethereumproject/sputnikvm-ffi.git
    cd sputnikvm-ffi\c\ffi
    cargo build --release
    copy %GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\c\ffi\target\release\sputnikvm.lib \
        %GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\c\sputnikvm.lib

    cd %GOPATH%\src\github.com\ethereumproject\go-ethereum\cmd\geth
    set CGO_LDFLAGS=-Wl,--allow-multiple-definition \
        %GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\c\sputnikvm.lib -lws2_32 -luserenv
    go build -tags=sputnikvm .
else
    ep_gopath=$GOPATH/src/github.com/ethereumproject
	sputnikffi_path="$ep_gopath/sputnikvm-ffi"

	# If sputnikvmffi has not already been cloned/existing
	if [ -d "$sputnikffi_path" ]; then
		# Ensure git is happening in svm-ffi.
		# Update if .git exists, otherwise don't try updating. We could possibly handle git-initing and adding remote but seems
		# like an edge case.
		if [ -d "$sputnikffi_path/.git" ]; then
			# Just grab the first remote name that shows up.
			remote_name=$(git remote -v | head -1 | awk '{print $1;}')
        	echo "Updating SputnikVM FFI..."
        	cd $sputnikffi_path
        	git pull "$remote_name" master
		fi
    else
        echo "Cloning SputnikVM FFI..."
        cd $ep_gopath
        git clone https://github.com/ethereumproject/sputnikvm-ffi.git
	fi
fi

