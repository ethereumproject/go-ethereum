
#### eth_getWork

Returns the hash of the current block, the seedHash, and the boundary condition to be met ("target").

##### Parameters
none

##### Returns

`Array` - Array with the following properties:
1. `DATA`, 32 Bytes - current block header pow-hash
2. `DATA`, 32 Bytes - the seed hash used for the DAG.
3. `DATA`, 32 Bytes - the boundary condition ("target"), 2^256 / difficulty.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getWork","params":[],"id":73}'

// Result
{
"id":1,
"jsonrpc":"2.0",
"result": [
"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
"0x5EED00000000000000000000000000005EED0000000000000000000000000000",
"0xd1ff1c01710000000000000000000000d1ff1c01710000000000000000000000"
]
}
```
