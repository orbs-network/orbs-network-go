// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

type MockTransportListener struct {
	mock.Mock
}

func (l *MockTransportListener) OnTransportMessageReceived(ctx context.Context, payloads [][]byte) {
	l.Called(ctx, payloads)
}

func ListenTo(transport adapter.Transport, nodeAddress primitives.NodeAddress) *MockTransportListener {
	l := &MockTransportListener{}
	transport.RegisterListener(l, nodeAddress)
	return l
}

func (l *MockTransportListener) ExpectReceive(payloads [][]byte) {
	l.WhenOnTransportMessageReceived(payloads).Return().AtLeast(1)
}

func (l *MockTransportListener) ExpectNotReceive() {
	l.Never("OnTransportMessageReceived", mock.Any, mock.Any)
}

func (l *MockTransportListener) BlockReceive() {
	l.When("OnTransportMessageReceived", mock.Any, mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
		<-ctx.Done()
	})
}

func (l *MockTransportListener) WhenOnTransportMessageReceived(arg interface{}) *mock.MockFunction {
	return l.When("OnTransportMessageReceived", mock.Any, arg)
}

func (l *MockTransportListener) ExpectTracingContextToPropagate(t *testing.T, originalTracingContext *trace.Context) *mock.MockFunction {
	return l.When("OnTransportMessageReceived", mock.Any, mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
		propagatedTracingContext, ok := trace.FromContext(ctx)
		require.True(t, ok, "memory transport did not create a tracing context")

		require.NotEmpty(t, propagatedTracingContext.NestedFields())
		require.Equal(t, propagatedTracingContext.NestedFields(), originalTracingContext.NestedFields())
	}).Times(1)
}
