## Integration
These additional API's follow the same conventions as the official DApp API. Web3 can be [extended](https://github.com/ethereumproject/web3.js/pull/229) and used to consume these additional API's.

The different functions are split into multiple smaller logically grouped API's. Examples are given for the [Javascript console](./JavaScript-Console) but can easily be converted to a rpc request.

**2 examples:**

* Console: `miner.start()`

* IPC: `echo '{"jsonrpc":"2.0","method":"miner_start","params":[],"id":1}' | socat -,ignoreeof  UNIX-CONNECT:$HOME/.ethereum/geth.ipc`

* RPC: `curl -X POST --data '{"jsonrpc":"2.0","method":"miner_start","params":[],"id":74}' localhost:8545`

With the number of THREADS as an arguments:

* Console: `miner.start(4)`

* IPC: `echo '{"jsonrpc":"2.0","method":"miner_start","params":["0x04"],"id":1}' | socat -,ignoreeof  UNIX-CONNECT:$HOME/.ethereum/geth.ipc`

* RPC: `curl -X POST --data '{"jsonrpc":"2.0","method":"miner_start","params":["0x04"],"id":74}' localhost:8545`