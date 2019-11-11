// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build unsafetests

package elections_systemcontract

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/env"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/safemath/safeuint64"
	"time"
)

var PUBLIC = sdk.Export(getTokenEthereumContractAddress, getGuardiansEthereumContractAddress, getVotingEthereumContractAddress, getValidatorsEthereumContractAddress, getValidatorsRegistryEthereumContractAddress,
	unsafetests_setTokenEthereumContractAddress, unsafetests_setGuardiansEthereumContractAddress,
	unsafetests_setVotingEthereumContractAddress, unsafetests_setValidatorsEthereumContractAddress, unsafetests_setValidatorsRegistryEthereumContractAddress,
	unsafetests_setVariables, unsafetests_setElectedValidators, unsafetests_setCurrentElectedBlockNumber,
	unsafetests_setCurrentElectionTimeNanos, unsafetests_setElectionMirrorPeriodInSeconds, unsafetests_setElectionVotePeriodInSeconds, unsafetests_setElectionPeriodInSeconds,
	mirrorDelegationByTransfer, mirrorDelegation,
	processVoting, isProcessingPeriod, hasProcessingStarted, processTrigger,
	getElectionPeriod, getCurrentElectionBlockNumber, getNextElectionBlockNumber, getEffectiveElectionBlockNumber, getNumberOfElections,
	getElectionPeriodInNanos, getEffectiveElectionTimeInNanos, getCurrentElectionTimeInNanos, getNextElectionTimeInNanos,
	getCurrentEthereumBlockNumber, getProcessingStartBlockNumber, isElectionOverdue, getMirroringEndBlockNumber,
	getElectedValidatorsOrbsAddress, getElectedValidatorsEthereumAddress, getElectedValidatorsEthereumAddressByBlockNumber, getElectedValidatorsOrbsAddressByBlockHeight,
	getElectedValidatorsOrbsAddressByIndex, getElectedValidatorsEthereumAddressByIndex, getElectedValidatorsBlockNumberByIndex, getElectedValidatorsBlockHeightByIndex,
	getCumulativeParticipationReward, getCumulativeGuardianExcellenceReward, getCumulativeValidatorReward,
	getGuardianStake, getGuardianVotingWeight, getTotalStake, getValidatorStake, getValidatorVote, getExcellenceProgramGuardians,
	switchToTimeBasedElections,
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
	if index == 0 {
		index = 1
	}
	_setNumberOfElections(index)
	_setElectedValidatorsOrbsAddressAtIndex(index, joinedAddresses)
	_setElectedValidatorsBlockHeightAtIndex(index, env.GetBlockHeight()-1) // so that election is valid from this block
}

func unsafetests_setCurrentElectedBlockNumber(blockNumber uint64) {
	_setElectedValidatorsBlockNumberAtIndex(getNumberOfElections(), safeuint64.Sub(blockNumber, getElectionPeriod()))
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

func unsafetests_setCurrentElectionTimeNanos(time uint64) {
	fmt.Printf("elections : set electiontime to %d period %d\n", time, getElectionPeriodInNanos())
	_setElectedValidatorsTimeInNanosAtIndex(getNumberOfElections(), safeuint64.Sub(time, getElectionPeriodInNanos()))
	fmt.Printf("elections : compare to current block %d, current block time :%d\n", ethereum.GetBlockNumber(), ethereum.GetBlockTime())
}

func unsafetests_setElectionMirrorPeriodInSeconds(period uint64) {
	MIRROR_PERIOD_LENGTH_IN_NANOS = period * uint64(time.Second.Nanoseconds())
}

func unsafetests_setElectionVotePeriodInSeconds(period uint64) {
	VOTE_PERIOD_LENGTH_IN_NANOS = period * uint64(time.Second.Nanoseconds())
}

func unsafetests_setElectionPeriodInSeconds(period uint64) {
	ELECTION_PERIOD_LENGTH_IN_NANOS = period * uint64(time.Second.Nanoseconds())
}
