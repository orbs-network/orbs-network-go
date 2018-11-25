package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type testConfig struct {
	timeout time.Duration
}

func (c *testConfig) NodePublicKey() primitives.Ed25519PublicKey {
	panic("implement me")
}

func (c *testConfig) NodePrivateKey() primitives.Ed25519PrivateKey {
	panic("implement me")
}

func (c *testConfig) LeanHelixConsensusRoundTimeoutInterval() time.Duration {
	return c.timeout
}

func (c *testConfig) ActiveConsensusAlgo() consensus.ConsensusAlgoType {
	return consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX
}

type testGossip struct{}

func (g *testGossip) SendLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	panic("implement me")
}

func (g *testGossip) RegisterLeanHelixHandler(handler gossiptopics.LeanHelixHandler) {

}

// TODO Extract a harness out of this mess after 3+ tests are written
func TestMessageRegistration(t *testing.T) {
	//t.Skipf("Skipped till NodePublicKey() is implemented")

	comm := NewNetworkCommunication(&testGossip{})

	f := func(ctx context.Context, message leanhelix.ConsensusRawMessage) {}
	g := func(ctx context.Context, message leanhelix.ConsensusRawMessage) {}

	unregisterToken1 := comm.RegisterOnMessage(f)
	unregisterToken2 := comm.RegisterOnMessage(g)
	require.NotEqual(t, unregisterToken1, unregisterToken2, "cannot return the same cancel token for both registered functions")
	countRegistered := comm.CountRegisteredOnMessage()
	require.Equal(t, 2, comm.CountRegisteredOnMessage(), "should have registered 2 but actually registered %d", countRegistered)

	// Delete existing
	comm.UnregisterOnMessage(unregisterToken1)
	comm.UnregisterOnMessage(unregisterToken2)

	// Delete non-existing (no-op)
	comm.UnregisterOnMessage(unregisterToken2)
	countRegistered = comm.CountRegisteredOnMessage()
	require.Equal(t, 0, comm.CountRegisteredOnMessage(), "no registered functions should have remained but actually %d are still registered", countRegistered)
}
