package core

import (
	"math/big"
	"testing"

	"github.com/ethereumproject/go-ethereum/ethdb"
	"github.com/ethereumproject/go-ethereum/params"
)

// TestMakeGenesisDump is a unit-ish test for MakeGenesisDump()
func TestMakeGenesisDump(t *testing.T) {
	// setup so we have a genesis block in this test db
	db, _ := ethdb.NewMemDatabase()
	genesisDump := &params.GenesisDump{
		Nonce:      "0x0000000000000042",
		Timestamp:  "0x0000000000000000000000000000000000000000000000000000000000000000",
		Coinbase:   "0x0000000000000000000000000000000000000000",
		Difficulty: "0x0000000000000000000000000000000000000000000000000000000400000000",
		ExtraData:  "0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa",
		GasLimit:   "0x0000000000000000000000000000000000000000000000000000000000001388",
		Mixhash:    "0x0000000000000000000000000000000000000000000000000000000000000000",
		ParentHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
		Alloc: map[params.Hex]*params.GenesisDumpAlloc{
			"000d836201318ec6899a67540690382780743280": {Balance: "200000000000000000000"},
			"001762430ea9c3a26e5749afdb70da5f78ddbb8c": {Balance: "200000000000000000000"},
			"001d14804b399c6ef80e64576f657660804fec0b": {Balance: "4200000000000000000000"},
			"0032403587947b9f15622a68d104d54d33dbd1cd": {Balance: "77500000000000000000"},
			"00497e92cdc0e0b963d752b2296acb87da828b24": {Balance: "194800000000000000000"},
			"004bfbe1546bc6c65b5c7eaa55304b38bbfec6d3": {Balance: "2000000000000000000000"},
			"005a9c03f69d17d66cbb8ad721008a9ebbb836fb": {Balance: "2000000000000000000000"},
		},
	}
	gBlock1, err := WriteGenesisBlock(db, genesisDump)
	if err != nil {
		t.Errorf("WriteGenesisBlock could not setup genesisDump: err: %v", err)
	}

	// ensure equivalent genesis dumps in and out
	gotGenesisDump, err := MakeGenesisDump(db)
	if gotGenesisDump.Nonce != genesisDump.Nonce {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump nonce: wanted: %v, got: %v", genesisDump.Nonce, gotGenesisDump.Nonce)
	}
	if gotGenesisDump.Timestamp != genesisDump.Timestamp {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump timestamp: wanted: %v, got: %v", genesisDump.Timestamp, gotGenesisDump.Timestamp)
	}
	if gotGenesisDump.Coinbase != genesisDump.Coinbase {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump coinbase: wanted: %v, got: %v", genesisDump.Coinbase, gotGenesisDump.Coinbase)
	}
	if gotGenesisDump.Difficulty != genesisDump.Difficulty {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump difficulty: wanted: %v, got: %v", genesisDump.Difficulty, gotGenesisDump.Difficulty)
	}
	if gotGenesisDump.ExtraData != genesisDump.ExtraData {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump extra data: wanted: %v, got: %v", genesisDump.ExtraData, gotGenesisDump.ExtraData)
	}
	if gotGenesisDump.GasLimit != genesisDump.GasLimit {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump gaslimit: wanted: %v, got: %v", genesisDump.GasLimit, gotGenesisDump.GasLimit)
	}
	if gotGenesisDump.Mixhash != genesisDump.Mixhash {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump mixhash: wanted: %v, got: %v", genesisDump.Mixhash, gotGenesisDump.Mixhash)
	}
	if gotGenesisDump.ParentHash != genesisDump.ParentHash {
		t.Errorf("MakeGenesisDump failed to make equivalent genesis dump parenthash: wanted: %v, got: %v", genesisDump.ParentHash, gotGenesisDump.ParentHash)
	}
	if !sameGenesisDumpAllocationsBalances(genesisDump, gotGenesisDump) {
		t.Error("MakeGenesisDump failed to make equivalent genesis dump allocations.")
	}

	// ensure equivalent genesis blocks in and out
	gBlock2, err := WriteGenesisBlock(db, gotGenesisDump)
	if err != nil {
		t.Errorf("WriteGenesisBlock could not setup gotGenesisDump: err: %v", err)
	}

	if gBlock1.Hash() != gBlock2.Hash() {
		t.Errorf("MakeGenesisDump failed to make genesis block with equivalent hashes: wanted: %v, got: %v", gBlock1.Hash(), gBlock2.Hash())
	}
	db.Close()
}

func TestMakeGenesisDump2(t *testing.T) {
	// setup so we have a genesis block in this test db
	for i, gen := range []*params.GenesisDump{params.DefaultConfigMainnet.Genesis, params.DefaultConfigMorden.Genesis} {
		db, _ := ethdb.NewMemDatabase()
		genesisDump := gen
		gBlock1, err := WriteGenesisBlock(db, genesisDump)
		if err != nil {
			t.Errorf("WriteGenesisBlock could not setup initial genesisDump: %v, err: %v", i, err)
		}

		// ensure equivalent genesis dumps in and out
		gotGenesisDump, err := MakeGenesisDump(db)

		// ensure equivalent genesis blocks in and out
		gBlock2, err := WriteGenesisBlock(db, gotGenesisDump)
		if err != nil {
			t.Errorf("WriteGenesisBlock could not setup gotGenesisDump: %v, err: %v", i, err)
		}

		if !sameGenesisDumpAllocationsBalances(genesisDump, gotGenesisDump) {
			t.Error("MakeGenesisDump failed to make equivalent genesis dump allocations.")
		}

		if gBlock1.Hash() != gBlock2.Hash() {
			t.Errorf(`MakeGenesisDump failed to make genesis block with equivalent hashes: %v: wanted: %v, got: %v
			WANTED:
			%v

			GOT:
			%v`, i, gBlock1.Hash().Hex(), gBlock2.Hash().Hex(), gBlock1.String(), gBlock2.String())

		}
		db.Close()
	}
}

func sameGenesisDumpAllocationsBalances(gd1, gd2 *params.GenesisDump) bool {
	for address, alloc := range gd2.Alloc {
		if gd1.Alloc[address] != nil {
			bal1, _ := new(big.Int).SetString(gd1.Alloc[address].Balance, 0)
			bal2, _ := new(big.Int).SetString(alloc.Balance, 0)
			if bal1.Cmp(bal2) != 0 {
				return false
			}
		} else if alloc.Balance != "" {
			return false
		}

	}
	return true
}
