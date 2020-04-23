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

const DEFAULT_REF_TIME = 1492983000
const DEFAULT_GENESIS_REF_TIME = 1492982000

type MemoryConfig interface {
	GossipPeers() adapter.GossipPeers
	GenesisValidatorNodes() map[string]config.ValidatorNode
}

type MemoryProvider struct {
	logger log.Logger

	sync.RWMutex
	currentReference      primitives.TimestampSeconds
	genesisReference      primitives.TimestampSeconds
	topology              adapter.GossipPeers
	committees            []management.CommitteeTerm
	protocolVersions      []management.ProtocolVersionTerm
	isSubscriptionActives []management.SubscriptionTerm
}

func NewMemoryProvider(cfg MemoryConfig, logger log.Logger) *MemoryProvider {
	committee := getCommitteeFromConfig(cfg)
	return &MemoryProvider{
		logger:                logger,
		currentReference:      DEFAULT_REF_TIME,
		genesisReference:      DEFAULT_GENESIS_REF_TIME,
		topology:              cfg.GossipPeers(),
		committees:            []management.CommitteeTerm{{AsOfReference: 0, Members: committee}},
		protocolVersions:      []management.ProtocolVersionTerm{{AsOfReference: 0, Version: config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE}},
		isSubscriptionActives: []management.SubscriptionTerm{{AsOfReference: 0, IsActive: true}},
	}
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

func (mp *MemoryProvider) Get(ctx context.Context) (*management.VirtualChainManagementData, error) {
	mp.RLock()
	defer mp.RUnlock()

	return &management.VirtualChainManagementData{
		CurrentReference: mp.currentReference,
		GenesisReference: mp.genesisReference,
		CurrentTopology:  mp.topology,
		Committees:       mp.committees,
		Subscriptions:    mp.isSubscriptionActives,
        ProtocolVersions: mp.protocolVersions,
	}, nil
}

// for acceptance tests
func (mp *MemoryProvider) AddCommittee(reference primitives.TimestampSeconds, committee []primitives.NodeAddress) error {
	mp.Lock()
	defer mp.Unlock()

	if mp.committees[len(mp.committees)-1].AsOfReference >= reference {
		return errors.Errorf("new committee must have an 'asOf' reference bigger than %d (and not %d)", mp.committees[len(mp.committees)-1].AsOfReference, reference)
	}

	mp.committees = append(mp.committees, management.CommitteeTerm{AsOfReference: reference, Members: committee})
	mp.currentReference = reference
	return nil
}

func (mp *MemoryProvider) AddSubscription(reference primitives.TimestampSeconds, isActive bool) error {
	mp.Lock()
	defer mp.Unlock()

	if mp.committees[len(mp.isSubscriptionActives)-1].AsOfReference >= reference {
		return errors.Errorf("new subscription must have an 'asOf' reference bigger than %d (and not %d)", mp.isSubscriptionActives[len(mp.isSubscriptionActives)-1].AsOfReference, reference)
	}

	mp.isSubscriptionActives = append(mp.isSubscriptionActives, management.SubscriptionTerm{AsOfReference: reference, IsActive: isActive})
	mp.currentReference = reference
	return nil
}
