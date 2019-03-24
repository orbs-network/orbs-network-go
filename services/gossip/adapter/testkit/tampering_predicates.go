// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type HeaderPredicate func(header *gossipmessages.Header) bool

func ATransactionRelayMessage(header *gossipmessages.Header) bool {
	return header.IsTopicTransactionRelay()
}

func ABenchmarkConsensusMessage(header *gossipmessages.Header) bool {
	return header.IsTopicBenchmarkConsensus()
}

func AConsensusMessage(header *gossipmessages.Header) bool {
	return header.IsTopicBenchmarkConsensus() || header.IsTopicLeanHelix()
}

func HasHeader(headerPredicate HeaderPredicate) MessagePredicate {
	return func(data *adapter.TransportData) bool {
		header, ok := parseHeader(data)

		return ok && headerPredicate(header)
	}
}

func Not(predicate MessagePredicate) MessagePredicate {
	return func(data *adapter.TransportData) bool {
		return !predicate(data)
	}
}

// a MessagePredicate for capturing Lean Helix Consensus Algorithm gossip messages of the given type
// TODO (v1) Maybe fix this another way - Orbs doesn't know specific LH messages anymore
//func LeanHelixMessage(messageType leanhelix.MessageType) MessagePredicate {
//	return HasHeader(func(header *gossipmessages.Header) bool {
//		return header.IsTopicLeanHelix() && header.LeanHelix() == consensus.LeanHelixMessageType(messageType)
//	})
//}

func BenchmarkConsensusMessage(messageType consensus.BenchmarkConsensusMessageType) MessagePredicate {
	return HasHeader(func(header *gossipmessages.Header) bool {
		return header.IsTopicBenchmarkConsensus() && header.BenchmarkConsensus() == messageType
	})
}

func BlockSyncMessage(messageType gossipmessages.BlockSyncMessageType) MessagePredicate {
	return HasHeader(func(header *gossipmessages.Header) bool {
		return header.IsTopicBlockSync() && header.BlockSync() == messageType
	})
}

func TransactionRelayMessage(messageType gossipmessages.TransactionsRelayMessageType) MessagePredicate {
	return HasHeader(func(header *gossipmessages.Header) bool {
		return header.IsTopicTransactionRelay() && header.TransactionRelay() == messageType
	})
}

func (this MessagePredicate) And(other MessagePredicate) MessagePredicate {
	return func(data *adapter.TransportData) bool {
		return this(data) && other(data)
	}
}

func parseHeader(data *adapter.TransportData) (*gossipmessages.Header, bool) {
	if data == nil || len(data.Payloads) == 0 {
		return nil, false
	}

	header := gossipmessages.HeaderReader(data.Payloads[0])
	if !header.IsValid() {
		return nil, false
	}

	return header, true
}
