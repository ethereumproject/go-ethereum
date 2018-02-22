
#### db_putHex

Stores binary data in the local database.

**Note** this function is deprecated and will be removed in the future.


##### Parameters

1. `String` - Database name.
2. `String` - Key name.
3. `DATA` - The data to store.

```js
params: [
"testDB",
"myKey",
"0x68656c6c6f20776f726c64"
]
```

##### Returns

`Boolean` - returns `true` if the value was stored, otherwise `false`.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"db_putHex","params":["testDB","myKey","0x68656c6c6f20776f726c64"],"id":73}'

// Result
{
"id":1,
"jsonrpc":"2.0",
"result": true
}
```
