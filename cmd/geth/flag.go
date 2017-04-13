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

package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/ethereumproject/ethash"
	"github.com/ethereumproject/go-ethereum/accounts"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/event"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/miner"
	"github.com/ethereumproject/go-ethereum/node"
	"github.com/ethereumproject/go-ethereum/p2p/discover"
	"github.com/ethereumproject/go-ethereum/p2p/nat"
	"github.com/ethereumproject/go-ethereum/pow"
	"github.com/ethereumproject/go-ethereum/whisper"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	cli.AppHelpTemplate = `{{.Name}} {{if .Flags}}[global options] {{end}}command{{if .Flags}} [command options]{{end}} [arguments...]

VERSION:
   {{.Version}}

COMMANDS:
   {{range .Commands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
   {{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{end}}
`

	cli.CommandHelpTemplate = `{{.Name}}{{if .Subcommands}} command{{end}}{{if .Flags}} [command options]{{end}} [arguments...]
{{if .Description}}{{.Description}}
{{end}}{{if .Subcommands}}
SUBCOMMANDS:
	{{range .Subcommands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
	{{end}}{{end}}{{if .Flags}}
OPTIONS:
	{{range .Flags}}{{.}}
	{{end}}{{end}}
`
}

// #chainconfigi
// MustMakeDataDir retrieves the currently requested data directory, terminating
// if none (or the empty string) is specified. If the node is starting a testnet,
// the a subdirectory of the specified datadir will be used.
func MustMakeDataDir(ctx *cli.Context) string {
	if path := ctx.GlobalString(DataDirFlag.Name); path != "" {
		if ctx.GlobalBool(TestNetFlag.Name) {
			return filepath.Join(path, "/testnet")
		}
		return path
	}
	log.Fatal("Cannot determine default data directory, please set manually (--datadir)")
	return ""
}

// MakeIPCPath creates an IPC path configuration from the set command line flags,
// returning an empty string if IPC was explicitly disabled, or the set path.
func MakeIPCPath(ctx *cli.Context) string {
	if ctx.GlobalBool(IPCDisabledFlag.Name) {
		return ""
	}
	return ctx.GlobalString(IPCPathFlag.Name)
}

// MakeNodeKey creates a node key from set command line flags, either loading it
// from a file or as a specified hex value. If neither flags were provided, this
// method returns nil and an emphemeral key is to be generated.
func MakeNodeKey(ctx *cli.Context) *ecdsa.PrivateKey {
	var (
		hex  = ctx.GlobalString(NodeKeyHexFlag.Name)
		file = ctx.GlobalString(NodeKeyFileFlag.Name)

		key *ecdsa.PrivateKey
		err error
	)
	switch {
	case file != "" && hex != "":
		log.Fatalf("Options %q and %q are mutually exclusive", NodeKeyFileFlag.Name, NodeKeyHexFlag.Name)

	case file != "":
		if key, err = crypto.LoadECDSA(file); err != nil {
			log.Fatalf("Option %q: %v", NodeKeyFileFlag.Name, err)
		}

	case hex != "":
		if key, err = crypto.HexToECDSA(hex); err != nil {
			log.Fatalf("Option %q: %v", NodeKeyHexFlag.Name, err)
		}
	}
	return key
}

// MakeBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func MakeBootstrapNodes(ctx *cli.Context) []*discover.Node {
	// Return pre-configured nodes if none were manually requested
	if !ctx.GlobalIsSet(BootnodesFlag.Name) {
		if ctx.GlobalBool(TestNetFlag.Name) {
			return TestNetBootNodes
		}
		return HomesteadBootNodes
	}
	// Otherwise parse and use the CLI bootstrap nodes
	bootnodes := []*discover.Node{}

	for _, url := range strings.Split(ctx.GlobalString(BootnodesFlag.Name), ",") {
		node, err := discover.ParseNode(url)
		if err != nil {
			glog.V(logger.Error).Infof("Bootstrap URL %s: %v\n", url, err)
			continue
		}
		bootnodes = append(bootnodes, node)
	}
	return bootnodes
}

// MakeListenAddress creates a TCP listening address string from set command
// line flags.
func MakeListenAddress(ctx *cli.Context) string {
	return fmt.Sprintf(":%d", ctx.GlobalInt(ListenPortFlag.Name))
}

// MakeNAT creates a port mapper from set command line flags.
func MakeNAT(ctx *cli.Context) nat.Interface {
	natif, err := nat.Parse(ctx.GlobalString(NATFlag.Name))
	if err != nil {
		log.Fatalf("Option %s: %v", NATFlag.Name, err)
	}
	return natif
}

// MakeRPCModules splits input separated by a comma and trims excessive white
// space from the substrings.
func MakeRPCModules(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}

// MakeHTTPRpcHost creates the HTTP RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func MakeHTTPRpcHost(ctx *cli.Context) string {
	if !ctx.GlobalBool(RPCEnabledFlag.Name) {
		return ""
	}
	return ctx.GlobalString(RPCListenAddrFlag.Name)
}

// MakeWSRpcHost creates the WebSocket RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func MakeWSRpcHost(ctx *cli.Context) string {
	if !ctx.GlobalBool(WSEnabledFlag.Name) {
		return ""
	}
	return ctx.GlobalString(WSListenAddrFlag.Name)
}

// MakeDatabaseHandles raises out the number of allowed file handles per process
// for Geth and returns half of the allowance to assign to the database.
func MakeDatabaseHandles() int {
	if err := raiseFdLimit(2048); err != nil {
		log.Fatal("Failed to raise file descriptor allowance: ", err)
	}
	limit, err := getFdLimit()
	if err != nil {
		log.Fatal("Failed to retrieve file descriptor allowance: ", err)
	}
	if limit > 2048 { // cap database file descriptors even if more is available
		limit = 2048
	}
	return limit / 2 // Leave half for networking and other stuff
}

// MakeAccountManager creates an account manager from set command line flags.
func MakeAccountManager(ctx *cli.Context) *accounts.Manager {
	// Create the keystore crypto primitive, light if requested
	scryptN := accounts.StandardScryptN
	scryptP := accounts.StandardScryptP
	if ctx.GlobalBool(LightKDFFlag.Name) {
		scryptN = accounts.LightScryptN
		scryptP = accounts.LightScryptP
	}
	datadir := MustMakeDataDir(ctx)

	keydir := filepath.Join(datadir, "keystore")
	if path := ctx.GlobalString(KeyStoreDirFlag.Name); path != "" {
		keydir = path
	}

	m, err := accounts.NewManager(keydir, scryptN, scryptP)
	if err != nil {
		glog.Fatalf("init account manager at %q: %s", keydir, err)
	}
	return m
}

// MakeAddress converts an account specified directly as a hex encoded string or
// a key index in the key store to an internal account representation.
func MakeAddress(accman *accounts.Manager, account string) (accounts.Account, error) {
	// If the specified account is a valid address, return it
	if common.IsHexAddress(account) {
		return accounts.Account{Address: common.HexToAddress(account)}, nil
	}
	// Otherwise try to interpret the account as a keystore index
	index, err := strconv.Atoi(account)
	if err != nil {
		return accounts.Account{}, fmt.Errorf("invalid account address or index %q", account)
	}
	return accman.AccountByIndex(index)
}

// MakeEtherbase retrieves the etherbase either from the directly specified
// command line flags or from the keystore if CLI indexed.
func MakeEtherbase(accman *accounts.Manager, ctx *cli.Context) common.Address {
	accounts := accman.Accounts()
	if !ctx.GlobalIsSet(EtherbaseFlag.Name) && len(accounts) == 0 {
		glog.V(logger.Error).Infoln("WARNING: No etherbase set and no accounts found as default")
		return common.Address{}
	}
	etherbase := ctx.GlobalString(EtherbaseFlag.Name)
	if etherbase == "" {
		return common.Address{}
	}
	// If the specified etherbase is a valid address, return it
	account, err := MakeAddress(accman, etherbase)
	if err != nil {
		log.Fatalf("Option %q: %v", EtherbaseFlag.Name, err)
	}
	return account.Address
}

// MakePasswordList reads password lines from the file specified by --password.
func MakePasswordList(ctx *cli.Context) []string {
	path := ctx.GlobalString(PasswordFileFlag.Name)
	if path == "" {
		return nil
	}
	text, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Failed to read password file: ", err)
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

// ExportChainConfig exports contextualized chain configuration to JSON file
func ExportExternalChainConfigJSON(ctx *cli.Context) error {

	chainConfigFilePath := ctx.GlobalString(ExportChainConfigFlag.Name)
	chainConfigFilePath = filepath.Clean(chainConfigFilePath)

	if chainConfigFilePath == "" || chainConfigFilePath == "/" {
		return fmt.Errorf("Filepath to export chain configuration was blank; it must be specified.")
	}
	chainConfigFilePath = common.AbsolutePath(common.DefaultDataDir(), chainConfigFilePath)

	config := MustMakeChainConfig(ctx)

	jsonConfig, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("Could not marshal json from chain config: %v", err)
	}

	if err := ioutil.WriteFile(chainConfigFilePath, jsonConfig, 0644); err != nil {
		return fmt.Errorf("Could not write chain config file: %v", err)
	}
	glog.V(logger.Info).Info(fmt.Sprintf("Wrote chain config file to \x1b[32m%s\x1b[39m  Closing down now.", chainConfigFilePath))
	return nil
}

// Check for preexisting **Un-classic** data directory, ie "/home/path/to/Ethereum".
// If it exists, check if the data therein belongs to Classic blockchain (ie not configged as "ETF"),
// and rename it to fit Classic naming convention ("/home/path/to/EthereumClassic") if that dir doesn't already exist.
// This case only applies to Default, ie when a user **doesn't** provide a custom --datadir flag;
// a user should be able to override a specified data dir if they want.
// TODO: make test(s)
func migrateExistingDirToClassicNamingScheme(ctx *cli.Context) error {

	unclassicDataDirPath := common.DefaultUnclassicDataDir()
	classicDataDirPath := common.DefaultDataDir()

	// only if using default data dir (flagged dir can override)
	if ctx.GlobalIsSet(DataDirFlag.Name) {
		log.Printf("INFO: Global --datadir flag is set: %v \n", DataDirFlag.Value.String())
		return nil
	}

	// only if default **classic** data dir doesn't already exist
	if _, err := os.Stat(classicDataDirPath); err == nil {
		// classic data dir already exists
		log.Printf("INFO: Using existing ETClassic data directory at: %v.\n", classicDataDirPath)
		return nil
	}

	// only if **unclassic** ("<home>/Ethereum") datadir chaindb path DOES already exist, so return nil if it doesn't;
	// otherwise NewLDBDatabase will create an empty one there.
	unclassicChainDBPath := filepath.Join(unclassicDataDirPath, "chaindata")
	if _, err := os.Stat(unclassicChainDBPath); os.IsNotExist(err) {
		log.Printf("INFO: No existing ETH/ETF default chaindata dir found at: %v \n Using default data directory at: %v.\n", unclassicChainDBPath, classicDataDirPath)
		return nil
	}

	// check if there is existing etf blockchain data in unclassic default dir (ie /<home>/Ethereum)
	chainDB, err := ethdb.NewLDBDatabase(unclassicChainDBPath, 0, 0)
	if err != nil {
		log.Fatalf("WARNING: Error checking blockchain version for existing Ethereum chaindata database at: %v \n  Using default data directory at: %v", err, classicDataDirPath)
		return nil
	}

	defer chainDB.Close()

	bcVersionNumber := core.GetBlockChainVersion(chainDB)
	isETF := core.DefaultConfig.IsETF(big.NewInt(int64(bcVersionNumber)))

	// if existing database in <home>/Ethereum isn't ETC, let it be
	if isETF {
		log.Printf("INFO: Existing default Ethereum database at: %v isn't an ETClassic blockchain, it has version number: %v. Will not alter. \nUsing ETC chaindata database at: %v \n", unclassicDataDirPath, bcVersionNumber, classicDataDirPath)
		return nil
	}

	log.Printf("INFO/WARNING: Found existing Ethereum data directory with ETC chaindata. \n Moving it from: %v, to: %v \n To specify a data directory use the `--datadir` flag.", unclassicDataDirPath, classicDataDirPath)

	return os.Rename(unclassicDataDirPath, classicDataDirPath)
}

// makeName makes the node name, which can be (in part) customized by the IdentityFlag
func makeName(version string, ctx *cli.Context) string {
	name := fmt.Sprintf("Geth/%s/%s/%s", version, runtime.GOOS, runtime.Version())
	if identity := ctx.GlobalString(IdentityFlag.Name); len(identity) > 0 {
		name += "/" + identity
	}
	return name
}

// MakeSystemNode sets up a local node, configures the services to launch and
// assembles the P2P protocol stack.
func MakeSystemNode(version string, ctx *cli.Context) *node.Node {
	name := makeName(version, ctx)

	// global settings
	if ctx.GlobalIsSet(ExtraDataFlag.Name) {
		s := ctx.GlobalString(ExtraDataFlag.Name)
		if len(s) > types.HeaderExtraMax {
			log.Fatalf("%s flag %q exceeds size limit of %d", ExtraDataFlag.Name, s, types.HeaderExtraMax)
		}
		miner.HeaderExtra = []byte(s)
	}

	if migrationError := migrateExistingDirToClassicNamingScheme(ctx); migrationError != nil {
		log.Fatalf("Failed to migrate existing Classic database: %v", migrationError)
	}

	// Avoid conflicting network flags
	networks, netFlags := 0, []cli.BoolFlag{DevModeFlag, TestNetFlag}
	for _, flag := range netFlags {
		if ctx.GlobalBool(flag.Name) {
			networks++
		}
	}
	if networks > 1 {
		log.Fatalf("The %v flags are mutually exclusive", netFlags)
	}

	// Configure the node's service container
	stackConf := &node.Config{
		DataDir:         MustMakeDataDir(ctx),
		PrivateKey:      MakeNodeKey(ctx),
		Name:            name,
		NoDiscovery:     ctx.GlobalBool(NoDiscoverFlag.Name),
		BootstrapNodes:  MakeBootstrapNodes(ctx),
		ListenAddr:      MakeListenAddress(ctx),
		NAT:             MakeNAT(ctx),
		MaxPeers:        ctx.GlobalInt(MaxPeersFlag.Name),
		MaxPendingPeers: ctx.GlobalInt(MaxPendingPeersFlag.Name),
		IPCPath:         MakeIPCPath(ctx),
		HTTPHost:        MakeHTTPRpcHost(ctx),
		HTTPPort:        ctx.GlobalInt(RPCPortFlag.Name),
		HTTPCors:        ctx.GlobalString(RPCCORSDomainFlag.Name),
		HTTPModules:     MakeRPCModules(ctx.GlobalString(RPCApiFlag.Name)),
		WSHost:          MakeWSRpcHost(ctx),
		WSPort:          ctx.GlobalInt(WSPortFlag.Name),
		WSOrigins:       ctx.GlobalString(WSAllowedOriginsFlag.Name),
		WSModules:       MakeRPCModules(ctx.GlobalString(WSApiFlag.Name)),
	}

	// Configure the Ethereum service
	accman := MakeAccountManager(ctx)

	ethConf := &eth.Config{
		ChainConfig:             MustMakeChainConfig(ctx),
		FastSync:                ctx.GlobalBool(FastSyncFlag.Name),
		BlockChainVersion:       ctx.GlobalInt(BlockchainVersionFlag.Name),
		DatabaseCache:           ctx.GlobalInt(CacheFlag.Name),
		DatabaseHandles:         MakeDatabaseHandles(),
		NetworkId:               ctx.GlobalInt(NetworkIdFlag.Name),
		AccountManager:          accman,
		Etherbase:               MakeEtherbase(accman, ctx),
		MinerThreads:            ctx.GlobalInt(MinerThreadsFlag.Name),
		NatSpec:                 ctx.GlobalBool(NatspecEnabledFlag.Name),
		DocRoot:                 ctx.GlobalString(DocRootFlag.Name),
		GasPrice:                new(big.Int),
		GpoMinGasPrice:          new(big.Int),
		GpoMaxGasPrice:          new(big.Int),
		GpoFullBlockRatio:       ctx.GlobalInt(GpoFullBlockRatioFlag.Name),
		GpobaseStepDown:         ctx.GlobalInt(GpobaseStepDownFlag.Name),
		GpobaseStepUp:           ctx.GlobalInt(GpobaseStepUpFlag.Name),
		GpobaseCorrectionFactor: ctx.GlobalInt(GpobaseCorrectionFactorFlag.Name),
		SolcPath:                ctx.GlobalString(SolcPathFlag.Name),
		AutoDAG:                 ctx.GlobalBool(AutoDAGFlag.Name) || ctx.GlobalBool(MiningEnabledFlag.Name),
	}

	if _, ok := ethConf.GasPrice.SetString(ctx.GlobalString(GasPriceFlag.Name), 0); !ok {
		log.Fatalf("malformed %s flag value %q", GasPriceFlag.Name, ctx.GlobalString(GasPriceFlag.Name))
	}
	if _, ok := ethConf.GpoMinGasPrice.SetString(ctx.GlobalString(GpoMinGasPriceFlag.Name), 0); !ok {
		log.Fatalf("malformed %s flag value %q", GpoMinGasPriceFlag.Name, ctx.GlobalString(GpoMinGasPriceFlag.Name))
	}
	if _, ok := ethConf.GpoMaxGasPrice.SetString(ctx.GlobalString(GpoMaxGasPriceFlag.Name), 0); !ok {
		log.Fatalf("malformed %s flag value %q", GpoMaxGasPriceFlag.Name, ctx.GlobalString(GpoMaxGasPriceFlag.Name))
	}

	// Configure the Whisper service
	shhEnable := ctx.GlobalBool(WhisperEnabledFlag.Name)

	// Override any default configs in dev mode or the test net
	switch {
	case ctx.GlobalBool(TestNetFlag.Name):
		if !ctx.GlobalIsSet(NetworkIdFlag.Name) {
			ethConf.NetworkId = 2
		}
		ethConf.Genesis = core.TestNetGenesis
		state.StartingNonce = 1048576 // (2**20)

	case ctx.GlobalBool(DevModeFlag.Name):
		// Override the base network stack configs
		if !ctx.GlobalIsSet(DataDirFlag.Name) {
			stackConf.DataDir = filepath.Join(os.TempDir(), "/ethereum_dev_mode")
		}
		if !ctx.GlobalIsSet(MaxPeersFlag.Name) {
			stackConf.MaxPeers = 0
		}
		if !ctx.GlobalIsSet(ListenPortFlag.Name) {
			stackConf.ListenAddr = ":0"
		}
		// Override the Ethereum protocol configs
		ethConf.Genesis = core.TestNetGenesis
		if !ctx.GlobalIsSet(GasPriceFlag.Name) {
			ethConf.GasPrice = new(big.Int)
		}
		if !ctx.GlobalIsSet(WhisperEnabledFlag.Name) {
			shhEnable = true
		}
		ethConf.PowTest = true
	}
	// Assemble and return the protocol stack
	stack, err := node.New(stackConf)
	if err != nil {
		log.Fatal("Failed to create the protocol stack: ", err)
	}
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return eth.New(ctx, ethConf)
	}); err != nil {
		log.Fatal("Failed to register the Ethereum service: ", err)
	}
	if shhEnable {
		if err := stack.Register(func(*node.ServiceContext) (node.Service, error) { return whisper.New(), nil }); err != nil {
			log.Fatal("Failed to register the Whisper service: ", err)
		}
	}

	if ctx.GlobalBool(Unused1.Name) {
		glog.V(logger.Info).Infoln(fmt.Sprintf("Geth started with --%s flag, which is unused by Geth Classic and can be omitted", Unused1.Name))
	}

	return stack
}

// MustMakeChainConfig reads the chain configuration from the database in ctx.Datadir.
func MustMakeChainConfig(ctx *cli.Context) *core.ChainConfig {
	db := MakeChainDatabase(ctx)
	defer db.Close()

	return MustMakeChainConfigFromDb(ctx, db)
}

// MustMakeChainConfigFromDb reads the chain configuration from the given database.
func MustMakeChainConfigFromDb(ctx *cli.Context, db ethdb.Database) *core.ChainConfig {
	c := core.DefaultConfig
	if ctx.GlobalBool(TestNetFlag.Name) {
		c = core.TestConfig
	}

	for i := range c.Forks {
		// Force override any existing configs if explicitly requested
		if c.Forks[i].Name == "ETF" {
			if ctx.GlobalBool(ETFChain.Name) {
				c.Forks[i].Support = true
			}
		}
	}

	separator := strings.Repeat("-", 110)
	glog.V(logger.Warn).Info(separator)
	glog.V(logger.Warn).Info(fmt.Sprintf("Starting Geth Classic \x1b[32m%s\x1b[39m", ctx.App.Version))

	genesis := core.GetBlock(db, core.GetCanonicalHash(db, 0))
	genesisHash := ""
	if genesis != nil {
		genesisHash = genesis.Hash().Hex()
	}
	glog.V(logger.Warn).Info(fmt.Sprintf("Loading blockchain: \x1b[36mgenesis\x1b[39m block \x1b[36m%s\x1b[39m.", genesisHash))
	glog.V(logger.Warn).Info(fmt.Sprintf("%v blockchain hard-forks associated with this genesis block:", len(c.Forks)))

	netsplitChoice := ""
	for i := range c.Forks {
		if c.Forks[i].NetworkSplit {
			netsplitChoice = fmt.Sprintf("resulted in a network split (support: %t)", c.Forks[i].Support)
		} else {
			netsplitChoice = ""
		}
		glog.V(logger.Warn).Info(fmt.Sprintf(" %7v %v hard-fork %v", c.Forks[i].Block, c.Forks[i].Name, netsplitChoice))
	}

	if ctx.GlobalBool(TestNetFlag.Name) {
		glog.V(logger.Warn).Info("Geth is configured to use the \x1b[33mEthereum (ETC) Testnet\x1b[39m blockchain!")
	} else {
		glog.V(logger.Warn).Info("Geth is configured to use the \x1b[32mEthereum (ETC) Classic\x1b[39m blockchain!")
	}
	glog.V(logger.Warn).Info(separator)
	return c
}

// MakeChainDatabase open an LevelDB using the flags passed to the client and will hard crash if it fails.
func MakeChainDatabase(ctx *cli.Context) ethdb.Database {
	var (
		datadir = MustMakeDataDir(ctx)
		cache   = ctx.GlobalInt(CacheFlag.Name)
		handles = MakeDatabaseHandles()
	)

	chainDb, err := ethdb.NewLDBDatabase(filepath.Join(datadir, "chaindata"), cache, handles)
	if err != nil {
		log.Fatal("Could not open database: ", err)
	}
	return chainDb
}

// MakeChain creates a chain manager from set command line flags.
func MakeChain(ctx *cli.Context) (chain *core.BlockChain, chainDb ethdb.Database) {
	var err error
	chainDb = MakeChainDatabase(ctx)

	chainConfig := MustMakeChainConfigFromDb(ctx, chainDb)

	pow := pow.PoW(core.FakePow{})
	if !ctx.GlobalBool(FakePoWFlag.Name) {
		pow = ethash.New()
	}
	chain, err = core.NewBlockChain(chainDb, chainConfig, pow, new(event.TypeMux))
	if err != nil {
		log.Fatal("Could not start chainmanager: ", err)
	}
	return chain, chainDb
}

// MakeConsolePreloads retrieves the absolute paths for the console JavaScript
// scripts to preload before starting.
func MakeConsolePreloads(ctx *cli.Context) []string {
	// Skip preloading if there's nothing to preload
	if ctx.GlobalString(PreloadJSFlag.Name) == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	preloads := []string{}

	assets := ctx.GlobalString(JSpathFlag.Name)
	for _, file := range strings.Split(ctx.GlobalString(PreloadJSFlag.Name), ",") {
		preloads = append(preloads, common.AbsolutePath(assets, strings.TrimSpace(file)))
	}
	return preloads
}