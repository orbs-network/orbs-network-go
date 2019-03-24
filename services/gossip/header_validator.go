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
	"github.com/pkg/errors"
)

type validatorConfig interface {
	NodeAddress() primitives.NodeAddress
	VirtualChainId() primitives.VirtualChainId
}

type headerValidator struct {
	config validatorConfig
	logger log.BasicLogger
}

func newHeaderValidator(config validatorConfig, logger log.BasicLogger) *headerValidator {
	return &headerValidator{config: config, logger: logger}
}

func (v *headerValidator) validateMessageHeader(header *gossipmessages.Header) error {
	if header.VirtualChainId() != v.config.VirtualChainId() {
		return errors.Errorf("message is addressed at virtual chain id %d but my virtual chain id is %d", header.VirtualChainId(), v.config.VirtualChainId())
	}

	if header.RecipientMode() == gossipmessages.RECIPIENT_LIST_MODE_LIST && !isInRecipientList(v.config.NodeAddress(), header.RecipientNodeAddressesIterator()) {
		return errors.Errorf("message is addressed to a list this node isn't a member of")
	}

	if header.RecipientMode() == gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST && isInRecipientList(v.config.NodeAddress(), header.RecipientNodeAddressesIterator()) {
		return errors.Errorf("message is addressed to a list excluding this node")
	}

	return nil
}

func isInRecipientList(me primitives.NodeAddress, recipientIterator *gossipmessages.HeaderRecipientNodeAddressesIterator) bool {
	for recipientIterator.HasNext() {
		if me.Equal(recipientIterator.NextRecipientNodeAddresses()) {
			return true
		}
	}

	return false
}
