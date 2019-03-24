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
	"github.com/pkg/errors"
	"net"
	"time"
)

func (t *directTransport) clientMainLoop(parentCtx context.Context, queue *transportQueue) {
	for {
		ctx := trace.NewContext(parentCtx, fmt.Sprintf("Gossip.Transport.TCP.Client.%s", queue.networkAddress))
		t.logger.Info("attempting outgoing transport connection", log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
		conn, err := net.DialTimeout("tcp", queue.networkAddress, t.config.GossipNetworkTimeout())

		if err != nil {
			t.logger.Info("cannot connect to gossip peer endpoint", log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
			time.Sleep(t.config.GossipConnectionKeepAliveInterval())
			continue
		}

		if !t.clientHandleOutgoingConnection(ctx, conn, queue) {
			return
		}
	}
}

// returns true if should attempt reconnect on error
func (t *directTransport) clientHandleOutgoingConnection(ctx context.Context, conn net.Conn, queue *transportQueue) bool {
	t.logger.Info("successful outgoing gossip transport connection", log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
	t.metrics.activeOutgoingConnections.Inc()
	defer t.metrics.activeOutgoingConnections.Dec()
	queue.Clear(ctx)
	queue.Enable()
	defer queue.Disable()

	for {

		ctxWithKeepAliveTimeout, cancelCtxWithKeepAliveTimeout := context.WithTimeout(ctx, t.config.GossipConnectionKeepAliveInterval())
		data := queue.Pop(ctxWithKeepAliveTimeout)
		cancelCtxWithKeepAliveTimeout()

		if data != nil {

			// ctxWithKeepAliveTimeout not closed, so no keep alive timeout nor system shutdown
			// meaning do a regular send (we have data)
			err := t.sendTransportData(ctx, conn, data)
			if err != nil {
				t.metrics.outgoingConnectionSendErrors.Inc()
				t.logger.Info("failed sending transport data, reconnecting", log.Error(err), log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
				conn.Close()
				return true
			}

		} else {
			// ctxWithKeepAliveTimeout is closed, so either keep alive timeout or system shutdown
			if ctx.Err() == nil {

				// parent ctx not closed, so no system shutdown
				// meaning keep alive timeout
				err := t.sendKeepAlive(ctx, conn)
				if err != nil {
					t.metrics.outgoingConnectionKeepaliveErrors.Inc()
					t.logger.Info("failed sending keepalive, reconnecting", log.Error(err), log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
					conn.Close()
					return true
				}

			} else {

				// parent ctx is closed, so system shutdown
				// meaning cleanup and exit
				t.logger.Info("client loop stopped since server is shutting down", trace.LogFieldFrom(ctx))
				conn.Close()
				return false

			}
		}
	}
}

func (t *directTransport) addDataToOutgoingPeerQueue(data *adapter.TransportData, outgoingQueue *transportQueue) {
	err := outgoingQueue.Push(data)
	if err != nil {
		t.metrics.outgoingConnectionSendQueueErrors.Inc()
		t.logger.Info("direct transport send queue error", log.Error(err), log.String("peer", outgoingQueue.networkAddress))
	}
	t.metrics.outgoingMessageSize.Record(int64(data.TotalSize()))
}

func (t *directTransport) sendTransportData(ctx context.Context, conn net.Conn, data *adapter.TransportData) error {
	timeout := t.config.GossipNetworkTimeout()
	zeroBuffer := make([]byte, 4)
	sizeBuffer := make([]byte, 4)

	// send num payloads
	membuffers.WriteUint32(sizeBuffer, uint32(len(data.Payloads)))
	err := write(ctx, conn, sizeBuffer, timeout)
	if err != nil {
		return err
	}

	for _, payload := range data.Payloads {
		// send payload size
		membuffers.WriteUint32(sizeBuffer, uint32(len(payload)))
		err := write(ctx, conn, sizeBuffer, timeout)
		if err != nil {
			return err
		}

		// send payload data
		err = write(ctx, conn, payload, timeout)
		if err != nil {
			return err
		}

		// send padding
		paddingSize := calcPaddingSize(uint32(len(payload)))
		if paddingSize > 0 {
			err = write(ctx, conn, zeroBuffer[:paddingSize], timeout)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *directTransport) sendKeepAlive(ctx context.Context, conn net.Conn) error {
	timeout := t.config.GossipNetworkTimeout()
	zeroBuffer := make([]byte, 4)

	// send zero num payloads
	err := write(ctx, conn, zeroBuffer, timeout)
	if err != nil {
		return err
	}

	return nil
}

func write(ctx context.Context, conn net.Conn, buffer []byte, timeout time.Duration) error {
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): consider whether the right approach is to poll context this way or have a single watchdog goroutine that closes all active connections when context is cancelled
	// make sure context is still open
	err := ctx.Err()
	if err != nil {
		return err
	}

	err = conn.SetWriteDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}
	written, err := conn.Write(buffer)
	if written != len(buffer) {
		if err == nil {
			return errors.Errorf("attempted to write %d bytes but only wrote %d", len(buffer), written)
		} else {
			return errors.Wrapf(err, "attempted to write %d bytes but only wrote %d", len(buffer), written)
		}
	}
	return nil
}
