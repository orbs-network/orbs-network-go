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
	stateBlockNumber := state.ReadUint64(_formatGuardianBlockNumberKey(e.Voter[:]))
	stateBlockTxIndex := state.ReadUint32(_formatGuardianBlockTxIndexKey(e.Voter[:]))
	if stateBlockNumber > eventBlockNumber || (stateBlockNumber == eventBlockNumber && stateBlockTxIndex > eventBlockTxIndex) {
		panic(fmt.Errorf("voteOut of guardian %v to %v with block-height %d and tx-index %d failed since already have newer block-height %d and tx-index %d",
			e.Voter, e.Validators, eventBlockNumber, eventBlockTxIndex, stateBlockNumber, stateBlockTxIndex))
	}

	if stateBlockNumber == 0 { // new guardian
		numOfGuardians := _getNumberOfGurdians()
		_setGuardianAtIndex(numOfGuardians, e.Voter[:])
		_setNumberOfGurdians(numOfGuardians + 1)
	}

	fmt.Printf("elections %10d: guardian %x voted against (%d) %v\n", _getCurrentElectionBlockNumber(), e.Voter, len(e.Validators), e.Validators)
	_setCandidates(e.Voter[:], e.Validators)
	state.WriteUint64(_formatGuardianBlockNumberKey(e.Voter[:]), eventBlockNumber)
	state.WriteUint32(_formatGuardianBlockTxIndexKey(e.Voter[:]), eventBlockTxIndex)
}

/***
 * Guardians - Data struct
 */
func _formatNumberOfGuardians() []byte {
	return []byte("Guardian_Address_Count")
}

func _getNumberOfGurdians() int {
	return int(state.ReadUint32(_formatNumberOfGuardians()))
}

func _setNumberOfGurdians(numberOfGuardians int) {
	state.WriteUint32(_formatNumberOfGuardians(), uint32(numberOfGuardians))
}

func _formatGuardianIterator(num int) []byte {
	return []byte(fmt.Sprintf("Guardian_Address_%d", num))
}

func _getGuardianAtIndex(index int) [20]byte {
	return _addressSliceToArray(state.ReadBytes(_formatGuardianIterator(index)))
}

func _setGuardianAtIndex(index int, guardian []byte) {
	state.WriteBytes(_formatGuardianIterator(index), guardian)
}

func _formatGuardianCandidateKey(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s_Candidates", hex.EncodeToString(guardian)))
}

func _getCandidates(guardian []byte) [][20]byte {
	candidates := state.ReadBytes(_formatGuardianCandidateKey(guardian))
	numCandidate := len(candidates) / 20
	candidatesList := make([][20]byte, numCandidate)
	for i := 0; i < numCandidate; i++ {
		copy(candidatesList[i][:], candidates[i*20:i*20+20])
	}
	return candidatesList
}

func _setCandidates(guardian []byte, candidateList [][20]byte) {
	candidates := make([]byte, 0, len(candidateList)*20)
	for _, v := range candidateList {
		candidates = append(candidates, v[:]...)
	}

	state.WriteBytes(_formatGuardianCandidateKey(guardian), candidates)
}

func _formatGuardianBlockNumberKey(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s_BlockNumber", hex.EncodeToString(guardian)))
}

func _formatGuardianBlockTxIndexKey(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s_BlockTxIndex", hex.EncodeToString(guardian)))
}

func _formatGuardianStakeKey(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s_Stake", hex.EncodeToString(guardian)))
}

func getGuardianStake(guardian []byte) uint64 {
	return state.ReadUint64(_formatGuardianStakeKey(guardian))
}

func _setGuardianStake(guardian []byte, stake uint64) {
	state.WriteUint64(_formatGuardianStakeKey(guardian), stake)
}

func _formatGuardianVoteWeightKey(guardian []byte) []byte {
	return []byte(fmt.Sprintf("Guardian_%s_Weight", hex.EncodeToString(guardian)))
}

func getGuardianVotingWeight(guardian []byte) uint64 {
	return state.ReadUint64(_formatGuardianVoteWeightKey(guardian))
}

func _setGuardianVotingWeight(guardian []byte, weight uint64) {
	state.WriteUint64(_formatGuardianVoteWeightKey(guardian), weight)
}
