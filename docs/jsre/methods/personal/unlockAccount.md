

#### personal.unlockAccount
personal.unlockAccount(addr, passwd, duration)

Unlock the account with the given address, password and an optional duration (in seconds). If password is not given you will be prompted for it.

#### Return
`boolean` indication if the account was unlocked

#### Example
` personal.unlockAccount(eth.coinbase, "mypasswd", 300)`
