package elections_systemcontract

import (
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsVotingContract_initCurrentElection(t *testing.T) {
	tests := []struct {
		name              string
		expectCurrentTime uint64
		ethereumBlockTime uint64
	}{
		{"before is 0", FIRST_ELECTION_TIME_IN_NANOS, 0},
		{"before is a smaller than first", FIRST_ELECTION_TIME_IN_NANOS, 1569920000000000000},
		{"before is after first but before second", FIRST_ELECTION_TIME_IN_NANOS + ELECTION_PERIOD_LENGTH_IN_NANOS, FIRST_ELECTION_TIME_IN_NANOS + 500000},
		{"before is after second", FIRST_ELECTION_TIME_IN_NANOS + 2*ELECTION_PERIOD_LENGTH_IN_NANOS, FIRST_ELECTION_TIME_IN_NANOS + ELECTION_PERIOD_LENGTH_IN_NANOS + 50000},
	}
	for i := range tests {
		cTest := tests[i]
		t.Run(cTest.name, func(t *testing.T) {
			InServiceScope(nil, nil, func(m Mockery) {
				_init()
				m.MockEthereumGetBlockTime(int(cTest.ethereumBlockTime))
				switchToTimeBasedElections()
				_initCurrentElection()
				after := getCurrentElectionTimeInNanos()
				require.EqualValues(t, cTest.expectCurrentTime, after, "'%s' failed ", cTest.name)
			})
		})
	}
}
