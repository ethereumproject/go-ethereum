
#### admin.startRPC

admin.startRPC(host, portNumber, corsheader, modules)

Starts the HTTP server for the [JSON-RPC](https://github.com/ethereumproject/wiki/wiki/JSON-RPC).

##### Returns

`true` on success, otherwise `false`.

##### Example

```javascript
admin.startRPC("127.0.0.1", 8545, "*", "web3,net,eth")
// true
```
