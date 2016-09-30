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

package core

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/core/state"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/core/vm"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/event"
	"github.com/ethereumproject/go-ethereum/pow/ezp"
)

func testChainConfig() *ChainConfig {
	return &ChainConfig{
		Forks: []*Fork{
			&Fork{
				Name:  "Homestead",
				Block: big.NewInt(0),
			},
		},
	}
}

func proc() (Validator, *BlockChain) {
	db, _ := ethdb.NewMemDatabase()
	var mux event.TypeMux

	WriteTestNetGenesisBlock(db)
	blockchain, err := NewBlockChain(db, testChainConfig(), thePow(), &mux)
	if err != nil {
		fmt.Println(err)
	}
	return blockchain.validator, blockchain
}

func TestNumber(t *testing.T) {
	pow := ezp.New()
	_, chain := proc()

	statedb, _ := state.New(chain.Genesis().Root(), chain.chainDb)
	header := makeHeader(chain.Genesis(), statedb)
	header.Number = big.NewInt(3)
	cfg := testChainConfig()
	err := ValidateHeader(cfg, pow, header, chain.Genesis().Header(), false, false)
	if err != BlockNumberErr {
		t.Errorf("expected block number error, got %q", err)
	}

	header = makeHeader(chain.Genesis(), statedb)
	err = ValidateHeader(cfg, pow, header, chain.Genesis().Header(), false, false)
	if err == BlockNumberErr {
		t.Errorf("didn't expect block number error")
	}
}

func TestPutReceipt(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	var addr common.Address
	addr[0] = 1
	var hash common.Hash
	hash[0] = 2

	receipt := new(types.Receipt)
	receipt.Logs = vm.Logs{&vm.Log{
		Address:     addr,
		Topics:      []common.Hash{hash},
		Data:        []byte("hi"),
		BlockNumber: 42,
		TxHash:      hash,
		TxIndex:     0,
		BlockHash:   hash,
		Index:       0,
	}}

	WriteReceipts(db, types.Receipts{receipt})
	receipt = GetReceipt(db, common.Hash{})
	if receipt == nil {
		t.Error("expected to get 1 receipt, got none.")
	}
}

func TestDifficultyBombFreeze(t *testing.T) {
	numIgnored := big.NewInt(3000000)
	var parentTime uint64

	// 20 seconds, parent diff 7654414978364
	parentTime = 1452838500
	act := calcDifficultyDiehard(parentTime+20, parentTime, numIgnored, big.NewInt(7654414978364))
	exp := big.NewInt(7650945906507)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 5 seconds, parent diff 7654414978364
	act = calcDifficultyDiehard(parentTime+5, parentTime, numIgnored, big.NewInt(7654414978364))
	exp = big.NewInt(7658420921133)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 80 seconds, parent diff 7654414978364
	act = calcDifficultyDiehard(parentTime+80, parentTime, numIgnored, big.NewInt(7654414978364))
	exp = big.NewInt(7628520862629)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 76 seconds, parent diff 12657404717367
	parentTime = 1469081721
	act = calcDifficultyDiehard(parentTime+76, parentTime, numIgnored, big.NewInt(12657404717367))
	exp = big.NewInt(12620590912441)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 1 second, parent diff 12620590912441
	parentTime = 1469164521
	act = calcDifficultyDiehard(parentTime+1, parentTime, numIgnored, big.NewInt(12620590912441))
	exp = big.NewInt(12627021745803)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	// 10 seconds, parent diff 12627021745803
	parentTime = 1469164522
	act = calcDifficultyDiehard(parentTime+10, parentTime, numIgnored, big.NewInt(12627021745803))
	exp = big.NewInt(12627290181259)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

}

func TestDifficultyBombExplode(t *testing.T) {
	var parentTime uint64

	// 6 seconds, blocks in 5m
	num := big.NewInt(5000102)
	parentTime = 1513175023
	act := calcDifficultyExplosion(parentTime+6, parentTime, num, big.NewInt(22627021745803))
	exp := big.NewInt(22638338531720)
	if exp.Cmp(act) != 0 {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(5001105)
	parentTime = 1513189406
	act = calcDifficultyExplosion(parentTime + 29, parentTime, num, big.NewInt(22727021745803))
	exp = big.NewInt(22716193002673)
	if (exp.Cmp(act) != 0) {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(5100123)
	parentTime = 1514609324
	act = calcDifficultyExplosion(parentTime + 41, parentTime, num, big.NewInt(22893437765583))
	exp = big.NewInt(22860439327271)
	if (exp.Cmp(act) != 0) {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(6150001)
	parentTime = 1529664575
	act = calcDifficultyExplosion(parentTime + 105, parentTime, num, big.NewInt(53134780363303))
	exp = big.NewInt(53451033724425)
	if (exp.Cmp(act) != 0) {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(6550001)
	parentTime = 1535431724
	act = calcDifficultyExplosion(parentTime + 60, parentTime, num, big.NewInt(82893437765583))
	exp = big.NewInt(91487154230751)
	if (exp.Cmp(act) != 0) {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(7000000)
	parentTime = 1535431724
	act = calcDifficultyExplosion(parentTime + 180, parentTime, num, big.NewInt(304334879167015))
	exp = big.NewInt(583283638618965)
	if (exp.Cmp(act) != 0) {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(8000000)
	parentTime = 1535431724
	act = calcDifficultyExplosion(parentTime + 420, parentTime, num, big.NewInt(78825323605416810))
	exp = big.NewInt(365477653727918567)
	if (exp.Cmp(act) != 0) {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}

	num = big.NewInt(9000000)
	parentTime = 1535431724
	act = calcDifficultyExplosion(parentTime + 2040, parentTime, num, big.NewInt(288253236054168103))
	exp = big.NewInt(0)
	exp.SetString("295422224299015703633", 10)
	if (exp.Cmp(act) != 0) {
		t.Errorf("Expected to have %d difficulty, got %d (difference: %d)",
			exp, act, big.NewInt(0).Sub(act, exp))
	}
}
