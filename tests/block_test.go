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
	"math/big"
	mrand "math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereumproject/go-ethereum/logger/glog"
)

func init() {
	glog.SetD(0)
	glog.SetV(0)
}

func TestBcValidBlockTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcValidBlockTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcUncleHeaderValidityTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcUncleHeaderValiditiy.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcUncleTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcUncleTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcForkUncleTests(t *testing.T) {
	// This test case depends on code that is non-deterministic by purpose - it's a protection against
	// selfish miners. By seeding random number generator here, just before test, with constant seed,
	// we ensure that correct path is selected - randomised chain reorganization action is not executed.
	// This enable to deterministically test non-deterministic code.
	mrand.Seed(123)
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcForkUncle.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcInvalidHeaderTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcInvalidHeaderTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcInvalidRLPTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcInvalidRLPTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcRPCAPITests(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcRPC_API_Test.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcForkBlockTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcForkBlockTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcForkStress(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcForkStressTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcTotalDifficulty(t *testing.T) {
	// skip because these will fail due to selfish mining fix
	t.Skip()

	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcTotalDifficultyTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcWallet(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcWalletTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcGasPricer(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcGasPricerTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

// TODO: iterate over files once we got more than a few
func TestBcRandom(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "RandomTests/bl201507071825GO.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcMultiChain(t *testing.T) {
	// skip due to selfish mining
	t.Skip()

	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcMultiChainTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcState(t *testing.T) {
	err := RunBlockTest(big.NewInt(1000000), big.NewInt(100000), filepath.Join(blockTestDir, "bcStateTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

// Homestead tests
func TestHomesteadBcValidBlockTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcValidBlockTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcUncleHeaderValidityTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcUncleHeaderValiditiy.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcUncleTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcUncleTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcInvalidHeaderTests(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcInvalidHeaderTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcRPCAPITests(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcRPC_API_Test.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcForkStress(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcForkStressTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcTotalDifficulty(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcTotalDifficultyTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcWallet(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcWalletTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcGasPricer(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcGasPricerTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcMultiChain(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcMultiChainTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHomesteadBcState(t *testing.T) {
	err := RunBlockTest(big.NewInt(0), big.NewInt(100000), filepath.Join(blockTestDir, "Homestead", "bcStateTest.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEIP150Bc(t *testing.T) {
	if _, err := os.Stat(filepath.Join(blockTestDir, "TestNetwork", "bcEIP150Test.json")); os.IsNotExist(err) {
		t.Skip("skipping test, need bcEIP150Test.json file")
	}
	err := RunBlockTest(big.NewInt(0), big.NewInt(10), filepath.Join(blockTestDir, "TestNetwork", "bcEIP150Test.json"), BlockSkipTests)
	if err != nil {
		t.Fatal(err)
	}
}
