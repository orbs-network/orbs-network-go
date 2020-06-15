package management

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"strings"
	"sync"
	"time"
)

type Config interface {
	ManagementPollingInterval() time.Duration
}

type Provider interface { // update of data provider
	Get(ctx context.Context, referenceTime primitives.TimestampSeconds) (*VirtualChainManagementData, error)
}

type TopologyConsumer interface { // consumer that needs to get topology update message
	UpdateTopology(ctx context.Context, input *services.UpdateTopologyInput) (*services.UpdateTopologyOutput, error)
}

type CommitteeTerm struct {
	AsOfReference primitives.TimestampSeconds
	Members       []primitives.NodeAddress
}

type SubscriptionTerm struct {
	AsOfReference primitives.TimestampSeconds
	IsActive      bool
}

type ProtocolVersionTerm struct {
	AsOfReference primitives.TimestampSeconds
	Version       primitives.ProtocolVersion
}

type VirtualChainManagementData struct {
	CurrentReference   primitives.TimestampSeconds
	GenesisReference   primitives.TimestampSeconds
	StartPageReference primitives.TimestampSeconds
	EndPageReference   primitives.TimestampSeconds
	CurrentTopology    []*services.GossipPeer
	Committees         []CommitteeTerm
	Subscriptions      []SubscriptionTerm
	ProtocolVersions   []ProtocolVersionTerm
}

type service struct {
	govnr.TreeSupervisor

	logger           log.Logger
	config           Config
	provider         Provider
	topologyConsumer TopologyConsumer

	metrics struct {
		currentRefTime             *metric.Gauge
		genesisRefTime             *metric.Gauge
		pageStartRefTime           *metric.Gauge
		pageEndRefTime             *metric.Gauge
		lasUpdateTime              *metric.Gauge
		numCommitteeEvents         *metric.Gauge
		currentCommittee           *metric.Text
		currentCommitteeRefTime    *metric.Gauge
		numSubscriptionEvents      *metric.Gauge
		currentSubscription        *metric.Text
		currentSubscriptionRefTime *metric.Gauge
		numProtocolEvents          *metric.Gauge
		currentProtocol            *metric.Gauge
		currentProtocolRefTime     *metric.Gauge
		currentTopology            *metric.Text
		pageCachedStartRefTime     *metric.Gauge
		pageCachedEndRefTime       *metric.Gauge
	}

	sync.RWMutex
	data *VirtualChainManagementData
	cachedHistoricData *VirtualChainManagementData // data holder cannot be nil !
}

func NewManagement(parentCtx context.Context, config Config, provider Provider, topologyConsumer TopologyConsumer, parentLogger log.Logger, metricFactory metric.Factory) *service {
	logger := parentLogger.WithTags(log.String("service", "management"))
	s := &service{
		logger:           logger,
		config:           config,
		provider:         provider,
		topologyConsumer: topologyConsumer,
		cachedHistoricData: &VirtualChainManagementData{}, // data holder cannot be nil !
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

func (s *service) getCurrentReference(systemRef primitives.TimestampSeconds) primitives.TimestampSeconds {
	s.RLock()
	defer s.RUnlock()
	if s.data.CurrentReference != 0 {
		return s.data.CurrentReference
	}

	committeeTerm := s.getCommittee(systemRef, s.data.Committees)
	subTerm := s.getSubscriptionStatus(systemRef, s.data.Subscriptions)
	pvTerm := s.getProtocolVersion(systemRef, s.data.ProtocolVersions)
	return maxOf(committeeTerm.AsOfReference, subTerm.AsOfReference, pvTerm.AsOfReference)
}

func maxOf(a, b, c primitives.TimestampSeconds) primitives.TimestampSeconds {
	if a > b {
		if a > c {
			return a
		} else {
			return c
		}
	} else if b > c {
		return b
	} else {
		return c
	}

}


func (s *service) GetCurrentReference(ctx context.Context, input *services.GetCurrentReferenceInput) (*services.GetCurrentReferenceOutput, error) {
	s.RLock()
	defer s.RUnlock()
	return &services.GetCurrentReferenceOutput{
		CurrentReference: s.getCurrentReference(input.SystemTime),
	}, nil
}

func (s *service) GetGenesisReference(ctx context.Context, input *services.GetGenesisReferenceInput) (*services.GetGenesisReferenceOutput, error) {
	s.RLock()
	defer s.RUnlock()
	return &services.GetGenesisReferenceOutput{
		CurrentReference: s.getCurrentReference(input.SystemTime),
		GenesisReference: s.data.GenesisReference,
	}, nil
}

func (s *service) validateRefTimeRequestNotInTheFuture(refTime primitives.TimestampSeconds) error {
	if refTime > s.data.EndPageReference {
		return errors.Errorf("ReferenceTime %d is in the future (current last valid is %d)", refTime, s.data.EndPageReference)
	}
	return nil
}

func (s *service) getCommittee(refTime primitives.TimestampSeconds, committees []CommitteeTerm) *CommitteeTerm {
	s.RLock()
	defer s.RUnlock()
	i := len(committees) - 1
	for ; i > 0 && refTime < committees[i].AsOfReference; i-- {
	}
	return &committees[i]
}

func (s *service) GetCommittee(ctx context.Context, input *services.GetCommitteeInput) (*services.GetCommitteeOutput, error) {
	if err := s.validateRefTimeRequestNotInTheFuture(input.Reference); err != nil {
		return nil, err
	}

	if input.Reference >= s.data.StartPageReference {
		return &services.GetCommitteeOutput{
			Members: s.getCommittee(input.Reference, s.data.Committees).Members,
		}, nil
	}

	if err := s.updateHistoric(ctx, input.Reference); err != nil {
		return nil, err
	}

	return &services.GetCommitteeOutput{
		Members: s.getCommittee(input.Reference, s.cachedHistoricData.Committees).Members,
	}, nil
}

func (s *service) getSubscriptionStatus(refTime primitives.TimestampSeconds, subscrptions []SubscriptionTerm) *SubscriptionTerm {
	s.RLock()
	defer s.RUnlock()
	i := len(subscrptions) - 1
	for ; i > 0 && refTime < subscrptions[i].AsOfReference; i-- {
	}
	return &subscrptions[i]
}

func (s *service) GetSubscriptionStatus(ctx context.Context, input *services.GetSubscriptionStatusInput) (*services.GetSubscriptionStatusOutput, error) {
	if err := s.validateRefTimeRequestNotInTheFuture(input.Reference); err != nil {
		return nil, err
	}

	if input.Reference >= s.data.StartPageReference {
		return &services.GetSubscriptionStatusOutput{
			SubscriptionStatusIsActive: s.getSubscriptionStatus(input.Reference, s.data.Subscriptions).IsActive,
		}, nil
	}

	if err := s.updateHistoric(ctx, input.Reference); err != nil {
		return nil, err
	}

	return &services.GetSubscriptionStatusOutput{
		SubscriptionStatusIsActive: s.getSubscriptionStatus(input.Reference, s.cachedHistoricData.Subscriptions).IsActive,
	}, nil
}

func (s *service) getProtocolVersion(refTime primitives.TimestampSeconds, protocolVersions []ProtocolVersionTerm) *ProtocolVersionTerm {
	s.RLock()
	defer s.RUnlock()
	i := len(protocolVersions) - 1
	for ; i > 0 && refTime < protocolVersions[i].AsOfReference; i-- {
	}
	return &protocolVersions[i]
}

func (s *service) GetProtocolVersion(ctx context.Context, input *services.GetProtocolVersionInput) (*services.GetProtocolVersionOutput, error) {
	if err := s.validateRefTimeRequestNotInTheFuture(input.Reference); err != nil {
		return nil, err
	}

	if input.Reference >= s.data.StartPageReference {
		return &services.GetProtocolVersionOutput{
			ProtocolVersion: s.getProtocolVersion(input.Reference, s.data.ProtocolVersions).Version,
		}, nil
	}

	if err := s.updateHistoric(ctx, input.Reference); err != nil {
		return nil, err
	}

	return &services.GetProtocolVersionOutput{
		ProtocolVersion: s.getProtocolVersion(input.Reference, s.cachedHistoricData.ProtocolVersions).Version,
	}, nil
}

func (s *service) write(newData *VirtualChainManagementData) {
	s.Lock()
	defer s.Unlock()
	s.data = newData
}

func (s *service) update(ctx context.Context) error {
	data, err := s.provider.Get(ctx, 0)
	if err != nil {
		return err
	}
	s.write(data)
	s.topologyConsumer.UpdateTopology(ctx, &services.UpdateTopologyInput{Peers: s.data.CurrentTopology})
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

func (s *service) updateHistoric(ctx context.Context, refTime primitives.TimestampSeconds) error {
	if refTime < s.cachedHistoricData.StartPageReference || refTime > s.cachedHistoricData.EndPageReference {
		historic, err := s.provider.Get(ctx, refTime)
		if err != nil {
			return errors.Wrapf(err, "management provider failed to get historic reference data for reference-time %d", refTime)
		}
		s.Lock()
		defer s.Unlock()
		//s.cachedHistoricData.CurrentReference = historic.CurrentReference
		//s.cachedHistoricData.GenesisReference = historic.GenesisReference
		//s.cachedHistoricData.StartPageReference = historic.StartPageReference
		//s.cachedHistoricData.EndPageReference = historic.EndPageReference
		//s.cachedHistoricData.CurrentTopology = historic.CurrentTopology
		//s.cachedHistoricData.Committees = historic.Committees
		//s.cachedHistoricData.Subscriptions = historic.Subscriptions
		//s.cachedHistoricData.ProtocolVersions = historic.ProtocolVersions
		s.cachedHistoricData = historic
		s.metrics.pageCachedStartRefTime.Update(int64(s.cachedHistoricData.StartPageReference))
		s.metrics.pageCachedEndRefTime.Update(int64(s.cachedHistoricData.EndPageReference))
	}
	return nil
}

func (s *service) initMetrics(metricFactory metric.Factory) {
	s.metrics.currentRefTime = metricFactory.NewGauge("Management.Data.CurrentRefTime")
	s.metrics.genesisRefTime = metricFactory.NewGauge("Management.Data.GenesisRefTime")
	s.metrics.pageStartRefTime = metricFactory.NewGauge("Management.Data.PageStartRefTime")
	s.metrics.pageEndRefTime = metricFactory.NewGauge("Management.Data.PageEndRefTime")
	s.metrics.lasUpdateTime = metricFactory.NewGauge("Management.Data.LastUpdateTime")
	s.metrics.pageCachedStartRefTime = metricFactory.NewGauge("Management.Data.CachedStartRefTime")
	s.metrics.pageCachedEndRefTime = metricFactory.NewGauge("Management.Data.CashedEndRefTime")
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

	currentRef := s.data.CurrentReference

	s.metrics.currentRefTime.Update(int64(currentRef))
	s.metrics.genesisRefTime.Update(int64(s.data.GenesisReference))
	s.metrics.pageStartRefTime.Update(int64(s.data.StartPageReference))
	s.metrics.pageEndRefTime.Update(int64(s.data.EndPageReference))
	s.metrics.lasUpdateTime.Update(time.Now().Unix())
	topologyStringArray := make([]string, len(s.data.CurrentTopology))
	for j, peer := range s.data.CurrentTopology {
		topologyStringArray[j] = fmt.Sprintf("{\"Address\":\"%s\",\"Endpoint\":\"%s\",\"Port\":%s,}", peer.StringAddress(), peer.StringEndpoint(), peer.StringPort())
	}
	s.metrics.currentTopology.Update("[" + strings.Join(topologyStringArray, ", ") + "]")

	s.metrics.numCommitteeEvents.Update(int64(len(s.data.Committees)))
	committeeTerm := s.getCommittee(currentRef, s.data.Committees)
	s.metrics.currentCommitteeRefTime.Update(int64(committeeTerm.AsOfReference))
	committeeStringArray := make([]string, len(committeeTerm.Members))
	for j, nodeAddress := range committeeTerm.Members {
		committeeStringArray[j] = fmt.Sprintf("\"%v\"", nodeAddress) // %v is because NodeAddress has .String()
	}
	s.metrics.currentCommittee.Update("[" + strings.Join(committeeStringArray, ", ") + "]")

	s.metrics.numSubscriptionEvents.Update(int64(len(s.data.Subscriptions)))
	subscriptionTerm := s.getSubscriptionStatus(currentRef, s.data.Subscriptions)
	s.metrics.currentSubscriptionRefTime.Update(int64(subscriptionTerm.AsOfReference))
	if subscriptionTerm.IsActive {
		s.metrics.currentSubscription.Update("Active")
	} else {
		s.metrics.currentSubscription.Update("Non-Active")
	}

	s.metrics.numProtocolEvents.Update(int64(len(s.data.ProtocolVersions)))
	pvTerm := s.getProtocolVersion(currentRef, s.data.ProtocolVersions)
	s.metrics.currentProtocolRefTime.Update(int64(pvTerm.AsOfReference))
	s.metrics.currentProtocol.Update(int64(pvTerm.Version))
}
