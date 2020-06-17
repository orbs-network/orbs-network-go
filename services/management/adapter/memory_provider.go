// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"bytes"
	"context"
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/management"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sort"
	"sync"
	"time"
)

const DEFAULT_GENESIS_ONSET = 1000

type MemoryConfig interface {
	GossipPeers() adapter.TransportPeers
	GenesisValidatorNodes() map[string]config.ValidatorNode
}

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

func NewMemoryProvider(cfg MemoryConfig, logger log.Logger) *MemoryProvider {
	committee := getCommitteeFromConfig(cfg)
	return &MemoryProvider{
		logger:                logger,
		currentReference:      primitives.TimestampSeconds(time.Now().Unix()),
		genesisReference:      primitives.TimestampSeconds(time.Now().Unix() - DEFAULT_GENESIS_ONSET),
		topology:              getTopologyFromConfig(cfg, logger),
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

func getTopologyFromConfig(cfg MemoryConfig, logger log.Logger) []*services.GossipPeer {
	peers := cfg.GossipPeers()
	topology := make([]*services.GossipPeer, 0, len(peers))
	for _, peer := range peers {
		if nodeAddress, err := hex.DecodeString(peer.HexOrbsAddress()); err != nil {
			// TODO post V2 moving all gossip out of config, there is nothing really to do here now
			logger.Error("Bad address for a configured gossip peer, ignored.")
		} else {
			topology = append(topology, &services.GossipPeer{Address: nodeAddress, Endpoint: peer.Endpoint(), Port: uint32(peer.Port())})
		}

	}
	return topology
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

// for acceptance tests use one or the other no support for cross "types" adding with respect to paging
func (mp *MemoryProvider) AddCommittee(reference primitives.TimestampSeconds, committee []primitives.NodeAddress) error {
	mp.Lock()
	defer mp.Unlock()

	if mp.committees[len(mp.committees)-1].AsOfReference > reference {
		return errors.Errorf("new committee cannot have an 'asOf' reference smaller than %d (and not %d)", mp.committees[len(mp.committees)-1].AsOfReference, reference)
	}

	mp.committees = append(mp.committees, management.CommitteeTerm{AsOfReference: reference, Members: committee})
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
