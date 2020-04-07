package management

import (
	"context"
	"github.com/orbs-network/lean-helix-go/test"
	adapterGossip "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestManagement_GetCommitteeWhenOnlyOneTerm(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])

			committee := getCommittee(cp, ctx, 0)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee = getCommittee(cp, ctx, 10)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeAfterAnUpdateExists(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])
			termChangeHeight := primitives.TimestampSeconds(10)
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])

			committee := getCommittee(cp, ctx, termChangeHeight-1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee = getCommittee(cp, ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = getCommittee(cp, ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWhenTwoChangesOneAfterOther(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])
			termChangeHeight := primitives.TimestampSeconds(10)
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
			cp.addCommittee(termChangeHeight+1, testKeys.NodeAddressesForTests()[5:9])

			committee := getCommittee(cp, ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = getCommittee(cp, ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = getCommittee(cp, ctx, termChangeHeight+2)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWhenTwoChangesClose(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			cp := newStaticCommitteeManagement(0, testKeys.NodeAddressesForTests()[:4])
			termChangeHeight := primitives.TimestampSeconds(10)
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
			cp.addCommittee(termChangeHeight+2, testKeys.NodeAddressesForTests()[5:9])

			committee := getCommittee(cp, ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = getCommittee(cp, ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = getCommittee(cp, ctx, termChangeHeight+2)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = getCommittee(cp, ctx, termChangeHeight+3)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func getCommittee(m *service, ctx context.Context, reference primitives.TimestampSeconds) []primitives.NodeAddress {
	committee, err := m.GetCommittee(ctx, &services.GetCommitteeInput{Reference:reference})
	if err != nil {
		return nil
	}
	return committee.Members
}

func newStaticCommitteeManagement(ref primitives.TimestampSeconds, committee []primitives.NodeAddress) *service {
	return &service{
		data: &VirtualChainManagementData{
			CurrentReference: ref,
			CurrentTopology:  nil,
			Committees:       []CommitteeTerm{{ref, committee}},
			Subscriptions:    nil,
			ProtocolVersions: nil,
		},
	}
}

func (s*service) addCommittee(ref primitives.TimestampSeconds, committee []primitives.NodeAddress) {
	s.Lock()
	defer s.Unlock()
	s.data.Committees = append (s.data.Committees, CommitteeTerm{ref, committee})
}

func TestManagement_InternalDataOnlyChangesAfterUpdateWhenAutoUpdateDisabled(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider(0, testKeys.NodeAddressesForTests()[:4])
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger )

			committee := getCommittee(cp, ctx, 5)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			p.changeCommittee(4, testKeys.NodeAddressesForTests()[1:5] )

			committee = getCommittee(cp, ctx, 5)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			err := cp.update(ctx)
			require.NoError(t, err)
			committee = getCommittee(cp, ctx, 5)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")
		})
	})
}

type staticProvider struct {
	sync.RWMutex
	ref primitives.TimestampSeconds
	committee []primitives.NodeAddress
}

func newStaticProvider(ref primitives.TimestampSeconds, committee []primitives.NodeAddress) *staticProvider{
	return &staticProvider{ ref:ref, committee:committee}
}

func (sp *staticProvider) Get(ctx context.Context) (*VirtualChainManagementData, error) {
	sp.RLock()
	defer sp.RUnlock()
	return &VirtualChainManagementData{
		CurrentReference: sp.ref,
		CurrentTopology:  make(adapterGossip.GossipPeers),
		Committees:       []CommitteeTerm{{sp.ref, sp.committee}},
		Subscriptions:    nil,
		ProtocolVersions: nil,
	}, nil
}

func (sp *staticProvider) UpdateTopology(bgCtx context.Context, newPeers adapterGossip.GossipPeers) {
	// does nothing just here for test to run
}

func (sp *staticProvider) changeCommittee(ref primitives.TimestampSeconds, committee []primitives.NodeAddress) {
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

func (tc *cfg) ManagementPollingInterval() time.Duration { // no auto update
	return 0
}
