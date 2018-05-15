# How to
It is possible to specify the set of API's which are offered over an interface with the `--${interface}api` command line argument for the go ethereum daemon. Where `${interface}` can be `rpc` for the `http` interface or `ipc` for an unix socket on unix or named pipe on Windows.

For example, `geth --ipcapi "admin,eth,miner" --rpcapi "eth,web3"` will
* enable the admin, official DApp and miner API over the IPC interface
* enable the eth and web3 API over the RPC interface

Please note that offering an API over the `rpc` interface will give everyone access to the API who can access this interface (e.g. DApp's). So be careful which API's you enable. By default geth enables all API's over the `ipc` interface and only the eth,net and web3 API's over the `rpc` and `ws` interface.

To determine which API's an interface provides the `modules` transaction can be used, e.g. over an `ipc` interface on unix systems:

```
echo '{"jsonrpc":"2.0","method":"rpc_modules","params":[],"id":1}' | socat -,ignoreeof  UNIX-CONNECT:$HOME/.ethereum/geth.ipc
```
will give all enabled modules including the version number:
```
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": {
    "admin": "1.0",
    "debug": "1.0",
    "eth": "1.0",
    "miner": "1.0",
    "net": "1.0",
    "personal": "1.0",
    "rpc": "1.0",
    "txpool": "1.0",
    "web3": "1.0",
    "geth": "1.0"
  }
}

```