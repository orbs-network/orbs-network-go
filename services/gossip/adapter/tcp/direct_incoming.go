// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/pkg/errors"
	"net"
	"time"
)

func (t *directTransport) serverListenForIncomingConnections(ctx context.Context, listenPort uint16) (net.Listener, error) {
	// TODO(v1): migrate to ListenConfig which has better support of contexts (go 1.11 required)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return nil, err
	}

	// this goroutine will shut down the server gracefully when context is done
	go func() {
		<-ctx.Done()
		t.mutex.Lock()
		defer t.mutex.Unlock()
		t.serverListeningUnderMutex = false
		listener.Close()
	}()

	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.serverListeningUnderMutex = true

	return listener, err
}

func (t *directTransport) isServerListening() bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.serverListeningUnderMutex
}

func (t *directTransport) serverMainLoop(parentCtx context.Context, listenPort uint16) {
	listener, err := t.serverListenForIncomingConnections(parentCtx, listenPort)
	if err != nil {
		panic(fmt.Sprintf("gossip transport failed to listen on port %d: %s", listenPort, err.Error()))
	}

	t.serverPort = listener.Addr().(*net.TCPAddr).Port
	t.logger.Info("gossip transport server listening", log.Uint32("port", uint32(t.serverPort)))

	for {
		if parentCtx.Err() != nil {
			t.logger.Info("ending server main loop (system shutting down)")
		}

		ctx := trace.NewContext(parentCtx, "Gossip.Transport.TCP.Server")

		conn, err := listener.Accept()
		if err != nil {
			if !t.isServerListening() {
				t.logger.Info("incoming connection accept stopped since server is shutting down", trace.LogFieldFrom(ctx))
				return
			}
			t.metrics.incomingConnectionAcceptErrors.Inc()
			t.logger.Info("incoming connection accept error", log.Error(err), trace.LogFieldFrom(ctx))
			continue
		}
		t.metrics.incomingConnectionAcceptSuccesses.Inc()
		supervised.GoOnce(t.logger, func() {
			t.serverHandleIncomingConnection(ctx, conn)
		})
	}
}

func (t *directTransport) serverHandleIncomingConnection(ctx context.Context, conn net.Conn) {
	t.logger.Info("successful incoming gossip transport connection", log.String("peer", conn.RemoteAddr().String()), trace.LogFieldFrom(ctx))
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): add a white list for IPs we're willing to accept connections from
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): make sure each IP from the white list connects only once
	t.metrics.activeIncomingConnections.Inc()
	defer t.metrics.activeIncomingConnections.Dec()

	for {
		payloads, err := t.receiveTransportData(ctx, conn)
		if err != nil {
			t.metrics.incomingConnectionTransportErrors.Inc()
			t.logger.Info("failed receiving transport data, disconnecting", log.Error(err), log.String("peer", conn.RemoteAddr().String()), trace.LogFieldFrom(ctx))
			conn.Close()

			return
		}

		// notify if not keepalive
		if len(payloads) > 0 {
			t.notifyListener(ctx, payloads)
		}
	}
}

func (t *directTransport) receiveTransportData(ctx context.Context, conn net.Conn) ([][]byte, error) {
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): think about timeout policy on receive, we might not want it
	timeout := t.config.GossipNetworkTimeout()
	res := [][]byte{}

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

func (t *directTransport) notifyListener(ctx context.Context, payloads [][]byte) {
	listener := t.getListener()

	if listener == nil {
		return
	}

	listener.OnTransportMessageReceived(ctx, payloads)
}

func (t *directTransport) getListener() adapter.TransportListener {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.transportListenerUnderMutex
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
