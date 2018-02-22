
#### eth_sign

Signs data with a given address.

The sign method calculates an Ethereum specific signature with: sign(keccak256("\x19Ethereum Signed Message:\n" + len(message) + message))).

By adding a prefix to the message makes the calculated signature recognisable as an Ethereum specific signature. This prevents misuse where a malicious DApp can sign arbitrary data (e.g. transaction) and use the signature to impersonate the victim.

**Note** the address to sign must be unlocked.

##### Parameters

1. `DATA`, 20 Bytes - address
2. `DATA`, Data to sign

##### Returns

`DATA`: Signed data

##### Example

```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sign","params":["0xd1ade25ccd3d550a7eb532ac759cac7be09c2719", "Schoolbus"],"id":1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": "0x2ac19db245478a06032e69cdbd2b54e648b78431d0a47bd1fbab18f79f820ba407466e37adbe9e84541cab97ab7d290f4a64a5825c876d22109f3bf813254e8601"
}
```
