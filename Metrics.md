# Orbs Network Metrics Overview

This overview provides a list of Orbs network node metrics that we deem significant because they provide valuable insights into node and virtual chain status.
We expect and encourage the Orbs community to develop multiple monitoring tools for the nodes, that can serve Validators, Guardians, and app developers. The following metrics might help with this task.

Our goal is to recommend metrics for 4 types of monitoring:
1. Single node monitoring - mostly relevant for Validators to monitor their own node
1. Virtual chain monitoring - mostly relevant for applications to monitor the performance of the virtual chain they paid for
1. Network monitoring - mostly relevant for Guardians to monitor the performance of the entire network
1. PoS monitoring - relevant for all parties interested in network status (this type is not discussed in this document)

## How to retrieve the metrics

All metrics are available in JSON and Prometheus format:
* JSON at `http://$NODE_IP/vchains/$CHAIN_ID/metrics.json` (and `http://$NODE_IP/vchains/$CHAIN_ID/metrics`)
* Prometheus at `http://$NODE_IP/vchains/$CHAIN_ID/metrics.prometheus`

You can see live examples from Orbs validator node [here](http://validator.orbs.com/vchains/1100000/metrics) and [here](http://validator.orbs.com/vchains/1100000/metrics.prometheus)

Please note, that **all metrics are available per virtual chain**, and when we are talking about the network we mean any single virtual chain and not all virtual chains.

## Metrics
### Block height
Retrieved from:
* `BlockStorage.BlockHeight` (since `v1.0.0`)
* `BlockSync.ProcessingBlocksState.LastCommitted.TimeNano` (since `v1.0.0`)

Meaning:
* Indicates liveness of the system
* Indicates if the node is in sync

Relevant values:
* Block height should advance at least once in 30s period. Does vary greatly depending on traffic, with constant traffic the blocks are closed as soon as new transactions arrive.
* Last committed time should be no farther than 30 minutes in the past or the node will be considered out of sync.

Useful to app developers and node operators.

### Time spent in queue by transaction
Retrieved from:
* `TransactionPool.PendingPool.TimeSpentInQueue.Millis` (since `v1.0.0`)

Meaning and relevant values:
* Indicates liveness of the system and response time. The shorter the better. TBD: clear range here (Good, Acceptable, Alert)

Useful to app developers and node operators.

### Transactions per second (block storage)
Retrieved from:
* `TransactionPool.CommitRate` (since `v2.0.1`) (was `TransactionPool.CommitRate.PerSecond` since `v1.0.0`)

Meaning:
* Amount of transactions from committed blocks that were moved from pending pool to committed pool.

Useful to app developers and node operators.

### Transactions per second (public API)
Retrieved from:
* `PublicA[i].Transactions` (since `v2.0.1`) (was `PublicAPI.Transactions.PerSecond` since `v1.1.0`)

Meaning:
* Indicates the number of transactions received by virtual chain public API.

Useful to app developers and node operators.

### Queries per second (public API)
Retrieved from:
* `PublicApi.Queries` (since `v2.0.1`) (was `PublicAPI.Queries.PerSecond` since `v1.1.0`)

Meaning:
* Indicates the number of queries received by virtual chain public API. Queries differ from transactions because they only use read-only methods of the smart contract and does not alter the state, therefore the node can serve many more queries than it can process transactions.

Useful to app developers and node operators.

### Ethereum
Retrieved from:
* `Ethereum.Node.LastBlock` (since `v1.0.0`)
* `Ethereum.Node.Sync.Status` (since `v1.0.0`)
* `Ethereum.Node.TransactionReceipts.Status` (since `v1.0.0`)

Meaning:
* Indicates whether Ethereum node operates properly.

Relevant values:
* `success` for sync status and transaction receipts status (anything should be treated as an error)
* A number for last block

Useful to node operators.

### Gossip
Retrieved from:
* `Gossip.IncomingConnection.Active.Count` (since `v1.0.0`)

Meaning:
* Indicates connectivity with network peers.

Relevant values:
* Equals (N-1) when N is the number of nodes in the network.

Useful to node operators.

### Resources metrics
Retrieved from:
* `OS.Process.CPU.Percent` (since `v2.0.1`) (was `OS.Process.CPU.PerCent` since `v1.0.0`)
* `OS.Process.Memory.Bytes` (since `v1.0.0`)
* `BlockStorage.FileSystemSize.Bytes` (since `v1.0.0`)

Meaning:
* Indicates the amount of CPU the virtual chain process consumes.
* Indicates the amount of RAM the virtual chain process consumes.
* Indicates the amount of storage  the virtual chain process consumes.

Relevant values:
* For CPU: below 50% - green, below 70% yellow, above - red
* For RAM: below 1Gb - green, below 2Gb yellow, above - red
* For disk: below 50Gb - green, below 70Gb yellow, above - red

Useful to node operators.

### Binary version
Retrieved from:
* `Version.Commit` (since `v1.0.0`)
* `Version.Semantic` (since `v1.0.0`)

Meaning:
* Git commit
* semantic version of the binary.

Relevant values:
* Specific to the tag on Github that corresponds with semantic version

Useful to node operators
