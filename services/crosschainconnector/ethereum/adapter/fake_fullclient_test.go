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

	require.EqualValues(t, 1506108783, ffc.data[NUMBER_OF_BLOCKS-1], "expected ffc last block to be of specific ts")
}
