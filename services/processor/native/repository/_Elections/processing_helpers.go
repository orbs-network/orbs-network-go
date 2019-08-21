// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/safemath/safeuint64"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

/***
 * processing helper functions
 */
func isProcessingPeriod() uint32 {
	if hasProcessingStarted() == 1 {
		return 1
	}
	if _isTimeBasedElections() {
		if ethereum.GetBlockTime() > safeuint64.Add(getCurrentElectionTimeInNanos(), MIRROR_PERIOD_LENGTH_IN_NANOS) {
			return 1
		}
		return 0
	} else {
		return _isProcessingPeriodBlockBased()
	}
}

func hasProcessingStarted() uint32 {
	if len(_getVotingProcessState()) > 0 {
		return 1
	}
	return 0
}

func _formatProcessCurrentElectionBlockNumber() []byte {
	return []byte("Current_Election_Block_Number")
}

func _formatProcessCurrentElectionEarliestValidVoteBlockNumber() []byte {
	return []byte("Current_Election_Earliest_Vote")
}

func _formatProcessCurrentElectionTime() []byte {
	return []byte("Current_Election_Time")
}

func _getProcessCurrentElectionBlockNumber() uint64 {
	return state.ReadUint64(_formatProcessCurrentElectionBlockNumber())
}

func _getProcessCurrentElectionEarliestValidVoteBlockNumber() uint64 {
	return state.ReadUint64(_formatProcessCurrentElectionEarliestValidVoteBlockNumber())
}

func _getProcessCurrentElectionTime() uint64 {
	return state.ReadUint64(_formatProcessCurrentElectionTime())
}

func _setProcessCurrentElection(electionTime, electionBlockNumber, earliestValidVoteBlockNumber uint64) {
	state.WriteUint64(_formatProcessCurrentElectionBlockNumber(), electionBlockNumber)
	state.WriteUint64(_formatProcessCurrentElectionEarliestValidVoteBlockNumber(), earliestValidVoteBlockNumber)
	state.WriteUint64(_formatProcessCurrentElectionTime(), electionTime)
}

func _calculateProcessCurrentElectionValues() {
	if hasProcessingStarted() == 0 {
		var electionBlockTime, electionBlockNumber, earliestValidVoteBlockNumber uint64
		label := "time based"
		if _isTimeBasedElections() {
			electionBlockTime = getCurrentElectionTimeInNanos()
			electionBlockNumber = ethereum.GetBlockNumberByTime(electionBlockTime) + 1
			earliestValidVoteBlockNumber = ethereum.GetBlockNumberByTime(getCurrentElectionTimeInNanos()-VOTE_PERIOD_LENGTH_IN_NANOS) + 1
		} else {
			label = "block based"
			electionBlockNumber = getCurrentElectionBlockNumber()
			electionBlockTime = ethereum.GetBlockTimeByNumber(electionBlockNumber)
			earliestValidVoteBlockNumber = safeuint64.Sub(electionBlockNumber, VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS-1)
		}
		_setProcessCurrentElection(electionBlockTime, electionBlockNumber, earliestValidVoteBlockNumber)
		fmt.Printf("elections %10d: set %s election parameters: time is %d, block is %d, earliest valid vote block is %d\n", electionBlockNumber, label, electionBlockTime, electionBlockNumber, earliestValidVoteBlockNumber)
	}
}

func processTrigger() {
	fmt.Printf("elections : processTrigger called\n")
}
