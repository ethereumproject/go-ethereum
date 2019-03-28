** All code blocks and line numbers are referencing go-ethereum, not etc

## [EIP 140](http://eips.ethereum.org/EIPS/eip-140) Addition of 'REVERT' opcode, to permit error handling without consuming all gas

[EIP PR](https://github.com/ethereum/EIPs/pull/206)  
[Yellowpaper PR](https://github.com/ethereum/yellowpaper/pull/242)  
[Parity ruby implementation](https://gitlab.parity.io/parity/parity-ethereum/commit/dd004aba9f9640091e2d9c3f95dd7c3142106a4d)

- Introduce the 0xfd op code (for blocks after byantium fork) to be able to stop and revert state changes

- Revert should not cost gas

- Provide a pointer to a memory section (can be interpreted as error code or message)

- Be able to provide a reason when reverted (I Assume that is in the form of the previous point).

- If not enough gas or stack underflow, all gas will be consumed

`core/vm/jump_table.go` line 138

```go
	instructionSet[REVERT] = operation{
		execute:    opRevert,
		dynamicGas: gasRevert,
		minStack:   minStack(2, 0),
		maxStack:   maxStack(2, 0),
		memorySize: memoryRevert,
		valid:      true,
		reverts:    true,
		returns:    true,
	}
```

`core/vm/instructions.go` line 867

```go
func opRevert(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	offset, size := stack.pop(), stack.pop()
	ret := memory.GetPtr(offset.Int64(), size.Int64())

	interpreter.intPool.put(offset, size)
	return ret, nil
}
```

`core/vm/gas_table.go` line 449

```go
func gasRevert(gt params.GasTable, evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	return memoryGasCost(mem, memorySize)
}
```

`core/vm/memory_table.go` line 107

```go
func memoryRevert(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.Back(0), stack.Back(1))
}
```

`eth/tracers/internal/tracers/call_tracer.js` line 110

```js
// If an existing call is returning, pop off the call stack
if (syscall && op == "REVERT") {
  this.callstack[this.callstack.length - 1].error = "execution reverted";
  return;
}
```

---

## [EIP 658](http://eips.ethereum.org/EIPS/eip-658) Transaction receipts to include a status field to indicate success or failure

- Continuation of EIP 140

- Success or failure status

```go
returns:    true,
```

of `core/vm/jump_table.go` line 146 (from previous)?

---

## Elliptic curve addition and scalar multiplication on alt_bn128 ([EIP 196](http://eips.ethereum.org/EIPS/eip-196)) and pairing checks ([EIP 197](http://eips.ethereum.org/EIPS/eip-197)), permitting ZK-Snarks and other cryptographic mathemagic™


- ? EIP specs say using alt_bn128 but all code seems to be split on referencing bn128 and bn256 ? Is there something I need to know to bridge this inconsistency?

`cmd/puppeth/genesis.go` line 405

```go
if genesis.Config.ByzantiumBlock != nil {
  blnum := math2.HexOrDecimal64(genesis.Config.ByzantiumBlock.Uint64())
  spec.setPrecompile(5, &parityChainSpecBuiltin{
    Name: "modexp", ActivateAt: blnum, Pricing: &parityChainSpecPricing{ModExp: &parityChainSpecModExpPricing{Divisor: 20}},
  })
  spec.setPrecompile(6, &parityChainSpecBuiltin{
    Name: "alt_bn128_add", ActivateAt: blnum, Pricing: &parityChainSpecPricing{Linear: &parityChainSpecLinearPricing{Base: 500}},
  })
  spec.setPrecompile(7, &parityChainSpecBuiltin{
    Name: "alt_bn128_mul", ActivateAt: blnum, Pricing: &parityChainSpecPricing{Linear: &parityChainSpecLinearPricing{Base: 40000}},
  })
  spec.setPrecompile(8, &parityChainSpecBuiltin{
    Name: "alt_bn128_pairing", ActivateAt: blnum, Pricing: &parityChainSpecPricing{AltBnPairing: &parityChainSpecAltBnPairingPricing{Base: 100000, Pair: 80000}},
  })
  }
```

`cmd/puppeth/genesis.go` line 158

```go
if genesis.Config.ByzantiumBlock != nil {
  spec.setPrecompile(5, &alethGenesisSpecBuiltin{Name: "modexp",
    StartingBlock: (hexutil.Uint64)(genesis.Config.ByzantiumBlock.Uint64())})
  spec.setPrecompile(6, &alethGenesisSpecBuiltin{Name: "alt_bn128_G1_add",
    StartingBlock: (hexutil.Uint64)(genesis.Config.ByzantiumBlock.Uint64()),
    Linear:        &alethGenesisSpecLinearPricing{Base: 500}})
  spec.setPrecompile(7, &alethGenesisSpecBuiltin{Name: "alt_bn128_G1_mul",
    StartingBlock: (hexutil.Uint64)(genesis.Config.ByzantiumBlock.Uint64()),
    Linear:        &alethGenesisSpecLinearPricing{Base: 40000}})
  spec.setPrecompile(8, &alethGenesisSpecBuiltin{Name: "alt_bn128_pairing_product",
    StartingBlock: (hexutil.Uint64)(genesis.Config.ByzantiumBlock.Uint64())})
}
```

`core/genesis.go` line 372

```go
common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
```

`cmd/puppeth/genesis.go` line 286

```go
// parityChainSpecPricing represents the different pricing models that builtin
// contracts might advertise using.
type parityChainSpecPricing struct {
	...
	ModExp       *parityChainSpecModExpPricing       `json:"modexp,omitempty"`
	AltBnPairing *parityChainSpecAltBnPairingPricing `json:"alt_bn128_pairing,omitempty"`
}
```

`core/vm/contracts.go` line 49

```go
// PrecompiledContractsByzantium contains the default set of pre-compiled Ethereum
// contracts used in the Byzantium release.
var PrecompiledContractsByzantium = map[common.Address]PrecompiledContract{
	...
	common.BytesToAddress([]byte{5}): &bigModExp{},
	common.BytesToAddress([]byte{6}): &bn256Add{},
	common.BytesToAddress([]byte{7}): &bn256ScalarMul{},
	common.BytesToAddress([]byte{8}): &bn256Pairing{},
}
```

`core/vm/contracts.go` line 274

```go
// bn256Add implements a native elliptic curve point addition.
type bn256Add struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256Add) RequiredGas(input []byte) uint64 {
	return params.Bn256AddGas
}

func (c *bn256Add) Run(input []byte) ([]byte, error) {
	x, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	y, err := newCurvePoint(getData(input, 64, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.Add(x, y)
	return res.Marshal(), nil
}

// bn256ScalarMul implements a native elliptic curve scalar multiplication.
type bn256ScalarMul struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256ScalarMul) RequiredGas(input []byte) uint64 {
	return params.Bn256ScalarMulGas
}

func (c *bn256ScalarMul) Run(input []byte) ([]byte, error) {
	p, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.ScalarMult(p, new(big.Int).SetBytes(getData(input, 64, 32)))
	return res.Marshal(), nil
}

var (
	// true32Byte is returned if the bn256 pairing check succeeds.
	true32Byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	// false32Byte is returned if the bn256 pairing check fails.
	false32Byte = make([]byte, 32)

	// errBadPairingInput is returned if the bn256 pairing input is invalid.
	errBadPairingInput = errors.New("bad elliptic curve pairing size")
)

// bn256Pairing implements a pairing pre-compile for the bn256 curve
type bn256Pairing struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256Pairing) RequiredGas(input []byte) uint64 {
	return params.Bn256PairingBaseGas + uint64(len(input)/192)*params.Bn256PairingPerPointGas
}

func (c *bn256Pairing) Run(input []byte) ([]byte, error) {
	// Handle some corner cases cheaply
	if len(input)%192 > 0 {
		return nil, errBadPairingInput
	}
	// Convert the input into a set of coordinates
	var (
		cs []*bn256.G1
		ts []*bn256.G2
	)
	for i := 0; i < len(input); i += 192 {
		c, err := newCurvePoint(input[i : i+64])
		if err != nil {
			return nil, err
		}
		t, err := newTwistPoint(input[i+64 : i+192])
		if err != nil {
			return nil, err
		}
		cs = append(cs, c)
		ts = append(ts, t)
	}
	// Execute the pairing checks and return the results
	if bn256.PairingCheck(cs, ts) {
		return true32Byte, nil
	}
	return false32Byte, nil
}

```

**precompiled tests:**

`core/vm/contracts_test.go` line 471

```go
// Tests the sample inputs from the elliptic curve pairing check EIP 197.
func TestPrecompiledBn256Pairing(t *testing.T) {
	for _, test := range bn256PairingTests {
		testPrecompiled("08", test, t)
	}
}

// Behcnmarks the sample inputs from the elliptic curve pairing check EIP 197.
func BenchmarkPrecompiledBn256Pairing(bench *testing.B) {
	for _, test := range bn256PairingTests {
		benchmarkPrecompiled("08", test, bench)
	}
}

```

`core/vm/contracts_test.go` line 429

```go
// Tests the sample inputs from the ModExp EIP 198.
func TestPrecompiledModExp(t *testing.T) {
	for _, test := range modexpTests {
		testPrecompiled("05", test, t)
	}
}

// Benchmarks the sample inputs from the ModExp EIP 198.
func BenchmarkPrecompiledModExp(bench *testing.B) {
	for _, test := range modexpTests {
		benchmarkPrecompiled("05", test, bench)
	}
}
```

---

## Support for big integer modular exponentiation ([EIP 198](http://eips.ethereum.org/EIPS/eip-198)), enabling RSA signature verification and other cryptographic applications

(Included most code in previous EIP because they were grouped)

All logic seems to be in this file:

`core/vm/contracts.go` line 150

```go
// bigModExp implements a native big integer exponential modular operation.
type bigModExp struct{}

var (
	big1      = big.NewInt(1)
	big4      = big.NewInt(4)
	big8      = big.NewInt(8)
	big16     = big.NewInt(16)
	big32     = big.NewInt(32)
	big64     = big.NewInt(64)
	big96     = big.NewInt(96)
	big480    = big.NewInt(480)
	big1024   = big.NewInt(1024)
	big3072   = big.NewInt(3072)
	big199680 = big.NewInt(199680)
)

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bigModExp) RequiredGas(input []byte) uint64 {
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32))
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32))
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32))
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Retrieve the head 32 bytes of exp for the adjusted exponent length
	var expHead *big.Int
	if big.NewInt(int64(len(input))).Cmp(baseLen) <= 0 {
		expHead = new(big.Int)
	} else {
		if expLen.Cmp(big32) > 0 {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), 32))
		} else {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), expLen.Uint64()))
		}
	}
	// Calculate the adjusted exponent length
	var msb int
	if bitlen := expHead.BitLen(); bitlen > 0 {
		msb = bitlen - 1
	}
	adjExpLen := new(big.Int)
	if expLen.Cmp(big32) > 0 {
		adjExpLen.Sub(expLen, big32)
		adjExpLen.Mul(big8, adjExpLen)
	}
	adjExpLen.Add(adjExpLen, big.NewInt(int64(msb)))

	// Calculate the gas cost of the operation
	gas := new(big.Int).Set(math.BigMax(modLen, baseLen))
	switch {
	case gas.Cmp(big64) <= 0:
		gas.Mul(gas, gas)
	case gas.Cmp(big1024) <= 0:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big4),
			new(big.Int).Sub(new(big.Int).Mul(big96, gas), big3072),
		)
	default:
		gas = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(gas, gas), big16),
			new(big.Int).Sub(new(big.Int).Mul(big480, gas), big199680),
		)
	}
	gas.Mul(gas, math.BigMax(adjExpLen, big1))
	gas.Div(gas, new(big.Int).SetUint64(params.ModExpQuadCoeffDiv))

	if gas.BitLen() > 64 {
		return math.MaxUint64
	}
	return gas.Uint64()
}

func (c *bigModExp) Run(input []byte) ([]byte, error) {
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32)).Uint64()
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32)).Uint64()
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32)).Uint64()
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Handle a special case when both the base and mod length is zero
	if baseLen == 0 && modLen == 0 {
		return []byte{}, nil
	}
	// Retrieve the operands and execute the exponentiation
	var (
		base = new(big.Int).SetBytes(getData(input, 0, baseLen))
		exp  = new(big.Int).SetBytes(getData(input, baseLen, expLen))
		mod  = new(big.Int).SetBytes(getData(input, baseLen+expLen, modLen))
	)
	if mod.BitLen() == 0 {
		// Modulo 0 is undefined, return zero
		return common.LeftPadBytes([]byte{}, int(modLen)), nil
	}
	return common.LeftPadBytes(base.Exp(base, exp, mod).Bytes(), int(modLen)), nil
}
```

---

## [EIP 211](http://eips.ethereum.org/EIPS/eip-211) Support for variable length return values

[Replaces EIP 5](http://eips.ethereum.org/EIPS/eip-5)

`core/vm/jump_table.go` line 123
```go
instructionSet[RETURNDATASIZE] = operation{
  execute:     opReturnDataSize,
  constantGas: GasQuickStep,
  minStack:    minStack(0, 1),
  maxStack:    maxStack(0, 1),
  valid:       true,
}
instructionSet[RETURNDATACOPY] = operation{
  execute:    opReturnDataCopy,
  dynamicGas: gasReturnDataCopy,
  minStack:   minStack(3, 0),
  maxStack:   maxStack(3, 0),
  memorySize: memoryReturnDataCopy,
  valid:      true,
}
```

**RETURNDATASIZE:**

`core/vm/instructions.go` line 455

```go
func opReturnDataSize(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	stack.push(interpreter.intPool.get().SetUint64(uint64(len(interpreter.returnData))))
	return nil, nil
}
```

`core/vm/gas.go` line 27  

```go
GasQuickStep   uint64 = 2
```

**RETURNDATACOPY:**

`core/vm/instructions.go` line 460

```go
func opReturnDataCopy(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	var (
		memOffset  = stack.pop()
		dataOffset = stack.pop()
		length     = stack.pop()

		end = interpreter.intPool.get().Add(dataOffset, length)
	)
	defer interpreter.intPool.put(memOffset, dataOffset, length, end)

	if !end.IsUint64() || uint64(len(interpreter.returnData)) < end.Uint64() {
		return nil, errReturnDataOutOfBounds
	}
	memory.Set(memOffset.Uint64(), length.Uint64(), interpreter.returnData[dataOffset.Uint64():end.Uint64()])

	return nil, nil
}
```

`core/vm/gas_table.go` line 86

```go
func gasReturnDataCopy(gt params.GasTable, evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = math.SafeAdd(gas, GasFastestStep); overflow {
		return 0, errGasUintOverflow
	}

	words, overflow := bigUint64(stack.Back(2))
	if overflow {
		return 0, errGasUintOverflow
	}

	if words, overflow = math.SafeMul(toWordSize(words), params.CopyGas); overflow {
		return 0, errGasUintOverflow
	}

	if gas, overflow = math.SafeAdd(gas, words); overflow {
		return 0, errGasUintOverflow
	}
	return gas, nil
}
```

`core/vm/memory_table.go` line 27

```go
func memoryReturnDataCopy(stack *Stack) (uint64, bool) {
	return calcMemSize64(stack.Back(0), stack.Back(2))
}
```

---

## [EIP 214](http://eips.ethereum.org/EIPS/eip-214) Addition of the 'STATICCALL' opcode, permitting non-state-changing calls to other contracts

- STATICCALL opcode at: 0xfa

- equivalent to CALL but with only 6 arguments (omitting "value" argument)

- Reset value of flag after call returns

- Throw exception to state changing exceptions when STATIC set to true (CALLCODE not included)

- Backwards compatible because checks will only be made if set to TRUE

`core/vm/jump_table.go` line 114

```go
instructionSet[STATICCALL] = operation{
  execute:    opStaticCall,
  dynamicGas: gasStaticCall,
  minStack:   minStack(6, 1),
  maxStack:   maxStack(6, 1),
  memorySize: memoryStaticCall,
  valid:      true,
  returns:    true,
}
```

`core/vm/evm.go` line 318

```go
// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	// We do an AddBalance of zero here, just in order to trigger a touch.
	// This doesn't matter on Mainnet, where all empties are gone at the time of Byzantium,
	// but is the correct thing to do and matters on other networks, in tests, and potential
	// future scenarios
	evm.StateDB.AddBalance(addr, bigZero)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for code storage gas errors.
	ret, err = run(evm, contract, input, true)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}
```

`core/vm/instructions.go` line 834

```go
func opStaticCall(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// Pop gas. The actual gas is in interpreter.evm.callGasTemp.
	interpreter.intPool.put(stack.pop())
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	addr, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()
	toAddr := common.BigToAddress(addr)
	// Get arguments from the memory.
	args := memory.Get(inOffset.Int64(), inSize.Int64())

	ret, returnGas, err := interpreter.evm.StaticCall(contract, toAddr, args, gas)
	if err != nil {
		stack.push(interpreter.intPool.getZero())
	} else {
		stack.push(interpreter.intPool.get().SetUint64(1))
	}
	if err == nil || err == errExecutionReverted {
		memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	contract.Gas += returnGas

	interpreter.intPool.put(addr, inOffset, inSize, retOffset, retSize)
	return ret, nil
}
```

**js tracer files:** (STATICCALL tracers follow same rules as DELEGATECALL)

`eth/tracers/internal/tracers/4byte_tracer.js` line 42

```js
case "DELEGATECALL": case "STATICCALL":
  // gas, addr, memin, meminsz, memout, memoutsz
  return 2; // stack ptr to memin
}
```

`eth/tracers/internal/tracers/call_tracer.js` line 67

```js
// If a new method invocation is being done, add to the call stack
if (
  syscall &&
  (op == "CALL" ||
    op == "CALLCODE" ||
    op == "DELEGATECALL" ||
    op == "STATICCALL")
) {
  // Skip any pre-compile invocations, those are just fancy opcodes
  var to = toAddress(log.stack.peek(1).toString(16));
  if (isPrecompiled(to)) {
    return;
  }
  var off = op == "DELEGATECALL" || op == "STATICCALL" ? 0 : 1;

  var inOff = log.stack.peek(2 + off).valueOf();
  var inEnd = inOff + log.stack.peek(3 + off).valueOf();

  // Assemble the internal call report and store for completion
  var call = {
    type: op,
    from: toHex(log.contract.getAddress()),
    to: toHex(to),
    input: toHex(log.memory.slice(inOff, inEnd)),
    gasIn: log.getGas(),
    gasCost: log.getCost(),
    outOff: log.stack.peek(4 + off).valueOf(),
    outLen: log.stack.peek(5 + off).valueOf()
  };
  if (op != "DELEGATECALL" && op != "STATICCALL") {
    call.value = "0x" + log.stack.peek(2).toString(16);
  }
  this.callstack.push(call);
  this.descended = true;
  return;
}
```

`eth/tracers/internal/tracers/prestate_tracer.js` line 67

```js
case "CALL": case "CALLCODE": case "DELEGATECALL": case "STATICCALL":
  this.lookupAccount(toAddress(log.stack.peek(1).toString(16)), db);
  break;
```

---

## [EIP 100](http://eips.ethereum.org/EIPS/eip-100) Changes to the difficulty adjustment formula to take uncles into account

[article outlining the flaw](https://bitslog.wordpress.com/2016/04/28/uncle-mining-an-ethereum-consensus-protocol-flaw/)

- Adjusting formula for computing difficulty of a block

`old`

```python
adj_factor = max(1 - ((timestamp - parent.timestamp) // 10), -99)

child_diff = int(max(parent.difficulty + (parent.difficulty // BLOCK_DIFF_FACTOR) * adj_factor, min(parent.difficulty, MIN_DIFF)))
...
```

`new`

```python
adj_factor = max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99)
```

`consensus/ethash/consensus.go` line 337

```go
// makeDifficultyCalculator creates a difficultyCalculator with the given bomb-delay.
// the difficulty is calculated with Byzantium rules, which differs from Homestead in
// how uncles affect the calculation
func makeDifficultyCalculator(bombDelay *big.Int) func(time uint64, parent *types.Header) *big.Int {
	// Note, the calculations below looks at the parent number, which is 1 below
	// the block number. Thus we remove one from the delay given
	bombDelayFromParent := new(big.Int).Sub(bombDelay, big1)
	return func(time uint64, parent *types.Header) *big.Int {
		// https://github.com/ethereum/EIPs/issues/100.
		// algorithm:
		// diff = (parent_diff +
		//         (parent_diff / 2048 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99))
		//        ) + 2^(periodCount - 2)

		bigTime := new(big.Int).SetUint64(time)
		bigParentTime := new(big.Int).Set(parent.Time)

		// holds intermediate values to make the algo easier to read & audit
		x := new(big.Int)
		y := new(big.Int)

		// (2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9
		x.Sub(bigTime, bigParentTime)
		x.Div(x, big9)
		if parent.UncleHash == types.EmptyUncleHash {
			x.Sub(big1, x)
		} else {
			x.Sub(big2, x)
		}
		// max((2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9, -99)
		if x.Cmp(bigMinus99) < 0 {
			x.Set(bigMinus99)
		}
		// parent_diff + (parent_diff / 2048 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99))
		y.Div(parent.Difficulty, params.DifficultyBoundDivisor)
		x.Mul(y, x)
		x.Add(parent.Difficulty, x)

		// minimum difficulty can ever be (before exponential factor)
		if x.Cmp(params.MinimumDifficulty) < 0 {
			x.Set(params.MinimumDifficulty)
		}
		// calculate a fake block number for the ice-age delay
		// Specification: https://eips.ethereum.org/EIPS/eip-1234
		fakeBlockNumber := new(big.Int)
		if parent.Number.Cmp(bombDelayFromParent) >= 0 {
			fakeBlockNumber = fakeBlockNumber.Sub(parent.Number, bombDelayFromParent)
		}
		// for the exponential factor
		periodCount := fakeBlockNumber
		periodCount.Div(periodCount, expDiffPeriod)

		// the exponential factor, commonly referred to as "the bomb"
		// diff = diff + 2^(periodCount - 2)
		if periodCount.Cmp(big1) > 0 {
			y.Sub(periodCount, big2)
			y.Exp(big2, y, nil)
			x.Add(x, y)
		}
		return x
	}
}
```
