// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"encoding/binary"
	"github.com/orbs-network/crypto-lib-go/crypto/hash"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/env"
)

/**
 * This function is meant ot be used via callSystemContract or sendTx ... it will not give same result when used with RunQuery
 * This function is used with state as last committed block but with env (block height) of the block being closed.
 */
func getOrderedCommittee() [][]byte {
	return _orderList(env.GetBlockCommittee(), _generateSeed(env.GetBlockHeight()))
}

/**
 * This function is meant ot be used via runQuery ... and gives the committee for the next block (compared with block height of the return value)
 */
func getNextOrderedCommittee() [][]byte {
	return _orderList(env.GetBlockCommittee(), _generateSeed(env.GetBlockHeight()+1))
}

func _generateSeed(blockHeight uint64) []byte {
	seedBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBytes, blockHeight)
	return seedBytes
}

func _orderList(addresses [][]byte, seed []byte) [][]byte {

	committeeSize := len(addresses)
	orderedCommitteeAddresses := make([][]byte, 0, committeeSize)
	random := seed
	accumulatedWeights, totalWeight := _calculateAccumulatedWeightArray(addresses)

	for j := 0; j < committeeSize; j++ {
		random = _nextRandom(random)
		curr := _getRandomWeight(random, totalWeight)

		for i := 0; i < committeeSize; i++ {
			if curr > accumulatedWeights[i] {
				continue
			}
			orderedCommitteeAddresses = append(orderedCommitteeAddresses, addresses[i])
			currWeight := _absoluteWeight(addresses[i])
			totalWeight -= currWeight
			accumulatedWeights[i] = 0
			for k := i + 1; k < committeeSize; k++ {
				if accumulatedWeights[k] != 0 {
					accumulatedWeights[k] -= currWeight
				}
			}
			break
		}
	}
	return orderedCommitteeAddresses
}

func _calculateAccumulatedWeightArray(addresses [][]byte) ([]int, int) {
	accumulatedWeights := make([]int, len(addresses))
	totalWeight := 0
	for i, address := range addresses {
		weight := _absoluteWeight(address)
		totalWeight += weight
		accumulatedWeights[i] = totalWeight
	}
	return accumulatedWeights, totalWeight
}

func _absoluteWeight(address []byte) int {
	return 1 << (_getMaxReputation() - getReputation(address))
}

func _nextRandom(random []byte) []byte {
	return hash.CalcSha256(random)
}

func _getRandomWeight(random []byte, maxWeight int) int {
	return int(binary.LittleEndian.Uint32(random[hash.SHA256_HASH_SIZE_BYTES-4:])%uint32(maxWeight)) + 1
}
