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
Reflects changes to __master__ branch, but not yet packaged in a stable release.

#### Added
- _Option_: `--index-accounts` - use persistent keystore key file indexing (recommended for use with greater than ~10k-100k+ key files)
- _Command_: `--index-accounts account index` - build or rebuild persisent key file index
- _Option_: `--log-dir` - specify directory in which to redirect logs to files
#### Changed
- _Command_: `dump <blockHash|blockNum>,<|blockHash|blockNum> <|address>,<|address>` - specify dump for _n_ block(s) for optionally _a_ address(es)
#### Fixed
- SIGSEGV crash on malformed ChainID signer for replay-protected blocks.

## [3.4.0] - 2017-05-15
Tagged commit: c18792d

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


