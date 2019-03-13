package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"testing"
)

// this test exists to make sure that the fake timestamp/block pairs remain constant, as other tests in the system (such as header_by_timestamp_finder_test.go) rely on these constant numbers
func TestFakeBlockHeaderFetcherRawData(t *testing.T) {
	ffc := NewFakeBlockAndTimestampGetter(log.DefaultTestingLogger(t))

	require.EqualValues(t, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED, ffc.data[FAKE_CLIENT_NUMBER_OF_BLOCKS], "expected ffc last block to be of specific ts")
}

func TestFakeBlockHeaderFetcherOfLatest(t *testing.T) {
	ffc := NewFakeBlockAndTimestampGetter(log.DefaultTestingLogger(t))

	b, err := ffc.ApproximateBlockAt(context.Background(), nil)
	require.NoError(t, err, "should not fail getting 'latest' from fake db")
	require.EqualValues(t, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED, b.TimeSeconds, "expected ffc last block to be of specific ts")
	require.EqualValues(t, FAKE_CLIENT_NUMBER_OF_BLOCKS, b.Number, "expected last block of constant number")
}

func TestSameTimeBlocks(t *testing.T) {
	ffc := NewFakeBlockAndTimestampGetter(log.DefaultTestingLogger(t)).WithSameTimeBlocks(0.95)

	dups := make(map[int64]int)
	for _, t := range ffc.data {
		dups[t]++
	}

	maxCount := 1
	for _, c := range dups {
		if c > 1 {
			maxCount = c
			break
		}
	}

	require.EqualValues(t, FAKE_CLIENT_NUMBER_OF_BLOCKS*0.95, maxCount, "expected number of same timestamp mismatch")
}
