package main

import "os"

func ExampleAppRunHelpCommand() {
	os.Args = []string{"geth help"}
	app := makeCLIApp()
	app.Run(os.Args)
	// Output:
	// NAME:
	//    geth - the go-ethereum command line interface

	// USAGE:
	//    geth [options] command [command options] [arguments...]
	// VERSION:
	//    source
	// COMMANDS:
	//    import			import a blockchain file
	//    export			export blockchain into file
	//    upgradedb			upgrade chainblock database
	//    removedb			Remove blockchain and state databases
	//    dump				dump a specific block from storage
	//    monitor			Geth Monitor: node metrics monitoring and visualization
	//    account			manage accounts
	//    wallet			ethereum presale wallet
	//    console			Geth Console: interactive JavaScript environment
	//    attach			Geth Console: interactive JavaScript environment (connect to node)
	//    js				executes the given JavaScript files in the Geth JavaScript VM
	//    makedag			generate ethash dag (for testing)
	//    gpuinfo			gpuinfo
	//    gpubench			benchmark GPU
	//    version			print ethereum version numbers
	//    init				bootstraps and initialises a new genesis block (JSON)
	//    dumpExternalChainConfig	dump current chain configuration to JSON file [REQUIRED argument: filepath.json]
	//    help, h			Shows a list of commands or help for one command

	// ETHEREUM OPTIONS:
	//   --datadir "/Users/ia/Library/EthereumClassic"	Data directory for the databases and keystore
	//   --keystore 					Directory for the keystore (default = inside the datadir)
	//   --networkid value				Network identifier (integer, 0=Olympic, 1=Homestead, 2=Morden) (default: 1)
	//   --testnet					Morden network: pre-configured test network with modified starting nonces (replay protection)
	//   --dev						Developer mode: pre-configured private network with several debugging flags
	//   --identity value				Custom node name
	//   --fast					Enable fast syncing through state downloads
	//   --lightkdf					Reduce key-derivation RAM & CPU usage at some expense of KDF strength
	//   --cache value					Megabytes of memory allocated to internal caching (min 16MB / database forced) (default: 128)
	//   --blockchainversion value			Blockchain version (integer) (default: 3)

	// ACCOUNT OPTIONS:
	//   --unlock value	Comma separated list of accounts to unlock
	//   --password value	Password file to use for non-inteactive password input

	// API AND CONSOLE OPTIONS:
	//   --rpc			Enable the HTTP-RPC server
	//   --rpcaddr value	HTTP-RPC server listening interface (default: "localhost")
	//   --rpcport value	HTTP-RPC server listening port (default: 8545)
	//   --rpcapi value	API's offered over the HTTP-RPC interface (default: "eth,net,web3")
	//   --ws			Enable the WS-RPC server
	//   --wsaddr value	WS-RPC server listening interface (default: "localhost")
	//   --wsport value	WS-RPC server listening port (default: 8546)
	//   --wsapi value		API's offered over the WS-RPC interface (default: "eth,net,web3")
	//   --wsorigins value	Origins from which to accept websockets requests
	//   --ipcdisable		Disable the IPC-RPC server
	//   --ipcapi value	API's offered over the IPC-RPC interface (default: "admin,debug,eth,miner,net,personal,shh,txpool,web3")
	//   --ipcpath "geth.ipc"	Filename for IPC socket/pipe within the datadir (explicit paths escape it)
	//   --rpccorsdomain value	Comma separated list of domains from which to accept cross origin requests (browser enforced)
	//   --jspath loadScript	JavaScript root path for loadScript and document root for `admin.httpGet` (default: ".")
	//   --exec value		Execute JavaScript statement (only in combination with console/attach)
	//   --preload value	Comma separated list of JavaScript files to preload into the console

	// NETWORKING OPTIONS:
	//   --bootnodes value	Comma separated enode URLs for P2P discovery bootstrap
	//   --port value		Network listening port (default: 30303)
	//   --maxpeers value	Maximum number of network peers (network disabled if set to 0) (default: 25)
	//   --maxpendpeers value	Maximum number of pending connection attempts (defaults used if set to 0) (default: 0)
	//   --nat value		NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>) (default: "any")
	//   --nodiscover		Disables the peer discovery mechanism (manual peer addition)
	//   --nodekey value	P2P node key file
	//   --nodekeyhex value	P2P node key as hex (for testing)

	// MINER OPTIONS:
	//   --mine			Enable mining
	//   --minerthreads value		Number of CPU threads to use for mining (default: 2)
	//   --minergpus value		List of GPUs to use for mining (e.g. '0,1' will use the first two GPUs found)
	//   --autodag			Enable automatic DAG pregeneration
	//   --etherbase value		Public address for block mining rewards (default = first account created) (default: "0")
	//   --targetgaslimit value	Target gas limit sets the artificial target gas floor for the blocks to mine (default: "4712388")
	//   --gasprice value		Minimal gas price to accept for mining a transactions (default: "20000000000")
	//   --extradata value		Freeform header field set by the miner

	// GAS PRICE ORACLE OPTIONS:
	//   --gpomin value	Minimum suggested gas price (default: "20000000000")
	//   --gpomax value	Maximum suggested gas price (default: "500000000000")
	//   --gpofull value	Full block threshold for gas price calculation (%) (default: 80)
	//   --gpobasedown value	Suggested gas price base step down ratio (1/1000) (default: 10)
	//   --gpobaseup value	Suggested gas price base step up ratio (1/1000) (default: 100)
	//   --gpobasecf value	Suggested gas price base correction factor (%) (default: 110)

	// LOGGING AND DEBUGGING OPTIONS:
	//   --verbosity value	Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=core, 5=debug, 6=detail (default: 3)
	//   --vmodule value	Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=6,p2p=5)
	//   --backtrace value	Request a stack trace at a specific logging statement (e.g. "block.go:271") (default: :0)
	//   --metrics value	Enables metrics reporting. When the value is a path, either relative or absolute, then a log is written to the respective file.
	//   --fakepow		Disables proof-of-work verification

	// EXPERIMENTAL OPTIONS:
	//   --shh		Enable Whisper
	//   --natspec	Enable NatSpec confirmation notice

	// MISCELLANEOUS OPTIONS:
	//   --solc value		Solidity compiler command to be used (default: "solc")
	//   --chainconfig value	Specify a JSON format chain configuration file to use.
	//   --chain value		Name of blockchain network to use (default='mainnnet', test='testnet'). These correlate to subdirectories under your base <EthereumClassic> data directory (--datadir). Chain name to use can also be configured from an external chain configuration file. (default: "mainnet")
	//   --oppose-dao-fork	Use classic blockchain (always set, flag is unused and exists for compatibility only)
	//   --help, -h		show help
}
