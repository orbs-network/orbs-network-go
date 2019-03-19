package elections_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsVotingContract_mirrorVote(t *testing.T) {
	txHex := "0xabcd"
	guardianAddr := [20]byte{0x01}
	candidateAddrs := [][20]byte{{0x02}, {0x03}, {0x04}}
	blockNumber := 100
	txIndex := 10

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setTimingInMirror(m)

		// prepare
		m.MockEthereumLog(getVotingEthereumContractAddress(), getVotingAbi(), txHex, VOTE_OUT_NAME, blockNumber, txIndex, func(out interface{}) {
			v := out.(*VoteOut)
			v.Voter = guardianAddr
			v.Validators = candidateAddrs
		})
		mockGuardianInEthereum(m, uint64(blockNumber), guardianAddr, true)

		// call
		mirrorVote(txHex)

		// assert
		m.VerifyMocks()
		candidates := make([]byte, 0, len(candidateAddrs)*20)
		for _, v := range candidateAddrs {
			candidates = append(candidates, v[:]...)
		}

		require.EqualValues(t, candidates, state.ReadBytes(_formatGuardianCandidateKey(guardianAddr[:])))
		require.EqualValues(t, blockNumber, state.ReadUint64(_formatGuardianBlockNumberKey(guardianAddr[:])))
		require.EqualValues(t, txIndex, state.ReadUint32(_formatGuardianBlockTxIndexKey(guardianAddr[:])))
	})
}

func TestOrbsVotingContract_mirrorVoteLessThanMaximum(t *testing.T) {
	txHex := "0xabcd"
	guardianAddr := [20]byte{0x01}
	candidateAddrs := [][20]byte{{0x02}, {0x03}}
	blockNumber := 100
	txIndex := 10

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setTimingInMirror(m)

		// prepare
		m.MockEthereumLog(getVotingEthereumContractAddress(), getVotingAbi(), txHex, VOTE_OUT_NAME, blockNumber, txIndex, func(out interface{}) {
			v := out.(*VoteOut)
			v.Voter = guardianAddr
			v.Validators = candidateAddrs
		})
		mockGuardianInEthereum(m, uint64(blockNumber), guardianAddr, true)

		// call
		mirrorVote(txHex)

		// assert
		m.VerifyMocks()
		candidates := make([]byte, 0, len(candidateAddrs)*20)
		for _, v := range candidateAddrs {
			candidates = append(candidates, v[:]...)
		}

		require.EqualValues(t, candidates, state.ReadBytes(_formatGuardianCandidateKey(guardianAddr[:])))
		require.EqualValues(t, blockNumber, state.ReadUint64(_formatGuardianBlockNumberKey(guardianAddr[:])))
		require.EqualValues(t, txIndex, state.ReadUint32(_formatGuardianBlockTxIndexKey(guardianAddr[:])))
	})
}

func TestOrbsVotingContract_mirrorVote_NotGuardian(t *testing.T) {
	txHex := "0xabcd"
	guardianAddr := [20]byte{0x01}
	candidateAddrs := [][20]byte{{0x02}, {0x03}, {0x04}}
	blockNumber := 100
	txIndex := 10

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setTimingInMirror(m)

		// prepare
		m.MockEthereumLog(getVotingEthereumContractAddress(), getVotingAbi(), txHex, VOTE_OUT_NAME, blockNumber, txIndex, func(out interface{}) {
			v := out.(*VoteOut)
			v.Voter = guardianAddr
			v.Validators = candidateAddrs
		})
		mockGuardianInEthereum(m, uint64(blockNumber), guardianAddr, false)

		require.Panics(t, func() {
			mirrorVote(txHex)
		}, "should panic because not guardian")
	})
}

func TestOrbsVotingContract_mirrorVote_NoCandidates(t *testing.T) {
	txHex := "0xabcd"
	guardianAddr := [20]byte{0x01}
	candidateAddrs := make([][20]byte, 0)
	blockNumber := 100
	txIndex := 10

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setTimingInMirror(m)

		// prepare
		m.MockEthereumLog(getVotingEthereumContractAddress(), getVotingAbi(), txHex, VOTE_OUT_NAME, blockNumber, txIndex, func(out interface{}) {
			v := out.(*VoteOut)
			v.Voter = guardianAddr
			v.Validators = candidateAddrs
		})
		mockGuardianInEthereum(m, uint64(blockNumber), guardianAddr, true)

		mirrorVote(txHex)

		// assert
		m.VerifyMocks()
		candidates := make([]byte, 0, len(candidateAddrs)*20)
		for _, v := range candidateAddrs {
			candidates = append(candidates, v[:]...)
		}

		require.EqualValues(t, candidates, state.ReadBytes(_formatGuardianCandidateKey(guardianAddr[:])))
		require.EqualValues(t, blockNumber, state.ReadUint64(_formatGuardianBlockNumberKey(guardianAddr[:])))
		require.EqualValues(t, txIndex, state.ReadUint32(_formatGuardianBlockTxIndexKey(guardianAddr[:])))
	})
}

func TestOrbsVotingContract_mirrorVote_TooManyCandidates(t *testing.T) {
	txHex := "0xabcd"
	guardianAddr := [20]byte{0x01}
	candidateAddrs := [][20]byte{{0x02}, {0x03}, {0x04}, {0x05}}
	blockNumber := 100
	txIndex := 10

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setTimingInMirror(m)

		// prepare
		m.MockEthereumLog(getVotingEthereumContractAddress(), getVotingAbi(), txHex, VOTE_OUT_NAME, blockNumber, txIndex, func(out interface{}) {
			v := out.(*VoteOut)
			v.Voter = guardianAddr
			v.Validators = candidateAddrs
		})

		require.Panics(t, func() {
			mirrorVote(txHex)
		}, "should panic because too many candidates")
	})
}

func TestOrbsVotingContract_mirrorVote_AlreadyHaveNewerEventBlockNumber(t *testing.T) {
	txHex := "0xabcd"
	guardianAddr := [20]byte{0x01}
	candidateAddrs := [][20]byte{{0x02}, {0x03}, {0x04}}
	blockNumber := 100

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setTimingInMirror(m)

		// prepare
		state.WriteUint64(_formatGuardianBlockNumberKey(guardianAddr[:]), 101)
		m.MockEthereumLog(getVotingEthereumContractAddress(), getVotingAbi(), txHex, VOTE_OUT_NAME, blockNumber, 10, func(out interface{}) {
			v := out.(*VoteOut)
			v.Voter = guardianAddr
			v.Validators = candidateAddrs
		})
		mockGuardianInEthereum(m, uint64(blockNumber), guardianAddr, true)

		require.Panics(t, func() {
			mirrorVote(txHex)
		}, "should panic because newer block")
	})
}

func TestOrbsVotingContract_mirrorVote_AlreadyHaveNewerEventBlockTxIndex(t *testing.T) {
	txHex := "0xabcd"
	guardianAddr := [20]byte{0x01}
	candidateAddrs := [][20]byte{{0x02}, {0x03}, {0x04}}
	blockNumber := 100

	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setTimingInMirror(m)

		// prepare
		state.WriteUint64(_formatGuardianBlockNumberKey(guardianAddr[:]), uint64(blockNumber))
		state.WriteUint64(_formatGuardianBlockTxIndexKey(guardianAddr[:]), 50)
		m.MockEthereumLog(getVotingEthereumContractAddress(), getVotingAbi(), txHex, VOTE_OUT_NAME, blockNumber, 10, func(out interface{}) {
			v := out.(*VoteOut)
			v.Voter = guardianAddr
			v.Validators = candidateAddrs
		})
		mockGuardianInEthereum(m, uint64(blockNumber), guardianAddr, true)

		require.Panics(t, func() {
			mirrorVote(txHex)
		}, "should panic because newer block")
	})
}

func TestOrbsVotingContract_mirrorVote_NotDueDiligent(t *testing.T) {
}
