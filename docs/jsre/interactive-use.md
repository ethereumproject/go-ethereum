## Interactive use: the JSRE REPL  Console

The `ethereum CLI` executable `geth` has a JavaScript console (a **Read, Evaluate & Print Loop** = REPL exposing the JSRE), which can be started with the `console` or `attach` subcommand. The `console` subcommands starts the geth node and then opens the console. The `attach` subcommand will not start the geth node but instead tries to open the console on a running geth instance.

```shell
$ geth 2>>out.log console # redirect logs to a file and begin interactive console session
```

```shell
$ geth # terminal 1
$ geth attach # terminal 2
```

The attach node accepts an endpoint in case the geth node is running with a non default ipc endpoint or you would like to connect over the rpc interface.

    $ geth attach ipc:/some/custom/path
    $ geth attach http://191.168.1.1:8545
    $ geth attach ws://191.168.1.1:8546

Note that by default the geth node doesn't start the http and weboscket service and not all functionality is provided over these interfaces due to security reasons. These defaults can be overridden when the `--rpcapi` and `--wsapi` arguments when the geth node is started, or with [admin.startRPC](admin_startRPC) and [admin.startWS](admin_startWS).

If you need log information, start with:

    $ geth --verbosity 5 console 2>> /tmp/eth.log

Otherwise mute your logs, so that it does not pollute your console:

    $ geth console 2>> /dev/null

or

    $ geth --verbosity 0 console

Geth has support to load custom JavaScript files into the console through the `--preload` argument. This can be used to load often used functions, setup web3 contract objects, or ...
```
geth --preload "/my/scripts/folder/utils.js,/my/scripts/folder/contracts.js" console
```

