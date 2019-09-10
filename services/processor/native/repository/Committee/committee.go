// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"bytes"
	"encoding/binary"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/env"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"math"
	"sort"
)

func getOrderedCommittee() []byte {
	return getOrderedCommitteeForAddresses(_getElectedValidators())
}

func getOrderedCommitteeForAddresses(addresses []byte) []byte {
	addressArray := _split(addresses)
	return _concat(_getOrderedCommitteeArray(addressArray))
}

func _getOrderedCommitteeArray(addresses[][]byte) [][]byte {
	return _orderList(addresses, _generateSeed())
}

func _generateSeed() []byte {
	seedBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBytes, env.GetBlockHeight())
	return seedBytes
}

func _orderList(addrs [][]byte, seed []byte) [][]byte {
	addrsToSort := addrsAndScores{addrs, make([]float64, len(addrs))}
	for i, addr := range addrs {
		addrsToSort.scores[i] = _calculateScoreWithReputation(addr, seed)
	}
	sort.Sort(addrsToSort)
	return addrsToSort.addresses
}

func _calculateScoreWithReputation(addr []byte, seed []byte) float64 {
	rep := getReputation(addr)
	return float64(_calculateScore(addr, seed)) / _reputationAsFactor(rep)
}

func _calculateScore(addr []byte, seed []byte) uint32 {
	random := hash.CalcSha256(addr, seed)
	return binary.LittleEndian.Uint32(random[hash.SHA256_HASH_SIZE_BYTES-4:])
}

func _reputationAsFactor(reputation uint32) float64 {
	return math.Pow(2, float64(reputation))
}

type addrsAndScores struct {
	addresses [][]byte
	scores    []float64
}

func (s addrsAndScores) Len() int {
	return len(s.addresses)
}

func (s addrsAndScores) Swap(i, j int) {
	s.addresses[i], s.addresses[j] = s.addresses[j], s.addresses[i]
	s.scores[i], s.scores[j] = s.scores[j], s.scores[i]
}

// descending order
func (s addrsAndScores) Less(i, j int) bool {
	return s.scores[i] > s.scores[j] || (s.scores[i] == s.scores[j] && bytes.Compare(s.addresses[i], s.addresses[j]) > 0)
}
