// Copyright 2016 The go-ethereum Authors
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

package core

import (
	"encoding/csv"
	hexlib "encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"sort"
	"sync"

	"path/filepath"
	"reflect"

	"io"
	"strings"

	"github.com/eth-classic/go-ethereum/common"
	"github.com/eth-classic/go-ethereum/core/state"
	"github.com/eth-classic/go-ethereum/core/types"
	"github.com/eth-classic/go-ethereum/core/vm"
	"github.com/eth-classic/go-ethereum/ethdb"
	"github.com/eth-classic/go-ethereum/logger"
	"github.com/eth-classic/go-ethereum/logger/glog"
	"github.com/eth-classic/go-ethereum/p2p/discover"
)

var (
	ErrChainConfigNotFound     = errors.New("chain config not found")
	ErrChainConfigForkNotFound = errors.New("chain config fork not found")

	ErrInvalidChainID = errors.New("invalid chainID")

	ErrHashKnownBad  = errors.New("known bad hash")
	ErrHashKnownFork = validateError("known fork hash mismatch")

	// Chain identities.
	ChainIdentitiesBlacklist = map[string]bool{
		"chaindata": true,
		"dapp":      true,
		"keystore":  true,
		"nodekey":   true,
		"nodes":     true,
	}
	ChainIdentitiesMain = map[string]bool{
		"main":    true,
		"mainnet": true,
	}
	ChainIdentitiesMorden = map[string]bool{
		"morden":  true,
		"testnet": true,
	}

	cacheChainIdentity string
	cacheChainConfig   *SufficientChainConfig
)

func SetCacheChainIdentity(s string) {
	cacheChainIdentity = s
}

func GetCacheChainIdentity() string {
	return cacheChainIdentity
}

func SetCacheChainConfig(c *SufficientChainConfig) *SufficientChainConfig {
	cacheChainConfig = c
	return cacheChainConfig
}

func GetCacheChainConfig() *SufficientChainConfig {
	return cacheChainConfig
}

// SufficientChainConfig holds necessary data for externalizing a given blockchain configuration.
type SufficientChainConfig struct {
	ID              string           `json:"id,omitempty"` // deprecated in favor of 'Identity', method decoding should id -> identity
	Identity        string           `json:"identity"`
	Name            string           `json:"name,omitempty"`
	State           *StateConfig     `json:"state"`     // don't omitempty for clarity of potential custom options
	Network         int              `json:"network"`   // eth.NetworkId (mainnet=1, morden=2)
	Consensus       string           `json:"consensus"` // pow type (ethash OR ethash-test)
	Genesis         *GenesisDump     `json:"genesis"`
	ChainConfig     *ChainConfig     `json:"chainConfig"`
	Bootstrap       []string         `json:"bootstrap"`
	ParsedBootstrap []*discover.Node `json:"-"`
	Include         []string         `json:"include"` // config files to include
}

// StateConfig hold variable data for statedb.
type StateConfig struct {
	StartingNonce uint64 `json:"startingNonce,omitempty"`
}

// GenesisDump is the geth JSON format.
// https://github.com/eth-classic/wiki/wiki/Ethereum-Chain-Spec-Format#subformat-genesis
type GenesisDump struct {
	Nonce      prefixedHex `json:"nonce"`
	Timestamp  prefixedHex `json:"timestamp"`
	ParentHash prefixedHex `json:"parentHash"`
	ExtraData  prefixedHex `json:"extraData"`
	GasLimit   prefixedHex `json:"gasLimit"`
	Difficulty prefixedHex `json:"difficulty"`
	Mixhash    prefixedHex `json:"mixhash"`
	Coinbase   prefixedHex `json:"coinbase"`

	// Alloc maps accounts by their address.
	Alloc map[hex]*GenesisDumpAlloc `json:"alloc"`
	// Alloc file contains CSV representation of Alloc
	AllocFile string `json:"alloc_file"`
}

// GenesisDumpAlloc is a GenesisDump.Alloc entry.
type GenesisDumpAlloc struct {
	Code    prefixedHex `json:"-"` // skip field for json encode
	Storage map[hex]hex `json:"-"`
	Balance string      `json:"balance"` // decimal string
}

type GenesisAccount struct {
	Address common.Address `json:"address"`
	Balance *big.Int       `json:"balance"`
}

// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	// Forks holds fork block requirements. See ErrHashKnownFork.
	Forks Forks `json:"forks"`

	// BadHashes holds well known blocks with consensus issues. See ErrHashKnownBad.
	BadHashes []*BadHash `json:"badHashes"`
}

type Fork struct {
	Name string `json:"name"`
	// Block is the block number where the hard-fork commences on
	// the Ethereum network.
	Block *big.Int `json:"block"`
	// Used to improve sync for a known network split
	RequiredHash common.Hash `json:"requiredHash"`
	// Configurable features.
	Features []*ForkFeature `json:"features"`
}

// Forks implements sort interface, sorting by block number
type Forks []*Fork

func (fs Forks) Len() int { return len(fs) }
func (fs Forks) Less(i, j int) bool {
	iF := fs[i]
	jF := fs[j]
	return iF.Block.Cmp(jF.Block) < 0
}
func (fs Forks) Swap(i, j int) {
	fs[i], fs[j] = fs[j], fs[i]
}

// ForkFeatures are designed to decouple the implementation feature upgrades from Forks themselves.
// For example, there are several 'set-gasprice' features, each using a different gastable,
// as well as protocol upgrades including 'eip155', 'ecip1010', ... etc.
type ForkFeature struct {
	ID                string                    `json:"id"`
	Options           ChainFeatureConfigOptions `json:"options"` // no * because they have to be iterable(?)
	optionsLock       sync.RWMutex
	ParsedOptions     map[string]interface{} `json:"-"` // don't include in JSON dumps, since its for holding parsed JSON in mem
	parsedOptionsLock sync.RWMutex
	// TODO Derive Oracle contracts from fork struct (Version, Registrar, Release)
}

// These are the raw key-value configuration options made available
// by an external JSON file.
type ChainFeatureConfigOptions map[string]interface{}

type BadHash struct {
	Block *big.Int
	Hash  common.Hash
}

func (c *SufficientChainConfig) IsValid() (string, bool) {
	// entirely empty
	if reflect.DeepEqual(c, SufficientChainConfig{}) {
		return "all empty", false
	}

	if c.Identity == "" {
		return "identity/id", false
	}

	if c.Network == 0 {
		return "networkId", false
	}

	if c := c.Consensus; c == "" || (c != "ethash" && c != "ethash-test") {
		return "consensus", false
	}

	if c.Genesis == nil {
		return "genesis", false
	}
	if len(c.Genesis.Nonce) == 0 {
		return "genesis.nonce", false
	}
	if len(c.Genesis.GasLimit) == 0 {
		return "genesis.gasLimit", false
	}
	if len(c.Genesis.Difficulty) == 0 {
		return "genesis.difficulty", false
	}
	if _, e := c.Genesis.Header(); e != nil {
		return "genesis.header(): " + e.Error(), false
	}

	if c.ChainConfig == nil {
		return "chainConfig", false
	}

	if len(c.ChainConfig.Forks) == 0 {
		return "forks", false
	}

	return "", true
}

// Header returns the mapping.
func (g *GenesisDump) Header() (*types.Header, error) {
	var h types.Header

	var err error
	if err = g.Nonce.Decode(h.Nonce[:]); err != nil {
		return nil, fmt.Errorf("malformed nonce: %s", err)
	}
	if h.Time, err = g.Timestamp.Int(); err != nil {
		return nil, fmt.Errorf("malformed timestamp: %s", err)
	}
	if err = g.ParentHash.Decode(h.ParentHash[:]); err != nil {
		return nil, fmt.Errorf("malformed parentHash: %s", err)
	}
	if h.Extra, err = g.ExtraData.Bytes(); err != nil {
		return nil, fmt.Errorf("malformed extraData: %s", err)
	}
	if h.GasLimit, err = g.GasLimit.Int(); err != nil {
		return nil, fmt.Errorf("malformed gasLimit: %s", err)
	}
	if h.Difficulty, err = g.Difficulty.Int(); err != nil {
		return nil, fmt.Errorf("malformed difficulty: %s", err)
	}
	if err = g.Mixhash.Decode(h.MixDigest[:]); err != nil {
		return nil, fmt.Errorf("malformed mixhash: %s", err)
	}
	if err := g.Coinbase.Decode(h.Coinbase[:]); err != nil {
		return nil, fmt.Errorf("malformed coinbase: %s", err)
	}

	return &h, nil
}

// SortForks sorts a ChainConfiguration's forks by block number smallest to bigget (chronologically).
// This should need be called only once after construction
func (c *ChainConfig) SortForks() *ChainConfig {
	sort.Sort(c.Forks)
	return c
}

// GetChainID gets the chainID for a chainconfig.
// It returns big.Int zero-value if no chainID is ever set for eip155/chainID.
// It uses ChainConfig#HasFeature, so it will return the last chronological value
// if the value is set multiple times.
func (c *ChainConfig) GetChainID() *big.Int {
	n := new(big.Int)
	feat, _, ok := c.HasFeature("eip155")
	if !ok {
		return n
	}
	if val, ok := feat.GetBigInt("chainID"); ok {
		n.Set(val)
	}
	return n
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	if c.ForkByName("Homestead").Block == nil || num == nil {
		return false
	}
	return num.Cmp(c.ForkByName("Homestead").Block) >= 0
}

// IsDiehard returns whether num is greater than or equal to the Diehard block, but less than explosion.
func (c *ChainConfig) IsDiehard(num *big.Int) bool {
	fork := c.ForkByName("Diehard")
	if fork.Block == nil || num == nil {
		return false
	}
	return num.Cmp(fork.Block) >= 0
}

// IsExplosion returns whether num is either equal to the explosion block or greater.
func (c *ChainConfig) IsExplosion(num *big.Int) bool {
	feat, fork, configured := c.GetFeature(num, "difficulty")

	if configured {
		//name, exists := feat.GetString("type")
		if name, exists := feat.GetString("type"); exists && name == "ecip1010" {
			block := big.NewInt(0)
			if length, ok := feat.GetBigInt("length"); ok {
				block = block.Add(fork.Block, length)
			} else {
				panic("Fork feature ecip1010 requires length value.")
			}
			return num.Cmp(block) >= 0
		}
	}
	return false
}

// IsAtlantis returns true if num is greater than atlantic config block
func (c *ChainConfig) IsAtlantis(num *big.Int) bool {
	fork := c.ForkByName("Atlantis")
	if fork.Block == nil || num == nil {
		return false
	}
	return num.Cmp(fork.Block) >= 0
}

// ForkByName looks up a Fork by its name, assumed to be unique
func (c *ChainConfig) ForkByName(name string) *Fork {
	for i := range c.Forks {
		if c.Forks[i].Name == name {
			return c.Forks[i]
		}
	}
	return &Fork{}
}

// GetFeature looks up fork features by id, where id can (currently) be [difficulty, gastable, eip155].
// GetFeature returns the feature|nil, the latest fork configuring a given id, and if the given feature id was found at all
// If queried feature is not found, returns ForkFeature{}, Fork{}, false.
// If queried block number and/or feature is a zero-value, returns ForkFeature{}, Fork{}, false.
func (c *ChainConfig) GetFeature(num *big.Int, id string) (*ForkFeature, *Fork, bool) {
	var okForkFeature = &ForkFeature{}
	var okFork = &Fork{}
	var found = false
	if num != nil && id != "" {
		for _, f := range c.Forks {
			if f.Block == nil {
				continue
			}
			if f.Block.Cmp(num) > 0 {
				continue
			}
			for _, ff := range f.Features {
				if ff.ID == id {
					okForkFeature = ff
					okFork = f
					found = true
				}
			}
		}
	}
	return okForkFeature, okFork, found
}

// HasFeature looks up if fork feature exists on any fork at any block in the configuration.
// In case of multiple same-'id'd features, returns latest (assuming forks are sorted).
func (c *ChainConfig) HasFeature(id string) (*ForkFeature, *Fork, bool) {
	var okForkFeature = &ForkFeature{}
	var okFork = &Fork{}
	var found = false
	if id != "" {
		for _, f := range c.Forks {
			for _, ff := range f.Features {
				if ff.ID == id {
					okForkFeature = ff
					okFork = f
					found = true
				}
			}
		}
	}
	return okForkFeature, okFork, found
}

func (c *ChainConfig) HeaderCheck(h *types.Header) error {
	for _, fork := range c.Forks {
		if fork.Block.Cmp(h.Number) != 0 {
			continue
		}
		if !fork.RequiredHash.IsEmpty() && fork.RequiredHash != h.Hash() {
			return ErrHashKnownFork
		}
	}

	for _, bad := range c.BadHashes {
		if bad.Block.Cmp(h.Number) != 0 {
			continue
		}
		if bad.Hash == h.Hash() {
			return ErrHashKnownBad
		}
	}

	return nil
}

// GetLatestRequiredHash returns the latest requiredHash from chain config for a given blocknumber n (eg. bc head).
// It does NOT depend on forks being sorted.
func (c *ChainConfig) GetLatestRequiredHashFork(n *big.Int) (f *Fork) {
	lastBlockN := new(big.Int)
	for _, ff := range c.Forks {
		if ff.RequiredHash.IsEmpty() {
			continue
		}
		// If this fork is chronologically later than lastSet fork with required hash AND given block n is greater than
		// the fork.
		if ff.Block.Cmp(lastBlockN) > 0 && n.Cmp(ff.Block) >= 0 {
			f = ff
			lastBlockN = ff.Block
		}
	}
	return
}

func (c *ChainConfig) GetSigner(blockNumber *big.Int) types.Signer {
	feature, _, configured := c.GetFeature(blockNumber, "eip155")
	if configured {
		if chainId, ok := feature.GetBigInt("chainID"); ok {
			return types.NewChainIdSigner(chainId)
		} else {
			panic(fmt.Errorf("chainID is not set for EIP-155 at %v", blockNumber))
		}
	}
	return types.BasicSigner{}
}

// GasTable returns the gas table corresponding to the current fork
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) *vm.GasTable {
	f, _, configured := c.GetFeature(num, "gastable")
	if !configured {
		return DefaultHomeSteadGasTable
	}
	name, ok := f.GetString("type")
	if !ok {
		name = ""
	} // will wall to default panic
	switch name {
	case "homestead":
		return DefaultHomeSteadGasTable
	case "eip150":
		return DefaultGasRepriceGasTable
	case "eip160":
		return DefaultDiehardGasTable
	default:
		panic(fmt.Errorf("Unsupported gastable value '%v' at block: %v", name, num))
	}
}

// WriteToJSONFile writes a given config to a specified file path.
// It doesn't run any checks on the file path so make sure that's already squeaky clean.
func (c *SufficientChainConfig) WriteToJSONFile(path string) error {
	jsonConfig, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return fmt.Errorf("Could not marshal json from chain config: %v", err)
	}

	if err := ioutil.WriteFile(path, jsonConfig, 0644); err != nil {
		return fmt.Errorf("Could not write external chain config file: %v", err)
	}
	return nil
}

// resolvePath builds a path based on adjacentPath's directory.
// It assumes that adjacentPath is the path of a file or immediate parent directory, and that
// 'path' is either an absolute path or a path relative to the adjacentPath.
func resolvePath(path, parentOrAdjacentPath string) string {
	if !filepath.IsAbs(path) {
		baseDir := filepath.Dir(parentOrAdjacentPath)
		path = filepath.Join(baseDir, path)
	}
	return path
}

func parseAllocationFile(config *SufficientChainConfig, open func(string) (io.ReadCloser, error), currentFile string) error {
	if config.Genesis == nil || config.Genesis.AllocFile == "" {
		return nil
	}

	if len(config.Genesis.Alloc) > 0 {
		return fmt.Errorf("error processing %s: \"alloc\" values already set, but \"alloc_file\" is provided", currentFile)
	}
	path := resolvePath(config.Genesis.AllocFile, currentFile)
	csvFile, err := open(path)
	if err != nil {
		return fmt.Errorf("failed to read allocation file: %v", err)
	}
	defer csvFile.Close()

	config.Genesis.Alloc = make(map[hex]*GenesisDumpAlloc)

	reader := csv.NewReader(csvFile)
	line := 1
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("error while reading allocation file: %v", err)
		}
		if len(row) != 2 {
			return fmt.Errorf("invalid number of values in line %d: expected 2, got %d", line, len(row))
		}
		line++

		config.Genesis.Alloc[hex(row[0])] = &GenesisDumpAlloc{Balance: row[1]}
	}

	config.Genesis.AllocFile = ""
	return nil
}

func parseExternalChainConfig(mainConfigFile string, open func(string) (io.ReadCloser, error)) (*SufficientChainConfig, error) {
	var config = &SufficientChainConfig{}
	var processed []string

	contains := func(hayStack []string, needle string) bool {
		for _, v := range hayStack {
			if needle == v {
				return true
			}
		}
		return false
	}

	var processFile func(path, parent string) error
	processFile = func(path, parent string) (err error) {
		path = resolvePath(path, parent)
		if contains(processed, path) {
			return nil
		}
		processed = append(processed, path)

		f, err := open(path)
		// return file close error as named return if another error is not already being returned
		defer func() {
			if closeErr := f.Close(); err == nil {
				err = closeErr
			}
		}()
		if err != nil {
			return fmt.Errorf("failed to read chain configuration file: %s", err)
		}
		if err := json.NewDecoder(f).Decode(config); err != nil {
			return fmt.Errorf("%v: %s", f, err)
		}

		// read csv alloc file
		if err := parseAllocationFile(config, open, path); err != nil {
			return err
		}

		includes := make([]string, len(config.Include))
		copy(includes, config.Include)
		config.Include = nil

		for _, include := range includes {
			err := processFile(include, path)
			if err != nil {
				return err
			}
		}
		return
	}

	err := processFile(mainConfigFile, ".")
	if err != nil {
		return nil, err
	}

	// Make JSON 'id' -> 'identity' (for backwards compatibility)
	if config.ID != "" && config.Identity == "" {
		config.Identity = config.ID
	}

	// Make 'ethash' default (backwards compatibility)
	if config.Consensus == "" {
		config.Consensus = "ethash"
	}

	// Parse bootstrap nodes
	config.ParsedBootstrap = ParseBootstrapNodeStrings(config.Bootstrap)

	if invalid, ok := config.IsValid(); !ok {
		return nil, fmt.Errorf("Invalid chain configuration file. Please check the existence and integrity of keys and values for: %v", invalid)
	}

	config.ChainConfig = config.ChainConfig.SortForks()
	return config, nil
}

// ReadExternalChainConfigFromFile reads a flagged external json file for blockchain configuration.
// It returns a valid and full ("hard") configuration or an error.
func ReadExternalChainConfigFromFile(incomingPath string) (*SufficientChainConfig, error) {

	// ensure flag arg cleanliness
	flaggedExternalChainConfigPath := filepath.Clean(incomingPath)

	// ensure file exists and that it is NOT a directory
	if info, err := os.Stat(flaggedExternalChainConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("ERROR: No existing chain configuration file found at: %s", flaggedExternalChainConfigPath)
	} else if info.IsDir() {
		return nil, fmt.Errorf("ERROR: Specified configuration file cannot be a directory: %s", flaggedExternalChainConfigPath)
	}

	config, err := parseExternalChainConfig(flaggedExternalChainConfigPath, func(path string) (io.ReadCloser, error) { return os.Open(path) })
	if err != nil {
		return nil, err
	}
	return config, nil
}

// ParseBootstrapNodeStrings is a helper function to parse stringified bs nodes, ie []"enode://e809c4a2fec7daed400e5e28564e23693b23b2cc5a019b612505631bbe7b9ccf709c1796d2a3d29ef2b045f210caf51e3c4f5b6d3587d43ad5d6397526fa6179@174.112.32.157:30303",...
// to usable Nodes. It takes a slice of strings and returns a slice of Nodes.
func ParseBootstrapNodeStrings(nodeStrings []string) []*discover.Node {
	// Otherwise parse and use the CLI bootstrap nodes
	bootnodes := []*discover.Node{}

	for _, url := range nodeStrings {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		node, err := discover.ParseNode(url)
		if err != nil {
			glog.V(logger.Error).Infof("Bootstrap URL %s: %v\n", url, err)
			continue
		}
		bootnodes = append(bootnodes, node)
	}
	return bootnodes
}

// GetString gets and option value for an options with key 'name',
// returning value as a string.
func (o *ForkFeature) GetString(name string) (string, bool) {
	o.parsedOptionsLock.Lock()
	defer o.parsedOptionsLock.Unlock()

	if o.ParsedOptions == nil {
		o.ParsedOptions = make(map[string]interface{})
	} else {
		val, ok := o.ParsedOptions[name]
		if ok {
			return val.(string), ok
		}
	}
	o.optionsLock.RLock()
	defer o.optionsLock.RUnlock()

	val, ok := o.Options[name].(string)
	o.ParsedOptions[name] = val //expect it as a string in config

	return val, ok
}

// GetBigInt gets and option value for an options with key 'name',
// returning value as a *big.Int and ok if it exists.
func (o *ForkFeature) GetBigInt(name string) (*big.Int, bool) {
	i := new(big.Int)

	o.parsedOptionsLock.Lock()
	defer o.parsedOptionsLock.Unlock()

	if o.ParsedOptions == nil {
		o.ParsedOptions = make(map[string]interface{})
	} else {
		val, ok := o.ParsedOptions[name]
		if ok {
			if vv, ok := val.(*big.Int); ok {
				return i.Set(vv), true
			}
		}
	}

	o.optionsLock.RLock()
	originalValue, ok := o.Options[name]
	o.optionsLock.RUnlock()
	if !ok {
		return nil, false
	}

	// interface{} type assertion for _61_ is float64
	if value, ok := originalValue.(float64); ok {
		i.SetInt64(int64(value))
		o.ParsedOptions[name] = i
		return i, true
	}
	// handle other user-generated incoming options with some, albeit limited, degree of lenience
	if value, ok := originalValue.(int64); ok {
		i.SetInt64(value)
		o.ParsedOptions[name] = i
		return i, true
	}
	if value, ok := originalValue.(int); ok {
		i.SetInt64(int64(value))
		o.ParsedOptions[name] = i
		return i, true
	}
	if value, ok := originalValue.(string); ok {
		ii, ok := new(big.Int).SetString(value, 0)
		if ok {
			i.Set(ii)
			o.ParsedOptions[name] = i
		}
		return i, ok
	}
	return nil, false
}

// WriteGenesisBlock writes the genesis block to the database as block number 0
func WriteGenesisBlock(chainDb ethdb.Database, genesis *GenesisDump) (*types.Block, error) {
	statedb, err := state.New(common.Hash{}, state.NewDatabase(chainDb))
	if err != nil {
		return nil, err
	}

	for addrHex, account := range genesis.Alloc {
		var addr common.Address
		if err := addrHex.Decode(addr[:]); err != nil {
			return nil, fmt.Errorf("malformed addres %q: %s", addrHex, err)
		}

		balance, ok := new(big.Int).SetString(account.Balance, 0)
		if !ok {
			return nil, fmt.Errorf("malformed account %q balance %q", addrHex, account.Balance)
		}
		statedb.AddBalance(addr, balance)

		code, err := account.Code.Bytes()
		if err != nil {
			return nil, fmt.Errorf("malformed account %q code: %s", addrHex, err)
		}
		statedb.SetCode(addr, code)

		for key, value := range account.Storage {
			var k, v common.Hash
			if err := key.Decode(k[:]); err != nil {
				return nil, fmt.Errorf("malformed account %q key: %s", addrHex, err)
			}
			if err := value.Decode(v[:]); err != nil {
				return nil, fmt.Errorf("malformed account %q value: %s", addrHex, err)
			}
			statedb.SetState(addr, k, v)
		}
	}
	root, err := statedb.CommitTo(chainDb, false)
	if err != nil {
		return nil, err
	}

	header, err := genesis.Header()
	if err != nil {
		return nil, err
	}
	header.Root = root

	gblock := types.NewBlock(header, nil, nil, nil)

	if block := GetBlock(chainDb, gblock.Hash()); block != nil {
		glog.V(logger.Debug).Infof("Genesis block %s already exists in chain -- writing canonical number", block.Hash().Hex())
		err := WriteCanonicalHash(chainDb, block.Hash(), block.NumberU64())
		if err != nil {
			return nil, err
		}
		return block, nil
	}

	//if err := stateBatch.Write(); err != nil {
	//	return nil, fmt.Errorf("cannot write state: %v", err)
	//}
	if err := WriteTd(chainDb, gblock.Hash(), header.Difficulty); err != nil {
		return nil, err
	}
	if err := WriteBlock(chainDb, gblock); err != nil {
		return nil, err
	}
	if err := WriteBlockReceipts(chainDb, gblock.Hash(), nil); err != nil {
		return nil, err
	}
	if err := WriteCanonicalHash(chainDb, gblock.Hash(), gblock.NumberU64()); err != nil {
		return nil, err
	}
	if err := WriteHeadBlockHash(chainDb, gblock.Hash()); err != nil {
		return nil, err
	}

	return gblock, nil
}

func WriteGenesisBlockForTesting(db ethdb.Database, accounts ...GenesisAccount) *types.Block {
	dump := GenesisDump{
		GasLimit:   "0x47E7C4",
		Difficulty: "0x020000",
		Alloc:      make(map[hex]*GenesisDumpAlloc, len(accounts)),
	}

	for _, a := range accounts {
		dump.Alloc[hex(hexlib.EncodeToString(a.Address[:]))] = &GenesisDumpAlloc{
			Balance: a.Balance.String(),
		}
	}

	block, err := WriteGenesisBlock(db, &dump)
	if err != nil {
		panic(err)
	}
	return block
}

// MakeGenesisDump makes a genesis dump
func MakeGenesisDump(chaindb ethdb.Database) (*GenesisDump, error) {

	genesis := GetBlock(chaindb, GetCanonicalHash(chaindb, 0))
	if genesis == nil {
		return nil, nil
	}

	// Settings.
	genesisHeader := genesis.Header()
	nonce := fmt.Sprintf(`0x%x`, genesisHeader.Nonce)
	time := common.BigToHash(genesisHeader.Time).Hex()
	parentHash := genesisHeader.ParentHash.Hex()
	gasLimit := common.BigToHash(genesisHeader.GasLimit).Hex()
	difficulty := common.BigToHash(genesisHeader.Difficulty).Hex()
	mixHash := genesisHeader.MixDigest.Hex()
	coinbase := genesisHeader.Coinbase.Hex()

	var dump = &GenesisDump{
		Nonce:      prefixedHex(nonce), // common.ToHex(n)), // common.ToHex(
		Timestamp:  prefixedHex(time),
		ParentHash: prefixedHex(parentHash),
		//ExtraData:  prefixedHex(extra),
		GasLimit:   prefixedHex(gasLimit),
		Difficulty: prefixedHex(difficulty),
		Mixhash:    prefixedHex(mixHash),
		Coinbase:   prefixedHex(coinbase),
		//Alloc: ,
	}
	if genesisHeader.Extra != nil && len(genesisHeader.Extra) > 0 {
		dump.ExtraData = prefixedHex(common.ToHex(genesisHeader.Extra))
	}

	// State allocations.
	genState, err := state.New(genesis.Root(), state.NewDatabase(chaindb))
	if err != nil {
		return nil, err
	}
	stateDump := genState.RawDump([]common.Address{})

	stateAccounts := stateDump.Accounts
	dump.Alloc = make(map[hex]*GenesisDumpAlloc, len(stateAccounts))

	for address, acct := range stateAccounts {
		if common.IsHexAddress(address) {
			dump.Alloc[hex(address)] = &GenesisDumpAlloc{
				Balance: acct.Balance,
			}
		} else {
			return nil, fmt.Errorf("Invalid address in genesis state: %v", address)
		}
	}
	return dump, nil
}
