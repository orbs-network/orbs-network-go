# Merkle BinaryTrees / Tries

##  Merkle Binary Algorithm
This is a basic non public implementation of a compact Binary Merkle Tree/Trie. 
It defines the common algorithm for both Tree and Trie.
Essentially it defines:
* The node in memory is:
   * value : hash code of the value assassinated with the path that is represented by the current node
   * left/right : left and right child pointers, 
   * prefix : decompressed partial path (list of bytes each one is a 1 or 0)
   * node-hash : hash of this node specified for [Tree](Merkle Binary Trie Forest Hash)/[Trie](Merkle Binary Ordered Tree Hash) implementation  
* The insert function that allows adding a new key/value to a tree by generating 
a new node pointer. This function should be called multiple times to update a tree with 
many values as it does not hash or trim the tree. Note insert needs a "dirty" cache also defined in this package.
* The collapseAndHash function is called after the insert to compact the tree and run the hash 
function from the leafs upwards (using the dirty cache to limit action only on changed nodes).

The algo-implementation supports multi-length keys, so non-leafs may have a value. 

These common functions are used to build two public implementations of Merkle:

##  Merkle Binary Trie Forest
The input is a list of Key/Value pair. Each tree after creation is immutable.
We use a forest implementation that keeps several past trees with their root pointers 
and in each update only new nodes (including root node) change. This has advantage of
using the go GC to remove unused nodes when we discard a root node.

The implementations assumes a fixed size of key so that only leaf nodes have Values.
This causes all nodes that are non leaf to have 2 children. Adding nodes with differnt length
will not fail, but the proof/verify functions may not work.
 
* Tree type: Binary Merkle Trie
* Key: []byte (assume fixed size, upto 32B)
* Value (hash): SHA256 (32B)
* Hash (of Node): SHA256 (32B)

#### Merkle Binary Trie Forest Hash
* Leaf node : `hash {value, prefix}`
* Core node : `hash {left child's node-hash, right child's node-hash, prefix}`

Please note, since only leaf nodes use value in hashing, entries with key length shorter than the
 maximal key length may not participate in the the Merkle root and will not be able to create valid proofs.

#### Binary Merkle Trie Forest Proof
Provides inclusion / exclusion authentication for arbitrary keys.

##### Structure:
For inclusion proofs PATH represents the requested key. 

For exclusion proofs PATH represents the existing key which has the longest key prefix matching the requested key.
A valid proof for the existence of a different PATH which diverges from the
requested key on the Final node, proves the exclusion of the requested key. 

A proof is made of a variable number of node of 33 bytes each, followed by
32 bytes of a value-hash and PATH. 

```{List of nodes (33B each), Value-Hash (32B), Path (32B)}```

* List of nodes:
    * Core node: {hash of **sibling** of the path node (32B), self prefix size (1B)}
    * Final node: {hash of last (32B), prefix of leaf (1B)}
* ValueHash : Hash Value of leaf (needed for exclusion)
* Path: Key of proof (needed for exclusion - will differ in LSB from queried key)

Each node is the **sibling** of a node on PATH.
 followed by the value-hash of the leaf that has the 



> _Proof validation_:
  * Step 1: check self consistance of proof with queried key
     * hash_state = hash part of last node
     * key_bit = size of key 
     * For each node in the proof starting from the one before last
        * key_bit -= size of prefix of hash_state node (one after)
        * if key[key_bit] == 0
            * hash_state = hash{hash_state, node hash, node prefix}
        * else 
            * hash_state = hash{node hash, hash_state, node prefix}
       * 
     * result must be equal to the merkle root hash (or error)
  * Step 2: inclusion/ exclusion
     * if queried value is non zerohash
        * calculate_hash = hash{value, prefix}
        * if calculate is equal to hash of last node - value is included in tree
     * if queried value is zerohash
        * calculate_hash = hash{value from proof, prefix from proof}
        * exclusion is prooved if
            * calulate_hash is equal to hash of last node
            * query key is equal to proof key - without LSB  prefix size of leaf node
            * query key is NOT equal to proof key - in LSB prefix size of leaf node
        * if query and proof key size are not equal - erro


## Merkle Binary Ordered Tree
> The input is a list of values. The order of the values determains the tree.
Tree is immutable and only used to get proofs by index of value in original list.

> Note we use order the two 

* Tree type: Binary Merkle Tree
* Value: SHA256 (32B)
* Hash: SHA256 (32B)

#### Merkle Binary Ordered Tree Hash
> 
* Leaf node : {Value}
* Core node : hash{Min(left_child_hash, right_child_hash), Max(left_child_hash, right_child_hash)}

#### Merkle Binary Ordered Tree Proof
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

