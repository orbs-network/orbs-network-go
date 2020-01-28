// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"encoding/binary"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/env"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
)

/**
 * This function is meant ot be used via the callsystemcontract func ... it will not give same result when used with RunQuery
 */
func getOrderedCommittee() []byte {
	return getOrderedCommitteeForAddresses(_getElectedValidators())
}

func getOrderedCommitteeForAddresses(addresses []byte) []byte {
	return _concat(_getOrderedCommitteeForAddresses(addresses))
}

func _getOrderedCommitteeForAddresses(addresses []byte) [][]byte {
	addressArray := _split(addresses)
	return _getOrderedCommitteeArray(addressArray)
}

func _getOrderedCommitteeArray(addresses [][]byte) [][]byte {
	return _orderList(addresses, _generateSeed())
}

func _generateSeed() []byte {
	seedBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBytes, env.GetBlockHeight())
	return seedBytes
}

func _orderList(addresses [][]byte, seed []byte) [][]byte {
	committeeSize := len(addresses)
	accumulatedWeights := make([]int, committeeSize)
	totalWeight := 0
	for i, address := range addresses {
		weight := _absoluteWeight(address)
		totalWeight += weight
		accumulatedWeights[i] = totalWeight
	}

	orderedCommitteeAddresses := make([][]byte, 0, committeeSize)
	random := seed

	for j := 0; j < committeeSize; j++ {
		random = _nextRandom(random)
		curr := _getRandomWeight(random, totalWeight)

		for i := 0; i < committeeSize; i++ {
			if curr > accumulatedWeights[i] || accumulatedWeights[i] == 0 {
				continue
			}
			orderedCommitteeAddresses = append(orderedCommitteeAddresses, addresses[i])
			currWeight := _absoluteWeight(addresses[i])
			totalWeight -= currWeight
			accumulatedWeights[i] = 0
			for k := i + 1;k < committeeSize;k++ {
				if accumulatedWeights[k] != 0 {
					accumulatedWeights[k] -= currWeight
				}
			}
			break
		}
	}
	return orderedCommitteeAddresses
}

func _absoluteWeight(address []byte) int {
	return 1 << (_getMaxReputation() - getReputation(address))
}

func _nextRandom(random []byte) []byte {
	return hash.CalcSha256(random)
}

func _getRandomWeight(random []byte, maxWeight int) int {
	return int(binary.LittleEndian.Uint32(random[hash.SHA256_HASH_SIZE_BYTES-4:]) % uint32(maxWeight)) + 1
}
