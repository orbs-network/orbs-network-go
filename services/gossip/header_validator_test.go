// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gossip

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateMessageHeader_Valid(t *testing.T) {
	v := newValidator(t)

	header := (&gossipmessages.HeaderBuilder{
		VirtualChainId:         42,
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{{0x1}},
	}).Build()

	require.NoError(t, v.validateMessageHeader(header))
}

func TestValidateMessageHeader_IncorrectVirtualChain(t *testing.T) {
	v := newValidator(t)

	header := (&gossipmessages.HeaderBuilder{
		VirtualChainId: 43,
		RecipientMode:  gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	require.Error(t, v.validateMessageHeader(header))
}

func TestValidateMessageHeader_NodeIsNotARecipient_ListMode(t *testing.T) {
	v := newValidator(t)

	header := (&gossipmessages.HeaderBuilder{
		VirtualChainId:         42,
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{{0x2}, {0x3}},
	}).Build()

	require.Error(t, v.validateMessageHeader(header))
}

func TestValidateMessageHeader_NodeIsARecipient_AllButListMode(t *testing.T) {
	v := newValidator(t)

	header := (&gossipmessages.HeaderBuilder{
		VirtualChainId:         42,
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{{0x1}},
	}).Build()

	require.Error(t, v.validateMessageHeader(header))
}

func newValidator(t *testing.T) *headerValidator {
	cfg := &hardcodedValidatorConfig{virtualChainId: 42, nodeAddress: []byte{0x1}}
	v := newHeaderValidator(cfg, log.DefaultTestingLogger(t))
	return v
}

type hardcodedValidatorConfig struct {
	virtualChainId primitives.VirtualChainId
	nodeAddress    primitives.NodeAddress
}

func (c *hardcodedValidatorConfig) NodeAddress() primitives.NodeAddress {
	return c.nodeAddress
}

func (c *hardcodedValidatorConfig) VirtualChainId() primitives.VirtualChainId {
	return c.virtualChainId
}
