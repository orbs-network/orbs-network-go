package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"testing"
)

// this test exists to make sure that the fake timestamp/block pairs remain constant, as other tests in the system (such as header_by_timestamp_finder_test.go) rely on these constant numbers
func TestFakeBlockHeaderFetcherRawData(t *testing.T) {
	btg := NewFakeBlockTimeGetter(log.DefaultTestingLogger(t))

	require.EqualValues(t, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS, btg.data[FAKE_CLIENT_NUMBER_OF_BLOCKS], "expected getter last block to be of specific ts")
}

func TestFakeBlockHeaderFetcherOfLatest(t *testing.T) {
	btg := NewFakeBlockTimeGetter(log.DefaultTestingLogger(t))

	b, err := btg.GetTimestampForLatestBlock(context.Background())
	require.NoError(t, err, "should not fail getting 'latest' from fake db")
	require.EqualValues(t, secondsToNano(FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS), b.BlockTimeNano, "expected getter last block to be of specific ts")
	require.EqualValues(t, FAKE_CLIENT_NUMBER_OF_BLOCKS, b.BlockNumber, "expected last block of constant number")
}
