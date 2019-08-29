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
	elected := _split(_getElectedValidators())
	return _concat(_getOrderedCommitteeArray(elected))
}

func _getOrderedCommitteeArray(addresses[][]byte) [][]byte {
	seedBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBytes, env.GetBlockHeight())

	return _orderList(addresses, seedBytes)
}
func _orderList(addrs [][]byte, seed []byte) [][]byte {
	addrsToSort := addrsAndGrades{ addrs, make([]float64, len(addrs))}
	for i, addr := range addrs {
		addrsToSort.grades[i] = _calculateGradeWithReputation(addr, seed)
	}
	sort.Sort(addrsToSort)
	return addrsToSort.addresses
}

func _calculateGradeWithReputation(addr []byte, seed []byte) float64 {
	rep := _getReputation(addr)
	return float64(_calculateGrade(addr, seed)) / _calculateReputationMarkDownFactor(rep)
}

func _calculateGrade(addr []byte, seed []byte) uint32 {
	random := hash.CalcSha256(addr, seed)
	return binary.LittleEndian.Uint32(random[hash.SHA256_HASH_SIZE_BYTES-4:])
}

func _calculateReputationMarkDownFactor(reputation uint32) float64 {
	if reputation < ToleranceLevel {
		return 1.0
	}
	return math.Pow(2, float64(reputation))
}

type addrsAndGrades struct {
	addresses [][]byte
	grades    []float64
}

func (s addrsAndGrades) Len() int {
	return len(s.addresses)
}

func (s addrsAndGrades) Swap(i, j int) {
	s.addresses[i], s.addresses[j] = s.addresses[j], s.addresses[i]
	s.grades[i], s.grades[j] = s.grades[j], s.grades[i]
}

// descending order
func (s addrsAndGrades) Less(i, j int) bool {
	return s.grades[i] > s.grades[j] || (s.grades[i] == s.grades[j] && bytes.Compare(s.addresses[i], s.addresses[j]) > 0)
}
