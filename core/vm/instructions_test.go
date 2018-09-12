package vm

import (
	"github.com/ethereumproject/go-ethereum/common"
	"github.com/ethereumproject/go-ethereum/ethdb"
	"math/big"
	"testing"
)

type twoOperandTest struct {
	x, y     string
	expected string
}

type VMTestEnv struct {
	ruleset    rs
	db         *ethdb.MemDatabase
	evm        *EVM
	readOnly   bool
	returnData []byte
}

func (VMTestEnv) RuleSet() RuleSet {
	panic("implement me")
}

func (VMTestEnv) Db() Database {
	panic("implement me")
}

func (VMTestEnv) SnapshotDatabase() int {
	panic("implement me")
}

func (VMTestEnv) RevertToSnapshot(int) {
	panic("implement me")
}

func (VMTestEnv) Origin() common.Address {
	panic("implement me")
}

func (VMTestEnv) BlockNumber() *big.Int {
	panic("implement me")
}

func (VMTestEnv) GetHash(uint64) common.Hash {
	panic("implement me")
}

func (VMTestEnv) Coinbase() common.Address {
	panic("implement me")
}

func (VMTestEnv) Time() *big.Int {
	panic("implement me")
}

func (VMTestEnv) Difficulty() *big.Int {
	panic("implement me")
}

func (VMTestEnv) GasLimit() *big.Int {
	panic("implement me")
}

func (VMTestEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	panic("implement me")
}

func (VMTestEnv) Transfer(from, to Account, amount *big.Int) {
	panic("implement me")
}

func (VMTestEnv) AddLog(*Log) {
	panic("implement me")
}

func (VMTestEnv) Vm() Vm {
	panic("implement me")
}

func (VMTestEnv) Depth() int {
	panic("implement me")
}

func (VMTestEnv) SetDepth(i int) {
	panic("implement me")
}

func (VMTestEnv) SetReadOnly(isReadOnly bool) {
	panic("implement me")
}

func (VMTestEnv) IsReadOnly() bool {
	panic("implement me")
}

func (VMTestEnv) SetReturnData(data []byte) {
	panic("implement me")
}

func (VMTestEnv) ReturnData() []byte {
	panic("implement me")
}

func (VMTestEnv) Call(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	panic("implement me")
}

func (VMTestEnv) CallCode(me ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	panic("implement me")
}

func (VMTestEnv) DelegateCall(me ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	panic("implement me")
}

func (VMTestEnv) StaticCall(me ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	panic("implement me")
}

func (VMTestEnv) Create(me ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	panic("implement me")
}

func (VMTestEnv) Create2(me ContractRef, data []byte, gas, price, value, salt *big.Int) ([]byte, common.Address, error) {
	panic("implement me")
}

type rs struct {
	HomesteadBlock           *big.Int
	HomesteadGasRepriceBlock *big.Int
	DiehardBlock             *big.Int
	ExplosionBlock           *big.Int
	ECIP1045BBlock           *big.Int
	ECIP1045CBlock           *big.Int
}

func newTestRS() rs {
	return rs{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: new(big.Int),
		DiehardBlock:             new(big.Int),
		ExplosionBlock:           new(big.Int),
		ECIP1045BBlock:           new(big.Int),
		ECIP1045CBlock:           new(big.Int),
	}
}

func (r rs) IsHomestead(n *big.Int) bool {
	if n == nil || r.HomesteadBlock == nil {
		return false
	}
	return n.Cmp(r.HomesteadBlock) >= 0
}

func (r rs) IsEIP150(n *big.Int) bool {
	if n == nil || r.HomesteadGasRepriceBlock == nil {
		return false
	}
	return n.Cmp(r.HomesteadGasRepriceBlock) >= 0
}

func (r rs) IsDiehard(n *big.Int) bool {
	if n == nil || r.DiehardBlock == nil {
		return false
	}
	return n.Cmp(r.DiehardBlock) >= 0
}

func (r rs) IsECIP1045B(n *big.Int) bool {
	if n == nil || r.ECIP1045BBlock == nil {
		return false
	}
	return n.Cmp(r.ECIP1045BBlock) >= 0
}

func (r rs) IsECIP1045C(n *big.Int) bool {
	if n == nil || r.ECIP1045CBlock == nil {
		return false
	}
	return n.Cmp(r.ECIP1045CBlock) >= 0
}

func (r rs) GasTable(num *big.Int) *GasTable {
	if r.IsECIP1045C(num) {
		return &GasTable{}
	} else if r.IsDiehard(num) {
		return &GasTable{}
	} else if r.IsEIP150(num) {
		return &GasTable{}
	} else if r.IsHomestead(num) {
		return &GasTable{}
	}
	return &GasTable{}
}

func newVMTestEnv() (VMTestEnv, error) {
	env := VMTestEnv{}
	db, err := ethdb.NewMemDatabase()
	if err != nil {
		return env, err
	}
	env.db = db
	env.ruleset = newTestRS()

	return env, err
}

func testTwoOperandOp(t *testing.T, tests []twoOperandTest, opFn func(instr instruction, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack)) {
	tenv, err := newVMTestEnv()
	if err != nil {
		t.Fatal(err)
	}
	stak := newstack()
	pc := uint64(0)
	for i, test := range tests {
		x := new(big.Int).SetBytes(common.Hex2Bytes(test.x))
		shift := new(big.Int).SetBytes(common.Hex2Bytes(test.y))
		expected := new(big.Int).SetBytes(common.Hex2Bytes(test.expected))
		stak.push(x)
		stak.push(shift)
		opFn(instruction{}, &pc, tenv, nil, nil, stak)
		actual := stak.pop()
		if actual.Cmp(expected) != 0 {
			t.Errorf("testcase %d, want=%v, got=%v", i, expected, actual)
		}
	}
}

func TestSHL(t *testing.T) {
	// Testcases from https://github.com/ethereum/EIPs/blob/master/EIPS/eip-145.md#shl-shift-left
	tests := []twoOperandTest{
		{"0000000000000000000000000000000000000000000000000000000000000001", "00", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "01", "0000000000000000000000000000000000000000000000000000000000000002"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "ff", "8000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "0101", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "00", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "01", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ff", "8000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000000", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "01", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe"},
	}
	testTwoOperandOp(t, tests, opSHL)
}
func TestSHR(t *testing.T) {
	// Testcases from https://github.com/ethereum/EIPs/blob/master/EIPS/eip-145.md#shr-logical-shift-right
	tests := []twoOperandTest{
		{"0000000000000000000000000000000000000000000000000000000000000001", "00", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "01", "4000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "ff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "0101", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "00", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "01", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000000", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
	}
	testTwoOperandOp(t, tests, opSHR)
}

func TestSAR(t *testing.T) {
	// Testcases from https://github.com/ethereum/EIPs/blob/master/EIPS/eip-145.md#sar-arithmetic-shift-right
	tests := []twoOperandTest{
		{"0000000000000000000000000000000000000000000000000000000000000001", "00", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "01", "c000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "ff", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "0100", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"8000000000000000000000000000000000000000000000000000000000000000", "0101", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "00", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "01", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ff", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0100", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{"0000000000000000000000000000000000000000000000000000000000000000", "01", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"4000000000000000000000000000000000000000000000000000000000000000", "fe", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "f8", "000000000000000000000000000000000000000000000000000000000000007f"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "fe", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0100", "0000000000000000000000000000000000000000000000000000000000000000"},
	}

	testTwoOperandOp(t, tests, opSAR)
}
func TestSGT(t *testing.T) {
	tests := []twoOperandTest{

		{"0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000001", "8000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000001", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "8000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb", "0000000000000000000000000000000000000000000000000000000000000000"},
	}
	testTwoOperandOp(t, tests, opSgt)
}

func TestSLT(t *testing.T) {
	tests := []twoOperandTest{
		{"0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"0000000000000000000000000000000000000000000000000000000000000001", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"8000000000000000000000000000000000000000000000000000000000000001", "8000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"8000000000000000000000000000000000000000000000000000000000000001", "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "8000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000001"},
		{"fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd", "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb", "0000000000000000000000000000000000000000000000000000000000000001"},
	}
	testTwoOperandOp(t, tests, opSlt)
}
