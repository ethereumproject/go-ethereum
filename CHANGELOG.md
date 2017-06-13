# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

__Legend__:
```md
<Added> for new features.
<Changed> for changes in existing functionality.
<Refactored> for improvements to codebase quality not impacting client interface.
<Deprecated> for once-stable features removed in upcoming releases.
<Removed> for deprecated features removed in this release.
<Fixed> for any bug fixes.
<Security> to invite users to upgrade in case of vulnerabilities.
```

Releases considered stable may be found on our [Releases Page](https://github.com/ethereumproject/go-ethereum/releases).

## [Unreleased]

#### Added
- Newly configurable in external `chain.json`:
    - `"state": { "startingNonce": NUMBER }` - _optional_ (mainnet: 0, morden: 1048576) - "dirty" starting world state
    - `"network": NUMBER` - _required_ (mainnet: 1, morden: 2) - network id used to identify valid peers
    - `"consensus": STRING` - _optional_ (default: "ethash", optional: "ethash-test") - specify smaller and faster pow algorithm, e.g. `--dev` mode sets "ethash-test"
    > See cmd/geth/config/*.json for updated examples.

- Dev mode (`--dev`) made compatible with `--chain`

#### Fixed
- `geth attach` command uses chain subdirectory schema by default, e.g. `datadir/mainnet/geth.ipc` instead of `datadir/geth.ipc`
- Sometimes ungraceful stopping on SIGTERM causing corrupted chaindata
- PublicKey method for protected transactions with malformed chain id causing SIGSEGV

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


