# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

__Legend__:
```text
<Added> for new features.
<Changed> for changes in existing functionality.
<Refactored> for improvements to codebase quality not impacting client interface.
<Deprecated> for once-stable features removed in upcoming releases.
<Removed> for deprecated features removed in this release.
<Fixed> for any bug fixes.
<Security> to invite users to upgrade in case of vulnerabilities.
<Consensus> to invite users to upgrade in case of consensus protocol changes.
```

Releases considered __stable__ may be found on our [Releases Page](https://github.com/ethereumproject/go-ethereum/releases).

Rolling builds for the master branch may be found at [builds.etcdevteam.com](builds.etcdevteam.com).

## [4.0.0] - 2017-09-05

#### Consensus
- [ECIP-1017](https://github.com/ethereumproject/ECIPs/blob/master/ECIPs/ECIP-1017.md) - implement monetary policy on Morden Testnet (2 million block era) and Mainnet (5 million block era)

#### Added
- JSON-RPC: `debug_traceTransaction` method
- JSON-RPC: `eth_chainId` method; returns configured Ethereum EIP-155 chain id for signing protected txs. For congruent behavior in Ethereum Foundation and Parity clients, please see https://github.com/ethereum/EIPs/pull/695 and https://github.com/paritytech/parity/pull/6329.
- P2P: improve peer discovery by allowing "good-will" for peers with unknown HF blocks
- _Option_: `--log-status` - enable interval-based status logging, e.g. `--log-status="sync=10"`, where `sync` is the context (currently the only one implemented) and `10` is interval in seconds.

#### Fixed
- geth/cmd: Improve chain configuration file handling to allow specifying a file instead
  of chain identity and allow flag overrides for bootnodes and network-id.
- _Command_: `monitor` - enables sexy terminal-based graphs for metrics around
  a specified set of modules, e.g.

  ```
  $ geth
  ```

  ```
  $ geth monitor "p2p/.*/(count|average)" "msg/txn/out/.*/count"
  ```

- P2P: Improve wording for logging as-yet-unknown nodes.


## [3.5.86] - 2017-07-19 - db60074

#### Added
- Newly configurable in external `chain.json`:
    - `"state": { "startingNonce": NUMBER }` - _optional_ (mainnet: 0, morden: 1048576) - "dirty" starting world state
    - `"network": NUMBER` - _required_ (mainnet: 1, morden: 2) - network id used to identify valid peers
    - `"consensus": STRING` - _optional_ (default: "ethash", optional: "ethash-test") - specify smaller and faster pow algorithm, e.g. `--dev` mode sets "ethash-test"
    > See cmd/geth/config/*.json for updated examples.

- Dev mode (`--dev`) made compatible with `--chain`
- `debug_AccountExist` method added to RPC and web3 extension methods (thanks @sorpaas)
- Additional Morden testnet bootnodes
- Add listen for `SIGTERM` to stop more gracefully, if possible

#### Changed
- Nightly and tagged release distribution builds now available at [builds.etcdevteam.com](builds.etcdevteam.com) (instead of Bintray)
- _Option_: `--chain <chainIdentifier|mychain.json>` - specify chain identifier OR path to JSON configuration file

#### Fixed
- `geth attach` command uses chain subdirectory schema by default, e.g. `datadir/mainnet/geth.ipc` instead of `datadir/geth.ipc`
- Sometimes ungraceful stopping on SIGTERM, potentially causing corrupted chaindata
- PublicKey method for protected transactions with malformed chain id causing SIGSEGV
- Concurrent map read/writes for State Objects
- Ignore reported neighbors coming from non-reserved addresses; prevents irrelevant discovery attempts on local and reserved IP's
- RLP-decoded transactions include EIP155 signer if applicable (thanks @shawdon)

## [3.5.0] - 2017-06-02 - 402c170

Wiki: https://github.com/ethereumproject/go-ethereum/wiki/Release-3.5.0-Notes

#### Security
- Hash map exploit opportunity (thanks @karalabe)
#### Added
- _Option_: `--index-accounts` - use persistent keystore key file indexing (recommended for use with greater than ~10k-100k+ key files)
- _Command_: `--index-accounts account index` - build or rebuild persisent key file index
- _Option_: `--log-dir` - specify directory in which to redirect logs to files
- _Command_: `status` - retrieve contextual status for node, ethereum, and chain configuration
#### Changed
- _Command_: `dump <blockHash|blockNum>,<|blockHash|blockNum> <|address>,<|address>` - specify dump for _n_ block(s) for optionally _a_ address(es)
- _Option__: `--chain` replaces `--chain-config` and expects consistent custom external chain config JSON path
#### Fixed
- SIGSEGV crash on malformed ChainID signer for replay-protected blocks.
#### Removed
- _Option_: `--chain-config`, replaced by `--chain`

## [3.4.0] - 2017-05-15 - c18792d

Wiki: https://github.com/ethereumproject/go-ethereum/wiki/Release-3.4.0-Notes

#### Added
- _Command_: `rollback <int>` - sets blockchain head and purges blocks antecedent to specified block number
- _Option_: `--chain <name>` - specify default or custom chain configuration by subdirectory
- _Option_: `--chain-config <path>` - specify an external JSON file to define chain configuration
- _Command_: `dump-chain-config <path>` - specify an external JSON file location in which to dump current chain configuration
- Default data directory now at `~/Library/EthereumClassic` or OS-sensible equivalent.
- Chaindata saves to respective subdirectory under parent data dir, _ie_ `/mainnet`.
#### Changed
- Commands and flags using compound/concatenated words (_ie_ `--datadir`) aliased to hypenated equivalent (_ie_ `--data-dir`).
#### Deprecated
- Un-hyphenated aliased commands and flags (see above).
- Public API `GetBlockByNumber` function populates `logsBloom` field.
- Logging `sync busy` relegated to `Debug` level.
#### Removed
- _Command_: `init <path>` - replaced with `--chain-config` flag


__Contributors:__
- @whilei
- @splix
- @tranvictor


## [3.3.0] - 2017-03-08
Tagged commit: 1f9eaca
#### Added
- New bootnodes
- Improved peer discovery
#### Refactored
- ~9k LOC refactored and cleaned
#### Removed
- JIT VM

__Contributors:__
- @pascaldekloe
- @splix
- @outragedhuman

## [3.2.3] - 2016-12-26
Tagged commit: 2b51918
#### Added
- Difficulty Bomb delay (ECIP-1010)
- EXP reprice (EIP-160)
- Replay Protection (EIP-155)


