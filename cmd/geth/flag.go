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
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"errors"

	"github.com/ethereumproject/ethash"
	"github.com/ethereumproject/go-ethereum/accounts"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core"
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
	"github.com/ethereumproject/go-ethereum/core/state"
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

var (
	// Errors.
	ErrInvalidFlag        = errors.New("invalid flag or context value")
	ErrInvalidChainID     = errors.New("invalid chainID")
	ErrDirectoryStructure = errors.New("error in directory structure")
	ErrStackFail          = errors.New("error in stack protocol")

	// Chain identities.
	chainIdentitiesBlacklist = map[string]bool{
		"chaindata": true,
		"dapp":      true,
		"keystore":  true,
		"nodekey":   true,
		"nodes":     true,
	}
	chainIdentitiesMain = map[string]bool{
		"main":    true,
		"mainnet": true,
	}
	chainIdentitiesMorden = map[string]bool{
		"morden":  true,
		"testnet": true,
	}

	devModeDataDirPath = filepath.Join(os.TempDir(), "/ethereum_dev_mode")
)

// chainIsMorden allows either
// '--testnet' (legacy), or
// '--chain=morden|testnet'
func chainIsMorden(ctx *cli.Context) bool {
	if ctx.GlobalIsSet(aliasableName(TestNetFlag.Name, ctx)) && ctx.GlobalIsSet(aliasableName(ChainIdentityFlag.Name, ctx)) {
		glog.Fatalf(`%v: used redundant/conflicting flags: %v, %v
		Please use one or the other, but not both.`, ErrInvalidFlag, aliasableName(TestNetFlag.Name, ctx), aliasableName(ChainIdentityFlag.Name, ctx))
		return false
	}
	return ctx.GlobalBool(aliasableName(TestNetFlag.Name, ctx)) || chainIdentitiesMorden[ctx.GlobalString(aliasableName(ChainIdentityFlag.Name, ctx))]
}

// getChainIdentity parses --chain and --testnet (legacy) flags.
// It will fatal if finds notok value.
// It returns one of valid strings: ["mainnet", "morden", or --chain="flaggedCustom"]
func mustMakeChainIdentity(ctx *cli.Context) string {
	if chainIsMorden(ctx) {
		return core.DefaultTestnetChainConfigID // this makes '--testnet', '--chain=testnet', and '--chain=morden' all use the same /morden subdirectory, if --chain isn't specified
	}
	// If --chain is in use.
	if chainFlagVal := ctx.GlobalString(aliasableName(ChainIdentityFlag.Name, ctx)); chainFlagVal != "" {
		if chainIdentitiesMain[chainFlagVal] {
			return core.DefaultChainConfigID
		}
		// Check for unallowed values.
		if chainIdentitiesBlacklist[chainFlagVal] {
			glog.Fatalf(`%v: %v: reserved word
					reserved words for --chain flag include: 'chaindata', 'dapp', 'keystore', 'nodekey', 'nodes',
					please use a different identifier`, ErrInvalidFlag, ErrInvalidChainID)
			return ""
		}
		// Check for arguments that look like paths (not allowed).
		if dir, _ := filepath.Split(chainFlagVal); dir != "" {
			glog.Fatalf(`%v: %v: chain identity should not be a path`, ErrInvalidFlag, ErrInvalidChainID)
			return ""
		}
		return chainFlagVal
	} else if ctx.GlobalIsSet(aliasableName(ChainIdentityFlag.Name, ctx)) {
		glog.Fatalf("%v: %v: chainID empty", ErrInvalidFlag, ErrInvalidChainID)
		return ""
	}
	// If no relevant flag is set, return default mainnet.
	return core.DefaultChainConfigID
}

// mustMakeChainConfigNameDefaulty gets mainnet or testnet defaults if in use.
// _If a custom net is in use, it echoes the name of the ChainConfigID._
// It is intended to be a human-readable name for a chain configuration.
// - It should only be called in reference to default configuration (name will be configured
// separately through external JSON config otherwise).
func mustMakeChainConfigNameDefaulty(ctx *cli.Context) string {
	if chainIsMorden(ctx) {
		return core.DefaultTestnetChainConfigName
	}
	return core.DefaultChainConfigName
}

// mustMakeDataDir retrieves the currently requested data directory, terminating
// if none (or the empty string) is specified.
// --> <home>/<EthereumClassic>(defaulty) or --datadir
func mustMakeDataDir(ctx *cli.Context) string {
	if !ctx.GlobalIsSet(aliasableName(DataDirFlag.Name, ctx)) {
		if ctx.GlobalBool(aliasableName(DevModeFlag.Name, ctx)) {
			return devModeDataDirPath
		}
	}
	if path := ctx.GlobalString(aliasableName(DataDirFlag.Name, ctx)); path != "" {
		return path
	}

	glog.Fatalf("%v: cannot determine data directory, please set manually (--%v)", ErrDirectoryStructure, DataDirFlag.Name)
	return ""
}

// MustMakeChainDataDir retrieves the currently requested data directory including chain-specific subdirectory.
// A subdir of the datadir is used for each chain configuration ("/mainnet", "/testnet", "/my-custom-net").
// --> <home>/<EthereumClassic>/<mainnet|testnet|custom-net>, per --chain
func MustMakeChainDataDir(ctx *cli.Context) string {
	rp := common.EnsurePathAbsoluteOrRelativeTo(mustMakeDataDir(ctx), mustMakeChainIdentity(ctx))
	if !filepath.IsAbs(rp) {
		af, e := filepath.Abs(rp)
		if e != nil {
			glog.Fatalf("cannot make absolute path for chain data dir: %v: %v", rp, e)
		}
		rp = af
	}
	return rp
}

// MakeIPCPath creates an IPC path configuration from the set command line flags,
// returning an empty string if IPC was explicitly disabled, or the set path.
func MakeIPCPath(ctx *cli.Context) string {
	if ctx.GlobalBool(aliasableName(IPCDisabledFlag.Name, ctx)) {
		return ""
	}
	return ctx.GlobalString(aliasableName(IPCPathFlag.Name, ctx))
}

// MakeNodeKey creates a node key from set command line flags, either loading it
// from a file or as a specified hex value. If neither flags were provided, this
// method returns nil and an emphemeral key is to be generated.
func MakeNodeKey(ctx *cli.Context) *ecdsa.PrivateKey {
	var (
		hex  = ctx.GlobalString(aliasableName(NodeKeyHexFlag.Name, ctx))
		file = ctx.GlobalString(aliasableName(NodeKeyFileFlag.Name, ctx))

		key *ecdsa.PrivateKey
		err error
	)
	switch {
	case file != "" && hex != "":
		log.Fatalf("Options %q and %q are mutually exclusive", aliasableName(NodeKeyFileFlag.Name, ctx), aliasableName(NodeKeyHexFlag.Name, ctx))

	case file != "":
		if key, err = crypto.LoadECDSA(file); err != nil {
			log.Fatalf("Option %q: %v", aliasableName(NodeKeyFileFlag.Name, ctx), err)
		}

	case hex != "":
		if key, err = crypto.HexToECDSA(hex); err != nil {
			log.Fatalf("Option %q: %v", aliasableName(NodeKeyHexFlag.Name, ctx), err)
		}
	}
	return key
}

// MakeBootstrapNodesFromContext creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func MakeBootstrapNodesFromContext(ctx *cli.Context) []*discover.Node {
	// Return pre-configured nodes if none were manually requested
	if !ctx.GlobalIsSet(aliasableName(BootnodesFlag.Name, ctx)) {

		// --testnet/--chain=morden flag overrides --config flag
		if chainIsMorden(ctx) {
			return TestNetBootNodes
		}
		return HomesteadBootNodes
	}
	return core.ParseBootstrapNodeStrings(strings.Split(ctx.GlobalString(aliasableName(BootnodesFlag.Name, ctx)), ","))
}

// MakeListenAddress creates a TCP listening address string from set command
// line flags.
func MakeListenAddress(ctx *cli.Context) string {
	return fmt.Sprintf(":%d", ctx.GlobalInt(aliasableName(ListenPortFlag.Name, ctx)))
}

// MakeNAT creates a port mapper from set command line flags.
func MakeNAT(ctx *cli.Context) nat.Interface {
	natif, err := nat.Parse(ctx.GlobalString(aliasableName(NATFlag.Name, ctx)))
	if err != nil {
		log.Fatalf("Option %s: %v", aliasableName(NATFlag.Name, ctx), err)
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
	if !ctx.GlobalBool(aliasableName(RPCEnabledFlag.Name, ctx)) {
		return ""
	}
	return ctx.GlobalString(aliasableName(RPCListenAddrFlag.Name, ctx))
}

// MakeWSRpcHost creates the WebSocket RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func MakeWSRpcHost(ctx *cli.Context) string {
	if !ctx.GlobalBool(aliasableName(WSEnabledFlag.Name, ctx)) {
		return ""
	}
	return ctx.GlobalString(aliasableName(WSListenAddrFlag.Name, ctx))
}

// MakeDatabaseHandles raises out the number of allowed file handles per process
// for Geth and returns half of the allowance to assign to the database.
func MakeDatabaseHandles() int {
	if err := raiseFdLimit(2048); err != nil {
		glog.V(logger.Warn).Info("Failed to raise file descriptor allowance: ", err)
	}
	limit, err := getFdLimit()
	if err != nil {
		glog.V(logger.Warn).Info("Failed to retrieve file descriptor allowance: ", err)
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
	if ctx.GlobalBool(aliasableName(LightKDFFlag.Name, ctx)) {
		scryptN = accounts.LightScryptN
		scryptP = accounts.LightScryptP
	}

	datadir := MustMakeChainDataDir(ctx)

	keydir := filepath.Join(datadir, "keystore")
	if path := ctx.GlobalString(aliasableName(KeyStoreDirFlag.Name, ctx)); path != "" {
		af, e := filepath.Abs(path)
		if e != nil {
			glog.V(logger.Error).Infof("keydir path could not be made absolute: %v: %v", path, e)
		} else {
			keydir = af
		}
	}

	m, err := accounts.NewManager(keydir, scryptN, scryptP, ctx.GlobalBool(aliasableName(AccountsIndexFlag.Name, ctx)))
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
		return accounts.Account{}, fmt.Errorf("invalid account address or index: %q", account)
	}
	return accman.AccountByIndex(index)
}

// MakeEtherbase retrieves the etherbase either from the directly specified
// command line flags or from the keystore if CLI indexed.
func MakeEtherbase(accman *accounts.Manager, ctx *cli.Context) common.Address {
	accounts := accman.Accounts()
	if !ctx.GlobalIsSet(aliasableName(EtherbaseFlag.Name, ctx)) && len(accounts) == 0 {
		glog.V(logger.Error).Infoln("WARNING: No etherbase set and no accounts found as default")
		return common.Address{}
	}
	etherbase := ctx.GlobalString(aliasableName(EtherbaseFlag.Name, ctx))
	if etherbase == "" {
		return common.Address{}
	}
	// If the specified etherbase is a valid address, return it
	account, err := MakeAddress(accman, etherbase)
	if err != nil {
		log.Fatalf("Option %q: %v", aliasableName(EtherbaseFlag.Name, ctx), err)
	}
	return account.Address
}

// MakePasswordList reads password lines from the file specified by --password.
func MakePasswordList(ctx *cli.Context) []string {
	path := ctx.GlobalString(aliasableName(PasswordFileFlag.Name, ctx))
	if path == "" {
		return nil
	}
	text, err := ioutil.ReadFile(path)
	if err != nil {
		glog.Fatal("Failed to read password file: ", err)
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

// makeName makes the node name, which can be (in part) customized by the NodeNameFlag
func makeNodeName(version string, ctx *cli.Context) string {
	name := fmt.Sprintf("Geth/%s/%s/%s", version, runtime.GOOS, runtime.Version())
	if identity := ctx.GlobalString(aliasableName(NodeNameFlag.Name, ctx)); len(identity) > 0 {
		name += "/" + identity
	}
	return name
}

// MakeSystemNode sets up a local node, configures the services to launch and
// assembles the P2P protocol stack.
func MakeSystemNode(version string, ctx *cli.Context) *node.Node {

	// global settings

	if ctx.GlobalIsSet(aliasableName(ExtraDataFlag.Name, ctx)) {
		s := ctx.GlobalString(aliasableName(ExtraDataFlag.Name, ctx))
		if len(s) > types.HeaderExtraMax {
			log.Fatalf("%s flag %q exceeds size limit of %d", aliasableName(ExtraDataFlag.Name, ctx), s, types.HeaderExtraMax)
		}
		miner.HeaderExtra = []byte(s)
	}

	// Data migrations if should.
	if shouldAttemptDirMigration(ctx) {
		// Rename existing default datadir <home>/<Ethereum>/ to <home>/<EthereumClassic>.
		// Only do this if --datadir flag is not specified AND <home>/<EthereumClassic> does NOT already exist (only migrate once and only for defaulty).
		// If it finds an 'Ethereum' directory, it will check if it contains default ETC or ETHF chain data.
		// If it contains ETC data, it will rename the dir. If ETHF data, if will do nothing.
		if migrationError := migrateExistingDirToClassicNamingScheme(ctx); migrationError != nil {
			glog.Fatalf("%v: failed to migrate existing Classic database: %v", ErrDirectoryStructure, migrationError)
		}

		// Move existing mainnet data to pertinent chain-named subdir scheme (ie ethereum-classic/mainnet).
		// This should only happen if the given (newly defined in this protocol) subdir doesn't exist,
		// and the dirs&files (nodekey, dapp, keystore, chaindata, nodes) do exist,
		if subdirMigrateErr := migrateToChainSubdirIfNecessary(ctx); subdirMigrateErr != nil {
			glog.Fatalf("%v: failed to migrate existing data to chain-specific subdir: %v", ErrDirectoryStructure, subdirMigrateErr)
		}
	}

	// Makes sufficient configuration from JSON file or DB pending flags.
	// Delegates flag usage.
	config := mustMakeSufficientChainConfig(ctx)
	logChainConfiguration(ctx, config)

	// Configure the Ethereum service
	ethConf := mustMakeEthConf(ctx, config)

	// Configure node's service container.
	name := makeNodeName(version, ctx)
	stackConf, shhEnable := mustMakeStackConf(ctx, name, config)

	// Assemble and return the protocol stack
	stack, err := node.New(stackConf)
	if err != nil {
		glog.Fatalf("%v: failed to create the protocol stack: ", ErrStackFail, err)
	}
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return eth.New(ctx, ethConf)
	}); err != nil {
		glog.Fatalf("%v: failed to register the Ethereum service: ", ErrStackFail, err)
	}
	if shhEnable {
		if err := stack.Register(func(*node.ServiceContext) (node.Service, error) { return whisper.New(), nil }); err != nil {
			glog.Fatalf("%v: failed to register the Whisper service: ", ErrStackFail, err)
		}
	}

	if ctx.GlobalBool(Unused1.Name) {
		glog.V(logger.Info).Infoln(fmt.Sprintf("Geth started with --%s flag, which is unused by Geth Classic and can be omitted", Unused1.Name))
	}

	return stack
}

// shouldAttemptDirMigration decides based on flags if
// should attempt to migration from old (<=3.3) directory schema to new.
func shouldAttemptDirMigration(ctx *cli.Context) bool {
	if !ctx.GlobalIsSet(aliasableName(DataDirFlag.Name, ctx)) {
		if chainVal := mustMakeChainIdentity(ctx); chainIdentitiesMain[chainVal] || chainIdentitiesMorden[chainVal] {
			return true
		}
	}
	return false
}

func mustMakeStackConf(ctx *cli.Context, name string, config *core.SufficientChainConfig) (stackConf *node.Config, shhEnable bool) {
	// Configure the node's service container
	stackConf = &node.Config{
		DataDir:         MustMakeChainDataDir(ctx),
		PrivateKey:      MakeNodeKey(ctx),
		Name:            name,
		NoDiscovery:     ctx.GlobalBool(aliasableName(NoDiscoverFlag.Name, ctx)),
		BootstrapNodes:  config.ParsedBootstrap,
		ListenAddr:      MakeListenAddress(ctx),
		NAT:             MakeNAT(ctx),
		MaxPeers:        ctx.GlobalInt(aliasableName(MaxPeersFlag.Name, ctx)),
		MaxPendingPeers: ctx.GlobalInt(aliasableName(MaxPendingPeersFlag.Name, ctx)),
		IPCPath:         MakeIPCPath(ctx),
		HTTPHost:        MakeHTTPRpcHost(ctx),
		HTTPPort:        ctx.GlobalInt(aliasableName(RPCPortFlag.Name, ctx)),
		HTTPCors:        ctx.GlobalString(aliasableName(RPCCORSDomainFlag.Name, ctx)),
		HTTPModules:     MakeRPCModules(ctx.GlobalString(aliasableName(RPCApiFlag.Name, ctx))),
		WSHost:          MakeWSRpcHost(ctx),
		WSPort:          ctx.GlobalInt(aliasableName(WSPortFlag.Name, ctx)),
		WSOrigins:       ctx.GlobalString(aliasableName(WSAllowedOriginsFlag.Name, ctx)),
		WSModules:       MakeRPCModules(ctx.GlobalString(aliasableName(WSApiFlag.Name, ctx))),
	}

	// Configure the Whisper service
	shhEnable = ctx.GlobalBool(aliasableName(WhisperEnabledFlag.Name, ctx))

	// Override any default configs in dev mode
	if ctx.GlobalBool(aliasableName(DevModeFlag.Name, ctx)) {
		if !ctx.GlobalIsSet(aliasableName(MaxPeersFlag.Name, ctx)) {
			stackConf.MaxPeers = 0
		}
		// From p2p/server.go:
		// If the port is zero, the operating system will pick a port. The
		// ListenAddr field will be updated with the actual address when
		// the server is started.
		if !ctx.GlobalIsSet(aliasableName(ListenPortFlag.Name, ctx)) {
			stackConf.ListenAddr = ":0"
		}
		if !ctx.GlobalIsSet(aliasableName(WhisperEnabledFlag.Name, ctx)) {
			shhEnable = true
		}
	}

	return stackConf, shhEnable
}

func mustMakeEthConf(ctx *cli.Context, sconf *core.SufficientChainConfig) *eth.Config {

	accman := MakeAccountManager(ctx)
	passwords := MakePasswordList(ctx)

	accounts := strings.Split(ctx.GlobalString(aliasableName(UnlockedAccountFlag.Name, ctx)), ",")
	for i, account := range accounts {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			unlockAccount(ctx, accman, trimmed, i, passwords)
		}
	}

	ethConf := &eth.Config{
		ChainConfig:             sconf.ChainConfig,
		Genesis: sconf.Genesis,
		FastSync:                ctx.GlobalBool(aliasableName(FastSyncFlag.Name, ctx)),
		BlockChainVersion:       ctx.GlobalInt(aliasableName(BlockchainVersionFlag.Name, ctx)),
		DatabaseCache:           ctx.GlobalInt(aliasableName(CacheFlag.Name, ctx)),
		DatabaseHandles:         MakeDatabaseHandles(),
		NetworkId:               sconf.Network,
		AccountManager:          accman,
		Etherbase:               MakeEtherbase(accman, ctx),
		MinerThreads:            ctx.GlobalInt(aliasableName(MinerThreadsFlag.Name, ctx)),
		NatSpec:                 ctx.GlobalBool(aliasableName(NatspecEnabledFlag.Name, ctx)),
		DocRoot:                 ctx.GlobalString(aliasableName(DocRootFlag.Name, ctx)),
		GasPrice:                new(big.Int),
		GpoMinGasPrice:          new(big.Int),
		GpoMaxGasPrice:          new(big.Int),
		GpoFullBlockRatio:       ctx.GlobalInt(aliasableName(GpoFullBlockRatioFlag.Name, ctx)),
		GpobaseStepDown:         ctx.GlobalInt(aliasableName(GpobaseStepDownFlag.Name, ctx)),
		GpobaseStepUp:           ctx.GlobalInt(aliasableName(GpobaseStepUpFlag.Name, ctx)),
		GpobaseCorrectionFactor: ctx.GlobalInt(aliasableName(GpobaseCorrectionFactorFlag.Name, ctx)),
		SolcPath:                ctx.GlobalString(aliasableName(SolcPathFlag.Name, ctx)),
		AutoDAG:                 ctx.GlobalBool(aliasableName(AutoDAGFlag.Name, ctx)) || ctx.GlobalBool(aliasableName(MiningEnabledFlag.Name, ctx)),
	}

	if _, ok := ethConf.GasPrice.SetString(ctx.GlobalString(aliasableName(GasPriceFlag.Name, ctx)), 0); !ok {
		log.Fatalf("malformed %s flag value %q", aliasableName(GasPriceFlag.Name, ctx), ctx.GlobalString(aliasableName(GasPriceFlag.Name, ctx)))
	}
	if _, ok := ethConf.GpoMinGasPrice.SetString(ctx.GlobalString(aliasableName(GpoMinGasPriceFlag.Name, ctx)), 0); !ok {
		log.Fatalf("malformed %s flag value %q", aliasableName(GpoMinGasPriceFlag.Name, ctx), ctx.GlobalString(aliasableName(GpoMinGasPriceFlag.Name, ctx)))
	}
	if _, ok := ethConf.GpoMaxGasPrice.SetString(ctx.GlobalString(aliasableName(GpoMaxGasPriceFlag.Name, ctx)), 0); !ok {
		log.Fatalf("malformed %s flag value %q", aliasableName(GpoMaxGasPriceFlag.Name, ctx), ctx.GlobalString(aliasableName(GpoMaxGasPriceFlag.Name, ctx)))
	}

	switch sconf.Consensus {
	case "ethash-test":
		ethConf.PowTest = true
	}

	// Override any default configs in dev mode
	if ctx.GlobalBool(aliasableName(DevModeFlag.Name, ctx)) {
		// Override the Ethereum protocol configs
		if !ctx.GlobalIsSet(aliasableName(GasPriceFlag.Name, ctx)) {
			ethConf.GasPrice = new(big.Int)
		}
	}

	return ethConf
}

// mustMakeSufficientChainConfig makes a sufficent chain configuration (id, chainconfig, nodes,...)
// based on --chain or defaults or fails hard.
// - User must provide a full and complete config file if any is specified located at /custom/chain.json
// - Note: Function reads custom config file each time it is called; this could be altered if desired, but I (whilei) felt
// reading a file a couple of times was more efficient than storing a global.
func mustMakeSufficientChainConfig(ctx *cli.Context) *core.SufficientChainConfig {

	config := &core.SufficientChainConfig{}
	defer func() {
		if ctx.GlobalBool(aliasableName(DevModeFlag.Name, ctx)) {
			config.Consensus = "ethash-test"
		}
	}()

	chainIdentity := mustMakeChainIdentity(ctx)

	if chainIdentitiesMain[chainIdentity] || chainIdentitiesMorden[chainIdentity] {
		// Initialise chain configuration before handling migrations or setting up node.
		config.Identity = chainIdentity
		config.Name = mustMakeChainConfigNameDefaulty(ctx)
		config.Network = eth.NetworkId // 1, default mainnet
		config.Consensus = "ethash"
		config.Genesis = core.DefaultGenesis
		config.ChainConfig = MustMakeChainConfigFromDefaults(ctx).SortForks()
		config.ParsedBootstrap = MakeBootstrapNodesFromContext(ctx)
		if chainIsMorden(ctx) {
			config.Network = 2
			config.Genesis = core.TestNetGenesis
			state.StartingNonce = state.DefaultTestnetStartingNonce // (2**20)
		}
		return config
	}

	// Returns surely valid suff chain config.
	chainDir := MustMakeChainDataDir(ctx)
	configPath := filepath.Join(chainDir, "chain.json")
	if _, de := os.Stat(configPath); de != nil && os.IsNotExist(de) {
		glog.Fatalf(`%v: %v
		It looks like you haven't set up your custom chain yet...
		Here's a possible workflow for that:

		$ geth --chain morden dump-chain-config %v/chain.json
		$ sed -i.bak s/morden/%v/ %v/chain.json
		$ vi %v/chain.json # <- make your customizations
		`, core.ErrChainConfigNotFound, configPath,
			chainDir, chainIdentity, chainDir, chainDir)
	}
	config, err := core.ReadExternalChainConfig(configPath)
	if err != nil {
		glog.Fatalf(`invalid external configuration JSON: '%v': %v
		Valid custom configuration JSON file must be named 'chain.json' and be located in respective chain subdir.`, configPath, err)
	}

	// Ensure JSON 'id' value matches name of parent chain subdir.
	if config.Identity != chainIdentity {
		glog.Fatalf(`%v: JSON 'id' value in external config file (%v) must match name of parent subdir (%v)`, ErrInvalidChainID, config.Identity, chainIdentity)
	}

	// Check for flagged bootnodes.
	if ctx.GlobalIsSet(aliasableName(BootnodesFlag.Name, ctx)) {
		glog.Fatalf("Conflicting custom chain config and --%s flags. Please use either but not both.", aliasableName(BootnodesFlag.Name, ctx))
	}

	// Set statedb StartingNonce from external config, if specified (is optional)
	if config.State != nil {
		if sn := config.State.StartingNonce; sn != 0 {
			state.StartingNonce = sn
		}
	}

	return config
}

func logChainConfiguration(ctx *cli.Context, config *core.SufficientChainConfig) {

	chainIdentity := mustMakeChainIdentity(ctx)
	chainIsCustom := !(chainIdentitiesMain[chainIdentity] || chainIdentitiesMorden[chainIdentity])
	if chainIsCustom {
		glog.V(logger.Info).Infof("Using custom chain configuration file: \x1b[32m%s\x1b[39m", filepath.Clean(filepath.Join(MustMakeChainDataDir(ctx), "chain.json")))
	}

	glog.V(logger.Info).Info(glog.Separator("-"))

	glog.V(logger.Info).Infof("Starting Geth Classic \x1b[32m%s\x1b[39m", ctx.App.Version)
	glog.V(logger.Info).Infof("Geth is configured to use blockchain: \x1b[32m%v (ETC)\x1b[39m", config.Name)
	glog.V(logger.Info).Infof("Using chain database at: \x1b[32m%s\x1b[39m", MustMakeChainDataDir(ctx)+"/chaindata")

	glog.V(logger.Info).Infof("%v blockchain upgrades associated with this configuration:", len(config.ChainConfig.Forks))

	for i := range config.ChainConfig.Forks {
		glog.V(logger.Info).Infof(" %7v %v", config.ChainConfig.Forks[i].Block, config.ChainConfig.Forks[i].Name)
		if !config.ChainConfig.Forks[i].RequiredHash.IsEmpty() {
			glog.V(logger.Info).Infof("         with block %v", config.ChainConfig.Forks[i].RequiredHash.Hex())
		}
		for _, feat := range config.ChainConfig.Forks[i].Features {
			glog.V(logger.Debug).Infof("    id: %v", feat.ID)
			for k, v := range feat.Options {
				glog.V(logger.Debug).Infof("        %v: %v", k, v)
			}
		}
	}

	if chainIsCustom {
		glog.V(logger.Info).Infof("State starting nonce: %v", colorGreen(state.StartingNonce))
	}

	glog.V(logger.Info).Info(glog.Separator("-"))
}

// MustMakeChainConfigFromDefaults reads the chain configuration from hardcode.
func MustMakeChainConfigFromDefaults(ctx *cli.Context) *core.ChainConfig {
	c := core.DefaultConfig
	if chainIsMorden(ctx) {
		c = core.TestConfig
	}
	return c
}

// MakeChainDatabase open an LevelDB using the flags passed to the client and will hard crash if it fails.
func MakeChainDatabase(ctx *cli.Context) ethdb.Database {
	var (
		datadir = MustMakeChainDataDir(ctx)
		cache   = ctx.GlobalInt(aliasableName(CacheFlag.Name, ctx))
		handles = MakeDatabaseHandles()
	)

	chainDb, err := ethdb.NewLDBDatabase(filepath.Join(datadir, "chaindata"), cache, handles)
	if err != nil {
		glog.Fatal("Could not open database: ", err)
	}
	return chainDb
}

// MakeChain creates a chain manager from set command line flags.
func MakeChain(ctx *cli.Context) (chain *core.BlockChain, chainDb ethdb.Database) {
	var err error
	sconf := mustMakeSufficientChainConfig(ctx)
	chainDb = MakeChainDatabase(ctx)

	pow := pow.PoW(core.FakePow{})
	if !ctx.GlobalBool(aliasableName(FakePoWFlag.Name, ctx)) {
		pow = ethash.New()
	} else {
		glog.V(logger.Info).Info("Consensus: fake")
	}

	chain, err = core.NewBlockChain(chainDb, sconf.ChainConfig, pow, new(event.TypeMux))
	if err != nil {
		glog.Fatal("Could not start chainmanager: ", err)
	}
	return chain, chainDb
}

// MakeConsolePreloads retrieves the absolute paths for the console JavaScript
// scripts to preload before starting.
func MakeConsolePreloads(ctx *cli.Context) []string {
	// Skip preloading if there's nothing to preload
	if ctx.GlobalString(aliasableName(PreloadJSFlag.Name, ctx)) == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	preloads := []string{}

	assets := ctx.GlobalString(aliasableName(JSpathFlag.Name, ctx))
	for _, file := range strings.Split(ctx.GlobalString(aliasableName(PreloadJSFlag.Name, ctx)), ",") {
		preloads = append(preloads, common.EnsurePathAbsoluteOrRelativeTo(assets, strings.TrimSpace(file)))
	}
	return preloads
}
