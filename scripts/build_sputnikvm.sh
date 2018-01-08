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
	if [ -d "$sputnikffi_path/.git" ]; then
        echo "Updating SputnikVM FFI..."
        cd $sputnikffi_path
        git pull origin master # TODO: handle alternate remote names, ala 'upstream'?
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

