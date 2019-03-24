// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

/// Test builders for: protocol.EventsArray, primitives.PackedEventsArray

func PackedEventsArrayEncode(eventBuilders []*protocol.EventBuilder) primitives.PackedEventsArray {
	eventsArray := (&protocol.EventsArrayBuilder{Events: eventBuilders}).Build()
	return eventsArray.RawEventsArray()
}
