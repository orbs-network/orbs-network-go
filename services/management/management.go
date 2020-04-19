package management

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
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

	metrics struct {
		currentRefTime *metric.Gauge
		genesisRefTime *metric.Gauge
		lasUpdateTime  *metric.Gauge
		numCommitteeEvents *metric.Gauge
		currentCommittee *metric.Text
		currentCommitteeRefTime *metric.Gauge
		numSubscriptionEvents *metric.Gauge
		currentSubscription *metric.Text
		currentSubscriptionRefTime *metric.Gauge
		numProtocolEvents *metric.Gauge
		currentProtocol *metric.Gauge
		currentProtocolRefTime *metric.Gauge
		currentTopology *metric.Text
	}

	sync.RWMutex
	data *VirtualChainManagementData
}

func NewManagement(parentCtx context.Context, config Config, provider Provider, topologyConsumer TopologyConsumer, parentLogger log.Logger, metricFactory metric.Factory) *service {
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

	s.initMetrics(metricFactory)

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
				s.updateMetrics()
			}
		}
	})
}

func (s *service) initMetrics(metricFactory metric.Factory) {
	s.metrics.currentRefTime = metricFactory.NewGauge("Management.CurrentRefTime")
	s.metrics.genesisRefTime = metricFactory.NewGauge("Management.GenesisRefTime")
	s.metrics.lasUpdateTime = metricFactory.NewGauge("Management.LastUpdateTime")
	s.metrics.numCommitteeEvents = metricFactory.NewGauge("Management.Committee.Count")
	s.metrics.currentCommittee = metricFactory.NewText("Management.Committee.Current")
	s.metrics.currentCommitteeRefTime = metricFactory.NewGauge("Management.Committee.CurrentRefTime")
	s.metrics.numSubscriptionEvents = metricFactory.NewGauge("Management.Subscription.Count")
	s.metrics.currentSubscription = metricFactory.NewText("Management.Subscription.Current")
	s.metrics.currentSubscriptionRefTime = metricFactory.NewGauge("Management.Subscription.CurrentRefTime")
	s.metrics.numProtocolEvents = metricFactory.NewGauge("Management.Protocol.Count")
	s.metrics.currentProtocol = metricFactory.NewGauge("Management.Protocol.Current")
	s.metrics.currentProtocolRefTime = metricFactory.NewGauge("Management.Protocol.CurrentRefTime")
	s.metrics.currentTopology = metricFactory.NewText("Management.Topology.Current")
}

func (s *service) updateMetrics() {
	s.RLock()
	defer s.RUnlock()
	s.metrics.currentRefTime.Update(int64(s.data.CurrentReference))
	s.metrics.genesisRefTime.Update(int64(s.data.GenesisReference))
	s.metrics.lasUpdateTime.Update(time.Now().Unix())
	s.metrics.currentTopology.Update(fmt.Sprintf("%v", s.data.CurrentTopology))

	s.metrics.numCommitteeEvents.Update(int64(len(s.data.Committees)))
	i := len(s.data.Committees) - 1
	for ; i > 0 && s.data.CurrentReference < s.data.Committees[i].AsOfReference; i-- {
	}
	s.metrics.currentCommitteeRefTime.Update(int64(s.data.Committees[i].AsOfReference))
	s.metrics.currentCommittee.Update(fmt.Sprintf("%v", s.data.Committees[i].Members))

	s.metrics.numSubscriptionEvents.Update(int64(len(s.data.Subscriptions)))
	i = len(s.data.Subscriptions) - 1
	for ; i > 0 && s.data.CurrentReference < s.data.Subscriptions[i].AsOfReference; i-- {
	}
	s.metrics.currentSubscriptionRefTime.Update(int64(s.data.Subscriptions[i].AsOfReference))
	if s.data.Subscriptions[i].IsActive {
		s.metrics.currentSubscription.Update("Active")
	} else {
		s.metrics.currentSubscription.Update("Non-Active")
	}

	s.metrics.numProtocolEvents.Update(int64(len(s.data.ProtocolVersions)))
	i = len(s.data.ProtocolVersions) - 1
	for ; i > 0 && s.data.CurrentReference < s.data.ProtocolVersions[i].AsOfReference; i-- {
	}
	s.metrics.currentProtocolRefTime.Update(int64(s.data.ProtocolVersions[i].AsOfReference))
	s.metrics.currentProtocol.Update(int64(s.data.ProtocolVersions[i].Version))
}
