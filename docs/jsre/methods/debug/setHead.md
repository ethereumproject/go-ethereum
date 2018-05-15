
#### debug.setHead

debug.setHead(blockNumber)

**Sets** the current head of the blockchain to the block referred to by _blockNumber_.
See [web3.eth.getBlock](https://github.com/ethereumproject/wiki/wiki/JavaScript-API#web3ethgetblock) for more details on block fields and lookup by number or hash.

##### Returns

`true` on success, otherwise `false`.

##### Example

debug.setHead(eth.blockNumber-1000)
