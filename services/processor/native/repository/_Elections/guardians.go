// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

/***
 * Guardians
 */
func _getGuardians() map[[20]byte]bool {
	numOfGuardians := _getNumberOfGuardians()
	guardians := make(map[[20]byte]bool, numOfGuardians)
	for i := 0; i < numOfGuardians; i++ {
		guardians[_getGuardianAtIndex(i)] = true
	}
	return guardians
}

func _setGuardians(guardians [][20]byte) {
	numOfGuardians := len(guardians)
	_setNumberOfGuardians(numOfGuardians)
	for i := 0; i < numOfGuardians; i++ {
		_setGuardianAtIndex(i, guardians[i][:])
		state.WriteUint32(_formatGuardian(guardians[i][:]), 1)
	}
}

func _clearGuardians() {
	numOfGuardians := _getNumberOfGuardians()
	for i := 0; i < numOfGuardians; i++ {
		g := _getGuardianAtIndex(i)
		guardian := g[:]
		state.Clear(_formatGuardian(guardian))
		state.Clear(_formatGuardianIterator(i))
		state.Clear(_formatGuardianCandidateKey(guardian))
		state.Clear(_formatGuardianStakeKey(guardian))
		state.Clear(_formatGuardianVoteBlockNumberKey(guardian))
	}
	_setNumberOfGuardians(0)
}

func _isGuardian(guardian [20]byte) bool {
	return state.ReadUint32(_formatGuardian(guardian[:])) != 0
}

/***
 * Guardians - data struct
 */
func _formatNumberOfGuardians() []byte {
	return []byte("Guardians_Count")
}

func _getNumberOfGuardians() int {
	return int(state.ReadUint32(_formatNumberOfGuardians()))
}

func _setNumberOfGuardians(numberOfGuardians int) {
	state.WriteUint32(_formatNumberOfGuardians(), uint32(numberOfGuardians))
}

func _formatGuardianIterator(num int) []byte {
	return []byte(fmt.Sprintf("Guardian_Address_%d", num))
}

func _getGuardianAtIndex(index int) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatGuardianIterator(index)))
}

func _setGuardianAtIndex(index int, guardian []byte) {
	state.WriteBytes(_formatGuardianIterator(index), guardian)
}

func _formatGuardian(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s", hex.EncodeToString(guardian)))
}

func _formatGuardianStakeKey(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s_Stake", hex.EncodeToString(guardian)))
}

func getGuardianStake(guardian []byte) uint64 {
	return state.ReadUint64(_formatGuardianStakeKey(guardian))
}

func _setGuardianStake(guardian []byte, stake uint64) {
	state.WriteUint64(_formatGuardianStakeKey(guardian), stake)
}

func _formatGuardianVoteBlockNumberKey(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s_VoteAt", hex.EncodeToString(guardian)))
}

func _getGuardianVoteBlockNumber(guardian []byte) uint64 {
	return state.ReadUint64(_formatGuardianVoteBlockNumberKey(guardian))
}

func _setGuardianVoteBlockNumber(guardian []byte, blockNumber uint64) {
	state.WriteUint64(_formatGuardianVoteBlockNumberKey(guardian), blockNumber)
}

func _formatGuardianCandidateKey(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s_Candidates", hex.EncodeToString(guardian)))
}

func _getCandidates(guardian []byte) [][20]byte {
	candidates := state.ReadBytes(_formatGuardianCandidateKey(guardian))
	numCandidate := len(candidates) / 20
	candidatesList := make([][20]byte, numCandidate)
	for i := 0; i < numCandidate; i++ {
		copy(candidatesList[i][:], candidates[i*20:i*20+20])
	}
	return candidatesList
}

func _setCandidates(guardian []byte, candidateList [][20]byte) {
	candidates := make([]byte, 0, len(candidateList)*20)
	for _, v := range candidateList {
		candidates = append(candidates, v[:]...)
	}

	state.WriteBytes(_formatGuardianCandidateKey(guardian), candidates)
}

func _formatGuardianVoteWeightKey(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s_Weight", hex.EncodeToString(guardian)))
}

func getGuardianVotingWeight(guardian []byte) uint64 {
	return state.ReadUint64(_formatGuardianVoteWeightKey(guardian))
}

func _setGuardianVotingWeight(guardian []byte, weight uint64) {
	state.WriteUint64(_formatGuardianVoteWeightKey(guardian), weight)
}
