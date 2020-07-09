package management

import (
	"context"
	"github.com/orbs-network/lean-helix-go/test"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestManagement_maxOf(t *testing.T) {
	groupsOfThree := [][]primitives.TimestampSeconds{{5,4,3},{5,3,4},{3,5,4},{4,3,5},{3,4,5},{3,5,4}}
	for i := range groupsOfThree {
		require.EqualValues(t, 5, maxOf(groupsOfThree[i][0], groupsOfThree[i][1], groupsOfThree[i][2]))
	}
}

func TestManagement_getCurrentReference_RefExists(t *testing.T) {
	s := &service{
		data: &VirtualChainManagementData{
			CurrentReference:   50,
		},
	}

	// input of systemRef is ignored
	require.EqualValues(t, 50, getCurrentReference(5, s.data))
	require.EqualValues(t, 50, getCurrentReference(primitives.TimestampSeconds(time.Now().Unix()), s.data))
}

func TestManagement_getCurrentReference_NoRef(t *testing.T) {
	now := primitives.TimestampSeconds(time.Now().Unix())

	s := &service{
		data: &VirtualChainManagementData{
			CurrentReference:   0,
			Committees:         []CommitteeTerm{{now-5000, nil, nil}, {now+5000, nil, nil}},
			Subscriptions:      []SubscriptionTerm{{now-40000, false}, {now-4000, false}, {now+100, false}},
			ProtocolVersions:   []ProtocolVersionTerm{{now-200, 5} /*unreachable*/, {now-4500, 5}, {now+1000, 5}},
		},
	}

	require.EqualValues(t, now-4000, getCurrentReference(now, s.data))
}

const ACurrentRef = primitives.TimestampSeconds(5501)
const AHistoricRef = primitives.TimestampSeconds(3500)

func TestManagement_GetCommitteeNotSupportFuture(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())

			committee := getCommitteeOrNil(cp, ctx, ACurrentRef + 2000)
			require.Nil(t, committee, "should not get a committee")
		})
	})
}

func TestManagement_GetCommitteeWhenOnlyOneTerm(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())

			committee := getCommitteeOrNil(cp, ctx, ACurrentRef)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, ACurrentRef+10)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeAfterAnUpdateExists(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())
			termChangeHeight := ACurrentRef + 10
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])

			committee := getCommitteeOrNil(cp, ctx, termChangeHeight-1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWhenTwoChangesOneAfterOther(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())
			termChangeHeight := ACurrentRef + 10
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
			cp.addCommittee(termChangeHeight+1, testKeys.NodeAddressesForTests()[5:9])

			committee := getCommitteeOrNil(cp, ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight+2)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWhenTwoChangesClose(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())
			termChangeHeight := ACurrentRef + 10
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
			cp.addCommittee(termChangeHeight+2, testKeys.NodeAddressesForTests()[5:9])

			committee := getCommitteeOrNil(cp, ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight+2)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight+3)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWhenTwoInSameRef(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())
			termChangeHeight := ACurrentRef + 10
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[1:5])
			cp.addCommittee(termChangeHeight, testKeys.NodeAddressesForTests()[5:9])
			cp.addCommittee(termChangeHeight+10, testKeys.NodeAddressesForTests()[2:5])

			committee := getCommitteeOrNil(cp, ctx, termChangeHeight)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight-1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight+1)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[5:9], committee, "wrong committee values")

			committee = getCommitteeOrNil(cp, ctx, termChangeHeight+11)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[2:5], committee, "wrong committee values")
		})
	})
}

func TestManagement_InternalDataUpdateLogic(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())

			committee := getCommitteeOrNil(cp, ctx, ACurrentRef)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			// update provider under the hood
			p.ref = ACurrentRef
			p.committee = testKeys.NodeAddressesForTests()[1:5]

			// before manual update of service
			committee = getCommitteeOrNil(cp, ctx, ACurrentRef)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")

			err := cp.update(ctx) // manual update of service
			require.NoError(t, err)
			committee = getCommitteeOrNil(cp, ctx, ACurrentRef)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[1:5], committee, "wrong committee values")
		})
	})
}

func TestManagement_InternalHistoricData_UpdateLogic(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())

			require.Nil(t, cp.cachedHistoricData.Committees, "cached reference was not empty to begin with")

			cp.tryGetHistoricData(ctx, 7) // any number to trigger

			require.Equal(t, AHistoricRef, cp.cachedHistoricData.CurrentReference, "cache should hold the static historic")

			// update provider under the hood if update succeeds cached value will change
			p.historicRef = ACurrentRef

			cp.tryGetHistoricData(ctx, cp.cachedHistoricData.StartPageReference) // min still in cache
			require.Equal(t, AHistoricRef, cp.cachedHistoricData.CurrentReference, "cache should not change while ask in range")
			cp.tryGetHistoricData(ctx, cp.cachedHistoricData.EndPageReference) // max still in cache
			require.Equal(t, AHistoricRef, cp.cachedHistoricData.CurrentReference, "cache should not change while ask in range")

			cp.tryGetHistoricData(ctx, ACurrentRef) // outside cached ref
			require.Equal(t, ACurrentRef, cp.cachedHistoricData.CurrentReference)
		})
	})
}

func TestManagement_GetCommitteeWithHistoric(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())
			require.Nil(t, cp.cachedHistoricData.Committees, "cached reference was not empty to begin with")

			committee := getCommitteeOrNil(cp, ctx, AHistoricRef)
			require.NotNil(t, cp.cachedHistoricData.Committees, "cached reference was not updated")
			require.EqualValues(t, testKeys.NodeAddressesForTests()[2:6], committee, "wrong committee values")

			cp.cachedHistoricData = nil // make sure cache is not used here
			committee = getCommitteeOrNil(cp, ctx, ACurrentRef)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[:4], committee, "wrong committee values")
		})
	})
}

func TestManagement_GetCommitteeWithTwoHistoricCalls(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		test.WithContext(func(ctx context.Context) {
			p := newStaticProvider()
			cp := NewManagement(ctx, newConfig(), p, p, harness.Logger, metric.NewRegistry())
			require.Nil(t, cp.cachedHistoricData.Committees, "cached reference was not empty to begin with")

			committee := getCommitteeOrNil(cp, ctx, AHistoricRef)
			require.NotNil(t, cp.cachedHistoricData.Committees, "cached reference was not updated")
			require.EqualValues(t, testKeys.NodeAddressesForTests()[2:6], committee, "wrong committee values")

			// update provider under the hood
			p.historicRef = AHistoricRef - 1500
			p.historicCommittee = testKeys.NodeAddressesForTests()[3:8]

			committee = getCommitteeOrNil(cp, ctx, AHistoricRef - 1500)
			require.EqualValues(t, testKeys.NodeAddressesForTests()[3:8], committee, "wrong committee values")
		})
	})
}

// helpers
func (s*service) addCommittee(ref primitives.TimestampSeconds, committee []primitives.NodeAddress) {
	s.Lock()
	defer s.Unlock()
	weights := make([]primitives.Weight, len(committee))
	for i := range weights {
		weights[i] = 1
	}
	s.data.EndPageReference = ref + 1000
	s.data.Committees = append (s.data.Committees, CommitteeTerm{ref, committee, weights})
}

func getCommitteeOrNil(m *service, ctx context.Context, reference primitives.TimestampSeconds) []primitives.NodeAddress {
	committee, err := m.GetCommittee(ctx, &services.GetCommitteeInput{Reference:reference})
	if err != nil {
		return nil
	}
	return committee.Members
}

type staticProvider struct {
	sync.RWMutex
	ref primitives.TimestampSeconds
	committee []primitives.NodeAddress
	weights []primitives.Weight
	historicRef primitives.TimestampSeconds
	historicCommittee []primitives.NodeAddress
	historicWeights []primitives.Weight
}

func newStaticProvider() *staticProvider{
	return &staticProvider{
		ref: ACurrentRef,
		committee: testKeys.NodeAddressesForTests()[:4],
		weights: []primitives.Weight{1,2,3,4},
		historicRef: AHistoricRef,
		historicCommittee: testKeys.NodeAddressesForTests()[2:6],
		historicWeights: []primitives.Weight{1,2,3,4},
	}
}

func (sp *staticProvider) Get(ctx context.Context, ref primitives.TimestampSeconds) (*VirtualChainManagementData, error) {
	sp.RLock()
	defer sp.RUnlock()
	if ref == 0 {
		return &VirtualChainManagementData{
			CurrentReference:   sp.ref,
			StartPageReference: sp.ref - 1000,
			EndPageReference:   sp.ref + 1000, // simulate the fact that endpage needs to be more current
			CurrentTopology:    []*services.GossipPeer{},
			Committees:         []CommitteeTerm{{sp.ref, sp.committee, sp.weights}},
			Subscriptions:      nil,
			ProtocolVersions:   nil,
		}, nil
	} else {
		return &VirtualChainManagementData{
			CurrentReference:   sp.historicRef,
			StartPageReference: sp.historicRef - 1000,
			EndPageReference:   sp.historicRef + 1000, // simulate the fact that endpage needs to be more current
			CurrentTopology:    []*services.GossipPeer{},
			Committees:         []CommitteeTerm{{sp.historicRef, sp.historicCommittee, sp.historicWeights}},
			Subscriptions:      nil,
			ProtocolVersions:   nil,
		}, nil
	}
}

func (sp *staticProvider) UpdateTopology(ctx context.Context, input *services.UpdateTopologyInput) (*services.UpdateTopologyOutput, error) {
	// does nothing just here for test to run
	return &services.UpdateTopologyOutput{}, nil
}

type cfg struct {
}

func newConfig() *cfg {
	return &cfg{}
}

func (tc *cfg) ManagementPollingInterval() time.Duration { // no auto update
	return 0
}
