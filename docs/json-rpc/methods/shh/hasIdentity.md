
#### shh_hasIdentity

Checks if the client hold the private keys for a given identity.


##### Parameters

1. `DATA`, 60 Bytes - The identity address to check.

```js
params: [
"0x04f96a5e25610293e42a73908e93ccc8c4d4dc0edcfa9fa872f50cb214e08ebf61a03e245533f97284d442460f2998cd41858798ddfd4d661997d3940272b717b1"
]
```

##### Returns

`Boolean` - returns `true` if the client holds the privatekey for that identity, otherwise `false`.


##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_hasIdentity","params":["0x04f96a5e25610293e42a73908e93ccc8c4d4dc0edcfa9fa872f50cb214e08ebf61a03e245533f97284d442460f2998cd41858798ddfd4d661997d3940272b717b1"],"id":73}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": true
}
```
