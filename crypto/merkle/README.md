# Merkle Trees / Tries

##  Merkle Trie Forest
> The input is a list of Key/Value pair. Each tree after creating is immutable.
We use a forest implementation that keeps several past trees with their root pointers 
and in each update only new nodes (including root node) change. This has advantage of
using the go GC to remove unused nodes when we discard a root node.

> The implementations supports multi-length keys, so non-leafs may have a value. 
The trie is compacted. Proof nodes may include a prefix with size to represent 
compacted nodes. Provides both inclusions and exclusion proofs. 

* Tree type: Binary Merkle Trie
* Key: []byte
* Value: SHA256 (32B)
* Hash: SHA256 (32B)

#### Binary Merkle Trie - Proof
> Provides inclusion / exclusion authentication for arbitrary keys.
* Leaf node serialization: {Value, prefix_size, masked_prefix}
* Core node serialization: {left_child_hash, right_child_hash, prefix_size, masked_prefix}


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

