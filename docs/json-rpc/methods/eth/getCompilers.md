
#### eth_getCompilers

Returns a list of available compilers in the client.

##### Parameters
none

##### Returns

`Array` - Array of available compilers.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getCompilers","params":[],"id":1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": ["solidity", "lll", "serpent"]
}
```
