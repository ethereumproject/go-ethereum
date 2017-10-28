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

package tests

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/crypto"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/machine/classic"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

const useRawVmByDefault = false

var restrtInRawCalssicVm = false

func init() {

	if len(os.Getenv("DEBUG_TESTS")) > 0 {
		glog.SetToStderr(true)
		glog.SetV(logger.Debug)
	}

	restrtInRawCalssicVm = !useRawVmByDefault && len(os.Getenv("RESTART_TESTS")) > 0
	SetTestVmConfig()

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.Signal(21))
		<-sigc
		signal.Stop(sigc)
		buf := make([]byte, 64*1024)
		stacklen := runtime.Stack(buf, true)
		glog.Errorf("=== TERMINATED ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
		os.Exit(-1)
	}()

}

func SetTestVmConfig() {
	if useRawVmByDefault {
		core.VmManager.SwitchToRawClassicVm()
	} else {
		core.VmManager.EnvConfig()
	}
}

func SetBackupVmConfig() bool {
	if restrtInRawCalssicVm {
		core.VmManager.SwitchToRawClassicVm()
		return true
	}
	return false
}

func checkLogs(tlog []Log, logs state.Logs) error {

	if len(tlog) != len(logs) {
		return fmt.Errorf("log length mismatch. Expected %d, got %d", len(tlog), len(logs))
	} else {
		for i, log := range tlog {
			if common.HexToAddress(log.AddressF) != logs[i].Address {
				return fmt.Errorf("log address expected %v got %x", log.AddressF, logs[i].Address)
			}

			if !bytes.Equal(logs[i].Data, common.FromHex(log.DataF)) {
				return fmt.Errorf("log data expected %v got %x", log.DataF, logs[i].Data)
			}

			if len(log.TopicsF) != len(logs[i].Topics) {
				return fmt.Errorf("log topics length expected %d got %d", len(log.TopicsF), logs[i].Topics)
			} else {
				for j, topic := range log.TopicsF {
					if common.HexToHash(topic) != logs[i].Topics[j] {
						return fmt.Errorf("log topic[%d] expected %v got %x", j, topic, logs[i].Topics[j])
					}
				}
			}
			genBloom := common.LeftPadBytes(types.LogsBloom(state.Logs{logs[i]}).Bytes(), 256)

			if !bytes.Equal(genBloom, common.Hex2Bytes(log.BloomF)) {
				return fmt.Errorf("bloom mismatch")
			}
		}
	}
	return nil
}

type Account struct {
	Balance string
	Code    string
	Nonce   string
	Storage map[string]string
}

type Log struct {
	AddressF string   `json:"address"`
	DataF    string   `json:"data"`
	TopicsF  []string `json:"topics"`
	BloomF   string   `json:"bloom"`
}

func (self Log) Address() []byte      { return common.Hex2Bytes(self.AddressF) }
func (self Log) Data() []byte         { return common.Hex2Bytes(self.DataF) }
func (self Log) RlpData() interface{} { return nil }
func (self Log) Topics() [][]byte {
	t := make([][]byte, len(self.TopicsF))
	for i, topic := range self.TopicsF {
		t[i] = common.Hex2Bytes(topic)
	}
	return t
}

func makePreState(db ethdb.Database, accounts map[string]Account) *state.StateDB {
	statedb, _ := state.New(common.Hash{}, db)
	for addr, account := range accounts {
		insertAccount(statedb, addr, account)
	}
	return statedb
}

func insertAccount(state *state.StateDB, saddr string, account Account) {
	if common.IsHex(account.Code) {
		account.Code = account.Code[2:]
	}
	addr := common.HexToAddress(saddr)
	state.SetCode(addr, common.Hex2Bytes(account.Code))
	if i, err := strconv.ParseUint(account.Nonce, 0, 64); err != nil {
		panic(err)
	} else {
		state.SetNonce(addr, i)
	}
	if i, ok := new(big.Int).SetString(account.Balance, 0); !ok {
		panic("malformed account balance")
	} else {
		state.SetBalance(addr, i)
	}
	for a, v := range account.Storage {
		state.SetState(addr, common.HexToHash(a), common.HexToHash(v))
	}
}

type VmEnv struct {
	CurrentCoinbase   string
	CurrentDifficulty string
	CurrentGasLimit   string
	CurrentNumber     string
	CurrentTimestamp  interface{}
	PreviousHash      string
}

type VmTest struct {
	Callcreates interface{}
	//Env         map[string]string
	Env           VmEnv
	Exec          map[string]string
	Transaction   map[string]string
	Logs          []Log
	Gas           string
	Out           string
	Post          map[string]Account
	Pre           map[string]Account
	PostStateRoot string
}

type RuleSet struct {
	HomesteadBlock           *big.Int
	HomesteadGasRepriceBlock *big.Int
	DiehardBlock             *big.Int
	ExplosionBlock           *big.Int
}

func (r RuleSet) IsHomestead(n *big.Int) bool {
	return n.Cmp(r.HomesteadBlock) >= 0
}
func (r RuleSet) GasTable(num *big.Int) *vm.GasTable {
	if r.HomesteadGasRepriceBlock == nil || num == nil || num.Cmp(r.HomesteadGasRepriceBlock) < 0 {
		return &vm.GasTable{
			ExtcodeSize:     big.NewInt(20),
			ExtcodeCopy:     big.NewInt(20),
			Balance:         big.NewInt(20),
			SLoad:           big.NewInt(50),
			Calls:           big.NewInt(40),
			Suicide:         big.NewInt(0),
			ExpByte:         big.NewInt(10),
			CreateBySuicide: nil,
		}
	}
	if r.DiehardBlock == nil || num == nil || num.Cmp(r.DiehardBlock) < 0 {
		return &vm.GasTable{
			ExtcodeSize:     big.NewInt(700),
			ExtcodeCopy:     big.NewInt(700),
			Balance:         big.NewInt(400),
			SLoad:           big.NewInt(200),
			Calls:           big.NewInt(700),
			Suicide:         big.NewInt(5000),
			ExpByte:         big.NewInt(10),
			CreateBySuicide: big.NewInt(25000),
		}
	}

	return &vm.GasTable{
		ExtcodeSize:     big.NewInt(700),
		ExtcodeCopy:     big.NewInt(700),
		Balance:         big.NewInt(400),
		SLoad:           big.NewInt(200),
		Calls:           big.NewInt(700),
		Suicide:         big.NewInt(5000),
		ExpByte:         big.NewInt(50),
		CreateBySuicide: big.NewInt(25000),
	}
}

func NewEnvFromMap(ruleSet RuleSet, db *state.StateDB, envValues map[string]string, exeValues map[string]string) *core.VmEnv {

	number, _ := new(big.Int).SetString(envValues["currentNumber"], 0)
	if number == nil {
		panic("malformed current number")
	}
	time, _ := new(big.Int).SetString(envValues["currentTimestamp"], 0)
	if time == nil {
		panic("malformed current timestamp")
	}
	difficulty, _ := new(big.Int).SetString(envValues["currentDifficulty"], 0)
	if difficulty == nil {
		panic("malformed current difficulty")
	}
	gasLimit, _ := new(big.Int).SetString(envValues["currentGasLimit"], 0)
	if gasLimit == nil {
		panic("malformed current gas limit")
	}

	fork := vm.Frontier
	if ruleSet.IsHomestead(number) {
		fork = vm.Homestead
	}

	hashFn := func(n uint64) common.Hash {
		return common.BytesToHash(crypto.Keccak256([]byte(big.NewInt(int64(n)).String())))
	}

	machine, err := core.VmManager.ConnectMachine()
	if machine == nil {
		panic(fmt.Sprintf("could not connect machine: %v", err))
	}

	env := &core.VmEnv{
		common.HexToAddress(envValues["currentCoinbase"]),
		number,
		difficulty,
		gasLimit,
		time,
		db,
		hashFn,
		fork,
		ruleSet,
		machine,
	}

	if machine.Type() == vm.ClassicRawVm {
		machine.(*classic.RawMachine).Bind(db, hashFn)
	}

	return env
}

type Message struct {
	from              common.Address
	to                *common.Address
	value, gas, price *big.Int
	data              []byte
	nonce             uint64
}

func NewMessage(from common.Address, to *common.Address, data []byte, value, gas, price *big.Int, nonce uint64) Message {
	return Message{from, to, value, gas, price, data, nonce}
}

func (self Message) Hash() []byte                          { return nil }
func (self Message) From() (common.Address, error)         { return self.from, nil }
func (self Message) FromFrontier() (common.Address, error) { return self.from, nil }
func (self Message) To() *common.Address                   { return self.to }
func (self Message) GasPrice() *big.Int                    { return self.price }
func (self Message) Gas() *big.Int                         { return self.gas }
func (self Message) Value() *big.Int                       { return self.value }
func (self Message) Nonce() uint64                         { return self.nonce }
func (self Message) Data() []byte                          { return self.data }
