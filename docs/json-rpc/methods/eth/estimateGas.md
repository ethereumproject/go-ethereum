
#### eth_estimateGas

Makes a call or transaction, which won't be added to the blockchain and returns the used gas, which can be used for estimating the used gas.

##### Parameters

See [eth_call](#eth-call) parameters, expect that all properties are optional.

##### Returns

`QUANTITY` - the amount of gas used.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_estimateGas","params":[{see above}],"id":1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": "0x5208" // 21000
}
```
