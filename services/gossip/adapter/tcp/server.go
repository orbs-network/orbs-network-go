// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"net"
	"sync"
	"time"
)

type serverConfig interface {
	GossipListenPort() uint16
	GossipNetworkTimeout() time.Duration
}

type transportServer struct {
	sync.RWMutex
	listener                          adapter.TransportListener
	listening                         bool
	port                              int
	logger                            log.Logger
	incomingConnectionAcceptSuccesses *metric.Gauge
	incomingConnectionAcceptErrors    *metric.Gauge
	incomingConnectionTransportErrors *metric.Gauge
	activeIncomingConnections         *metric.Gauge
	config                            serverConfig
}

func newDirectTransportServer(config serverConfig, logger log.Logger, registry metric.Registry) *transportServer {
	return &transportServer{
		config:                            config,
		listener:                          nil,
		listening:                         false,
		port:                              0,
		logger:                            logger,
		incomingConnectionAcceptSuccesses: registry.NewGauge("Gossip.IncomingConnection.ListeningOnTCPPortSuccess.Count"),
		incomingConnectionAcceptErrors:    registry.NewGauge("Gossip.IncomingConnection.ListeningOnTCPPortErrors.Count"),
		incomingConnectionTransportErrors: registry.NewGauge("Gossip.IncomingConnection.TransportErrors.Count"),
		activeIncomingConnections:         registry.NewGauge("Gossip.IncomingConnection.Active.Count"),
	}
}

func (t *transportServer) getPort() int {
	t.Lock()
	defer t.Unlock()

	return t.port
}

func (t *transportServer) listenForIncomingConnections(ctx context.Context) (net.Listener, error) {
	// TODO(v1): migrate to ListenConfig which has better support of contexts (go 1.11 required)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", t.config.GossipListenPort()))
	if err != nil {
		return nil, err
	}

	// this goroutine will shut down the server gracefully when context is done
	go func() {
		<-ctx.Done()
		t.Lock()
		defer t.Unlock()
		t.listening = false
		err := listener.Close()
		if err != nil {
			t.logger.Error("Failed to close direct transport lister", log.Error(err))
		}
	}()

	t.Lock()
	defer t.Unlock()
	t.listening = true
	t.port = listener.Addr().(*net.TCPAddr).Port
	t.logger.Info("gossip transport server listening", log.Int("port", t.port))

	return listener, err
}

func (t *transportServer) IsListening() bool {
	t.RLock()
	defer t.RUnlock()

	return t.listening
}

func (t *transportServer) mainLoop(parentCtx context.Context, listener net.Listener) {
	for {
		if parentCtx.Err() != nil {
			t.logger.Info("ending server main loop (system shutting down)")
		}

		ctx := trace.NewContext(parentCtx, "Gossip.Transport.TCP.Server")

		conn, err := listener.Accept()
		if err != nil {
			if !t.IsListening() {
				t.logger.Info("incoming connection accept stopped since server is shutting down", trace.LogFieldFrom(ctx))
				return
			}
			t.incomingConnectionAcceptErrors.Inc()
			t.logger.Info("incoming connection accept error", log.Error(err), trace.LogFieldFrom(ctx))
			continue
		}
		t.incomingConnectionAcceptSuccesses.Inc()
		govnr.Once(logfields.GovnrErrorer(t.logger), func() {
			t.handleIncomingConnection(ctx, conn)
		})
	}
}

func (t *transportServer) handleIncomingConnection(ctx context.Context, conn net.Conn) {
	t.logger.Info("successful incoming gossip transport connection", log.String("peer", conn.RemoteAddr().String()), trace.LogFieldFrom(ctx))
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): add a white list for IPs we're willing to accept connections from
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): make sure each IP from the white list connects only once
	t.activeIncomingConnections.Inc()
	defer t.activeIncomingConnections.Dec()

	for {
		payloads, err := t.receiveTransportData(ctx, conn)
		if err != nil {
			t.incomingConnectionTransportErrors.Inc()
			t.logger.Info("failed receiving transport data, disconnecting", log.Error(err), log.String("peer", conn.RemoteAddr().String()), trace.LogFieldFrom(ctx))
			conn.Close()

			return
		}

		// notify if not keepalive
		if len(payloads) > 0 {
			ctxWithPeer := context.WithValue(ctx, "peer-ip", conn.RemoteAddr().String())
			t.notifyListener(ctxWithPeer, payloads)
		}
	}
}

func (t *transportServer) receiveTransportData(ctx context.Context, conn net.Conn) ([][]byte, error) {
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): think about timeout policy on receive, we might not want it
	timeout := t.config.GossipNetworkTimeout()
	var res [][]byte

	// receive num payloads
	sizeBuffer, err := readTotal(ctx, conn, 4, timeout)
	if err != nil {
		return nil, err
	}
	numPayloads := membuffers.GetUint32(sizeBuffer)

	if numPayloads > MAX_PAYLOADS_IN_MESSAGE {
		return nil, errors.Errorf("received message with too many payloads: %d", numPayloads)
	}

	for i := uint32(0); i < numPayloads; i++ {
		// receive payload size
		sizeBuffer, err := readTotal(ctx, conn, 4, timeout)
		if err != nil {
			return nil, err
		}
		payloadSize := membuffers.GetUint32(sizeBuffer)
		if payloadSize > MAX_PAYLOAD_SIZE_BYTES {
			return nil, errors.Errorf("received message with a payload too big: %d bytes", payloadSize)
		}

		// receive payload data
		payload, err := readTotal(ctx, conn, payloadSize, timeout)
		if err != nil {
			return nil, err
		}
		res = append(res, payload)

		// receive padding
		paddingSize := calcPaddingSize(uint32(len(payload)))
		if paddingSize > 0 {
			_, err := readTotal(ctx, conn, paddingSize, timeout)
			if err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}

func (t *transportServer) notifyListener(ctx context.Context, payloads [][]byte) {
	listener := t.getListener()

	if listener == nil {
		return
	}

	listener.OnTransportMessageReceived(ctx, payloads)
}

func (t *transportServer) getListener() adapter.TransportListener {
	t.RLock()
	defer t.RUnlock()

	return t.listener
}

func (t *transportServer) startSupervisedMainLoop(ctx context.Context) *govnr.ForeverHandle {
	listener, err := t.listenForIncomingConnections(ctx)
	if err != nil {
		panic(fmt.Sprintf("gossip transport failed to listen on port %d: %s", t.config.GossipListenPort(), err.Error()))
	}
	return govnr.Forever(ctx, "TCP server", logfields.GovnrErrorer(t.logger), func() {
		t.mainLoop(ctx, listener)
	})
}

func readTotal(ctx context.Context, conn net.Conn, totalSize uint32, timeout time.Duration) ([]byte, error) {
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): consider whether the right approach is to poll context this way or have a single watchdog goroutine that closes all active connections when context is cancelled
	// make sure context is still open
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, totalSize)
	totalRead := uint32(0)
	for totalRead < totalSize {
		err := conn.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			return nil, err
		}
		read, err := conn.Read(buffer[totalRead:])
		totalRead += uint32(read)
		if totalRead == totalSize {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return buffer, nil
}
