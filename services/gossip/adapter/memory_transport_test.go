package adapter

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
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

		transport := NewMemoryTransport(parentContext, log.GetLogger(), makeFederation(address))
		listener := listenTo(transport, address)

		childContext, cancel := context.WithCancel(parentContext) // this is required so that the parent context does not get polluted
		defer cancel()

		contextWithTrace := trace.NewContext(childContext, "foo")
		originalTracingContext, _ := trace.FromContext(contextWithTrace)

		listener.expectTracingContextToPropagate(t, originalTracingContext)

		transport.Send(contextWithTrace, &TransportData{
			SenderNodeAddress: primitives.NodeAddress{0x02},
			RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		})

		test.EventuallyVerify(100*time.Millisecond, listener)
	})
}

func (l *MockTransportListener) expectTracingContextToPropagate(t *testing.T, originalTracingContext *trace.Context) *mock.MockFunction {
	return l.When("OnTransportMessageReceived", mock.Any, mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
		propagatedTracingContext, ok := trace.FromContext(ctx)
		require.True(t, ok, "memory transport did not create a tracing context")

		require.NotEmpty(t, propagatedTracingContext.NestedFields())
		require.Equal(t, propagatedTracingContext.NestedFields(), originalTracingContext.NestedFields())
	}).Times(1)
}

func makeFederation(address primitives.NodeAddress) map[string]config.FederationNode {
	federationNodes := make(map[string]config.FederationNode)
	federationNodes[address.KeyForMap()] = config.NewHardCodedFederationNode(address)
	return federationNodes
}
