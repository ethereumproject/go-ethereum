
#### eth_protocolVersion

Returns the current ethereum protocol version.

##### Parameters
none

##### Returns

`String` - The current ethereum protocol version

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_protocolVersion","params":[],"id":67}'

// Result
{
"id":67,
"jsonrpc": "2.0",
"result": "54"
}
```
