// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build unsafetests

package elections_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
)

var PUBLIC = sdk.Export(getTokenEthereumContractAddress, getGuardiansEthereumContractAddress, getVotingEthereumContractAddress, getValidatorsEthereumContractAddress, getValidatorsRegistryEthereumContractAddress,
	unsafetests_setTokenEthereumContractAddress, unsafetests_setGuardiansEthereumContractAddress,
	unsafetests_setVotingEthereumContractAddress, unsafetests_setValidatorsEthereumContractAddress, unsafetests_setValidatorsRegistryEthereumContractAddress,
	unsafetests_setVariables, unsafetests_setElectedValidators, unsafetests_setElectedBlockNumber,
	mirrorDelegationByTransfer, mirrorDelegation,
	processVoting,
	getElectionPeriod, getCurrentElectionBlockNumber, getNextElectionBlockNumber, getEffectiveElectionBlockNumber, getNumberOfElections,
	getCurrentEthereumBlockNumber, getProcessingStartBlockNumber, getMirroringEndBlockNumber,
	getElectedValidatorsOrbsAddress, getElectedValidatorsEthereumAddress, getElectedValidatorsEthereumAddressByBlockNumber, getElectedValidatorsOrbsAddressByBlockHeight,
	getElectedValidatorsOrbsAddressByIndex, getElectedValidatorsEthereumAddressByIndex, getElectedValidatorsBlockNumberByIndex, getElectedValidatorsBlockHeightByIndex,
	getCumulativeParticipationReward, getCumulativeGuardianExcellenceReward, getCumulativeValidatorReward,
	getGuardianStake, getGuardianVotingWeight, getTotalStake, getValidatorStake, getValidatorVote, getExcellenceProgramGuardians,
)
var SYSTEM = sdk.Export(_init)

/***
 * unsafetests functions
 */
func unsafetests_setVariables(voteMirrorPeriod uint64, voteValidPeriod uint64, electionPeriod uint64, maxElectedValidators uint32, minElectedValidators uint32) {
	VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS = voteMirrorPeriod
	VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS = voteValidPeriod
	ELECTION_PERIOD_LENGTH_IN_BLOCKS = electionPeriod
	MAX_ELECTED_VALIDATORS = int(maxElectedValidators)
	MIN_ELECTED_VALIDATORS = int(minElectedValidators)
}

func unsafetests_setElectedValidators(joinedAddresses []byte) {
	index := getNumberOfElections()
	_setElectedValidatorsOrbsAddressAtIndex(index, joinedAddresses)
}

func unsafetests_setElectedBlockNumber(blockNumber uint64) {
	_setCurrentElectionBlockNumber(blockNumber)
}

func unsafetests_setTokenEthereumContractAddress(addr string) {
	ETHEREUM_TOKEN_ADDR = addr
}

func unsafetests_setVotingEthereumContractAddress(addr string) {
	ETHEREUM_VOTING_ADDR = addr
}

func unsafetests_setValidatorsEthereumContractAddress(addr string) {
	ETHEREUM_VALIDATORS_ADDR = addr
}

func unsafetests_setValidatorsRegistryEthereumContractAddress(addr string) {
	ETHEREUM_VALIDATORS_REGISTRY_ADDR = addr
}

func unsafetests_setGuardiansEthereumContractAddress(addr string) {
	ETHEREUM_GUARDIANS_ADDR = addr
}
