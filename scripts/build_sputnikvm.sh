#!/usr/bin/env bash

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

if ["$OS" == "Windows"]; then
    cd %GOPATH%\src\github.com\ethereumproject
    git clone https://github.com/ethereumproject/sputnikvm-ffi
    cd sputnikvm-ffi\c\ffi
    cargo build --release
    copy %GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\c\ffi\target\release\sputnikvm.lib \
        %GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\c\sputnikvm.lib

    cd %GOPATH%\src\github.com\ethereumproject\go-ethereum\cmd\geth
    set CGO_LDFLAGS=-Wl,--allow-multiple-definition \
        %GOPATH%\src\github.com\ethereumproject\sputnikvm-ffi\c\sputnikvm.lib -lws2_32 -luserenv
    go build -tags=sputnikvm .
else
    cd $GOPATH/src/github.com/ethereumproject
    git clone https://github.com/ethereumproject/sputnikvm-ffi
    cd sputnikvm-ffi/c/ffi
    cargo build --release
    cp $GOPATH/src/github.com/ethereumproject/sputnikvm-ffi/c/ffi/target/release/libsputnikvm_ffi.a \
        $GOPATH/src/github.com/ethereumproject/sputnikvm-ffi/c/libsputnikvm.a

    if ["$OS" == "Linux"]; then
        cd $GOPATH/src/github.com/ethereumproject/go-ethereum/cmd/geth
        CGO_LDFLAGS="$GOPATH/src/github.com/ethereumproject/sputnikvm-ffi/c/libsputnikvm.a -ldl" go build -tags=sputnikvm .
    else
        cd $GOPATH/src/github.com/ethereumproject/go-ethereum/cmd/geth
        CGO_LDFLAGS="$GOPATH/src/github.com/ethereumproject/sputnikvm-ffi/c/libsputnikvm.a -ldl -lresolv" go build -tags=sputnikvm .
    fi
fi

