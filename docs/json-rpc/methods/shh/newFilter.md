
#### shh_newFilter

Creates filter to notify, when client receives whisper message matching the filter options.


##### Parameters

1. `Object` - The filter options:
- `to`: `DATA`, 60 Bytes - (optional) Identity of the receiver. *When present it will try to decrypt any incoming message if the client holds the private key to this identity.*
- `topics`: `Array of DATA` - Array of `DATA` topics which the incoming message's topics should match.  You can use the following combinations:
- `[A, B] = A && B`
- `[A, [B, C]] = A && (B || C)`
- `[null, A, B] = ANYTHING && A && B` `null` works as a wildcard

```js
params: [{
"topics": ['0x12341234bf4b564f'],
"to": "0x04f96a5e25610293e42a73908e93ccc8c4d4dc0edcfa9fa872f50cb214e08ebf61a03e245533f97284d442460f2998cd41858798ddfd4d661997d3940272b717b1"
}]
```

##### Returns

`QUANTITY` - The newly created filter.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_newFilter","params":[{"topics": ['0x12341234bf4b564f'],"to": "0x2341234bf4b2341234bf4b564f..."}],"id":73}'

// Result
{
"id":1,
"jsonrpc":"2.0",
"result": "0x7" // 7
}
```
