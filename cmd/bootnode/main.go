// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// bootnode runs a bootstrap node for the Ethereum Discovery Protocol.
package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/p2p/discover"
	"github.com/ethereumproject/go-ethereum/p2p/nat"
)

// Version is the application revision identifier. It can be set with the linker
// as in: go build -ldflags "-X main.Version="`git describe --tags`
var Version = "unknown"

var (
	listenAddr  = flag.String("addr", ":30301", "listen address")
	genKey      = flag.String("genkey", "", "generate a node key and quit")
	nodeKeyFile = flag.String("nodekey", "", "private key filename")
	nodeKeyHex  = flag.String("nodekeyhex", "", "private key as hex (for testing)")
	natdesc     = flag.String("nat", "none", "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)")
	versionFlag = flag.Bool("version", false, "Prints the revision identifier and exit immediatily.")
)

func main() {
	flag.Var(glog.GetVerbosity(), "verbosity", "log verbosity (0-9)")
	flag.Var(glog.GetVModule(), "vmodule", "log verbosity pattern")
	glog.SetToStderr(true)
	flag.Parse()

	if *versionFlag {
		fmt.Println("bootnode version", Version)
		os.Exit(0)
	}

	if *genKey != "" {
		key, err := crypto.GenerateKey()
		if err != nil {
			log.Fatalf("could not generate key: %s", err)
		}
		if err := crypto.SaveECDSA(*genKey, key); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}

	natm, err := nat.Parse(*natdesc)
	if err != nil {
		log.Fatalf("nat: %s", err)
	}

	var nodeKey *ecdsa.PrivateKey
	switch {
	case *nodeKeyFile == "" && *nodeKeyHex == "":
		log.Fatal("Use -nodekey or -nodekeyhex to specify a private key")
	case *nodeKeyFile != "" && *nodeKeyHex != "":
		log.Fatal("Options -nodekey and -nodekeyhex are mutually exclusive")
	case *nodeKeyFile != "":
		var err error
		nodeKey, err = crypto.LoadECDSA(*nodeKeyFile)
		if err != nil {
			log.Fatalf("nodekey: %s", err)
		}
	case *nodeKeyHex != "":
		var err error
		nodeKey, err = crypto.HexToECDSA(*nodeKeyHex)
		if err != nil {
			log.Fatalf("nodekeyhex: %s", err)
		}
	}

	if _, err := discover.ListenUDP(nodeKey, *listenAddr, natm, ""); err != nil {
		log.Fatal(err)
	}
	select {}
}
