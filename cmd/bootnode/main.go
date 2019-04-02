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

	"github.com/eth-classic/go-ethereum/crypto"
	"github.com/eth-classic/go-ethereum/logger/glog"
	"github.com/eth-classic/go-ethereum/p2p/discover"
	"github.com/eth-classic/go-ethereum/p2p/nat"
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

// onlyDoGenKey exits 0 if successful.
// It does the -genkey flag feature and that is all.
func onlyDoGenKey() {
	key, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("could not generate key: %s", err)
	}
	f, e := os.Create(*genKey)
	defer f.Close()
	if e != nil {
		log.Fatalf("coult not open genkey file: %v", e)
	}
	if _, err := crypto.WriteECDSAKey(f, key); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

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
		// exits 0 if successful
		onlyDoGenKey()
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
		f, err := os.Open(*nodeKeyFile)
		if err != nil {
			log.Fatalf("error opening node key file: %v", err)
		}
		nodeKey, err = crypto.LoadECDSA(f)
		if err := f.Close(); err != nil {
			log.Fatalf("error closing key file: %v", err)
		}
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
