[![MacOS Build Status](https://circleci.com/gh/ethereumproject/go-ethereum/tree/master.svg?style=shield)](https://circleci.com/gh/ethereumproject/go-ethereum/tree/master)
[![Windows Build Status](https://ci.appveyor.com/api/projects/status/github/ethereumproject/go-ethereum?svg=true)](https://ci.appveyor.com/project/splix/go-ethereum)
[![Go Report Card](https://goreportcard.com/badge/github.com/ethereumproject/go-ethereum)](https://goreportcard.com/report/github.com/ethereumproject/go-ethereum)
[![API Reference](https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://godoc.org/github.com/ethereumproject/go-ethereum)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/ethereumproject/go-ethereum?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

## Ethereum Go (Ethereum Classic Blockchain)

Official Go language implementation of the Ethereum protocol supporting the
_original_ chain. Ethereum Classic (ETC) offers a censorship-resistant and powerful application platform for developers in parallel to Ethereum (ETHF), while differentially rejecting the DAO bailout.

## Install

### :rocket: From a release binary
The simplest way to get started running a node is to visit our [Releases page](https://github.com/ethereumproject/go-ethereum/releases) and download a zipped executable binary (matching your operating system, of course), then moving the unzipped file `geth` to somewhere in your `$PATH`. Now you should be able to open a terminal and run `$ geth help` to make sure it's working. For additional installation instructions please check out the [Installation Wiki](https://github.com/ethereumproject/go-ethereum/wiki/Home#Developers).

#### :beers: Using Homebrew (OSX only)
```
$ brew install ethereumproject/classic/geth
```

### :hammer: Building the source

If your heart is set on the bleeding edge, install from source. However, please be advised that you may encounter some strange things, and we can't prioritize support beyond the release versions. Recommended for developers only.

#### Dependencies
Building geth requires both Go >=1.9 and a C compiler. On Linux systems,
a C compiler can, for example, by installed with `sudo apt-get install
build-essential`. On Mac: `xcode-select --install`.

#### Get source and package dependencies
```
$ go get -v github.com/ethereumproject/go-ethereum/...`
```

#### Install and build command executables

Executables installed from source will, by default, be installed in `$GOPATH/bin/`.

##### With go:

- the full suite of utilities:
```
$ go install github.com/ethereumproject/go-ethereum/cmd/...`
```

- just __geth__:
```
$ go install github.com/ethereumproject/go-ethereum/cmd/geth`
```

##### With make:
```
$ cd $GOPATH/src/github.com/ethereumproject/go-ethereum
```

- the full suite of utilities:
```
$ make install
```

- just __geth__:
```
$ make install_geth
```

> For further `make` information, use `make help` to see a list and description of available make
> commands.


##### Building a specific release
All the above commands results with building binaries from `HEAD`. To use a specific release/tag, use the following before installing:

```shell
$ go get -d github.com/ethereumproject/go-ethereum/...
$ cd $GOPATH/src/github.com/ethereumproject/go-ethereum
$ git checkout <TAG OR REVISION>
# Use a go or make command above.
```

##### Using a release source code tarball
Because of strict Go directory structure, the tarball needs to be extracted into the proper subdirectory under `$GOPATH`.
The following commands are an example of building the v4.1.1 release:

```shell
$ mkdir -p $GOPATH/src/github.com/ethereumproject
$ cd $GOPATH/src/github.com/ethereumproject
$ tar xzf /path/to/go-ethereum-4.1.1.tar.gz
$ mv go-ethereum-4.1.1 go-ethereum
$ cd go-ethereum
# Use a go or make command above.
```

## Executables

This repository includes several wrappers/executables found in the `cmd` directory.

| Command    | Description |
|:----------:|-------------|
| **`geth`** | The main Ethereum CLI client. It is the entry point into the Ethereum network (main-, test-, or private net), capable of running as a full node (default) archive node (retaining all historical state) or a light node (retrieving data live). It can be used by other processes as a gateway into the Ethereum network via JSON RPC endpoints exposed on top of HTTP, WebSocket and/or IPC transports. Please see our [Command Line Options](https://github.com/ethereumproject/go-ethereum/wiki/Command-Line-Options) wiki page for details. |
| `abigen` | Source code generator to convert Ethereum contract definitions into easy to use, compile-time type-safe Go packages. It operates on plain [Ethereum contract ABIs](https://github.com/ethereumproject/wiki/wiki/Ethereum-Contract-ABI) with expanded functionality if the contract bytecode is also available. However it also accepts Solidity source files, making development much more streamlined. Please see our [Native DApps](https://github.com/ethereumproject/go-ethereum/wiki/Native-DApps-in-Go) wiki page for details. |
| `bootnode` | Stripped down version of our Ethereum client implementation that only takes part in the network node discovery protocol, but does not run any of the higher level application protocols. It can be used as a lightweight bootstrap node to aid in finding peers in private networks. |
| `disasm` | Bytecode disassembler to convert EVM (Ethereum Virtual Machine) bytecode into more user friendly assembly-like opcodes (e.g. `echo "6001" | disasm`). For details on the individual opcodes, please see pages 22-30 of the [Ethereum Yellow Paper](http://gavwood.com/paper.pdf). |
| `evm` | Developer utility version of the EVM (Ethereum Virtual Machine) that is capable of running bytecode snippets within a configurable environment and execution mode. Its purpose is to allow insolated, fine graned debugging of EVM opcodes (e.g. `evm --code 60ff60ff --debug`). |
| `gethrpctest` | Developer utility tool to support our [ethereum/rpc-test](https://github.com/ethereumproject/rpc-tests) test suite which validates baseline conformity to the [Ethereum JSON RPC](https://github.com/ethereumproject/wiki/wiki/JSON-RPC) specs. Please see the [test suite's readme](https://github.com/ethereumproject/rpc-tests/blob/master/README.md) for details. |
| `rlpdump` | Developer utility tool to convert binary RLP ([Recursive Length Prefix](https://github.com/ethereumproject/wiki/wiki/RLP)) dumps (data encoding used by the Ethereum protocol both network as well as consensus wise) to user friendlier hierarchical representation (e.g. `rlpdump --hex CE0183FFFFFFC4C304050583616263`). |

## :green_book: Geth: the basics

### Data directory
By default, geth will store all node and blockchain data in a __parent directory__ depending on your OS:

- Linux: `$HOME/.ethereum-classic/`
- Mac: `$HOME/Library/EthereumClassic/`
- Windows: `$HOME/AppData/Roaming/EthereumClassic/`

__You can specify this directory__ with `--data-dir=$HOME/id/rather/put/it/here`.

Within this parent directory, geth will use a __/subdirectory__ to hold data for each network you run. The defaults are:

 - `/mainnet` for the Mainnet
 - `/morden` for the Morden Testnet

__You can specify this subdirectory__ with `--chain=mycustomnet`.

> __Migrating__: If you have existing data created prior to the [3.4 Release](https://github.com/ethereumproject/go-ethereum/releases), geth will attempt to migrate your existing standard ETC data to this structure. To learn more about managing this migration please read our [3.4 release notes on our Releases page](https://github.com/ethereumproject/go-ethereum/wiki/Release-3.4.0-Notes).

### Full node on the main Ethereum network

```
$ geth
```

It's that easy! This will establish an ETC blockchain node and download ("sync") the full blocks for the entirety of the ETC blockchain. __However__, before you go ahead with plain ol' `geth`, we would encourage reviewing the following section...

#### :speedboat: `--fast`

The most common scenario is users wanting to simply interact with the Ethereum Classic network: create accounts; transfer funds; deploy and interact with contracts, and mine. For this particular use-case the user doesn't care about years-old historical data, so we can _fast-sync_ to the current state of the network. To do so:

```
$ geth --fast
```

Using geth in fast sync mode causes it to download only block _state_ data -- leaving out bulky transaction records -- which avoids a lot of CPU and memory intensive processing.

Fast sync will be automatically __disabled__ (and full sync enabled) when:

- your chain database contains *any* full blocks
- your node has synced up to the current head of the network blockchain

In case of using `--mine` together with `--fast`, geth will operate as described; syncing in fast mode up to the head, and then begin mining once it has synced its first full block at the head of the chain.

*Note:* To further increase geth's performace, you can use a `--cache=2054` flag to bump the memory allowance of the database (e.g. 2054MB) which can significantly improve sync times, especially for HDD users. This flag is optional and you can set it as high or as low as you'd like, though we'd recommend the 1GB - 2GB range.

### Create or manage account(s)

Geth is able to create, import, update, unlock, and otherwise manage your private (encrypted) key files. Key files are in JSON format and, by default, stored in the respective chain folder's `/keystore` directory; you can specify a custom location with the `--keystore` flag.

```
$ geth account new
```

This command will create a new account and prompt you to enter a passphrase to protect your account.

Other `account` subcommands include:
```
SUBCOMMANDS:

        list    print account addresses
        new     create a new account
        update  update an existing account
        import  import a private key into a new account

```

Learn more at the [Accounts Wiki Page](https://github.com/ethereumproject/go-ethereum/wiki/Managing-Accounts). If you're interested in using geth to manage a lot (~100,000+) of accounts, please visit the [Indexing Accounts Wiki page](https://github.com/ethereumproject/go-ethereum/wiki/Indexing-Accounts).


### Interact with the Javascript console
```
$ geth console
```

This command will start up Geth's built-in interactive [JavaScript console](https://github.com/ethereumproject/go-ethereum/wiki/JavaScript-Console), through which you can invoke all official [`web3` methods](https://github.com/ethereumproject/wiki/wiki/JavaScript-API) as well as Geth's own [management APIs](https://github.com/ethereumproject/go-ethereum/wiki/Management-APIs). This too is optional and if you leave it out you can always attach to an already running Geth instance with `geth attach`.

Learn more at the [Javascript Console Wiki page](https://github.com/ethereumproject/go-ethereum/wiki/JavaScript-Console).


### And so much more!

For a comprehensive list of command line options, please consult our [CLI Wiki page](https://github.com/ethereumproject/go-ethereum/wiki/Command-Line-Options).

## :orange_book: Geth: developing and advanced useage

### Morden Testnet
If you'd like to play around with creating Ethereum contracts, you
almost certainly would like to do that without any real money involved until you get the hang of the entire system. In other words, instead of attaching to the main network, you want to join the **test** network with your node, which is fully equivalent to the main network, but with play-Ether only.

```
$ geth --chain=morden --fast console
```

The `--fast` flag and `console` subcommand have the exact same meaning as above and they are equally useful on the testnet too. Please see above for their explanations if you've skipped to here.

Specifying the `--chain=morden` flag will reconfigure your Geth instance a bit:

 -  As mentioned above, Geth will host its testnet data in a `morden` subfolder (`~/.ethereum-classic/morden`).
 - Instead of connecting the main Ethereum network, the client will connect to the test network, which uses different P2P bootnodes, different network IDs and genesis states.

You may also optionally use `--testnet` or `--chain=testnet` to enable this configuration.

> *Note: Although there are some internal protective measures to prevent transactions from crossing over between the main network and test network (different starting nonces), you should make sure to always use separate accounts for play-money and real-money. Unless you manually move accounts, Geth
will by default correctly separate the two networks and will not make any accounts available between them.*

### Programatically interfacing Geth nodes

As a developer, sooner rather than later you'll want to start interacting with Geth and the Ethereum network via your own programs and not manually through the console. To aid this, Geth has built in support for a JSON-RPC based APIs ([standard APIs](https://github.com/ethereumproject/wiki/wiki/JSON-RPC) and
[Geth specific APIs](https://github.com/ethereumproject/go-ethereum/wiki/Management-APIs)). These can be exposed via HTTP, WebSockets and IPC (unix sockets on unix based platroms, and named pipes on Windows).

The IPC interface is enabled by default and exposes all the APIs supported by Geth, whereas the HTTP and WS interfaces need to manually be enabled and only expose a subset of APIs due to security reasons. These can be turned on/off and configured as you'd expect.

HTTP based JSON-RPC API options:

  * `--rpc` Enable the HTTP-RPC server
  * `--rpc-addr` HTTP-RPC server listening interface (default: "localhost")
  * `--rpc-port` HTTP-RPC server listening port (default: 8545)
  * `--rpc-api` API's offered over the HTTP-RPC interface (default: "eth,net,web3")
  * `--rpc-cors-domain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
  * `--ws` Enable the WS-RPC server
  * `--ws-addr` WS-RPC server listening interface (default: "localhost")
  * `--ws-port` WS-RPC server listening port (default: 8546)
  * `--ws-api` API's offered over the WS-RPC interface (default: "eth,net,web3")
  * `--ws-origins` Origins from which to accept websockets requests
  * `--ipc-disable` Disable the IPC-RPC server
  * `--ipc-api` API's offered over the IPC-RPC interface (default: "admin,debug,eth,miner,net,personal,shh,txpool,web3")
  * `--ipc-path` Filename for IPC socket/pipe within the datadir (explicit paths escape it)

You'll need to use your own programming environments' capabilities (libraries, tools, etc) to connect via HTTP, WS or IPC to a Geth node configured with the above flags and you'll need to speak [JSON-RPC](http://www.jsonrpc.org/specification) on all transports. You can reuse the same connection for multiple requests!

> Note: Please understand the security implications of opening up an HTTP/WS based transport before doing so! Hackers on the internet are actively trying to subvert Ethereum nodes with exposed APIs! Further, all browser tabs can access locally running webservers, so malicious webpages could try to subvert locally available APIs!*

### Operating a private/custom network

As of [Geth 3.4](https://github.com/ethereumproject/go-ethereum/releases) you are now able to configure a private chain by specifying an __external chain configuration__ JSON file, which includes necessary genesis block data as well as feature configurations for protocol forks, bootnodes, and chainID.

Please find full [example  external configuration files representing the Mainnet and Morden Testnet specs in the /config subdirectory of this repo](). You can use either of these files as a starting point for your own customizations.

It is important for a private network that all nodes use compatible chains. In the case of custom chain configuration, the chain configuration file (`chain.json`) should be equivalent for each node.

#### Define external chain configuration

Specifying an external chain configuration file will allow fine-grained control over a custom blockchain/network configuration, including the genesis state and extending through bootnodes and fork-based protocol upgrades.

```shell
$ geth --chain=morden dump-chain-config <datadir>/customnet/chain.json
$ sed s/mainnet/customnet/ <datadir>/customnet/chain.json
$ vi <datadir>/customnet/chain.json # make your custom edits
$ geth --chain=customnet [--flags] [command]
```

The external chain configuration file specifies valid settings for the following top-level fields:

| JSON Key | Notes |
| --- | --- |
| `chainID` |  Chain identity. Determines local __/subdir__ for chain data, with required `chain.json` located in it. It is required, but must not be identical for each node. Please note that this is _not_ the chainID validation introduced in _EIP-155_; that is configured as a protocal upgrade within `forks.features`. |
| `name` | _Optional_. Human readable name, ie _Ethereum Classic Mainnet_, _Morden Testnet._ |
| `state.startingNonce` | _Optional_. Initialize state db with a custom nonce. |
| `network` | Determines Network ID to identify valid peers. |
| `consensus` | _Optional_. Proof of work algorithm to use, either "ethash" or "ethast-test" (for development) |
| `genesis` | Determines __genesis state__. If running the node for the first time, it will write the genesis block. If configuring an existing chain database with a different genesis block, it will overwrite it. |
| `chainConfig` | Determines configuration for fork-based __protocol upgrades__, ie _EIP-150_, _EIP-155_, _EIP-160_, _ECIP-1010_, etc ;-). Subkeys are `forks` and `badHashes`. |
| `bootstrap` | _Optional_. Determines __bootstrap nodes__ in [enode format](https://github.com/ethereumproject/wiki/wiki/enode-url-format). |
| `include` | _Optional_. Other configuration files to include. Paths can be relative (to the config file with `include` field, or absolute). Each of configuration files has the same structure as "main" configuration. Included files are processed after the "main" configuration in the same order as specified in the array; values processed later overwrite the previously defined ones. |


*Fields `name`, `state.startingNonce`, and `consensus` are optional. Geth will panic if any required field is missing, invalid, or in conflict with another flag. This renders `--chain` __incompatible__ with `--testnet`. It remains __compatible__ with `--data-dir`.*

To learn more about external chain configuration, please visit the [External Command Line Options Wiki page](https://github.com/ethereumproject/go-ethereum/wiki/Command-Line-Options).

##### Create the rendezvous point

Once all participating nodes have been initialized to the desired genesis state, you'll need to start a __bootstrap node__ that others can use to find each other in your network and/or over the internet. The clean way is to configure and run a dedicated bootnode:

```
$ bootnode --genkey=boot.key
$ bootnode --nodekey=boot.key
```

With the bootnode online, it will display an `enode` URL that other nodes can use to connect to it and exchange peer information. Make sure to replace the
displayed IP address information (most probably `[::]`) with your externally accessible IP to get the actual `enode` URL.

*Note: You could also use a full fledged Geth node as a bootnode, but it's the less recommended way.*

To learn more about enodes and enode format, visit the [Enode Wiki page](https://github.com/ethereumproject/wiki/wiki/enode-url-format).

##### Starting up your member nodes

With the bootnode operational and externally reachable (you can try `telnet <ip> <port>` to ensure it's indeed reachable), start every subsequent Geth node pointed to the bootnode for peer discovery via the `--bootnodes` flag. It will probably be desirable to keep private network data separate from defaults; to do so, specify a custom `--datadir` and/or `--chain` flag.

```
$ geth --datadir=path/to/custom/data/folder \
       --chain=kittynet \
       --bootnodes=<bootnode-enode-url-from-above>
```

*Note: Since your network will be completely cut off from the main and test networks, you'll also need to configure a miner to process transactions and create new blocks for you.*

#### Running a private miner

Mining on the public Ethereum network is a complex task as it's only feasible using GPUs, requiring an OpenCL or CUDA enabled `ethminer` instance. For information on such a setup, please consult the [EtherMining subreddit](https://www.reddit.com/r/EtherMining/) and the [Genoil miner](https://github.com/Genoil/cpp-ethereum) repository.

In a private network setting however, a single CPU miner instance is more than enough for practical purposes as it can produce a stable stream of blocks at the correct intervals without needing heavy resources (consider running on a single thread, no need for multiple ones either). To start a Geth instance for mining, run it with all your usual flags, extended by:

```
$ geth <usual-flags> --mine --minerthreads=1 --etherbase=0x0000000000000000000000000000000000000000
```

Which will start mining blocks and transactions on a single CPU thread, crediting all proceedings to the account specified by `--etherbase`. You can further tune the mining by changing the default gas limit blocks converge to (`--targetgaslimit`) and the price transactions are accepted at (`--gasprice`).

For more information about managing accounts, please see the [Managing Accounts Wiki page](https://github.com/ethereumproject/go-ethereum/wiki/Managing-Accounts).


## Contribution

Thank you for considering to help out with the source code!

The core values of democratic engagement, transparency, and integrity run deep with us. We welcome contributions from everyone, and are grateful for even the smallest of fixes.  :clap:

This project is migrated from the now hard-forked [Ethereum (ETHF) Github project](https://github.com/ethereum), and we will need to incrementally migrate pieces of the infrastructure required to maintain the project.

If you'd like to contribute to go-ethereum, please fork, fix, commit and send a pull request for the maintainers to review and merge into the main code base. If you wish to submit more complex changes, please check up with the core devs first on [our Slack channel (#development)](http://ethereumclassic.herokuapp.com/) or [our Discord channel (#development)](https://discord.gg/wpwSGWn) to ensure those changes are in line with the general philosophy of the project and/or get some early feedback which can make both your efforts much lighter as well as our review and merge procedures quick and simple.

Please see the [Wiki](https://github.com/ethereumproject/go-ethereum/wiki) for more details on configuring your environment, managing project dependencies, and testing procedures.

## License

The go-ethereum library (i.e. all code outside of the `cmd` directory) is licensed under the [GNU Lesser General Public License v3.0](http://www.gnu.org/licenses/lgpl-3.0.en.html), also included in our repository in the `COPYING.LESSER` file.

The go-ethereum binaries (i.e. all code inside of the `cmd` directory) is licensed under the [GNU General Public License v3.0](http://www.gnu.org/licenses/gpl-3.0.en.html), also included in our repository in the `COPYING` file.
