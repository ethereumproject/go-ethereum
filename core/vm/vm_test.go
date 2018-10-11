package vm

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type testRuleSet string

type eip1283TestCase struct {
	index      int
	code       string
	wantGas    *big.Int
	wantRefund *big.Int
	originalV  *big.Int
	iters      [3]int64 // potential for 3 possible iterations, -1 will be used when only 2 iterations wanted
}

func (c eip1283TestCase) String() string {
	return fmt.Sprintf("i=%d code=%s gas=%v refund=%v original=%v iters=%v", c.index, c.code, c.wantGas, c.wantRefund, c.originalV, c.iters)
}

// TestEIP1283SStoreGas reads the specified test cases from the table in the raw EIP1283 markdown document
// and runs them as test cases. By reading directly from the EIP document, we're able to sidestep potential
// human errors in test implementation (read here, write there - human challenge).
func TestEIP1283SStoreGas(t *testing.T) {
	cases := []eip1283TestCase{}
	eipSpecPath, err := filepath.Abs("../../tests/files/VMTests/ECIP1045/vmEIP1283/eip-1283.md")
	if err != nil {
		t.Fatal(err)
	}
	readBytes, err := ioutil.ReadFile(eipSpecPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(readBytes, []byte("\n"))
	i := 0
	for _, line := range lines {
		lineString := string(line)
		if !strings.HasPrefix(lineString, "|") || !strings.Contains(lineString, "`") || strings.Count(lineString, "|") != 8 {
			continue
		}
		cols := strings.Split(lineString, "|")
		_code, _gas, _refund, _original, _1st, _2nd, _3rd := cols[1], cols[2], cols[3], cols[4], cols[5], cols[6], cols[7]
		_code = strings.TrimSpace(_code)
		_code = strings.Trim(_code, "`")

		_gas = strings.TrimSpace(_gas)
		_refund = strings.TrimSpace(_refund)
		_original = strings.TrimSpace(_original)
		_1st = strings.TrimSpace(_1st)
		_2nd = strings.TrimSpace(_2nd)
		_3rd = strings.TrimSpace(_3rd)

		_gasI, _ := strconv.Atoi(_gas)
		_refundI, _ := strconv.Atoi(_refund)
		_originalI, _ := strconv.Atoi(_original)
		_1stI, _ := strconv.Atoi(_1st)
		_2ndI, _ := strconv.Atoi(_2nd)
		var _3rdI int
		if _3rd == "" {
			_3rdI = -1
		} else {
			_3rdI, _ = strconv.Atoi(_3rd)
		}

		c := eip1283TestCase{
			index:      i,
			code:       _code,
			wantGas:    big.NewInt(int64(_gasI)),
			wantRefund: big.NewInt(int64(_refundI)),
			originalV:  big.NewInt(int64(_originalI)),
			iters:      [3]int64{int64(_1stI), int64(_2ndI), int64(_3rdI)},
		}
		cases = append(cases, c)
		i++
	}

	if len(cases) < 17 {
		t.Fatal("unexpected number of cases", len(cases))
	}

	for i, test := range cases {
		gotGas := big.NewInt(0)
		gotRefund := big.NewInt(0)
		currentV := new(big.Int).Set(test.originalV)
		for _, iter := range test.iters {
			if iter == -1 {
				break
			}
			newV := new(big.Int).SetUint64(uint64(iter))

			gas, ref := eip1283sstoreGas(test.originalV, currentV, newV)
			currentV.Set(newV)

			gotGas.Add(gotGas, gas)
			gotRefund.Add(gotRefund, ref)
		}

		// Add gas for PUSH1 ops.
		pushN := strings.Count(test.code, "60")
		gotGas.Add(gotGas, new(big.Int).Mul(big.NewInt(int64(pushN)), big.NewInt(3)))

		if gotGas.Cmp(test.wantGas) != 0 {
			t.Log(test)
			t.Errorf("test: %v; [gas] got=%v, want=%v, diff=%v", i, gotGas, test.wantGas, new(big.Int).Sub(gotGas, test.wantGas))
		}
		if gotRefund.Cmp(test.wantRefund) != 0 {
			t.Log(test)
			t.Errorf("test: %v; [refund] got=%v, want=%v, diff=%v", i, gotRefund, test.wantRefund, new(big.Int).Sub(gotRefund, test.wantRefund))
		}
	}
}
