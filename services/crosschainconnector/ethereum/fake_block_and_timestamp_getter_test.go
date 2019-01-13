package ethereum

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// this test exists to make sure that the fake timestamp/block pairs remain constant, as other tests in the system (such as header_by_timestamp_finder_test.go) rely on these constant numbers
func TestFakeBlockHeaderFetcher(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	ffc := NewFakeBlockAndTimestampGetter(logger)

	require.EqualValues(t, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED, ffc.data[FAKE_CLIENT_NUMBER_OF_BLOCKS-1], "expected ffc last block to be of specific ts")
}
