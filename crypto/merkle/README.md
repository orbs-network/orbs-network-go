# Merkle BinaryTrees / Tries

##  Merkle Binary Algorithm
> This is a basic non public implementation of a compact Binary Merkle Tree/Trie. 
It defines the common algorithm for both Tree and Trie.
Essentially it defines:
* The node in memory with hashed value, left/right pointer, 
decompressed path (list of bytes each one is a 1 or 0) and a hash of the from this node
downwards.
* The insert function that allows adding a new key/value to a tree by generating 
a new node pointer. This function should be called multiple times to update a tree with 
many values as it doesn't hash or trim the tree. Note insert needs a "dirty" cache also defined in this package.
* The collapseAndHash function is called after the insert to compact the tree and run the hash 
function from the leafs upwards
> The implementations supports multi-length keys, so non-leafs may have a value. 
  The trie is compacted.
> These common functions are used to build two public implementations of Merkle:

##  Merkle Binary Trie Forest
> The input is a list of Key/Value pair. Each tree after creating is immutable.
We use a forest implementation that keeps several past trees with their root pointers 
and in each update only new nodes (including root node) change. This has advantage of
using the go GC to remove unused nodes when we discard a root node.

> The implementations assumes a fixed size of key so that only leaf nodes have Values.
 This causes all nodes that are non leaf have 2 children. 

* Tree type: Binary Merkle Trie
* Key: []byte (assume fixed size upto 1-256)
* Value: SHA256 (32B)
* Hash: SHA256 (32B)

#### Merkle Binary Trie Forest - Hash
> Please note, since only leaf nodes use value in hashing any entry with non fixed size key
will not participate in the the Merkle root and will not be able to create valid proofs.
* Leaf node : hash{Value, prefix (MSB shifted)}
* Core node : hash{left_child_hash, right_child_hash, prefix (MSB shifted)}

#### Binary Merkle Trie Forest - Proof
> Provides inclusion / exclusion authentication for arbitrary keys.

* Structure:
    * First node {Root hash(32B), Root prefix size(B)}
    * Core node: {Sibling hash (32B), Self prefix size (B)}
    * List is of max depth of tree * 33B each node.

* Proof validation:
  * node prefix size = the size of prefeix (lsb) of the key 
  * node hash = hash of sibling of value testes.
  * hash_state = hash of (the value tested, key prefix (prefix size lsb of key)
  * key left over = key size - node prefix size
  * For each node in the proof starting from the last
    * if key in key left over size = 0
      * hash_state = hash{hash_state, node hash, Max(hash_state, node)}
    * key_bit--
  * Compare the hash_state with the tree root.


## Merkle Binary Ordered Tree
> The input is a list of values. The order of the values determains the tree.
Tree is immutable and only used to get proofs by index of value in original list.

> Note we use order the two 

* Tree type: Binary Merkle Tree
* Value: SHA256 (32B)
* Hash: SHA256 (32B)

#### Merkle Binary Ordered Tree - Hash
> 
* Leaf node : {Value}
* Core node : hash{Min(left_child_hash, right_child_hash), Max(left_child_hash, right_child_hash)}

#### Merkle Binary Ordered Tree - Proof
> Proofs : Provides inclusion authentication for sequential values (0 - max_index)..
> To maintain a short proof the key size if the ceiling of the log2 of the number of values.

* Structure:
  * List of ceiling(log(max_index)) core nodes' hash.

* Proof validation:
  * hash_state = the value tested
  * key_bit = proof length - 1
  * For each node in the proof starting from the last
      * hash_state = hash{Min(hash_state, node), Max(hash_state, node)}
    * key_bit--
  * Compare the hash_state with the tree root.

