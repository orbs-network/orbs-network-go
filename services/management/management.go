package management

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	adapterGossip "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"sync"
	"time"
)

type Config interface {
	ManagementPollingInterval() time.Duration
}

type Provider interface { // update of data provider
	Get(ctx context.Context) (*VirtualChainManagementData, error)
}

type TopologyConsumer interface { // consumer that needs to get topology update message
	UpdateTopology(bgCtx context.Context, newPeers adapterGossip.GossipPeers)
}

type CommitteeTerm struct {
	AsOfReference primitives.TimestampSeconds
	Members       []primitives.NodeAddress
}

type SubscriptionTerm struct {
	AsOfReference primitives.TimestampSeconds
	IsActive bool
}

type ProtocolVersionTerm struct {
	AsOfReference primitives.TimestampSeconds
	Version      primitives.ProtocolVersion
}

type VirtualChainManagementData struct {
	CurrentReference primitives.TimestampSeconds
	GenesisReference primitives.TimestampSeconds
	CurrentTopology  adapterGossip.GossipPeers
	Committees       []CommitteeTerm
	Subscriptions    []SubscriptionTerm
	ProtocolVersions []ProtocolVersionTerm
}

type service struct {
	govnr.TreeSupervisor

	logger           log.Logger
	config           Config
	provider         Provider
	topologyConsumer TopologyConsumer

	sync.RWMutex
	data *VirtualChainManagementData
}

func NewManagement(parentCtx context.Context, config Config, provider Provider, topologyConsumer TopologyConsumer, parentLogger log.Logger) *service {
	logger := parentLogger.WithTags(log.String("service", "management"))
	s := &service{
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

	if config.ManagementPollingInterval() > 0 {
		s.Supervise(s.startPollingForUpdates(parentCtx))
	}

	return s
}

func (s *service) GetCurrentReference(ctx context.Context, input *services.GetCurrentReferenceInput) (*services.GetCurrentReferenceOutput, error) {
	s.RLock()
	defer s.RUnlock()
	return &services.GetCurrentReferenceOutput{
		CurrentReference: s.data.CurrentReference,
	}, nil
}

func (s *service) GetGenesisReference(ctx context.Context, input *services.GetGenesisReferenceInput) (*services.GetGenesisReferenceOutput, error) {
	s.RLock()
	defer s.RUnlock()
	return &services.GetGenesisReferenceOutput{
		CurrentReference: s.data.CurrentReference,
		GenesisReference: s.data.GenesisReference,
	}, nil
}

func (s *service) GetProtocolVersion(ctx context.Context, input *services.GetProtocolVersionInput) (*services.GetProtocolVersionOutput, error) {
	s.RLock()
	defer s.RUnlock()
	i := len(s.data.ProtocolVersions) - 1
	for ; i > 0 && input.Reference < s.data.ProtocolVersions[i].AsOfReference; i-- {
	}

	return &services.GetProtocolVersionOutput{
		ProtocolVersion: s.data.ProtocolVersions[i].Version,
	}, nil
}

func (s *service) GetCommittee(ctx context.Context, input *services.GetCommitteeInput) (*services.GetCommitteeOutput, error) {
	s.RLock()
	defer s.RUnlock()
	i := len(s.data.Committees) - 1
	for ; i > 0 && input.Reference < s.data.Committees[i].AsOfReference; i-- {
	}

	return &services.GetCommitteeOutput{
		Members: s.data.Committees[i].Members,
	}, nil
}

func (s *service) GetSubscriptionStatus(ctx context.Context, input *services.GetSubscriptionStatusInput) (*services.GetSubscriptionStatusOutput, error) {
	s.RLock()
	defer s.RUnlock()
	i := len(s.data.Subscriptions) - 1
	for ; i > 0 && input.Reference < s.data.Subscriptions[i].AsOfReference; i-- {
	}

	return &services.GetSubscriptionStatusOutput{
		SubscriptionStatusIsActive: s.data.Subscriptions[i].IsActive,
	}, nil
}

func (s *service) write(newData *VirtualChainManagementData) {
	s.Lock()
	defer s.Unlock()
	s.data = newData
}

func (s *service) update(ctx context.Context) error {
	data, err := s.provider.Get(ctx)
	if err != nil {
		return err
	}
	s.write(data)
	s.topologyConsumer.UpdateTopology(ctx, s.data.CurrentTopology)
	return nil
}

func (s *service) startPollingForUpdates(bgCtx context.Context) govnr.ShutdownWaiter {
	return govnr.Forever(bgCtx, "management-service-updater", logfields.GovnrErrorer(s.logger), func() {
		for {
			select {
			case <-bgCtx.Done():
				return
			case <-time.After(s.config.ManagementPollingInterval()):
				err := s.update(bgCtx)
				if err != nil {
					s.logger.Info("management provider failed to update the topology", log.Error(err))
				}
			}
		}
	})
}
