package elections_systemcontract

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
)

/***
 * Helpers
 */
func _addressSliceToArray(a []byte) [20]byte {
	var array [20]byte
	copy(array[:], a)
	return array
}

func _isAfterElectionMirroring(blockNumber uint64) bool {
	return blockNumber > getMirroringEndBlockNumber()
}

func _mirrorPeriodValidator() {
	currentBlock := ethereum.GetBlockNumber()
	if _getVotingProcessState() != "" && _isAfterElectionMirroring(currentBlock) {
		panic(fmt.Errorf("current block number (%d) indicates mirror period for election (%d) has ended, resubmit next election", currentBlock, _getCurrentElectionBlockNumber()))
	}
}
