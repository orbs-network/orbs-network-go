// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/management"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sort"
	"sync"
)

type MemoryConfig interface {
	GossipPeers() adapter.GossipPeers
	GenesisValidatorNodes() map[string]config.ValidatorNode
	ProtocolVersion() primitives.ProtocolVersion
}

type MemoryProvider struct {
	logger log.Logger

	sync.RWMutex
	currentReference      uint64
	topology              adapter.GossipPeers
	committees            []management.CommitteeTerm
	protocolVersions      []management.ProtocolVersionTerm
	isSubscriptionActives []management.SubscriptionTerm
}

func NewMemoryProvider(config MemoryConfig, logger log.Logger) *MemoryProvider {
	committee := getCommitteeFromConfig(config)
	return &MemoryProvider{
		logger:                logger,
		currentReference:      0,
		topology:              config.GossipPeers(),
		committees:            []management.CommitteeTerm{{AsOfReference: 0, Committee: committee}},
		protocolVersions:      []management.ProtocolVersionTerm{{AsOfReference: 0, Version: config.ProtocolVersion()}},
		isSubscriptionActives: []management.SubscriptionTerm{{AsOfReference: 0, IsActive: true}},
	}
}

func (mp *MemoryProvider) Get(ctx context.Context) (*management.VirtualChainManagementData, error) {
	mp.RLock()
	defer mp.RUnlock()

	return &management.VirtualChainManagementData{
		CurrentReference: mp.currentReference,
		Topology:         mp.topology,
		Committees:       mp.committees,
		Subscriptions:    mp.isSubscriptionActives,
        ProtocolVersions: mp.protocolVersions,
	}, nil
}

// for acceptance tests
func (mp *MemoryProvider) AddCommittee(reference uint64, committee []primitives.NodeAddress) error {
	mp.Lock()
	defer mp.Unlock()

	if mp.committees[len(mp.committees)-1].AsOfReference >= reference {
		return errors.Errorf("new committee must have an 'asOf' reference bigger than %d (and not %d)", mp.committees[len(mp.committees)-1].AsOfReference, reference)
	}

	mp.committees = append(mp.committees, management.CommitteeTerm{AsOfReference: reference, Committee: committee})
	mp.currentReference = reference
	return nil
}

func getCommitteeFromConfig(config MemoryConfig) []primitives.NodeAddress {
	allNodes := config.GenesisValidatorNodes()
	var committee []primitives.NodeAddress

	for _, nodeAddress := range allNodes {
		committee = append(committee, nodeAddress.NodeAddress())
	}

	sort.SliceStable(committee, func(i, j int) bool {
		return bytes.Compare(committee[i], committee[j]) > 0
	})
	return committee
}
