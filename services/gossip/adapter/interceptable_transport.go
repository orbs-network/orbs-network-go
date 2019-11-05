package adapter

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type TransmitFunc func(ctx context.Context, peerAddress primitives.NodeAddress, data *TransportData)

type InterceptorFunc func(ctx context.Context, peerAddress primitives.NodeAddress, data *TransportData, transmit TransmitFunc) error

type InterceptableTransport interface {
	Transport
	SendWithInterceptor(ctx context.Context, data *TransportData, intercept InterceptorFunc) error
}
