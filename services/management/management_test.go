package management

import (
	"context"
	"github.com/orbs-network/lean-helix-go/test"
	adapterGossip "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestManagement_GetCommitteeWhenOnlyOneTerm(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])

			committee := cp.GetCommittee(ctx, 0)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, 10)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeAfterAnUpdateExists(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])
			termChangeHeight := uint64(10)
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])

			committee := cp.GetCommittee(ctx, termChangeHeight-1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWhenTwoChangesOneAfterOther(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])
			termChangeHeight := uint64(10)
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
			cp.addCommittee(termChangeHeight+1, testKeys.NodeAddressesForTests()[5:9])

			committee := cp.GetCommittee(ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+2)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWhenTwoChangesClose(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])
			termChangeHeight := uint64(10)
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
			cp.addCommittee(termChangeHeight+2, testKeys.NodeAddressesForTests()[5:9])

			committee := cp.GetCommittee(ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+2)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = cp.GetCommittee(ctx, termChangeHeight+3)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func newStaticCommitteeManagement(ref uint64, committee []primitives.NodeAddress) *Service{
	return &Service{
		data: &VirtualChainManagementData{
			CurrentReference: ref,
			Topology:         nil,
			Committees:       []CommitteeTerm{{ref, committee}},
			Subscriptions:    nil,
			ProtocolVersions: nil,
		},
	}
}

func (s* Service) addCommittee(ref uint64, committee []primitives.NodeAddress) {
	s.Lock()
	defer s.Unlock()
	s.data.Committees = append (s.data.Committees, CommitteeTerm{ref, committee})
}

func TestManagement_InternalDataOnlyChangesAfterUpdateWhenAutoUpdateDisabled(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider(0, testKeys.NodeAddressesForTests()[:4])
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger )

			committee := cp.GetCommittee(ctx, 5)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			p.changeCommittee(4, testKeys.NodeAddressesForTests()[1:5] )

			committee = cp.GetCommittee(ctx, 5)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			err := cp.update(ctx)
			require.NoError(t, err)
			committee = cp.GetCommittee(ctx, 5)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")
		})
	})
}

type staticProvider struct {
	sync.RWMutex
	ref uint64
	committee []primitives.NodeAddress
}

func newStaticProvider(ref uint64, committee []primitives.NodeAddress) *staticProvider{
	return &staticProvider{ ref:ref, committee:committee}
}

func (sp *staticProvider) Get(ctx context.Context) (*VirtualChainManagementData, error) {
	sp.RLock()
	defer sp.RUnlock()
	return &VirtualChainManagementData{
		CurrentReference: sp.ref,
		Topology:         make(adapterGossip.GossipPeers),
		Committees:       []CommitteeTerm{{sp.ref, sp.committee}},
		Subscriptions:    nil,
		ProtocolVersions: nil,
	}, nil
}

func (sp *staticProvider) UpdateTopology(bgCtx context.Context, newPeers adapterGossip.GossipPeers) {
	// does nothing just here for test to run
}

func (sp *staticProvider) changeCommittee(ref uint64, committee []primitives.NodeAddress) {
	sp.Lock()
	defer sp.Unlock()
	sp.ref = ref
	sp.committee = committee
}

type cfg struct {
}

func newConfig() *cfg {
	return &cfg{}
}

func (tc *cfg) ManagementUpdateInterval() time.Duration { // no auto update
	return 0
}
