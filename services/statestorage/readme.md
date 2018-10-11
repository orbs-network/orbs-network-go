# State storage working assumptions

### State Entry key encoding
1. Each state entry is addressed by a key which is a hash of the contract name 
appended by a hash of the variable name 
(contract programming model will determine the logical name of each variable): 
`hash(contract)+hash(variable_unique_id)`
1. As a side effect of #1 we assume every key has a known length. This limitation manifests in the way we serialize nodes in that we do not permit storing values on branch nodes. This is made possible only due to the stipulation that all keys are of the same height. Should we ever want to relax this requirement and allow paths of different lengths we would have to change the serialization scheme as it would require allowing two different paths where one is a prefix of the other, which is not possible to represent in the current serialization.
1. A value key must be a byte array: when represented in base64 no odd number of digits is allowed (because of merkle parity requirements).

#### TBD
1. Should we include virtual chain ID in the contract name before or after hashing? compare with V0 addressing... 
1. What hash function should be used? Should we use two hash functions? What is the length of a key hash?
1. Should we include type information in each entry?

### Startup time 
A node must reboot in a constant time, and then sync with other nodes in a time complexity that is at most O(n) where n are the number of blocks finalized since it last synced, roughly
* For first time installation a node may sync in a time complexity of O(h) where h is the current block height

##### Size estimates
Size of merkle tree structure must be benchmarked due to structure complexity. While we may assume that the number of entries in a persistent key/value store may not change in an order of magnitude (when appending the state with a merkle trie), the extensive use of hash codes may make the actual memory usage significantly larger than the state alone.

State of a 1 Million user contract holding ~500 bytes on average for each user is roughly 0.5GB

While its reasoneable to assume a single machine can hold this load, This may be doubled for each contract in the system
if multiple dApps are employed. and more importantly, bugs and human error outside of our control may drive the number
up drastically in a very short period of time essentially causing the system to halt until the contaminated blocks are
manually removed from block storage.

### Persistence
optimal performance is achieved when the entire state and Merkle trie are stored in memory.

Persistent storage may be used to satisfy two requirements:
1. Scale beyond the RAM capacity of a node (see [merkle](merkle/readme.md))
1. Using regular interval snapshots we can ensure [Startup time]() requirement 

### A Single full State snapshot with subsequent diffs in memory
Regardless if the main state snapshot is maintained in persistent storage or in memory, the following mechanism 
can be used to track the state of a few (5) consecutive block heights:
 
A single full state snapshot is maintained at all times, of the least recent block height supported for queries. The block height of the full-state snapshot is constantly maintained and updated (“full-state snapshot block height”). 

1. As new blocks are committed perform under write lock:
    1. Record the state diff of incoming new block and associate it with the corresponding block height
    1. If the full-state snapshot block height becomes less than the most recent known block height minus 5 (block height - 5) - advance the full-state snapshot block height and apply it’s successive block height’s state diff to the image. 
1. As state queries are received for specific block heights perform under read lock:
    1. Set selected block height to the query requested block height
    1. If selected block height > full-state snapshot block height:
        1. Search for requested key in the selected block’s recorded state diff image. If none exists return an error.
        1. If found return the value
        1. If not found, decrement selected block height and return to b 
    1. If selected block height == full-state snapshot block height
        1. Return the value found in full state snapshot
    1. If selected block height < full-state snapshot block height
        1. Return an error

* State diffs must include zero values, But the full-state snapshot must not include zero values 
* If persistence is implemented we: 
    1. persist the full-state snapshot in a key/value store 
    1. Cache must be applied at the DB level
    1. Full-state snapshot block height and state diffs updates must be written together atomically - when applied incrementally to a previous height state snapshot database.

### Possible DB Choices:

1. LevelDB
1. RocksDB - an alternative/enhanced fork of LevelDB
1. Redis - Since don't care about lags in persistence we may benefit from using Redis as an in memory with full state dumps in the background.

There may be 

### Single state database
It stands to reason to keep a single database for one block height of a given virtual blockchain.

But it's possible to break down the physical representations to the contract level for easy cleansing of contracts. 
This is also true about Merkle trie implementations but currently for simplicity we assume that we always only need one 
database for the merkle trie and one for state entries. And the minimum requirement is that each
database will contain a single state image belonging to a single block height.

This is possible because subsequent blocks are quick to sync from block storage and we never maintain state for far behind block heights.

