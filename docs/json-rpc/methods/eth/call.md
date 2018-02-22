
#### eth_call

Executes a new message call immediately without creating a transaction on the block chain.


##### Parameters

1. `Object` - The transaction call object
- `from`: `DATA`, 20 Bytes - (optional) The address the transaction is send from.
- `to`: `DATA`, 20 Bytes  - The address the transaction is directed to.
- `gas`: `QUANTITY`  - (optional) Integer of the gas provided for the transaction execution. eth_call consumes zero gas, but this parameter may be needed by some executions.
- `gasPrice`: `QUANTITY`  - (optional) Integer of the gasPrice used for each paid gas
- `value`: `QUANTITY`  - (optional) Integer of the value send with this transaction
- `data`: `DATA`  - (optional) The compiled code of a contract
2. `QUANTITY|TAG` - integer block number, or the string `"latest"`, `"earliest"` or `"pending"`, see the [default block parameter](#the-default-block-parameter)

##### Returns

`DATA` - the return value of executed contract.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_call","params":[{see above}],"id":1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": "0x0"
}
```
