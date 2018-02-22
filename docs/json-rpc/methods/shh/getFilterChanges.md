
#### shh_getFilterChanges

Polling method for whisper filters. Returns new messages since the last call of this method.

**Note** calling the [shh_getMessages](#shh_getmessages) method, will reset the buffer for this method, so that you won't receive duplicate messages.


##### Parameters

1. `QUANTITY` - The filter id.

```js
params: [
"0x7" // 7
]
```

##### Returns

`Array` - Array of messages received since last poll:

- `hash`: `DATA`, 32 Bytes (?) - The hash of the message.
- `from`: `DATA`, 60 Bytes - The sender of the message, if a sender was specified.
- `to`: `DATA`, 60 Bytes - The receiver of the message, if a receiver was specified.
- `expiry`: `QUANTITY` - Integer of the time in seconds when this message should expire (?).
- `ttl`: `QUANTITY` -  Integer of the time the message should float in the system in seconds (?).
- `sent`: `QUANTITY` -  Integer of the unix timestamp when the message was sent.
- `topics`: `Array of DATA` - Array of `DATA` topics the message contained.
- `payload`: `DATA` - The payload of the message.
- `workProved`: `QUANTITY` - Integer of the work this message required before it was send (?).

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_getFilterChanges","params":["0x7"],"id":73}'

// Result
{
"id":1,
"jsonrpc":"2.0",
"result": [{
"hash": "0x33eb2da77bf3527e28f8bf493650b1879b08c4f2a362beae4ba2f71bafcd91f9",
"from": "0x3ec052fc33..",
"to": "0x87gdf76g8d7fgdfg...",
"expiry": "0x54caa50a", // 1422566666
"sent": "0x54ca9ea2", // 1422565026
"ttl": "0x64" // 100
"topics": ["0x6578616d"],
"payload": "0x7b2274797065223a226d657373616765222c2263686...",
"workProved": "0x0"
}]
}
```
