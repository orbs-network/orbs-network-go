package elections_systemcontract

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

/***
 * Validators
 */
func _setValidators(validators [][20]byte) {
	numberOfValidators := len(validators)
	_setNumberOfValidators(numberOfValidators)
	for i := 0; i < numberOfValidators; i++ {
		_setValidatorEthereumAddressAtIndex(i, validators[i][:])
	}
}

func _getValidators() (validValidtors [][20]byte) {
	numOfValidators := _getNumberOfValidators()
	validValidtors = make([][20]byte, numOfValidators)
	for i := 0; i < numOfValidators; i++ {
		validValidtors[i] = _getValidatorEthereumAddressAtIndex(i)
	}
	return
}

/***
 * Validators - data struct
 */
func _formatNumberOfValidators() []byte {
	return []byte("Valid_Validators_Count")
}

func _getNumberOfValidators() int {
	return int(state.ReadUint32(_formatNumberOfValidators()))
}

func _setNumberOfValidators(numberOfValidators int) {
	state.WriteUint32(_formatNumberOfValidators(), uint32(numberOfValidators))
}

func _formatValidValidaorIterator(num int) []byte {
	return []byte(fmt.Sprintf("Valid_Validator_Address_%d", num))
}

func _getValidatorEthereumAddressAtIndex(index int) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatValidValidaorIterator(index)))
}

func _setValidatorEthereumAddressAtIndex(index int, guardian []byte) {
	state.WriteBytes(_formatValidValidaorIterator(index), guardian)
}

func _formatValidatorOrbsAddressKey(validator []byte) []byte {
	return []byte(fmt.Sprintf("Valid_Validator_%s_Orbs", hex.EncodeToString(validator)))
}

func _getValidatorOrbsAddress(validator []byte) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatValidatorOrbsAddressKey(validator)))
}

func _setValidatorOrbsAddress(validator []byte, orbsAddress []byte) {
	state.WriteBytes(_formatValidatorOrbsAddressKey(validator), orbsAddress)
}

func _formatValidatorStakeKey(validator []byte) []byte {
	return []byte(fmt.Sprintf("Valid_Validator_%s_Stake", hex.EncodeToString(validator)))
}

func getValidatorStake(validator []byte) uint64 {
	return state.ReadUint64(_formatValidatorStakeKey(validator))
}

func _setValidatorStake(validator []byte, stake uint64) {
	state.WriteUint64(_formatValidatorStakeKey(validator), stake)
}

func _formatValidatorVoteKey(validator []byte) []byte {
	return []byte(fmt.Sprintf("Valid_Validator_%s_Vote", hex.EncodeToString(validator)))
}

func getValidatorVote(validator []byte) uint64 {
	return state.ReadUint64(_formatValidatorVoteKey(validator))
}

func _setValidatorVote(validator []byte, stake uint64) {
	state.WriteUint64(_formatValidatorVoteKey(validator), stake)
}
