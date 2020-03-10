package management

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	adapterGossip "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/management/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"sync"
	"time"
)

type Config interface {
	ManagementUpdateInterval() time.Duration
//	ManagementFilePath() string
//	VirtualChainId() primitives.VirtualChainId
}

type CommitteeTerm struct {
	AsOfReference uint64
	Committee     []primitives.NodeAddress
}

type Service struct {
	govnr.TreeSupervisor

	logger   log.Logger
	config   Config
	provider adapter.Provider

	sync.RWMutex
	currentReference uint64
	topology         adapterGossip.GossipPeers
	committees       []*CommitteeTerm
}

func NewManagement(parentCtx context.Context, config Config, provider adapter.Provider, parentLogger log.Logger) *Service {
	logger := parentLogger.WithTags(log.String("service", "management"))
	s := &Service{
		logger:   logger,
		config:   config,
		provider: provider,
	}

	s.update(parentCtx, true)

	s.Supervise(s.startUpdating(parentCtx))

	return s
}

func (s *Service) GetCurrentReference(ctx context.Context) uint64 {
	s.RLock()
	defer s.RUnlock()
	return s.currentReference
}

func (s *Service) GetTopology(ctx context.Context) adapterGossip.GossipPeers {
	s.RLock()
	defer s.RUnlock()
	return s.topology
}

func (s *Service) GetCommittee(ctx context.Context, referenceNumber uint64) []primitives.NodeAddress {
	s.RLock()
	defer s.RUnlock()
	termIndex := len(s.committees) - 1
	for ; termIndex > 0 && referenceNumber < s.committees[termIndex].AsOfReference; termIndex-- {
	}
	return s.committees[termIndex].Committee
}

func (s *Service) isNewer (referenceNumber uint64) bool {
	return s.currentReference >= referenceNumber
}

func (s *Service) write(referenceNumber uint64, peers adapterGossip.GossipPeers, committees []*CommitteeTerm) {
	s.Lock()
	defer s.Unlock()
	s.currentReference = referenceNumber
	s.topology = peers
	s.committees = committees
}

func (s *Service) update(ctx context.Context, shouldPanic bool) {
	reference, peers, committees, err := s.provider.Update(ctx)
	if err != nil {
		s.logger.Info("management provider failed to update the topology", log.Error(err))
		if shouldPanic {
			panic(fmt.Sprintf("failed initializing management provider, err=%s", err.Error()))
		}
	}
	if s.isNewer(reference) { // currently not error if it's not newer
		s.write(reference, peers, committees)
	}
}


func (s *Service) startUpdating(bgCtx context.Context) govnr.ShutdownWaiter {
	return govnr.Forever(bgCtx, "management-service-updater", logfields.GovnrErrorer(s.logger), func() {
		for {
			select {
			case <-bgCtx.Done():
				return
			case <-time.After(s.config.ManagementUpdateInterval()):
				s.update(bgCtx, false)
			}
		}
	})
}

