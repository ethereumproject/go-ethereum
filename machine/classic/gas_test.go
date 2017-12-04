package classic

import (
	"github.com/ethereumproject/go-ethereum/core/vm"
	"math/big"
	"testing"
)

func TestGasIsEmpty(t *testing.T) {
	var DefaultGasRepriceGasTable = &vm.GasTable{
		ExtcodeSize:     big.NewInt(700),
		ExtcodeCopy:     big.NewInt(700),
		Balance:         big.NewInt(400),
		SLoad:           big.NewInt(200),
		Calls:           big.NewInt(700),
		Suicide:         big.NewInt(5000),
		ExpByte:         big.NewInt(10),
		CreateBySuicide: big.NewInt(25000),
	}
	if DefaultGasRepriceGasTable.IsEmpty() {
		t.Error("Unexpected IsEmpty() for nonempty gas table.")
	}
}
