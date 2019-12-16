// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build !unsafetests

package elections_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
)

var PUBLIC = sdk.Export(getTokenEthereumContractAddress, getGuardiansEthereumContractAddress, getVotingEthereumContractAddress, getValidatorsEthereumContractAddress, getValidatorsRegistryEthereumContractAddress,
	mirrorDelegationByTransfer, mirrorDelegation,
	processVoting, isProcessingPeriod, hasProcessingStarted, processTrigger,
	getNumberOfElections, isElectionOverdue,
	getElectionPeriodInNanos, getEffectiveElectionTimeInNanos, getCurrentElectionTimeInNanos, getNextElectionTimeInNanos,
	getElectedValidatorsOrbsAddress, getElectedValidatorsEthereumAddress, getElectedValidatorsEthereumAddressByBlockNumber, getElectedValidatorsOrbsAddressByBlockHeight,
	getElectedValidatorsOrbsAddressByIndex, getElectedValidatorsEthereumAddressByIndex, getElectedValidatorsBlockNumberByIndex, getElectedValidatorsBlockHeightByIndex,
	getCumulativeParticipationReward, getCumulativeGuardianExcellenceReward, getCumulativeValidatorReward,
	getGuardianStake, getGuardianVotingWeight, getTotalStake, getValidatorStake, getValidatorVote, getExcellenceProgramGuardians,
	getCurrentEthereumBlockNumber,

	_fixRewardsDrift9108900,

	// feature flag
	switchToTimeBasedElections,

	// block based
	getElectionPeriod, getCurrentElectionBlockNumber, getNextElectionBlockNumber, getEffectiveElectionBlockNumber,
	getProcessingStartBlockNumber, getMirroringEndBlockNumber,
)
var SYSTEM = sdk.Export(_init)
