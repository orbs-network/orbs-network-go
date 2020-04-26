// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gossip

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/codec"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
)

func (s *Service) RegisterHeaderSyncHandler(handler gossiptopics.HeaderSyncHandler) {
	s.handlers.Lock()
	defer s.handlers.Unlock()

	s.handlers.headerSyncHandlers = append(s.handlers.headerSyncHandlers, handler)
}

// handles both client and server sides of header sync
// server side (responding to requests) fires a new goroutine to handle each requests so as to not block the topic
// client side will be handled in the header sync client main loop
func (s *Service) receivedHeaderSyncMessage(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	switch header.HeaderSync() {
	case gossipmessages.HEADER_SYNC_AVAILABILITY_REQUEST:
		govnr.Once(logfields.GovnrErrorer(s.logger), func() {
			s.receivedHeaderSyncAvailabilityRequest(createHeaderSyncServerChildContextFrom(ctx), header, payloads)
		})
	case gossipmessages.HEADER_SYNC_AVAILABILITY_RESPONSE:
		s.receivedHeaderSyncAvailabilityResponse(ctx, header, payloads)
	case gossipmessages.HEADER_SYNC_REQUEST:
		govnr.Once(logfields.GovnrErrorer(s.logger), func() {
			s.receivedHeaderSyncRequest(createHeaderSyncServerChildContextFrom(ctx), header, payloads)
		})
	case gossipmessages.HEADER_SYNC_RESPONSE:
		s.receivedHeaderSyncResponse(ctx, header, payloads)
	}
}

func createHeaderSyncServerChildContextFrom(ctx context.Context) context.Context {
	return trace.NewContext(ctx, "HeaderSyncServer")
}

func (s *Service) BroadcastHeaderAvailabilityRequest(ctx context.Context, input *gossiptopics.HeaderAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:          gossipmessages.HEADER_TOPIC_HEADER_SYNC,
		HeaderSync:     gossipmessages.HEADER_SYNC_AVAILABILITY_REQUEST,
		RecipientMode:  gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		VirtualChainId: s.config.VirtualChainId(),
	}).Build()
	payloads, err := codec.EncodeHeaderAvailabilityRequest(header, input.Message)
	if err != nil {
		return nil, err
	}
	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderNodeAddress: s.config.NodeAddress(),
		RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:          payloads,
	})
}

func (s *Service) receivedHeaderSyncAvailabilityRequest(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	message, err := codec.DecodeHeaderAvailabilityRequest(payloads)
	if err != nil {
		return
	}

	s.handlers.RLock()
	defer s.handlers.RUnlock()

	for _, l := range s.handlers.headerSyncHandlers {
		_, err := l.HandleHeaderAvailabilityRequest(ctx, &gossiptopics.HeaderAvailabilityRequestInput{Message: message})
		if err != nil {
			s.logger.Info("HandleHeaderAvailabilityRequest failed", log.Error(err), logfields.ContextStringValue(ctx, "peer-ip"))
		}
	}
}

func (s *Service) SendHeaderAvailabilityResponse(ctx context.Context, input *gossiptopics.HeaderAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:                  gossipmessages.HEADER_TOPIC_HEADER_SYNC,
		HeaderSync:             gossipmessages.HEADER_SYNC_AVAILABILITY_RESPONSE,
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{input.RecipientNodeAddress},
		VirtualChainId:         s.config.VirtualChainId(),
	}).Build()
	payloads, err := codec.EncodeHeaderAvailabilityResponse(header, input.Message)
	if err != nil {
		return nil, err
	}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderNodeAddress:      s.config.NodeAddress(),
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{input.RecipientNodeAddress},
		Payloads:               payloads,
	})
}

func (s *Service) receivedHeaderSyncAvailabilityResponse(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	message, err := codec.DecodeHeaderAvailabilityResponse(payloads)
	if err != nil {
		return
	}

	s.handlers.RLock()
	defer s.handlers.RUnlock()

	for _, l := range s.handlers.headerSyncHandlers {
		_, err := l.HandleHeaderAvailabilityResponse(ctx, &gossiptopics.HeaderAvailabilityResponseInput{Message: message})
		if err != nil {
			s.logger.Info("HandleHeaderAvailabilityResponse failed", log.Error(err))
		}
	}
}

func (s *Service) SendHeaderSyncRequest(ctx context.Context, input *gossiptopics.HeaderSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:                  gossipmessages.HEADER_TOPIC_HEADER_SYNC,
		HeaderSync:             gossipmessages.HEADER_SYNC_REQUEST,
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{input.RecipientNodeAddress},
		VirtualChainId:         s.config.VirtualChainId(),
	}).Build()
	payloads, err := codec.EncodeHeaderSyncRequest(header, input.Message)
	if err != nil {
		return nil, err
	}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderNodeAddress:      s.config.NodeAddress(),
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{input.RecipientNodeAddress},
		Payloads:               payloads,
	})
}

func (s *Service) receivedHeaderSyncRequest(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	message, err := codec.DecodeHeaderSyncRequest(payloads)
	if err != nil {
		return
	}

	s.handlers.RLock()
	defer s.handlers.RUnlock()

	for _, l := range s.handlers.headerSyncHandlers {
		_, err := l.HandleHeaderSyncRequest(ctx, &gossiptopics.HeaderSyncRequestInput{Message: message})
		if err != nil {
			s.logger.Info("HandleHeaderSyncRequest failed", log.Error(err))
		}
	}
}

//func IsChunkTooBigError(err error) bool {
//	return tcp.IsQueueFullError(err)
//}

func (s *Service) SendHeaderSyncResponse(ctx context.Context, input *gossiptopics.HeaderSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:                  gossipmessages.HEADER_TOPIC_HEADER_SYNC,
		HeaderSync:             gossipmessages.HEADER_SYNC_RESPONSE,
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{input.RecipientNodeAddress},
		VirtualChainId:         s.config.VirtualChainId(),
	}).Build()
	payloads, err := codec.EncodeHeaderSyncResponse(header, input.Message)
	if err != nil {
		return nil, err
	}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderNodeAddress:      s.config.NodeAddress(),
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{input.RecipientNodeAddress},
		Payloads:               payloads,
	})
}

func (s *Service) receivedHeaderSyncResponse(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	message, err := codec.DecodeHeaderSyncResponse(payloads)
	if err != nil {
		return
	}

	s.handlers.RLock()
	defer s.handlers.RUnlock()

	for _, l := range s.handlers.headerSyncHandlers {
		_, err := l.HandleHeaderSyncResponse(ctx, &gossiptopics.HeaderSyncResponseInput{Message: message})
		if err != nil {
			s.logger.Info("HandleHeaderSyncResponse failed", log.Error(err))
		}
	}
}
