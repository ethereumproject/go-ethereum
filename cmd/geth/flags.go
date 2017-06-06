package main

import (
	"math/big"
	"runtime"

	"strings"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/eth"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/rpc"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
)

// These are all the command line flags we support.
// If you add to this list, please remember to include the
// flag in the appropriate command definition.
//
// The flags are defined here so their names and help texts
// are the same for all commands.

var (
	// General settings
	DataDirFlag = DirectoryFlag{
		Name:  "data-dir,datadir",
		Usage: "Data directory for the databases and keystore",
		Value: DirectoryString{common.DefaultDataDir()},
	}
	KeyStoreDirFlag = DirectoryFlag{
		Name:  "keystore",
		Usage: "Directory path for the keystore",
	}
	ChainIdentityFlag = cli.StringFlag{
		Name: "chain",
		Usage: `Identifier of blockchain network to use (default='mainnet', test='morden').
	If using a custom identity (i.e. --chain=custom),
	there must be a valid JSON chain configuration
	file at <datadir>/custom/chain.json`,
		Value: core.DefaultChainConfigID,
	}
	NetworkIdFlag = cli.IntFlag{
		Name:  "network-id, networkid",
		Usage: "Network identifier (integer: 1=Homestead, 2=Morden)",
		Value: eth.NetworkId,
	}
	TestNetFlag = cli.BoolFlag{
		Name:  "testnet",
		Usage: "[Use: --chain=morden] Morden network: pre-configured test network with modified starting nonces (replay protection)",
	}
	DevModeFlag = cli.BoolFlag{
		Name:  "dev",
		Usage: "Developer mode: pre-configured private network with several debugging flags",
	}
	NodeNameFlag = cli.StringFlag{
		Name:  "identity,name",
		Usage: "Custom node name",
	}
	NatspecEnabledFlag = cli.BoolFlag{
		Name:  "natspec",
		Usage: "Enable NatSpec confirmation notice",
	}
	DocRootFlag = DirectoryFlag{
		Name:  "doc-root,docroot",
		Usage: "Document Root for HTTPClient file scheme",
		Value: DirectoryString{common.HomeDir()},
	}
	CacheFlag = cli.IntFlag{
		Name:  "cache",
		Usage: "Megabytes of memory allocated to internal caching (min 16MB / database forced)",
		Value: 128,
	}
	BlockchainVersionFlag = cli.IntFlag{
		Name:  "blockchain-version,blockchainversion",
		Usage: "Blockchain version (integer)",
		Value: core.BlockChainVersion,
	}
	FastSyncFlag = cli.BoolFlag{
		Name:  "fast",
		Usage: "Enable fast syncing through state downloads",
	}
	LightKDFFlag = cli.BoolFlag{
		Name:  "light-kdf,lightkdf",
		Usage: "Reduce key-derivation RAM & CPU usage at some expense of KDF strength",
	}
	// Network Split settings
	ETFChain = cli.BoolFlag{
		Name:  "etf",
		Usage: "Updates the chain rules to use the ETF hard-fork blockchain",
	}
	// Miner settings
	// TODO Refactor CPU vs GPU mining flags
	MiningEnabledFlag = cli.BoolFlag{
		Name:  "mine",
		Usage: "Enable mining",
	}
	MinerThreadsFlag = cli.IntFlag{
		Name:  "miner-threads,minerthreads",
		Usage: "Number of CPU threads to use for mining",
		Value: runtime.NumCPU(),
	}
	MiningGPUFlag = cli.StringFlag{
		Name:  "miner-gpus,minergpus",
		Usage: "List of GPUs to use for mining (e.g. '0,1' will use the first two GPUs found)",
	}
	TargetGasLimitFlag = cli.StringFlag{
		Name:  "target-gas-limit,targetgaslimit",
		Usage: "Target gas limit sets the artificial target gas floor for the blocks to mine",
		Value: core.TargetGasLimit.String(),
	}
	AutoDAGFlag = cli.BoolFlag{
		Name:  "auto-dag,autodag",
		Usage: "Enable automatic DAG pregeneration",
	}
	EtherbaseFlag = cli.StringFlag{
		Name:  "etherbase",
		Usage: "Public address for block mining rewards (default = first account created)",
		Value: "0",
	}
	GasPriceFlag = cli.StringFlag{
		Name:  "gas-price,gasprice",
		Usage: "Minimal gas price to accept for mining a transactions",
		Value: new(big.Int).Mul(big.NewInt(20), common.Shannon).String(),
	}
	ExtraDataFlag = cli.StringFlag{
		Name:  "extra-data,extradata",
		Usage: "Freeform header field set by the miner",
	}
	// Account settings
	UnlockedAccountFlag = cli.StringFlag{
		Name:  "unlock",
		Usage: "Comma separated list of accounts to unlock",
		Value: "",
	}
	PasswordFileFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Password file to use for non-inteactive password input",
		Value: "",
	}

	// logging and debug settings
	VerbosityFlag = cli.GenericFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=core, 5=debug, 6=detail",
		Value: glog.GetVerbosity(),
	}
	VModuleFlag = cli.GenericFlag{
		Name:  "vmodule",
		Usage: "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=6,p2p=5)",
		Value: glog.GetVModule(),
	}
	LogDirFlag = DirectoryFlag{
		Name:  "log-dir,logdir",
		Usage: "Directory in which to write log files, redirecting terminal out (stderr)",
		Value: DirectoryString{filepath.Join(common.DefaultDataDir(), "logs")},
	}
	BacktraceAtFlag = cli.GenericFlag{
		Name:  "backtrace",
		Usage: "Request a stack trace at a specific logging statement (e.g. \"block.go:271\")",
		Value: glog.GetTraceLocation(),
	}
	MetricsFlag = cli.StringFlag{
		Name:  "metrics",
		Usage: "Enables metrics reporting. When the value is a path, either relative or absolute, then a log is written to the respective file.",
	}
	FakePoWFlag = cli.BoolFlag{
		Name:  "fake-pow, fakepow",
		Usage: "Disables proof-of-work verification",
	}

	// RPC settings
	RPCEnabledFlag = cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}
	RPCListenAddrFlag = cli.StringFlag{
		Name:  "rpc-addr,rpcaddr",
		Usage: "HTTP-RPC server listening interface",
		Value: common.DefaultHTTPHost,
	}
	RPCPortFlag = cli.IntFlag{
		Name:  "rpc-port,rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: common.DefaultHTTPPort,
	}
	RPCCORSDomainFlag = cli.StringFlag{
		Name:  "rpc-cors-domain,rpccorsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value: "",
	}
	RPCApiFlag = cli.StringFlag{
		Name:  "rpc-api,rpcapi",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: rpc.DefaultHTTPApis,
	}
	IPCDisabledFlag = cli.BoolFlag{
		Name:  "ipc-disable,ipcdisable",
		Usage: "Disable the IPC-RPC server",
	}
	IPCApiFlag = cli.StringFlag{
		Name:  "ipc-api,ipcapi",
		Usage: "API's offered over the IPC-RPC interface",
		Value: rpc.DefaultIPCApis,
	}
	IPCPathFlag = DirectoryFlag{
		Name:  "ipc-path,ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
		Value: DirectoryString{common.DefaultIPCSocket},
	}
	WSEnabledFlag = cli.BoolFlag{
		Name:  "ws",
		Usage: "Enable the WS-RPC server",
	}
	WSListenAddrFlag = cli.StringFlag{
		Name:  "ws-addr,wsaddr",
		Usage: "WS-RPC server listening interface",
		Value: common.DefaultWSHost,
	}
	WSPortFlag = cli.IntFlag{
		Name:  "ws-port,wsport",
		Usage: "WS-RPC server listening port",
		Value: common.DefaultWSPort,
	}
	WSApiFlag = cli.StringFlag{
		Name:  "ws-api,wsapi",
		Usage: "API's offered over the WS-RPC interface",
		Value: rpc.DefaultHTTPApis,
	}
	WSAllowedOriginsFlag = cli.StringFlag{
		Name:  "ws-origins,wsorigins",
		Usage: "Origins from which to accept websockets requests",
		Value: "",
	}
	ExecFlag = cli.StringFlag{
		Name:  "exec",
		Usage: "Execute JavaScript statement (only in combination with console/attach)",
	}
	PreloadJSFlag = cli.StringFlag{
		Name:  "preload",
		Usage: "Comma separated list of JavaScript files to preload into the console",
	}

	// Network Settings
	MaxPeersFlag = cli.IntFlag{
		Name:  "max-peers,maxpeers",
		Usage: "Maximum number of network peers (network disabled if set to 0)",
		Value: 25,
	}
	MaxPendingPeersFlag = cli.IntFlag{
		Name:  "max-pend-peers,maxpendpeers",
		Usage: "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value: 0,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port",
		Usage: "Network listening port",
		Value: 30303,
	}
	BootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated enode URLs for P2P discovery bootstrap",
		Value: "",
	}
	NodeKeyFileFlag = cli.StringFlag{
		Name:  "nodekey",
		Usage: "P2P node key file",
	}
	NodeKeyHexFlag = cli.StringFlag{
		Name:  "nodekey-hex,nodekeyhex",
		Usage: "P2P node key as hex (for testing)",
	}
	NATFlag = cli.StringFlag{
		Name:  "nat",
		Usage: "NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value: "any",
	}
	NoDiscoverFlag = cli.BoolFlag{
		Name:  "no-discover,nodiscover",
		Usage: "Disables the peer discovery mechanism (manual peer addition)",
	}
	WhisperEnabledFlag = cli.BoolFlag{
		Name:  "shh",
		Usage: "Enable Whisper",
	}
	// ATM the url is left to the user and deployment to
	JSpathFlag = cli.StringFlag{
		Name:  "js-path,jspath",
		Usage: "JavaScript root path for `loadScript` and document root for `admin.httpGet`",
		Value: ".",
	}
	SolcPathFlag = cli.StringFlag{
		Name:  "solc",
		Usage: "Solidity compiler command to be used",
		Value: "solc",
	}

	// Gas price oracle settings
	GpoMinGasPriceFlag = cli.StringFlag{
		Name:  "gpo-min,gpomin",
		Usage: "Minimum suggested gas price",
		Value: new(big.Int).Mul(big.NewInt(20), common.Shannon).String(),
	}
	GpoMaxGasPriceFlag = cli.StringFlag{
		Name:  "gpo-max,gpomax",
		Usage: "Maximum suggested gas price",
		Value: new(big.Int).Mul(big.NewInt(500), common.Shannon).String(),
	}
	GpoFullBlockRatioFlag = cli.IntFlag{
		Name:  "gpo-full,gpofull",
		Usage: "Full block threshold for gas price calculation (%)",
		Value: 80,
	}
	GpobaseStepDownFlag = cli.IntFlag{
		Name:  "gpo-base-down,gpobasedown",
		Usage: "Suggested gas price base step down ratio (1/1000)",
		Value: 10,
	}
	GpobaseStepUpFlag = cli.IntFlag{
		Name:  "gpo-base-up,gpobaseup",
		Usage: "Suggested gas price base step up ratio (1/1000)",
		Value: 100,
	}
	GpobaseCorrectionFactorFlag = cli.IntFlag{
		Name:  "gpo-base-cf,gpobasecf",
		Usage: "Suggested gas price base correction factor (%)",
		Value: 110,
	}
	Unused1 = cli.BoolFlag{
		Name:  "oppose-dao-fork",
		Usage: "Use classic blockchain (always set, flag is unused and exists for compatibility only)",
	}
)

// aliasableName check global vars for aliases flag and directoryFlag names.
// These can be "comma,sep-arated", and ensures that if one is set (and/or not the other),
// only one var will be returned and it will be decided in order of appearance.
func aliasableName(commaSeparatedName string, ctx *cli.Context) string {
	names := strings.Split(commaSeparatedName, ",")
	returnName := names[0] // first name as default
	last := len(names) - 1
	// for in reverse, so that we prioritize the first appearing
	for i := last; i >= 0; i-- {
		if ctx.GlobalIsSet(strings.TrimSpace(names[i])) {
			returnName = names[i]
		}
	}
	return returnName
}
