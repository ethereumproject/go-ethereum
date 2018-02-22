
#### shh_post

Sends a whisper message.

##### Parameters

1. `Object` - The whisper post object:
- `from`: `DATA`, 60 Bytes - (optional) The identity of the sender.
- `to`: `DATA`, 60 Bytes - (optional) The identity of the receiver. When present whisper will encrypt the message so that only the receiver can decrypt it.
- `topics`: `Array of DATA` - Array of `DATA` topics, for the receiver to identify messages.
- `payload`: `DATA` - The payload of the message.
- `priority`: `QUANTITY` - The integer of the priority in a rang from ... (?).
- `ttl`: `QUANTITY` - integer of the time to live in seconds.

```js
params: [{
from: "0x04f96a5e25610293e42a73908e93ccc8c4d4dc0edcfa9fa872f50cb214e08ebf61a03e245533f97284d442460f2998cd41858798ddfd4d661997d3940272b717b1",
to: "0x3e245533f97284d442460f2998cd41858798ddf04f96a5e25610293e42a73908e93ccc8c4d4dc0edcfa9fa872f50cb214e08ebf61a0d4d661997d3940272b717b1",
topics: ["0x776869737065722d636861742d636c69656e74", "0x4d5a695276454c39425154466b61693532"],
payload: "0x7b2274797065223a226d6",
priority: "0x64",
ttl: "0x64",
}]
```

##### Returns

`Boolean` - returns `true` if the message was send, otherwise `false`.


##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_post","params":[{"from":"0xc931d93e97ab07fe42d923478ba2465f2..","topics": ["0x68656c6c6f20776f726c64"],"payload":"0x68656c6c6f20776f726c64","ttl":0x64,"priority":0x64}],"id":73}'

// Result
{
"id":1,
"jsonrpc":"2.0",
"result": true
}
```
