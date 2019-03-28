#!/usr/bin/env bash

set -e

OUTPUT="$1"

if [ ! "$OUTPUT" == "build" ] && [ ! "$OUTPUT" == "install" ]; then
	echo "Specify 'install' or 'build' as first argument."
	exit 1
else
	echo "With SputnikVM, running geth $OUTPUT ..."
fi

installrust() {
    curl https://sh.rustup.rs -sSf | sh
    source $HOME/.cargo/env
}

# Prompt to install rust and cargo if not already installed.
if hash cargo 2>/dev/null; then
    echo "Cargo installed OK, continuing"
else
    while true; do
        read -p "Install/build with SputnikVM requires Rust and cargo to be installed. Would you like to install them? [Yy|Nn]" yn
        case $yn in
            [yY]* ) installrust; echo "Rust and cargo have been installed and temporarily added to your PATH"; break;;
            [nN]* ) echo "Can't compile SputniKVM. Exiting."; exit 0;;
        esac
    done
fi

OS=`uname -s`

geth_path="github.com/ethereumproject/go-ethereum"
sputnik_path="github.com/ETCDEVTeam/sputnikvm-ffi"
sputnik_dir="$GOPATH/src/$geth_path/vendor/$sputnik_path"

geth_bindir="$GOPATH/src/$geth_path/bin"

echo "Building SputnikVM"
make -C "$sputnik_dir/c"

echo "Doing geth $OUTPUT ..."
cd "$GOPATH/src/$geth_path"

LDFLAGS="$sputnik_dir/c/libsputnikvm.a "
case $OS in
	"Linux")
		LDFLAGS+="-ldl"
		;;

	"Darwin")
		LDFLAGS+="-ldl -lresolv"
		;;

    CYGWIN*|MINGW32*|MSYS*)
		LDFLAGS="-Wl,--allow-multiple-definition $sputnik_dir/c/sputnikvm.lib -lws2_32 -luserenv"
		;;
esac


if [ "$OUTPUT" == "install" ]; then
	CGO_LDFLAGS=$LDFLAGS go install -ldflags '-X main.Version='$(git describe --tags) -tags="sputnikvm netgo" ./cmd/geth
elif [ "$OUTPUT" == "build" ]; then
	mkdir -p "$geth_bindir"
	CGO_LDFLAGS=$LDFLAGS go build -ldflags '-X main.Version='$(git describe --tags) -o $geth_bindir/geth -tags="sputnikvm netgo" ./cmd/geth
fi

