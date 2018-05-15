
#### miner.makeDAG

miner.makeDAG(blockNumber, dir)

Generates the DAG for epoch `blockNumber/epochLength`. dir specifies a target directory,
If `dir` is the empty string, then ethash will use the default directories `~/.ethash` on Linux and MacOS, and `~\AppData\Ethash` on Windows. The DAG file's name is `full-<revision-number>R-<seedhash>`

##### Returns

`true` on success, otherwise `false`.
