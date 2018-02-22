
#### eth_compileLLL

Returns compiled LLL code.

##### Parameters

1. `String` - The source code.

```js
params: [
"(returnlll (suicide (caller)))",
]
```

##### Returns

`DATA` - The compiled source code.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_compileSolidity","params":["(returnlll (suicide (caller)))"],"id":1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": "0x603880600c6000396000f3006001600060e060020a600035048063c6888fa114601857005b6021600435602b565b8060005260206000f35b600081600702905091905056" // the compiled source code
}
```
