package management

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	adapterGossip "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"sync"
	"time"
)

type Config interface {
	ManagementUpdateInterval() time.Duration
}

type Provider interface { // update of data provider
	Get(ctx context.Context) (*VirtualChainManagementData, error)
}

type TopologyConsumer interface { // consumer that needs to get topology update message
	UpdateTopology(bgCtx context.Context, newPeers adapterGossip.GossipPeers)
}

type CommitteeTerm struct {
	AsOfReference uint64
	Committee     []primitives.NodeAddress
}

type SubscriptionTerm struct {
	AsOfReference uint64
	IsActive bool
}

type ProtocolVersionTerm struct {
	AsOfReference uint64
	Version      primitives.ProtocolVersion
}

type VirtualChainManagementData struct {
	CurrentReference uint64
	Topology         adapterGossip.GossipPeers
	Committees       []CommitteeTerm
	Subscriptions    []SubscriptionTerm
	ProtocolVersions []ProtocolVersionTerm
}

type Service struct {
	govnr.TreeSupervisor

	logger           log.Logger
	config           Config
	provider         Provider
	topologyConsumer TopologyConsumer

	sync.RWMutex
	data *VirtualChainManagementData
}

func NewManagement(parentCtx context.Context, config Config, provider Provider, topologyConsumer TopologyConsumer, parentLogger log.Logger) *Service {
	logger := parentLogger.WithTags(log.String("service", "management"))
	s := &Service{
		logger:           logger,
		config:           config,
		provider:         provider,
		topologyConsumer: topologyConsumer,
	}

	err := s.update(parentCtx)
	if err != nil {
		s.logger.Error("management provider failed to initializing the topology", log.Error(err))
		panic(fmt.Sprintf("failed initializing management provider, err=%s", err.Error())) // can't continue if no management
	}

	if config.ManagementUpdateInterval() > 0 {
		s.Supervise(s.startPollingForUpdates(parentCtx))
	}

	return s
}

func (s *Service) GetCurrentReference(ctx context.Context) uint64 {
	s.RLock()
	defer s.RUnlock()
	return s.data.CurrentReference
}

func (s *Service) GetTopology(ctx context.Context) adapterGossip.GossipPeers {
	s.RLock()
	defer s.RUnlock()
	return s.data.Topology
}

func (s *Service) GetCommittee(ctx context.Context, reference uint64) []primitives.NodeAddress {
	s.RLock()
	defer s.RUnlock()
	i := len(s.data.Committees) - 1
	for ; i > 0 && reference < s.data.Committees[i].AsOfReference; i-- {
	}
	return s.data.Committees[i].Committee
}

func (s *Service) GetSubscriptionStatus(ctx context.Context, reference uint64) bool {
	s.RLock()
	defer s.RUnlock()
	i := len(s.data.Subscriptions) - 1
	for ; i > 0 && reference < s.data.Subscriptions[i].AsOfReference; i-- {
	}
	return s.data.Subscriptions[i].IsActive
}

func (s *Service) GetProtocolVersion(ctx context.Context, reference uint64) primitives.ProtocolVersion {
	s.RLock()
	defer s.RUnlock()
	i := len(s.data.ProtocolVersions) - 1
	for ; i > 0 && reference < s.data.ProtocolVersions[i].AsOfReference; i-- {
	}
	return s.data.ProtocolVersions[i].Version
}

func (s *Service) write(newData *VirtualChainManagementData) {
	s.Lock()
	defer s.Unlock()
	s.data = newData
}

func (s *Service) update(ctx context.Context) error {
	data, err := s.provider.Get(ctx)
	if err != nil {
		return err
	}
	s.write(data)
	s.topologyConsumer.UpdateTopology(ctx, s.data.Topology)
	return nil
}

func (s *Service) startPollingForUpdates(bgCtx context.Context) govnr.ShutdownWaiter {
	return govnr.Forever(bgCtx, "management-service-updater", logfields.GovnrErrorer(s.logger), func() {
		for {
			select {
			case <-bgCtx.Done():
				return
			case <-time.After(s.config.ManagementUpdateInterval()):
				err := s.update(bgCtx)
				if err != nil {
					s.logger.Info("management provider failed to update the topology", log.Error(err))
				}
			}
		}
	})
}
