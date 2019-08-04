package elections_systemcontract

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/safemath/safeuint64"
)

func getElectionPeriod() uint64 {
	if _isTimeBasedElections() {
		panic(fmt.Sprintf("Election priod time in nanoseconds: %d", getElectionPeriodInNanos()))
	}
	return ELECTION_PERIOD_LENGTH_IN_BLOCKS
}

func getCurrentElectionBlockNumber() uint64 {
	if _isTimeBasedElections() {
		panic(fmt.Sprintf("Current election time in nanoseconds: %d", getCurrentElectionTimeInNanos()))
	}
	return safeuint64.Add(getEffectiveElectionBlockNumber(), getElectionPeriod())
}

func getNextElectionBlockNumber() uint64 {
	if _isTimeBasedElections() {
		panic(fmt.Sprintf("Next election time in nanoseconds: %d", getNextElectionTimeInNanos()))
	}
	return safeuint64.Add(getCurrentElectionBlockNumber(), getElectionPeriod())
}

func getCurrentEthereumBlockNumber() uint64 {
	return ethereum.GetBlockNumber()
}

func getProcessingStartBlockNumber() uint64 {
	if _isTimeBasedElections() {
		panic(fmt.Sprintf("Processing start time in nanoseconds: %d", safeuint64.Add(getCurrentElectionTimeInNanos(), MIRROR_PERIOD_LENGTH_IN_NANOS)))
	}
	return safeuint64.Add(getCurrentElectionBlockNumber(), VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS)
}

func getMirroringEndBlockNumber() uint64 {
	if _isTimeBasedElections() {
		panic(fmt.Sprintf("Mirroring end time in nanoseconds: %d", safeuint64.Add(getCurrentElectionTimeInNanos(), MIRROR_PERIOD_LENGTH_IN_NANOS)))
	}
	return safeuint64.Add(getCurrentElectionBlockNumber(), VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS)
}

func _isProcessingPeriodBlockBased() uint32 {
	currentBlockNumber := getCurrentEthereumBlockNumber()
	processStartBlockNumber := getProcessingStartBlockNumber()

	if currentBlockNumber >= processStartBlockNumber {
		return 1
	}
	return 0
}

func _isElectionOverdueBlockBased() uint32 {
	processStartBlockNumber := getProcessingStartBlockNumber()
	currentBlockNumber := getCurrentEthereumBlockNumber()

	if processStartBlockNumber == 0 || currentBlockNumber >= safeuint64.Add(processStartBlockNumber, 600) {
		return 1
	}
	return 0
}

func _initCurrentElectionBlockNumber() {
	currentElectionBlockNumber := getEffectiveElectionBlockNumber()
	if currentElectionBlockNumber == 0 {
		currBlock := getCurrentEthereumBlockNumber()
		nextElectionBlock := FIRST_ELECTION_BLOCK
		if currBlock > FIRST_ELECTION_BLOCK {
			blocksSinceFirstEver := safeuint64.Sub(currBlock, FIRST_ELECTION_BLOCK)
			blocksSinceStartOfAnElection := safeuint64.Mod(blocksSinceFirstEver, getElectionPeriod())
			blocksUntilNextElection := safeuint64.Sub(getElectionPeriod(), blocksSinceStartOfAnElection)
			nextElectionBlock = safeuint64.Add(currBlock, blocksUntilNextElection)
		}
		_setElectedValidatorsBlockNumberAtIndex(0, safeuint64.Sub(nextElectionBlock, getElectionPeriod()))
	}
}
