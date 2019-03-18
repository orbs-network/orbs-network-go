package elections_systemcontract

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

/***
 * Valid Validators
 */
func _setValidValidators(validValidators [][20]byte) {
	_setNumberOfValidValidaors(len(validValidators))
	for i := 0; i < len(validValidators); i++ {
		_setValidValidatorEthereumAddressAtIndex(i, validValidators[i][:])
	}
}

func _getValidValidators() (validValidtors [][20]byte) {
	numOfValidators := _getNumberOfValidValidaors()
	validValidtors = make([][20]byte, numOfValidators)
	for i := 0; i < numOfValidators; i++ {
		validValidtors[i] = _getValidValidatorEthereumAddressAtIndex(i)
	}
	return
}

/***
 * Valid Validators - data struct
 */
func _formatNumberOfValidators() []byte {
	return []byte("Valid_Validators_Count")
}

func _getNumberOfValidValidaors() int {
	return int(state.ReadUint32(_formatNumberOfValidators()))
}

func _setNumberOfValidValidaors(numberOfValidators int) {
	state.WriteUint32(_formatNumberOfValidators(), uint32(numberOfValidators))
}

func _formatValidValidaorIterator(num int) []byte {
	return []byte(fmt.Sprintf("Valid_Validator_Address_%d", num))
}

func _getValidValidatorEthereumAddressAtIndex(index int) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatValidValidaorIterator(index)))
}

func _setValidValidatorEthereumAddressAtIndex(index int, guardian []byte) {
	state.WriteBytes(_formatValidValidaorIterator(index), guardian)
}

func _formatValidValidatorOrbsAddressKey(validator []byte) []byte {
	return []byte(fmt.Sprintf("Valid_Validator_%s_Orbs", hex.EncodeToString(validator)))
}

func _getValidValidatorOrbsAddress(validator []byte) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatValidValidatorOrbsAddressKey(validator)))
}

func _setValidValidatorOrbsAddress(validator []byte, orbsAddress []byte) {
	state.WriteBytes(_formatValidValidatorOrbsAddressKey(validator), orbsAddress)
}

func _formatValidValidatorStakeKey(validator []byte) []byte {
	return []byte(fmt.Sprintf("Valid_Validator_%s_Stake", hex.EncodeToString(validator)))
}

func getValidValidatorStake(validator []byte) uint64 {
	return state.ReadUint64(_formatValidValidatorStakeKey(validator))
}

func _setValidValidatorStake(validator []byte, stake uint64) {
	state.WriteUint64(_formatValidValidatorStakeKey(validator), stake)
}

func _formatValidValidatorVoteKey(validator []byte) []byte {
	return []byte(fmt.Sprintf("Valid_Validator_%s_Vote", hex.EncodeToString(validator)))
}

func getValidValidatorVote(validator []byte) uint64 {
	return state.ReadUint64(_formatValidValidatorVoteKey(validator))
}

func _setValidValidatorVote(validator []byte, stake uint64) {
	state.WriteUint64(_formatValidValidatorVoteKey(validator), stake)
}
