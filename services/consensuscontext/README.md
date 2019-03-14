# Consensus Context
This file explains non-trivial implementation details in the Consensus Context.

## Committees
A *committee* is a subset of validator nodes, with size between some minimum (currently set at 4) and all of validator nodes.
The committee nodes' purpose is to participate in a consensus round, that is to reach consensus for a specific block height.

After consensus is reached, a new committee is chosen.

### Committee Selection

See `RequestValidationCommittee` in [`committee.go`](/committee.go)

#### Algorithm
A weighted random sort over the nodes is used for deciding which nodes become committee members.
The *random* part is achieved by hashing the concatenation of the random seed and the node's public key.
The *weighted* part is achieved by multiplying the node's resulting hash by its reputation,
thus increasing its chance of inclusion in a committee.
Note that at present, reputation is set to 1, so in effect it is not used by the calculation.

The actual committee is the `committeeSize` nodes with the highest `weighted_grade`.
The descending weighted_grade ordering is important - the first node is the initial leader of this consensus round.

```
for each node
   hash = sha256(concat(random_seed, node.public_key))
   low_4_bytes_hash := hash[:4]
   weighted_grade := to_uint32(low_4_bytes_hash) * reputation

sort_nodes := sort(nodes, by descending weighted_grade)
committee = first [committeeSize] sort_nodes

```

