# Merkle trie implementation notes

## Representation
A Merkle trie has two possible representations.
#### Location-Addressed nodes 
One is in memory where each node has a distinct memory address,
and the merkle hash is computed and stored inside the node as a hash label.

In this representation which we call location-addressed nodes, the node
is represented differently than it is when we construct a proof. 

In this representation we also maintain the notion of a tree structure as it is
possible for two identical nodes to exist independently at two memory addresses and be referenced
by other parent nodes, as is the case where two keys have the same suffix and value.  

This representation has an advantage of making trie manipulations simpler and it
leverages go built in  garbage collection almost seamlessly.

A disadvantage is that when we want to generate a proof we must serialize all nodes

#### Contect-Addressed nodes
In this representation we maintain nodes in a dictionary or key/value store where
each node is addressed by it's own hash code.

The advantages of this approach is in the fact that the data in the nodes is all 
that is really needed to construct and traverse the tree structure (and the root of each trie). 
The resulting collection of nodes closely resembles the nodes structure
we find in the proof. So we can rely on similar mechanisms for serialization to storing nodes.

This also makes nodes inherently immutable as any change in state results in copying 
of a node by allocating a new hash code address. This makes it easier to maintain
historical revisions of the tree in a single address space we call a merkle forest.

The main advantage of this approach also leads to some disadvantage when considering
garbage collection of old stale nodes. In other words, cleaning our forest from nodes which
are not part of any recent block state merkle trie, becomes more of a challenge in this representation,
as we need to manage references as nodes may be shared across tires as well as inside a trie.

#### Hybrid

As content addressing is more suitable for persistent storage and
location addressed approach in more useful when constructing in memory tries, it's possible to develop a hybrid or dual 
representation.

For a persistent storage layer we may benefit from the advantages of content addressed 
nodes in a persistent key / value store.

While for in memory structures (including partial tries which are loaded for the purpose of creating new
merkle roots and new tries for newly committed blocks) it may be better to retain 
a location-addressed representation of the trie or sub-tries.

There are two flavors for hybrid solutions - solutions where the entire forest 
is expected to fit in memory, and where it is not expected to fit in memory.
In the first one, we only store single block snapshots to speed up node init time
in case of system restart. In this case, the node will find the relatively recent 
state snapshot on disk, load it to memory when it loads and then playback all subsequent blocks
to reach an updated state on startup. This may still take some time but we assume loading
a snapshot to memory will be faster than replaying the entire block-chain history in memory

The second flavor is similar to the first one, only we don't load all the state
initially, but rather load nodes lazily - only when they are needed in the process of
appending new tries or when proofs are requested.
When it's time to write a new snapshot to the disk we have two options here
1. select a single block height from memory and write all nodes. Whenever we reach 
a node with a hash code that already exists on disk we stop and continue in another branch
of the tree. If the process fails in the middle we need to revert to the old snapshot.
1. Independently start writing a trie from a single generation - copying in essence most of the original
file. This may be more intensive in disk load and may take long time for large tries but
is may be a simple first stab at the problem.  

## Appending a trie with new state diffs

When we receive a new block we essentially want to create a new Merkle root and some new leaf nodes.
But the majority of the nodes in the new block trie should not change and should be reused and referenced directly in previous block's merkle tries.

The process of creating a new root with the minimal amount of node additions for K, a set of keys can be summarized:

1. Load all Merkle Proof nodes for K keys from the previous block's merkle trie. Create the tree structure from these nodes combined (translate to location-addressed node representation). 
1. Iterate over K keys and for each one make tree adjustments to reflect the newly set values. This process may introduce new nodes, and possibly delete some nodes.
1. During the manipulation of the new nodes we:
    * retain any parent/child relations until all values have been applied to the sandbox nodes. This is to prevent intermediate states from spawning unreferenced nodes. 
    * retain references to nodes outside the sandbox.
1. Once we have applied all state diffs we can scan the trie subset in the sandbox and correct all hash labels or references (depending on the addressing scheme as above)
1. The newly created nodes can be added to the main address space and the new merkle root returned to the caller

As a side effect of this process we may be able to produce a list of nodes which "disappeared" from the new trie by inferring that any node that was initially loaded into the sandbox and did not remain in it unchanged becomes a candidate for garbage collection once this generation expires and needs collecting.

## Garbage Collection
Before discussing garbage collection, it's important to stress that at this point it's not clear we need it in high priority.
However, it is expected that this will become an issue relatively quickly

the problem is discussed and addressed in the [Ethereum project](https://blog.ethereum.org/2015/06/26/state-tree-pruning/) project but 
its unclear if these recommendations [have been adopted](https://medium.com/codechain/ethereums-state-trie-pruning-45ea73ed2c78) on the matter.  

For example, if we decide to go with persistent storage there may not be strong reason to perform garbage collection at all.

Possible approaches are differing on the representation method used:
1. Content Addressing
    1. Concurrent mark sweep - There are several implementation options but the gist remains that during the mark phase we scan the forest from X most recent roots. During the mark phase any newly added nodes or merkle roots are also marked utilizing short lived locks.
    1. Managed nodes - In the process of appending a new trie to the forest as new blocks arrive [described above](Appending a trie) we can extract the list of nodes that are no longer being referenced by the new trie. If we record this list for each block height, 
    it can be used to later delete these nodes when the preceding block goes out of circulation. This approach adds some complexity to the code and requires traversing all existing "purge" lists to eradicate any mention of node hashes that have "returned" to a later 
    trie whenever we add new nodes in subsequent block additions.
1. Location Addressing
    1. In memory only - Rely on go garbage collector entirely. Loosing reference to the expired merkle root node should 
    lead 
    1. In memory with periodic snapshot dumps to disk
        1. If dumping the full image every time no need  to handle garbage collection in the persisted file
        1. If writing state increments to previously persisted snapshot we must either
            1. handle cleaning up nodes that are no longer in the current forest (how?)
            1. ignore this inefficiency and accept that the persistent image may have unused nodes (as in go-ethereum as per blog post above)
1. Hybrid
    1. In go-Ethereum the   

## Persistence

The 3 modes of persistence are:

There are two requirements that are affected most by this choice:
1. The ability to scale state beyond RAM capacity - this property isn't only relevant for natural state accumulation but also for possible contract bugs and mistakes. it compromises robustness. although, we can take a calculated risk and say we can respond quickly to any unlikely bug or mistake if they occur and at the same time assume natural state growth will be slow enough before an next release
1. The amount of time spent on syncing a node on system restart

There 4 possibilities, and some can be implemented as a milestone to get to another:

1. Purely in memory
    1. Pros 
        1. Fast & Simple
        1. GoLang GC may work seamlessly for node GC (subject to tests and benchmarks)
    1. Cons 
        1. Contracts may overflow memory and crash node
        1. Must implement memory limits and overflow handling policy
        1. When a node restarts we need to scan all blocks in the chain to replay all state changes
1. In Memory with persisted snapshots
    1. Pros 
        1. Fast
        1. On restart replay blocks later than last snapshot
        1. GoLang GC may work seamlessly for node GC (subject to tests and benchmarks)
    1. Cons 
        1. Contracts may overflow memory and crash node
        1. Must implement memory limits and overflow handling policy
        1. Introduce persistence complexity without the gains of scaling beyond RAM capacity
1. Fully Persisted with cache
    1. Pros 
        1. Fast
        1. Scales beyond RAM capacity
    1. Cons 
        1. Must define strategy to deal with memory overflow
        1. Most complicated option - YAGNI?
        1. Garbage collection more complicated
1. Fully persisted without cache
    1. Pros
        1. Simpler than #3
        1. Scales beyond RAM capacity
    1. Cons  
        1. may be too slow to be practical
        1. complicated
        1. Garbage collection more complicated
        
    