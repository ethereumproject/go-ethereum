## JSON-RPC support

| | cpp-ethereum | go-ethereum | py-ethereum|
|-------|:------------:|:-----------:|:-----------:|
| JSON-RPC 1.0 | &#x2713; | | |
| JSON-RPC 2.0 | &#x2713; | &#x2713; | &#x2713; |
| Batch requests | &#x2713; |  &#x2713; |  &#x2713; |
| HTTP | &#x2713; | &#x2713; | &#x2713; |

### Go

You can start the HTTP JSON-RPC with the `--rpc` flag
```bash
geth --rpc
```

change the default port (8545) and listing address (localhost) with:

```bash
geth --rpc --rpc-addr <ip> --rpc-port <portnumber>
```

If accessing the RPC from a browser, CORS will need to be enabled with the appropriate domain set. Otherwise, JavaScript calls are limit by the same-origin policy and requests will fail:

```bash
geth --rpc --rpc-cors-domain "http://localhost:3000"
```

The JSON RPC can also be started from the [geth console](https://github.com/ethereumproject/go-ethereum/wiki/JavaScript-Console) using the `admin.startRPC(addr, port)` command.

Not all RPC API's are enabled by default. To enable specific APIs, use the `--rpc-api` flag:

```bash
geth --rpc --rpc-api="admin,debug,eth,miner"
```

For a list of available APIs, use `geth help | grep rpc-api`.


### C++

You can start it by running `eth` application with `-j` option:
```bash
./eth -j
```

You can also specify JSON-RPC port (default is 8545):
```bash
./eth -j --json-rpc-port 8079
```

### Python
In python the JSONRPC server is currently started by default and listens on `127.0.0.1:4000`

You can change the port and listen address by giving a config option.

`pyethapp -c jsonrpc.listen_port=4002 -c jsonrpc.listen_host=127.0.0.2 run`