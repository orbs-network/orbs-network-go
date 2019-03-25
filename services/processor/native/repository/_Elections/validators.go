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
 * Validators
 */
func _setValidators(validators [][20]byte) {
	numberOfValidators := len(validators)
	_setNumberOfValidators(numberOfValidators)
	for i := 0; i < numberOfValidators; i++ {
		_setValidatorEthereumAddressAtIndex(i, validators[i][:])
	}
}

func _getValidators() (validtors [][20]byte) {
	numOfValidators := _getNumberOfValidators()
	validtors = make([][20]byte, numOfValidators)
	for i := 0; i < numOfValidators; i++ {
		validtors[i] = _getValidatorEthereumAddressAtIndex(i)
	}
	return
}

/***
 * Validators - data struct
 */
func _formatNumberOfValidators() []byte {
	return []byte("Validators_Count")
}

func _getNumberOfValidators() int {
	return int(state.ReadUint32(_formatNumberOfValidators()))
}

func _setNumberOfValidators(numberOfValidators int) {
	state.WriteUint32(_formatNumberOfValidators(), uint32(numberOfValidators))
}

func _formatValidaorIterator(num int) []byte {
	return []byte(fmt.Sprintf("Validator_Address_%d", num))
}

func _getValidatorEthereumAddressAtIndex(index int) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatValidaorIterator(index)))
}

func _setValidatorEthereumAddressAtIndex(index int, validator []byte) {
	state.WriteBytes(_formatValidaorIterator(index), validator)
}

func _formatValidatorOrbsAddressKey(validator []byte) []byte {
	return []byte(fmt.Sprintf("Validator_%s_Orbs", hex.EncodeToString(validator)))
}

func _getValidatorOrbsAddress(validator []byte) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatValidatorOrbsAddressKey(validator)))
}

func _setValidatorOrbsAddress(validator []byte, orbsAddress []byte) {
	state.WriteBytes(_formatValidatorOrbsAddressKey(validator), orbsAddress)
}

func _formatValidatorStakeKey(validator []byte) []byte {
	return []byte(fmt.Sprintf("Validator_%s_Stake", hex.EncodeToString(validator)))
}

func getValidatorStake(validator []byte) uint64 {
	return state.ReadUint64(_formatValidatorStakeKey(validator))
}

func _setValidatorStake(validator []byte, stake uint64) {
	state.WriteUint64(_formatValidatorStakeKey(validator), stake)
}

func _formatValidatorVoteKey(validator []byte) []byte {
	return []byte(fmt.Sprintf("Validator_%s_Vote", hex.EncodeToString(validator)))
}

func getValidatorVote(validator []byte) uint64 {
	return state.ReadUint64(_formatValidatorVoteKey(validator))
}

func _setValidatorVote(validator []byte, stake uint64) {
	state.WriteUint64(_formatValidatorVoteKey(validator), stake)
}
