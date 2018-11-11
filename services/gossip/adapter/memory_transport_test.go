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
		key := primitives.Ed25519PublicKey{0x01}
		federationNodes := make(map[string]config.FederationNode)
		federationNodes[key.KeyForMap()] = config.NewHardCodedFederationNode(key)

		transport := NewMemoryTransport(parentContext, log.GetLogger(), federationNodes)
		listener := listenTo(transport, key)

		childContext, cancel := context.WithCancel(parentContext)
		defer cancel()

		contextWithTrace := trace.NewContext(childContext, "foo")
		originalTracingContext, _ := trace.FromContext(contextWithTrace)

		data := &TransportData{
			SenderPublicKey: primitives.Ed25519PublicKey{0x02},
			RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
			Payloads:        [][]byte{},
		}

		listener.When("OnTransportMessageReceived", mock.Any, mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
			propagatedTracingContext, ok := trace.FromContext(ctx)
			require.True(t, ok, "memory transport did not create a tracing context")

			require.NotEmpty(t, propagatedTracingContext.NestedFields())
			require.Equal(t, propagatedTracingContext.NestedFields(), originalTracingContext.NestedFields())
		}).Times(1)

		transport.Send(contextWithTrace, data)

		test.EventuallyVerify(100 * time.Millisecond, listener)
	})
}

