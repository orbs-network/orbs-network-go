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
		guardian := _getGuardianAtIndex(i)
		state.Clear(_formatGuardian(guardian[:]))
		state.Clear(_formatGuardianIterator(i))
	}
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

func getGuardianStake(guardian []byte) uint64 {
	return state.ReadUint64(_formatVoterStakeKey(guardian))
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
