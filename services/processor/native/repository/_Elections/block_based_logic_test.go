package elections_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/safemath/safeuint64"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsVotingContract_initCurrentElectionBlockNumber(t *testing.T) {
	tests := []struct {
		name                     string
		expectCurrentBlockNumber uint64
		ethereumBlockNumber      uint64
	}{
		{"before is 0", FIRST_ELECTION_BLOCK, 0},
		{"before is a small number", FIRST_ELECTION_BLOCK, 5000000},
		{"before is after first but before second", FIRST_ELECTION_BLOCK + ELECTION_PERIOD_LENGTH_IN_BLOCKS, FIRST_ELECTION_BLOCK + 5000},
		{"before is after second", FIRST_ELECTION_BLOCK + 2*ELECTION_PERIOD_LENGTH_IN_BLOCKS, FIRST_ELECTION_BLOCK + ELECTION_PERIOD_LENGTH_IN_BLOCKS + 5000},
	}
	for i := range tests {
		cTest := tests[i]
		t.Run(cTest.name, func(t *testing.T) {
			InServiceScope(nil, nil, func(m Mockery) {
				_init()
				m.MockEthereumGetBlockNumber(int(cTest.ethereumBlockNumber))
				_initCurrentElectionBlockNumber()
				after := getCurrentElectionBlockNumber()
				require.EqualValues(t, cTest.expectCurrentBlockNumber, after, "'%s' failed ", cTest.name)
			})
		})
	}
}

func TestOrbsVotingContract_blockBased_IsProccessPeriod_yes(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		m.MockEthereumGetBlockNumber(100 + int(ELECTION_PERIOD_LENGTH_IN_BLOCKS+VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS) + 1)
		_setCurrentElectionBlockNumber_InTests(100 + ELECTION_PERIOD_LENGTH_IN_BLOCKS)

		require.EqualValues(t, 1, _isProcessingPeriodBlockBased(), "should be process period (1)")
	})
}

func TestOrbsVotingContract_blockBased_IsProccessPeriod_no(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		m.MockEthereumGetBlockNumber(100 + int(ELECTION_PERIOD_LENGTH_IN_BLOCKS+VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS) - 1)
		_setCurrentElectionBlockNumber_InTests(100 + ELECTION_PERIOD_LENGTH_IN_BLOCKS)

		require.EqualValues(t, 0, _isProcessingPeriodBlockBased(), "should not be process period (0)")
	})
}

func TestOrbsElectionResultsContract_getEffectiveElectionBlockNumber_emptyElection(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()

		// call
		b := getEffectiveElectionBlockNumber()

		// assert
		require.EqualValues(t, 0, b)
	})
}

func TestOrbsElectionResultsContract_getEffectiveElectionBlockNumber(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		setPastElection(1, 10000, 10000, 50, []byte{}, []byte{})
		_setNumberOfElections(1)

		// call
		b := getEffectiveElectionBlockNumber()

		// assert
		require.EqualValues(t, 10000, b)
	})
}

func TestOrbsElectionResultsContract_isElectionOverDueBlockNumber_Yes(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		m.MockEthereumGetBlockNumber(500000 + int(VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS) + 601)
		_setCurrentElectionBlockNumber_InTests(500000)

		// call
		b := _isElectionOverdueBlockBased()

		// assert
		require.EqualValues(t, 1, b)
	})
}

func TestOrbsElectionResultsContract_isElectionOverDueBlockNumber_No(t *testing.T) {
	InServiceScope(nil, nil, func(m Mockery) {
		_init()
		m.MockEthereumGetBlockNumber(500000 + int(VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS) + 599)
		_setCurrentElectionBlockNumber_InTests(500000)

		// call
		b := _isElectionOverdueBlockBased()

		// assert
		require.EqualValues(t, 0, b)
	})
}

func _setCurrentElectionBlockNumber_InTests(blockNumber uint64) {
	_setElectedValidatorsBlockNumberAtIndex(getNumberOfElections(), safeuint64.Sub(blockNumber, getElectionPeriod()))
}
