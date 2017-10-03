package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"math/big"
	"bytes"

	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/types"
)

func init() {
	glog.SetV(0)
}

var (
	baseDir            = filepath.Join("..", "..", "tests", "files")
	blockTestDir       = filepath.Join(baseDir, "BlockchainTests")
	stateTestDir       = filepath.Join(baseDir, "StateTests")
	transactionTestDir = filepath.Join(baseDir, "TransactionTests")
	vmTestDir          = filepath.Join(baseDir, "VMTests")
	rlpTestDir         = filepath.Join(baseDir, "RLPTests")

	BlockSkipTests = []string{
		// These tests are not valid, as they are out of scope for RLP and
		// the consensus protocol.
		"BLOCK__RandomByteAtTheEnd",
		"TRANSCT__RandomByteAtTheEnd",
		"BLOCK__ZeroByteAtTheEnd",
		"TRANSCT__ZeroByteAtTheEnd",

		"ChainAtoChainB_blockorder2",
		"ChainAtoChainB_blockorder1",
	}

	/* Go client does not support transaction (account) nonces above 2^64. This
	technically breaks consensus but is regarded as "reasonable
	engineering constraint" as accounts cannot easily reach such high
	nonce values in practice
	*/
	TransSkipTests = []string{"TransactionWithHihghNonce256"}
	StateSkipTests = []string{}
	VmSkipTests    = []string{}
)

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

func (self Log) Address() []byte {
	return common.Hex2Bytes(self.AddressF)
}

func (self Log) Data() []byte {
	return common.Hex2Bytes(self.DataF)
}

func (self Log) RlpData() interface{} {
	return nil
}

func (self Log) Topics() [][]byte {
	t := make([][]byte, len(self.TopicsF))
	for i, topic := range self.TopicsF {
		t[i] = common.Hex2Bytes(topic)
	}
	return t
}

func readJson(reader io.Reader, value interface{}) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading JSON file: %v", err)
	}
	if err = json.Unmarshal(data, &value); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(data, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at line %v: %v", line, err)
		}
		return fmt.Errorf("JSON unmarshal error: %v", err)
	}
	return nil
}

func readJsonFile(fn string, value interface{}) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer file.Close()

	err = readJson(file, value)
	if err != nil {
		return fmt.Errorf("%s in file %s", err.Error(), fn)
	}
	return nil
}

// findLine returns the line number for the given offset into data.
func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
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
