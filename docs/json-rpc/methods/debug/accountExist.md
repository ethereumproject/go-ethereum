
#### debug_accountExist

Returns whether a given account exists at a given block. Whether an account
exists affects the gas cost of a transaction.


##### Parameters

1. `String` - Account address.
2. `Uint64` - Block number.

```js
params: [
"address": "0x234adf3q4jalksdjfg...",
"number": 14,
]
```

##### Returns

`BOOL` - If the account exists.


##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"debug_accountExist","params":["address": ,"number": 14],"id":79}'

// Result
{
"id":1,
"jsonrpc":"2.0",
"result": true
}
```
