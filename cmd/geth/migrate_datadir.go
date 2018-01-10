// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"os"
	"path/filepath"
)

// handleIfDataDirSchemaMigrations is a handlers for the conditional logic around
// data/chain dir migrations from geth versions < 3.4 and in consideration of EF geth schemas used for ETC.
func handleIfDataDirSchemaMigrations(ctx *cli.Context) error {

	origV := int(*glog.GetVerbosity())
	if origV == 0 {
		origV = glog.DefaultVerbosity
	}

	// Turn verbosity down for migration check. If migration happens, it will print to Warn.
	// Otherwise logs are just debuggers.
	glog.SetToStderr(true)
	glog.SetV(3)

	if shouldAttemptDirMigration(ctx) {
		// Rename existing default datadir <home>/<Ethereum>/ to <home>/<EthereumClassic>.
		// Only do this if --datadir flag is not specified AND <home>/<EthereumClassic> does NOT already exist (only migrate once and only for defaulty).
		// If it finds an 'Ethereum' directory, it will check if it contains default ETC or ETHF chain data.
		// If it contains ETC data, it will rename the dir. If ETHF data, if will do nothing.
		if err := migrateExistingDirToClassicNamingScheme(ctx); err != nil {
			return err
		}

		// Move existing mainnet data to pertinent chain-named subdir scheme (ie ethereum-classic/mainnet).
		// This should only happen if the given (newly defined in this protocol) subdir doesn't exist,
		// and the dirs&files (nodekey, dapp, keystore, chaindata, nodes) do exist,
		if err := migrateToChainSubdirIfNecessary(ctx); err != nil {
			return err
		}
	}
	// (Re)set default debug verbosity level.
	glog.SetV(origV)
	return nil
}

// migrateExistingDirToClassicNamingScheme renames default base data directory ".../Ethereum" to ".../EthereumClassic", pending os customs, etc... ;-)
///
// Check for preexisting **Un-classic** data directory, ie "/home/path/to/Ethereum".
// If it exists, check if the data therein belongs to Classic blockchain (ie not configged as "ETF"),
// and rename it to fit Classic naming convention ("/home/path/to/EthereumClassic") if that dir doesn't already exist.
// This case only applies to Default, ie when a user **doesn't** provide a custom --datadir flag;
// a user should be able to override a specified data dir if they want.
func migrateExistingDirToClassicNamingScheme(ctx *cli.Context) error {

	ethDataDirPath := common.DefaultUnclassicDataDir()
	etcDataDirPath := common.DefaultDataDir()

	// only if default <EthereumClassic>/ datadir doesn't already exist
	if _, err := os.Stat(etcDataDirPath); err == nil {
		// classic data dir already exists
		glog.V(logger.Debug).Infof("Using existing ETClassic data directory at: %v\n", etcDataDirPath)
		return nil
	}

	ethChainDBPath := filepath.Join(ethDataDirPath, "chaindata")
	if chainIsMorden(ctx) {
		ethChainDBPath = filepath.Join(ethDataDirPath, "testnet", "chaindata")
	}

	// only if ETHdatadir chaindb path DOES already exist, so return nil if it doesn't;
	// otherwise NewLDBDatabase will create an empty one there.
	// note that this uses the 'old' non-subdirectory way of holding default data.
	// it must be called before migrating to subdirectories
	// NOTE: Since ETH stores chaindata by default in Ethereum/geth/..., this path
	// will not exist if the existing data belongs to ETH, so it works as a valid check for us as well.
	if _, err := os.Stat(ethChainDBPath); os.IsNotExist(err) {
		glog.V(logger.Debug).Warnf(`No existing default chaindata dir found at: %v
		  	Using default data directory at: %v`,
			ethChainDBPath, etcDataDirPath)
		return nil
	}

	foundCorrectLookingFiles := []string{}
	requiredFiles := []string{"LOG", "LOCK", "CURRENT"}
	for _, f := range requiredFiles {
		p := filepath.Join(ethChainDBPath, f)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			glog.V(logger.Debug).Warnf(`No existing default file found at: %v
		  	Using default data directory at: %v`,
				p, etcDataDirPath)
		} else {
			foundCorrectLookingFiles = append(foundCorrectLookingFiles, f)
		}
	}
	hasRequiredFiles := len(requiredFiles) == len(foundCorrectLookingFiles)
	if !hasRequiredFiles {
		return nil
	}

	// check if there is existing etf blockchain data in unclassic default dir (ie /<home>/Ethereum)
	chainDB, err := ethdb.NewLDBDatabase(ethChainDBPath, 0, 0)
	if err != nil {
		glog.V(logger.Debug).Warnf(`Failed to check blockchain compatibility for existing Ethereum chaindata database at: %v
		 	Using default data directory at: %v`,
			err, etcDataDirPath)
		return nil
	}

	defer chainDB.Close()

	// Only move if defaulty ETC (mainnet or testnet).
	// Get head block if testnet, fork block if mainnet.
	hh := core.GetHeadBlockHash(chainDB) // get last block in fork
	if ctx.GlobalBool(aliasableName(FastSyncFlag.Name, ctx)) {
		hh = core.GetHeadFastBlockHash(chainDB)
	}
	if hh.IsEmpty() {
		glog.V(logger.Debug).Warnln("There was no head block for the old database. It could be very young.")
	}

	hasRequiredForkIfSufficientHeight := true
	if !hh.IsEmpty() {
		// if head block < 1920000, then its compatible
		// if head block >= 1920000, then it must have a hash matching required hash

		// Use default configuration to check if known fork, if block 1920000 exists.
		// If block1920000 doesn't exist, given above checks for directory structure expectations,
		// I think it's safe to assume that the chaindata directory is just too 'young', where it hasn't
		// synced until block 1920000, and therefore can be migrated.
		conf := core.DefaultConfigMainnet.ChainConfig
		if chainIsMorden(ctx) {
			conf = core.DefaultConfigMorden.ChainConfig
		}

		hf := conf.ForkByName("The DAO Hard Fork")
		if hf == nil || hf.Block == nil || new(big.Int).Cmp(hf.Block) == 0 || hf.RequiredHash.IsEmpty() {
			glog.V(logger.Debug).Warnln("DAO Hard Fork required hash not configured for database chain. Not migrating.")
			return nil
		}

		b := core.GetBlock(chainDB, hh)
		if b == nil {
			glog.V(logger.Debug).Warnf("There was a problem checking the head block of old-namespaced database. The head hash was: %v", hh.Hex())
			return nil
		}

		// if head block >= 1920000
		if b.Number().Cmp(hf.Block) >= 0 {
			// now, since we know that the height is bigger than the hardfork, we have to check that the db contains the required hardfork hash
			glog.V(logger.Debug).Infof("Existing head block in old data dir has sufficient height: %v", b.String())

			hasRequiredForkIfSufficientHeight = false
			bf := core.GetBlock(chainDB, hf.RequiredHash)
			// does not have required block by hash
			if bf != nil {
				glog.V(logger.Debug).Infof("Head block has sufficient height AND required hash: %v", b.String())
				hasRequiredForkIfSufficientHeight = true
			} else {
				glog.V(logger.Debug).Infof("Head block has sufficient height but not required hash: %v", b.String())
			}
			// head block < 1920000
		} else {
			glog.V(logger.Debug).Infof("Existing head block in old data dir has INSUFFICIENT height to differentiate ETC/ETF: %v", b.String())
		}
	}

	if hasRequiredForkIfSufficientHeight {
		// if any of the LOG, LOCK, or CURRENT files are missing from old chaindata/, don't migrate
		glog.V(logger.Warn).Warnf(`Found existing data directory named 'Ethereum' with default ETC chaindata.
		  	Moving it from: %v, to: %v
		  	To specify a different data directory use the '--datadir' flag.`,
			ethDataDirPath, etcDataDirPath)
		return os.Rename(ethDataDirPath, etcDataDirPath)
	}

	glog.V(logger.Debug).Infof(`Existing default Ethereum database at: %v isn't an Ethereum Classic default blockchain.
	  	Will not migrate.
	  	Using ETC chaindata database at: %v`,
		ethDataDirPath, etcDataDirPath)
	return nil
}

// migrateToChainSubdirIfNecessary migrates ".../EthereumClassic/nodes|chaindata|...|nodekey" --> ".../EthereumClassic/mainnet/nodes|chaindata|...|nodekey"
func migrateToChainSubdirIfNecessary(ctx *cli.Context) error {
	chainIdentity := mustMakeChainIdentity(ctx) // "mainnet", "morden", "custom"

	datapath := mustMakeDataDir(ctx) // ".../EthereumClassic/ | --datadir"

	subdirPath := MustMakeChainDataDir(ctx) // ie, <EthereumClassic>/mainnet

	// check if default subdir "mainnet" exits
	// NOTE: this assumes that if the migration has been run once, the "mainnet" dir will exist and will have necessary datum inside it
	subdirPathInfo, err := os.Stat(subdirPath)
	if err == nil {
		// dir already exists
		return nil
	}
	if subdirPathInfo != nil && !subdirPathInfo.IsDir() {
		return fmt.Errorf(`%v: found file named '%v' in EthereumClassic datadir,
			which conflicts with default chain directory naming convention: %v`, ErrDirectoryStructure, chainIdentity, subdirPath)
	}

	// 3.3 testnet uses subdir '/testnet'
	if chainIdentitiesMorden[chainIdentity] {
		exTestDir := filepath.Join(subdirPath, "../testnet")
		exTestDirInfo, e := os.Stat(exTestDir)
		if e != nil && os.IsNotExist(e) {
			return nil // ex testnet dir doesn't exist
		}
		if !exTestDirInfo.IsDir() {
			return nil // don't interfere with user *file* that won't be relevant for geth
		}
		return os.Rename(exTestDir, subdirPath) // /testnet -> /morden
	}

	// mkdir -p ".../mainnet"
	if err := os.MkdirAll(subdirPath, 0755); err != nil {
		return err
	}

	// move if existing (nodekey, dapp/, keystore/, chaindata/, nodes/) into new subdirectories
	for _, dir := range []string{"dapp", "keystore", "chaindata", "nodes"} {

		dirPath := filepath.Join(datapath, dir)

		dirInfo, e := os.Stat(dirPath)
		if e != nil && os.IsNotExist(e) {
			continue // dir doesn't exist
		}
		if !dirInfo.IsDir() {
			continue // don't interfere with user *file* that won't be relevant for geth
		}

		dirPathUnderSubdir := filepath.Join(subdirPath, dir)
		if err := os.Rename(dirPath, dirPathUnderSubdir); err != nil {
			return err
		}
	}

	// ensure nodekey exists and is file (loop lets us stay consistent in form here, an keep options open for easy other files to include)
	for _, file := range []string{"nodekey", "geth.ipc"} {
		filePath := filepath.Join(datapath, file)

		// ensure exists and is a file
		fileInfo, e := os.Stat(filePath)
		if e != nil && os.IsNotExist(e) {
			continue
		}
		if fileInfo.IsDir() {
			continue // don't interfere with user dirs that won't be relevant for geth
		}

		filePathUnderSubdir := filepath.Join(subdirPath, file)
		if err := os.Rename(filePath, filePathUnderSubdir); err != nil {
			return err
		}
	}
	return nil
}
