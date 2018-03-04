#!/usr/bin/env bash

set -e

OUTPUT="$1"

if [ ! "$OUTPUT" == "build" ] && [ ! "$OUTPUT" == "install" ]; then
	echo "Specify 'install' or 'build' as first argument."
	exit 1
else
	echo "With SputnikVM, running geth $OUTPUT ..."
fi

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
				echo "Updating SputnikVM FFI from branch [%remote_name%] ..."
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
	if [ "$OUTPUT" == "install" ]; then
		go install -ldflags '-X main.Version='$(git describe --tags) -tags=sputnikvm .
	elif [ "$OUTPUT" == "build" ]; then
		mkdir -p %GOPATH%\src\github.com\ethereumproject\go-ethereum\bin
		go build -ldflags '-X main.Version='$(git describe --tags) -o %GOPATH%\src\github.com\ethereumproject\go-ethereum\bin\geth -tags=sputnikvm .
	fi
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
				echo "Updating SputnikVM FFI from branch [$remote_name] ..."
				git pull "$remote_name" master
			fi
		fi
    else
        echo "Cloning SputnikVM FFI ..."
        cd $ep_gopath
        git clone https://github.com/ethereumproject/sputnikvm-ffi.git
	fi
    cd "$sputnikffi_path/c/ffi"
	echo "Building SputnikVM FFI ..."
    cargo build --release
    cp $sputnikffi_path/c/ffi/target/release/libsputnikvm_ffi.a \
        $sputnikffi_path/c/libsputnikvm.a

	geth_binpath="$ep_gopath/go-ethereum/bin"
	echo "Doing geth $OUTPUT ..."
	cd "$ep_gopath/go-ethereum"
    if [ "$OS" == "Linux" ]; then
		if [ "$OUTPUT" == "install" ]; then
			CGO_LDFLAGS="$sputnikffi_path/c/libsputnikvm.a -ldl" go install -ldflags '-X main.Version='$(git describe --tags) ./cmd/geth
		elif [ "$OUTPUT" == "build" ]; then
			mkdir -p "$geth_binpath"
			CGO_LDFLAGS="$sputnikffi_path/c/libsputnikvm.a -ldl" go build -ldflags '-X main.Version='$(git describe --tags) -o $geth_binpath/geth -tags=sputnikvm ./cmd/geth
		fi
    else
		if [ "$OUTPUT" == "install" ]; then
			CGO_LDFLAGS="$sputnikffi_path/c/libsputnikvm.a -ldl -lresolv" go install -ldflags '-X main.Version='$(git describe --tags) -tags=sputnikvm ./cmd/geth
		elif [ "$OUTPUT" == "build" ]; then
			mkdir -p "$geth_binpath"
			CGO_LDFLAGS="$sputnikffi_path/c/libsputnikvm.a -ldl -lresolv" go build -ldflags '-X main.Version='$(git describe --tags) -o $geth_binpath/geth -tags=sputnikvm ./cmd/geth
		fi
    fi
fi

