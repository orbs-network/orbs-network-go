# Merkle Binary Trees / Tries

This package implements two variations of Merkle tree structures used for:
* Transactions / Receipts 
    * Compact Binary Merkle **Tree** - Tracks an ordered list of values. Entries are identified by their index location in the list: 0 to the length - 1. the tree is a Complete binary tree.
    * Insertions are strictly in the natural order of the list
    * There are no additions/subtractions
    * no need for exclusion proofs, only inclusion proofs
* State Merkle Trie - 
    * Compact Binary Merkle **Trie** - Tracks a set of key => values pairs. Where keys are of fixed length and are 256 bytes. The tree is a compact binary trie
    * There is no natural order and insertions may not be in order. The structure implicitly sorts entries by the keys binary representation
    * keys may be added or removed, possibly leading to a new root as a result of the change in specific branches in the trie
    * exclusion proofs are required as well as inclusion proofs

## Shared Algorithm (common package)
A common layer used by both flavors of our Merkle tree solutions:  **Compact Binary Merkle Tree/Trie**. If we regard the first structure (compact tree) as a
specific case of the second one (trie) we can construct a set of key => value pairs where key represents the ordinal number in the list for each value. 
This illustrates how a tree may be implemented using an underlying trie structure.

The shared algorithm package provides a general purpose binary trie data structure that may be used to implement both trie and tree merkle for data sets of key/value pairs or ordered value lists respectively.
It provides basic operations on the structure for construction and manipulation.

This layer does not address proof representation or proof verification as they are more specific for properties and semantics of each type of data set.

To support tries of varying key size, the algorithm layer makes no assumptions about the length of entry keys and permits attaching values to non lead nodes.
However, because the requirements listed above do not include using keys of variable length in same trie/tree, it's safe to assume that all keys are in fact of same length. 
 
Shared Algorithm package defines the common node structure for both Tree and Trie.
* The node in memory has the following properties:
   * `value-hash`: if the path leading from the root down to this node represents a the key with an associated value this field holds the hash of value. SHA256 (32B).
   * `left`, `right`: pointers to the left and right child nodes or null if this is a leaf node. the `left` node extends the path with a `0` bit, and the `right` one extends it with a `1` bit
   * `prefix`: a part of a key or path, extending the beyond any parent prefix and branch bits. prefixes are represented in inflated form: each bit in the actual key becomes a byte with either 1 or 0 matching the corresponding bit value in the key
   * `node-hash`: The hash value of the current node. Computed by an injectable hash function, see [Tree](Merkle Binary Trie Forest Hash) and [Trie](Merkle Binary Ordered Tree Hash) for possible implementations. SHA256 (32B). 
* `insert` function. Allows adding a new key/value to a tree by generating 
a new node pointer. This function should be called multiple times to update a tree with 
many values as it does not hash or trim the tree. Note, this function  uses a _dirty cache_ to optimize multiple subsequent 
invocations when several updates occur before determining the new root.
* `collapseAndHash` function. Called after a series of calls to insert. It acts on the _dirty cache_ applying changes back into the tree, 
while compacting the resulting structure to ensure compactness. Finally, it scans all altered branches, applying the provided hash function from the leafs upwards, 
altering node-hash to eventually arrive at the resulting new merkle root.

Supports mixed-length keys, so non-leafs may have a value 

`Zero-Hash` is defined as `SHA256([32]byte{})` - of hash of 32 bytes of 0. 

#  Merkle Binary Trie Forest
An implementation of a merkle trie used for virtual block-chain state structure:
* each entry is a key/value with similar length keys (32 bytes long) and arbitrary values. (see implications below)
* values are byte arrays where length(value) > 0. entries with empty values are not included
* supports proof for inclusion of specified key/value pairs
* supports special proofs for the exclusion of a specified key/value pair, which is equivalent to prooving the logical inclusion of key => empty value.

The input is a set of Key/Value pairs. Each tree after creation is immutable.
Since state evolves over time (as new blocks arrive), there is overlap between merkle roots pertaining to different block 
heights in so far as much of the state may remain unchanged.
But, for each distinct state there is a distinguished root node, and at least one path to a value node which is different.

We use the notion of a _forest_ to refer to the over-structure that contains several state merkle tries each for different block height.
Because nodes are immutable, each node may be referenced by, or included in, more than one state tree. This allows for significant savings in memory usage
when multiple state trees of neighbouring block heights are kept in memory. 

In each update only new nodes (including a new root node) are created. In addition to saving memory, this has the implicit advantage of
utilizing GoLang GC to remove unused nodes once we discard the final reference to a root node (corresponding to a past block height).

The implementation assumes a fixed size of key up to 32 Bytes. A fixed key length guarantees these properties:
* Only leaf nodes have assigned Values.
* Non leaf nodes have **exactly 2 children** (left/right). 

>Note: Adding nodes with different length will not fail, but the proof/verify functions may not work.

#### Merkle Binary Trie Forest Hash
* Leaf nodes: `node-hash := SHA256(value-hash, prefix)`
* Non leaf node: `node-hash := SHA256(left.node-hash, right.node-hash, prefix)`

>Please note, since only leaf nodes use value in hashing, entries with key length shorter than the
 maximal key length may not participate in the the Merkle root and will not be able to create valid proofs. This is resolved by the restriction to same-length keys

#### Binary Merkle Trie Forest Proof
Provides inclusion / exclusion authentication for arbitrary keys. 

A proof is generated by providing the root (relevant for _forests_ with more than one root), and a key of the correct length (same length as all keys in the trie). 
The key for which the proof was generated will be called `requested_key`.

A Path in the tree is a list of inter-connected nodes starting at a root node and extending down towards a leaf node. Each Path in the tree has an associated _path-prefix_ which
is the concatenation of each node's `prefix` field, together with a branch bit leading to the next node on the Path (`0` for `left` and `1` for `right`) 

###### Inclusion Proof
If `requested_key` is a valid entry in the key/value set represented by the merkle tree indicated by the requested root, there will be (exactly) one Path extending from the root node
indicated by the proof requester, and ending with a leaf node having a non-empty value. The associated _path-prefix_ will be identical to `requested_key`, and the generated proof
is called an _inclusion proof_. 

The proof will enclose information allowing the proof validator to compute the merkle root based on a path (list of nodes) with a _prefix-path_
fully overlapping with `requested_key`, enforcing the use of a single pre-image value used for constructing the leaf node's hash.

###### Exclusion Proof
If `requested_key` is not a valid entry in the key/value set represented by the merkle tree indicated by the requested root, there will be (exactly) one Path extending from the
root node indicated by the proof requester, where at each non-leaf node the branch followed is the bit indicated by `requested_key`. The path terminates on the first node whose
`prefix`es contribution to the _path-prefix_ contradicts `requested_key`. This results in the Path having the longest overlapping substring in both `requested_key` and _path-prefix_,
and ending with a node (maybe leaf or non-leaf) proving that the `requested_key`s path is not branched towards in the original trie.
* Since the key is missing from the set there cannot be a _prefix-path_ completely identical to `requested_key`.
* Since all nodes are either leaf nodes or have two children (always zero or two children - as required from the compactness of the trie) the Path for an exclusion proof 
ends with the first node who's `prefix` does not match`requested_key`. The contradiction may not be on the branch bit because there are always two branch bits available.

The proof will enclose information allowing to proof validator to compute the merkle root based on a path (list of nodes) which diverges from `requested_key` in a valid node's `prefix`  

##### Proof Structure:
Each proof, weather an inclusion or exclusion must satisfy these goals:
1. represent a pre-image to the merkle root that was specified when it was constructed
1. represent a Path along the trie whose _path-prefix_ coincides with `requested_path`:
    1. inclusion: entirely
    1. exclusion: all the way down to the last node in the path where it diverges

The proof is generated by traversing the tree from the root core node along the Path towards a requested key
leaf and its value. For each core node one child will be the next element on the Path (_child-on-path_) and it's sibling will be off path (_child-off-path_).
In each step we record the node's _child-off-path_'s `node-hash` and the current `prefix` size, then we travel down to the _child-on-path_ node and reiterate.
This process continues until a terminating node is reached (`term`). a terminating node satisfies one of these conditions:
1. inclusion: a leaf node (`leaf`) satisfying the `requested_key` is reached. this last node is represented as: `term.node-hash + len(leaf.prefix)`. notice the child node is replaced by this leaf's `node-hash` value. 
1. exclusion: a node whose `prefix` contributes to a _path-prefix_ which does not coincide with `requested_key`.

`term` node has a different representation in the proof than it's ancestors on the path. since it may or may not have child nodes, we defer it's pre-image representation to the end of the proof and instead include
it's own `node-hash` rather than parts of it's pre-image. In addition, we include the prefix length as we do with it's ancestors:   
terminating node is represented as `term.node-hash + len(term.prefix)`

This allows verifying the proof's path by calculating "bottom-up" all node hashes starting with the provided `term.node-hash`, and arriving back at the Merkle Root.

To summarize the above, here is the anatomy of a trie merkle proof: 
1. List _(A)_ of pre-image complements for nodes along the proof's path __except for the last node on the path__. each 33 bytes: _child-off-path_.`node-hash` +  `prefix` length indicator in bits
1. terminating node on path: `node-hash` + `prefix` length indicator in bits. let `node-hash` of the last node on path be `term.node-hash`, and `term.prefix` be the prefix of that node
1. _path-prefix_ - padded up to 32 bytes. if this is an exclusion proof the _path-prefix_ may be shorter than 32 bytes, however, the trailing bits may be ignored
1. terminating node pre-image complement:
    1. leaf-node: `value-hash` _(B)_
    1. non-leaf-node: `left.node-hash` + `right.node-hash` (Bl, Br)
    
>* _path-prefix_ is provided as a whole, with each node referencing it only by indicating the length of it's own prefix. When reading the proof this allows a validator to infer each
node's `prefix` by tracking the number of bits on the path consumed by previous nodes `prefix`es and branch bits and extracting the next n bits from the attached _path_prefix_.
>* in an inclusion proof the final node is always a leaf node. 
>* in an exclusion proof the final node may be a leaf or a core node. It is the first node whose `prefix` diverges from `requested_key`.

##### Proof validation:

We validate a proof in the presence of:
1. `purported-value`, which is a byte array, 
1. `requested_key` which should be the same key provided when generating the proof
1. `merkle-hash` 

are provided to the validator. If `len(purported-value) == 0` we expect the proof to be a valid
exclusion proof, since our merkle tree does not contain any zero values. if `len(purported-value) > 0` we tread the proof as an inclusion proof:

   
  * Step 1: check consistency of path with `requested_key` and `merkle-hash`. 
     * `current_hash := term.node-hash`
     * `path-length-without-terminating-node := sum(\[A[i] prefix length + 1\])`
     * `branch_bit_pos := path-length-without-terminating-node + len(term.prefix)` initialize branch bit index to the end of the path 
     * scan list _(A)_ from last to first:
        * `branch_bit_pos -= A[i] prefix length` determine the position of the branch bit leading up to the current node
        * if `path-prefix[branch_bit_pos] == 0`
            * `current_hash = SHA256(current_hash, A[i](child-off-path).node-hash, A[i].prefix)` on-path-node was left
        * else 
            * `current_hash = SHA256(A[i](child-off-path).node-hash, current_hash, A[i].prefix)` on-path-node was right
       * 
     * Verify Merkle root: assert that `current_hash == merkle-hash` (otherwise reject proof)
  * Step 2: Validate `term.node-hash` and 
     * inclusion: 
        * `calculated_term_node-hash = SHA256(purported value, term.prefix)`
        * asset `calculated_term_node-hash == term.node-hash` (otherwise we prove that the real value is NOT `purported-value`)
     * exclusion: 
        * if _B_ provided (the terminating node is a leaf node)
            * `calculated_term_node-hash = SHA256(B, term.prefix)`
        * if _Bl_ and _Br_ provided:
             * `calculated_term_node-hash = SHA256(Bl, Br, term.prefix)`
        * asset `calculated_term_node-hash == term.node-hash` (reject proof otherwise)
        * asset `path-prefix[0:path-length-without-terminating-node] == requested_key[0:path-length-without-terminating-node]`
        * asset `term.prefix != requested_key[path-length-without-terminating-node:]` to prove that the final node indicates exclusion of the requested key from any valid path
        * asset `path-length-without-terminating-node + len(term.prefix) == len(requested_key)` (reject proof otherwise)


# Merkle Binary Ordered Tree
The input is a list of values. Using the ordinal number for each item as a trie key, the order of the values determine the tree structure.
Tree is never appended to and is immutable from creation. and only used to get proofs by index of value in original list (ordinal value of the entry).

Proof structure is simplified as a result of the tree being complete as the keys are consecutive values and the LSB sequences are exhaustive. 

NOTE - because we only prove inclusion in a set and the index is irrelevant for the proof (if a value is included the verifier does not care that 
the index matches the expectation) we formulate the proof such that a key is not required for verifying a proof. We define each node's hash function such
that the order of concatenation of child hash values is determined by their relation to each other. the hash function always includes the the 
smallest hash value first and the greater hash value later. this allows for a very simple proof validation. and no key is required to validate a proof  

#### Merkle Binary Ordered Tree Hash
* leaf nodes: `node-hash := SHA256(value)`
* non leaf nodes: `min := Min(left_child_hash, right_child_hash); max := Max(left_child_hash, right_child_hash); node-hash := SHA256(min, max)`

#### Merkle Binary Ordered Tree Proof
Proofs : Provides inclusion authentication for sequential values (0 - max_index)..
To maintain a short proof the key size if the ceiling of the log2 of the number of values.

We validate a proof in the presence of:
1. `purported-value`, which is a byte array, 
1. `merkle-hash`

* Structure:
  * List of hashes, bound by log(max_index) nodes hashes.

* Proof validation:
  * `current_hash := SHA256(purported-value)`
  * `key_bit = proof length - 1`
  * For each node hash `N[i]` in the proof starting from the last
      * `current_hash := hash(current_hash, N[i])` the hash function sorts the parameters internally as described above so the order is irrelevant
  * assert `current_hash == merkle-hash`

