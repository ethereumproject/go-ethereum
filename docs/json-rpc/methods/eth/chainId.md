
### eth_chainId

Returns the currently configured chain id, a value used in replay-protected transaction
signing as introduced by EIP-155.

##### Parameters
none

##### Returns

`QUANTITY` - big integer of the current chain id. Defaults are mainnet=61, morden=62.

##### Example
```js
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'

// Result
{
"id":83,
"jsonrpc": "2.0",
"result": "0x3d" // 61
}
```
