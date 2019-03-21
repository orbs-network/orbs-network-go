package elections_systemcontract

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

/***
 * Mirror vote
 */
type VoteOut struct {
	Voter      [20]byte
	Validators [][20]byte
}

func mirrorVote(hexEncodedEthTxHash string) {
	_mirrorPeriodValidator()
	e := &VoteOut{}
	eventBlockNumber, eventBlockTxIndex := ethereum.GetTransactionLog(getVotingEthereumContractAddress(), getVotingAbi(), hexEncodedEthTxHash, VOTE_OUT_NAME, e)
	if len(e.Validators) > MAX_CANDIDATE_VOTES {
		panic(fmt.Errorf("voteOut of guardian %v to %v failed since voted to too many (%d) candidate",
			e.Voter, e.Validators, len(e.Validators)))
	}
	isGuardian := false
	ethereum.CallMethodAtBlock(eventBlockNumber, getGuardiansEthereumContractAddress(), getGuardiansAbi(), "isGuardian", &isGuardian, e.Voter)
	if !isGuardian {
		panic(fmt.Errorf("voteOut of guardian %v to %v failed since it is not a guardian at blockNumber %d",
			e.Voter, e.Validators, eventBlockNumber))
	}

	electionBlockNumber := _getCurrentElectionBlockNumber()
	if eventBlockNumber > electionBlockNumber {
		panic(fmt.Errorf("voteOut of guardian %v to %v failed since it happened in block number %d which is after election date (%d), resubmit next election",
			e.Voter, e.Validators, eventBlockNumber, electionBlockNumber))
	}
	stateBlockNumber := state.ReadUint64(_formatVoterBlockNumberKey(e.Voter[:]))
	stateBlockTxIndex := state.ReadUint32(_formatVoterBlockTxIndexKey(e.Voter[:]))
	if stateBlockNumber > eventBlockNumber || (stateBlockNumber == eventBlockNumber && stateBlockTxIndex > eventBlockTxIndex) {
		panic(fmt.Errorf("voteOut of guardian %v to %v with block-height %d and tx-index %d failed since already have newer block-height %d and tx-index %d",
			e.Voter, e.Validators, eventBlockNumber, eventBlockTxIndex, stateBlockNumber, stateBlockTxIndex))
	}

	if stateBlockNumber == 0 { // new voter
		numOfVoters := _getNumberOfVoters()
		_setVoterAtIndex(numOfVoters, e.Voter[:])
		_setNumberOfVoters(numOfVoters + 1)
	}

	fmt.Printf("elections %10d: guardian %x voted against (%d) %v\n", _getCurrentElectionBlockNumber(), e.Voter, len(e.Validators), e.Validators)
	_setCandidates(e.Voter[:], e.Validators)
	state.WriteUint64(_formatVoterBlockNumberKey(e.Voter[:]), eventBlockNumber)
	state.WriteUint32(_formatVoterBlockTxIndexKey(e.Voter[:]), eventBlockTxIndex)
}

/***
 * Voters - Data struct
 */
func _formatNumberOfVoters() []byte {
	return []byte("Voter_Address_Count")
}

func _getNumberOfVoters() int {
	return int(state.ReadUint32(_formatNumberOfVoters()))
}

func _setNumberOfVoters(numberOfVoters int) {
	state.WriteUint32(_formatNumberOfVoters(), uint32(numberOfVoters))
}

func _formatVoterIterator(num int) []byte {
	return []byte(fmt.Sprintf("Voter_Address_%d", num))
}

func _getVoterAtIndex(index int) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatVoterIterator(index)))
}

func _setVoterAtIndex(index int, voter []byte) {
	state.WriteBytes(_formatVoterIterator(index), voter)
}

func _formatVoterCandidateKey(voter []byte) []byte {
	return []byte(fmt.Sprintf("Voter_%s_Candidates", hex.EncodeToString(voter)))
}

func _getCandidates(voter []byte) [][20]byte {
	candidates := state.ReadBytes(_formatVoterCandidateKey(voter))
	numCandidate := len(candidates) / 20
	candidatesList := make([][20]byte, numCandidate)
	for i := 0; i < numCandidate; i++ {
		copy(candidatesList[i][:], candidates[i*20:i*20+20])
	}
	return candidatesList
}

func _setCandidates(voter []byte, candidateList [][20]byte) {
	candidates := make([]byte, 0, len(candidateList)*20)
	for _, v := range candidateList {
		candidates = append(candidates, v[:]...)
	}

	state.WriteBytes(_formatVoterCandidateKey(voter), candidates)
}

func _formatVoterBlockNumberKey(voter []byte) []byte {
	return []byte(fmt.Sprintf("Voter_%s_BlockNumber", hex.EncodeToString(voter)))
}

func _formatVoterBlockTxIndexKey(voter []byte) []byte {
	return []byte(fmt.Sprintf("Voter_%s_BlockTxIndex", hex.EncodeToString(voter)))
}

func _formatVoterStakeKey(voter []byte) []byte {
	return []byte(fmt.Sprintf("Voter_%s_Stake", hex.EncodeToString(voter)))
}

func _getVoterStake(voter []byte) uint64 {
	return state.ReadUint64(_formatVoterStakeKey(voter))
}

func _setVoterStake(voter []byte, stake uint64) {
	state.WriteUint64(_formatVoterStakeKey(voter), stake)
}
