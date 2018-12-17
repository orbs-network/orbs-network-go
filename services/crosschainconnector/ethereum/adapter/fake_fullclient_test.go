package adapter

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestFakeFullClientInit(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	ffc := NewFakeFullClientConnection(logger)

	require.EqualValues(t, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED, ffc.data[FAKE_CLIENT_NUMBER_OF_BLOCKS-1], "expected ffc last block to be of specific ts")
}
