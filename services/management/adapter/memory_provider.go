// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/management"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sync"
	"time"
)

const DEFAULT_GENESIS_ONSET = 1000

type MemoryProvider struct {
	logger log.Logger

	sync.RWMutex
	currentReference      primitives.TimestampSeconds
	genesisReference      primitives.TimestampSeconds
	topology              []*services.GossipPeer
	committees            []management.CommitteeTerm
	protocolVersions      []management.ProtocolVersionTerm
	isSubscriptionActives []management.SubscriptionTerm
}

// order of committee is important so make sure all calls from all nodes in a single network have same order !!
func NewMemoryProvider(committee []primitives.NodeAddress, topology []*services.GossipPeer, logger log.Logger) *MemoryProvider {
	return &MemoryProvider{
		logger:                logger,
		currentReference:      primitives.TimestampSeconds(time.Now().Unix()),
		genesisReference:      primitives.TimestampSeconds(time.Now().Unix() - DEFAULT_GENESIS_ONSET),
		topology:              topology,
		committees:            []management.CommitteeTerm{{AsOfReference: 0, Members: committee, Weights: generateWeightsForCommittee(committee)}},
		protocolVersions:      []management.ProtocolVersionTerm{{AsOfReference: 0, Version: config.MAXIMAL_CONSENSUS_BLOCK_PROTOCOL_VERSION}},
		isSubscriptionActives: []management.SubscriptionTerm{{AsOfReference: 0, IsActive: true}},
	}
}

func (mp *MemoryProvider) Get(ctx context.Context, referenceTime primitives.TimestampSeconds) (*management.VirtualChainManagementData, error) {
	mp.RLock()
	defer mp.RUnlock()

	return &management.VirtualChainManagementData{
		CurrentReference:   mp.currentReference,
		GenesisReference:   mp.genesisReference,
		StartPageReference: 0,
		EndPageReference:   mp.currentReference,
		CurrentTopology:    mp.topology,
		Committees:         mp.committees,
		Subscriptions:      mp.isSubscriptionActives,
		ProtocolVersions:   mp.protocolVersions,
	}, nil
}

func generateWeightsForCommittee(committee []primitives.NodeAddress) (weights []primitives.Weight) {
	weights = make([]primitives.Weight, len(committee))
	for i := range weights {
		weights[i] = 1
	}
	return
}

// for acceptance tests use one or the other no support for cross "types" adding with respect to paging
func (mp *MemoryProvider) AddCommittee(reference primitives.TimestampSeconds, committee []primitives.NodeAddress) error {
	mp.Lock()
	defer mp.Unlock()

	if mp.committees[len(mp.committees)-1].AsOfReference > reference {
		return errors.Errorf("new committee cannot have an 'asOf' reference smaller than %d (and not %d)", mp.committees[len(mp.committees)-1].AsOfReference, reference)
	}

	mp.committees = append(mp.committees, management.CommitteeTerm{AsOfReference: reference, Members: committee, Weights: generateWeightsForCommittee(committee)})
	mp.currentReference = reference
	return nil
}

func (mp *MemoryProvider) AddSubscription(reference primitives.TimestampSeconds, isActive bool) error {
	mp.Lock()
	defer mp.Unlock()

	if mp.committees[len(mp.isSubscriptionActives)-1].AsOfReference > reference {
		return errors.Errorf("new subscription cannot have an 'asOf' reference smaller than %d (and not %d)", mp.isSubscriptionActives[len(mp.isSubscriptionActives)-1].AsOfReference, reference)
	}

	mp.isSubscriptionActives = append(mp.isSubscriptionActives, management.SubscriptionTerm{AsOfReference: reference, IsActive: isActive})
	mp.currentReference = reference
	return nil
}
