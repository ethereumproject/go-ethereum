
#### shh_version

Returns the current whisper protocol version.

##### Parameters
none

##### Returns

`String` - The current whisper protocol version

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}'

// Result
{
"id":67,
"jsonrpc": "2.0",
"result": "2"
}
```
