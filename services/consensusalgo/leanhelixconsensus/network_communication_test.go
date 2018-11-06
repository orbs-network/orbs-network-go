package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"os"
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
	log := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	timeout := 1 * time.Millisecond
	metricFactory := metric.NewRegistry()
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	res := NewLeanHelixConsensusAlgo(
		ctx,
		&testGossip{},
		nil,
		nil,
		log,
		&testConfig{timeout: timeout},
		metricFactory,
	)
	s := res.(*service)

	f := func(ctx context.Context, message leanhelix.ConsensusRawMessage) {}
	g := func(ctx context.Context, message leanhelix.ConsensusRawMessage) {}

	unregisterToken1 := s.RegisterOnMessage(f)
	unregisterToken2 := s.RegisterOnMessage(g)
	require.NotEqual(t, unregisterToken1, unregisterToken2, "cannot return the same cancel token for both registered functions")
	countRegistered := s.CountRegisteredOnMessage()
	require.Equal(t, 2, s.CountRegisteredOnMessage(), "should have registered 2 but actually registered %d", countRegistered)

	// Delete existing
	s.UnregisterOnMessage(unregisterToken1)
	s.UnregisterOnMessage(unregisterToken2)

	// Delete non-existing (no-op)
	s.UnregisterOnMessage(unregisterToken2)
	countRegistered = s.CountRegisteredOnMessage()
	require.Equal(t, 0, s.CountRegisteredOnMessage(), "no registered functions should have remained but actually %d are still registered", countRegistered)
}
