// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

const TEST_KEEP_ALIVE_INTERVAL = 20 * time.Millisecond
const TEST_NETWORK_TIMEOUT = 1 * time.Second

const HARNESS_PEER_READ_TIMEOUT = 1 * time.Second
const HARNESS_OUTGOING_CONNECTIONS_INIT_TIMEOUT = 3 * time.Second

type directHarness struct {
	*with.ConcurrencyHarness
	config    config.GossipTransportConfig
	transport *DirectTransport

	peerTalkerConnection net.Conn
	listenerMock         *testkit.MockTransportListener
}

func newDirectHarnessWithConnectedPeers(t *testing.T, ctx context.Context, parent *with.ConcurrencyHarness) *directHarness {
	address := keys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress()
	cfg := config.ForDirectTransportTests(address, TEST_KEEP_ALIVE_INTERVAL, TEST_NETWORK_TIMEOUT) // this gossipPeers is just a stub, it's mostly a client gossipPeers and this is a server harness
	transport := makeTransport(ctx, parent.Logger, cfg)

	peerTalkerConnection := establishPeerClient(t, transport.GetServerPort()) // establish connection from test to server port ( test harness ==> SUT )

	h := &directHarness{
		ConcurrencyHarness:   parent,
		config:               cfg,
		transport:            transport,
		listenerMock:         &testkit.MockTransportListener{},
		peerTalkerConnection: peerTalkerConnection,
	}

	h.Supervise(transport)

	return h
}

func makeTransport(ctx context.Context, logger log.Logger, cfg config.GossipTransportConfig) *DirectTransport {
	registry := metric.NewRegistry()

	transport := NewDirectTransport(ctx, cfg, logger, registry)
	// to synchronize tests, wait until server is ready
	test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
		return transport.IsServerListening()
	})
	return transport
}

func establishPeerClient(t *testing.T, serverPort int) net.Conn {
	peerTalkerConnection, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	require.NoError(t, err, "test should be able connect to local transport")
	return peerTalkerConnection
}

func (h *directHarness) cleanupConnectedPeers() {
	h.peerTalkerConnection.Close()
}

func (h *directHarness) expectTransportListenerCalled(payloads [][]byte) {
	h.listenerMock.When("OnTransportMessageReceived", mock.Any, payloads).Return().Times(1)
}

func (h *directHarness) verifyTransportListenerCalled(t *testing.T) {
	err := test.EventuallyVerify(test.EVENTUALLY_ADAPTER_TIMEOUT, h.listenerMock)
	require.NoError(t, err, "transport listener mock should be called as expected")
}

func (h *directHarness) expectTransportListenerNotCalled() {
	h.listenerMock.When("OnTransportMessageReceived", mock.Any, mock.Any).Return().Times(0)
}

func (h *directHarness) verifyTransportListenerNotCalled(t *testing.T) {
	err := test.ConsistentlyVerify(test.CONSISTENTLY_ADAPTER_TIMEOUT, h.listenerMock)
	require.NoError(t, err, "transport listener mock should be called as expected")
}

func concatSlices(slices ...[]byte) []byte {
	var tmp []byte
	for _, s := range slices {
		tmp = append(tmp, s...)
	}
	return tmp
}

// encoded examples of the gossip wire protocol spec:
// https://github.com/orbs-network/orbs-spec/blob/master/encoding/gossip/membuffers-over-tcp.md

func exampleWireProtocolEncoding_Payloads_0x11_0x2233() []byte {
	// encoding payloads: [][]byte{{0x11}, {0x22, 0x33}}
	field_NumPayloads := []byte{0x02, 0x00, 0x00, 0x00}      // little endian
	field_FirstPayloadSize := []byte{0x01, 0x00, 0x00, 0x00} // little endian
	field_FirstPayloadData := []byte{0x11}
	field_FirstPayloadPadding := []byte{0x00, 0x00, 0x00}     // round payload data to 4 bytes
	field_SecondPayloadSize := []byte{0x02, 0x00, 0x00, 0x00} // little endian
	field_SecondPayloadData := []byte{0x22, 0x33}
	field_SecondPayloadPadding := []byte{0x00, 0x00} // round payload data to 4 bytes
	return concatSlices(field_NumPayloads, field_FirstPayloadSize, field_FirstPayloadData, field_FirstPayloadPadding, field_SecondPayloadSize, field_SecondPayloadData, field_SecondPayloadPadding)
}

func exampleWireProtocolEncoding_CorruptNumPayloads() []byte {
	field_NumPayloads := []byte{0x99, 0x99, 0x99, 0x99}      // corrupt value (too big)
	field_FirstPayloadSize := []byte{0x01, 0x00, 0x00, 0x00} // little endian
	field_FirstPayloadData := []byte{0x11}
	field_FirstPayloadPadding := []byte{0x00, 0x00, 0x00} // round payload data to 4 bytes
	return concatSlices(field_NumPayloads, field_FirstPayloadSize, field_FirstPayloadData, field_FirstPayloadPadding)
}

func exampleWireProtocolEncoding_CorruptPayloadSize() []byte {
	field_NumPayloads := []byte{0x01, 0x00, 0x00, 0x00}      // little endian
	field_FirstPayloadSize := []byte{0x99, 0x99, 0x99, 0x99} // corrupt value (too big)
	field_FirstPayloadData := []byte{0x11}
	field_FirstPayloadPadding := []byte{0x00, 0x00, 0x00} // round payload data to 4 bytes
	return concatSlices(field_NumPayloads, field_FirstPayloadSize, field_FirstPayloadData, field_FirstPayloadPadding)
}

func exampleWireProtocolEncoding_KeepAlive() []byte {
	// encoding payloads: [][]byte{} (this is how a keep alive looks like = zero payloads)
	field_NumPayloads := []byte{0x00, 0x00, 0x00, 0x00} // little endian
	return concatSlices(field_NumPayloads)
}
