
#### geth.getAddressTransactions

Returns transactions for an address.

Usage requires address-transaction indexes using `geth --atxi` to enable and create indexes during chain sync/import, optionally  using `geth atxi-build` to index pre-existing chain data.


##### Parameters
1. `DATA`, 20 Bytes - address to check for transactions
2. `QUANTITY` - integer block number to filter transactions floor
3. `QUANTITY` - integer block number to filter transactions ceiling
4. `STRING` - `[t|f|tf|]`, use `t` for transactions _to_ the address, `f` for _from_, or `tf`/`''` for both
5. `STRING` - `[s|c|sc|]`, use `s` for _standard_ transactions, `c` for _contracts_, or `sc`/``''` for both
6. `QUANTITY` - integer of index to begin pagination. Using `-1` equivalent to `0`.
7. `QUANTITY` - integer of index to end pagination. Using `-1` equivalent to last transaction _n_.
8. `BOOL` - whether to return transactions in order of oldest first. By default `false` returns transaction hashes ordered by newest transactions first.

```
params: [
'0x407d73d8a49eeb85d32cf465507dd71d507100c1',
123, // earliest block
456, // latest block, use 0 for "undefined", ie. eth.blockNumber
't', // only transactions to this address
'', // both standard and contract transactions
-1, // do not trim transactions for pagination (start)
-1, // do not trim transactions for pagination (end)
false // do not reverse order ('true' will reverse order to be oldest first)
]
```

##### Returns

`Array` - Array of transaction hashes, or an empty array if no transactions found

##### Example
```js
> geth.getAddressTransactions("0xf4e6FeA8C10C05fe9E2C2FA7545e4c9dd3993a26", 0, 0, "tf", "sc", -1, -1, true)

["0x148809a063efc39e66e35a27a72a82747905071bc2c3b7fc12370dd979eac650", "0x5649e7346ed868bde9ef3a532f8140aeb4171392d278da8e030b26540e248f8a", "0x11bc379dd4f42db7bce759e89dbfda8420fa489e785b1989374f719dac1923dd", "0xdc983ced410b96a95d9a27c9ed88c20ddf735c45797237a34c8f103bbad30caa", "0x58532fa1492a77df622fac57e5e4853f417dcbc3c92940c9df0a2bde72b303f9", "0x223c8589024914b293b92b1fbda08636dd1d3a121fc75532aaa046d579ff641d", "0xd5332daa2e8cb8621912ada4ce09bb1ed8d5831844f8260c4bd07e39677f1201", "0x15bced756880910783272beafc644b9d755291e9fa643ae7c305c6cae961fa26", "0x099e6323f5f9a09197fa5032a546f0f0706b3b8e31404e297e80fcea89210ccd", "0xb5c8b065561e1ee144c2999786accdb5626c10b31c691e01b3c94f22380c0143", "0x743156d73d92595d6005124b6130e4ed85e52312af08aa1303e7ea53741c8cff", "0x38d4128c12c1c4b7a0aab8ab028f95860e2a5a5deb4cd9c992d6cea5f3c45c2b", "0xf87d4e67aa21fda8e749fbedf2e6a6f9bb499d9e4f94e2faae65b473718d6905", "0x4aa8bb43108488e247d52e57ae50ba115b5e95452b89aa4cee92458cb2c9e148", "0x13dc8baa1f4bc0076095e7d73a9aa22e049a30e064e7ea13b34d1498f108730c", "0x351f388bd8271feef0b3b81dcbc500b1f8a0b16064722fca76612a6d04e37378", "0x02ecebe4cc15179991c202b249f358bbc81c26613c66d16bdcc201a550557a7b", "0xd5a50c70909b9f494495449df6cd3e3f5621de41ff5ab4174b066b61468ddbcc"]
```
