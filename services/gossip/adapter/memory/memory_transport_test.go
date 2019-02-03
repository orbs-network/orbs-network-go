package memory

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMemoryTransport_PropagatesTracingContext(t *testing.T) {

	test.WithContext(func(parentContext context.Context) {
		address := primitives.NodeAddress{0x01}

		transport := NewTransport(parentContext, log.DefaultTestingLogger(t), makeFederation(address))
		listener := testkit.ListenTo(transport, address)

		childContext, cancel := context.WithCancel(parentContext) // this is required so that the parent context does not get polluted
		defer cancel()

		contextWithTrace := trace.NewContext(childContext, "foo")
		originalTracingContext, _ := trace.FromContext(contextWithTrace)

		listener.ExpectTracingContextToPropagate(t, originalTracingContext)

		_ = transport.Send(contextWithTrace, &adapter.TransportData{
			SenderNodeAddress: primitives.NodeAddress{0x02},
			RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		})

		require.NoError(t, test.EventuallyVerify(100*time.Millisecond, listener))
	})
}

func makeFederation(address primitives.NodeAddress) map[string]config.FederationNode {
	federationNodes := make(map[string]config.FederationNode)
	federationNodes[address.KeyForMap()] = config.NewHardCodedFederationNode(address)
	return federationNodes
}
