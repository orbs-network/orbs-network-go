package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/ipfs"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

func TestIPFSWithLocalNode(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		t := harness.T
		h := NewIPFSDaemonHarness()
		h.StartDaemon()
		defer h.StopDaemon()

		err := h.AddFile(ExampleJSONPath())
		require.NoError(t, err)

		s := ipfs.NewIPFS(nil, harness.Logger)

		readme, err := ioutil.ReadFile(ExampleJSONPath())
		require.NoError(t, err)

		out, err := s.Read(context.Background(), &ipfs.IPFSReadInput{
			Hash: EXAMPLE_JSON_HASH,
		})

		require.NoError(t, err)
		require.EqualValues(t, readme, out.Content)
	})
}
