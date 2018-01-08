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
	# Check if git is happening in svm-ffi.
	if [ -d "%GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi" ]; then
		cd sputnikvm-ffi
		if [ -d "%GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\.git" ]; then
			remote_name=$(git remote -v | head -1 | awk '{print $1;}')
			if [ ! "%remote_name%" == "" ]; then
				echo "Updating SputnikVM FFI from branch [%remote_name%]..."
				git pull %remote_name% master
			fi
		fi
    	cd c\ffi
	else
    	git clone https://github.com/ethereumproject/sputnikvm-ffi.git
    	cd sputnikvm-ffi\c\ffi
	fi
    cargo build --release
    copy %GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\c\ffi\target\release\sputnikvm.lib \
        %GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\c\sputnikvm.lib

    cd %GOPATH%\src\github.com\ethereumproject\go-ethereum\cmd\geth
    set CGO_LDFLAGS=-Wl,--allow-multiple-definition \
        %GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\c\sputnikvm.lib -lws2_32 -luserenv
	mkdir -p %GOPATH%\src\github.com\ethereumproject\go-ethereum\bin
    go build -o %GOPATH%\src\github.com\ethereumproject\go-ethereum\bin\geth -tags=sputnikvm .
else
    ep_gopath=$GOPATH/src/github.com/ethereumproject
	sputnikffi_path="$ep_gopath/sputnikvm-ffi"

	# If sputnikvmffi has already been cloned/existing
	if [ -d "$sputnikffi_path" ]; then
		# Ensure git is happening in svm-ffi.
		# Update if .git exists, otherwise don't try updating. We could possibly handle git-initing and adding remote but seems
		# like an edge case.
		if [ -d "$sputnikffi_path/.git" ]; then
        	cd $sputnikffi_path
			remote_name=$(git remote -v | head -1 | awk '{print $1;}')
			if [ ! "$remote_name" == "" ]; then
				echo "Updating SputnikVM FFI from branch [$remote_name]..."
				git pull "$remote_name" master
			fi
		fi
    else
        echo "Cloning SputnikVM FFI..."
        cd $ep_gopath
        git clone https://github.com/ethereumproject/sputnikvm-ffi.git
	fi
    cd "$sputnikffi_path/c/ffi"
	echo "Building SputnikVM FFI..."
    cargo build --release
    cp $sputnikffi_path/c/ffi/target/release/libsputnikvm_ffi.a \
        $sputnikffi_path/c/libsputnikvm.a

	geth_binpath="$ep_gopath/go-ethereum/bin"
	echo "Building geth to $geth_binpath/geth..."
	mkdir -p "$geth_binpath"
    if [ "$OS" == "Linux" ]; then
        cd $GOPATH/src/github.com/ethereumproject/go-ethereum/cmd/geth
        CGO_LDFLAGS="$sputnikffi_path/c/libsputnikvm.a -ldl" go build -o $geth_binpath/geth -tags=sputnikvm .
    else
        cd $GOPATH/src/github.com/ethereumproject/go-ethereum/cmd/geth
        CGO_LDFLAGS="$sputnikffi_path/c/libsputnikvm.a -ldl -lresolv" go build -o $geth_binpath/geth -tags=sputnikvm .
    fi
fi

