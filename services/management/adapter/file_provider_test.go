// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/management"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestFileTopology_ReadFile(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			topologyFilePath := filepath.Join(config.GetCurrentSourceFileDirPath(), "_data", "good.json")
			cfg := newConfig(42, topologyFilePath)
			fileProvider := NewFileProvider(cfg, parent.Logger)
			expectFileProviderToReadCorrectly(t, ctx, fileProvider)
		})
	})
}

func TestFileTopology_ReadUrl(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			const url = "https://gist.githubusercontent.com/noambergIL/8131667fda382905e1c3997c7522a9c3/raw/30eec201954808b070adf5dc1f1ea459846997b6/management.json"
			cfg := newConfig(42, url)
			fileProvider := NewFileProvider(cfg, parent.Logger)
			expectFileProviderToReadCorrectly(t, ctx, fileProvider)
		})
	})
}

func expectFileProviderToReadCorrectly(t *testing.T, ctx context.Context, fp management.Provider) {
	data, err := fp.Get(ctx)
	require.NoError(t, err)
	require.EqualValues(t, data.CurrentReference, 1582616070)
	require.EqualValues(t, data.GenesisReference, 1582615603)
	require.Len(t, data.CurrentTopology, 4)
	requireTopologyToBeSameAsStatic(t, data.CurrentTopology)
	require.Len(t, data.Committees, 3)
	requireCommitteeToBeSameAsStatic(t, data.Committees)
}

func requireTopologyToBeSameAsStatic(t *testing.T, peers adapter.GossipPeers) {
	staticTopology := make(adapter.GossipPeers)
	staticTopology[testKeys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress().KeyForMap()] = adapter.NewGossipPeer(4400, "192.168.199.2", "a328846cd5b4979d68a8c58a9bdfeee657b34de7")
	staticTopology[testKeys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress().KeyForMap()] = adapter.NewGossipPeer(4400, "192.168.199.3", "d27e2e7398e2582f63d0800330010b3e58952ff6")
	staticTopology[testKeys.EcdsaSecp256K1KeyPairForTests(2).NodeAddress().KeyForMap()] = adapter.NewGossipPeer(4400, "192.168.199.4", "6e2cb55e4cbe97bf5b1e731d51cc2c285d83cbf9")
	staticTopology[testKeys.EcdsaSecp256K1KeyPairForTests(3).NodeAddress().KeyForMap()] = adapter.NewGossipPeer(4400, "192.168.199.5", "c056dfc0d1fbc7479db11e61d1b0b57612bf7f17")

	require.EqualValues(t, staticTopology, peers)
}

func requireCommitteeToBeSameAsStatic(t *testing.T, c []management.CommitteeTerm) {
	committee := []primitives.NodeAddress{testKeys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress(), testKeys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()}
	// notice order of ref from small to big.
	require.EqualValues(t, 1582613000, c[0].AsOfReference)
	require.ElementsMatch(t, committee, c[0].Members)
	require.EqualValues(t, 1582614000, c[1].AsOfReference)
	require.ElementsMatch(t, committee, c[1].Members)
	require.EqualValues(t, 1582616000, c[2].AsOfReference)
	require.ElementsMatch(t, committee, c[2].Members)
}

type fconfig struct {
	vcId primitives.VirtualChainId
	path string
}

func newConfig(vcId primitives.VirtualChainId, path string) *fconfig {
	return &fconfig{
		vcId: vcId,
		path: path,
	}
}

func (tc *fconfig) VirtualChainId() primitives.VirtualChainId {
	return tc.vcId
}

func (tc *fconfig) ManagementFilePath() string {
	return tc.path
}

func (tc *fconfig) ManagementMaxFileSize() uint32 {
	return 1 << 20 * 50
}
