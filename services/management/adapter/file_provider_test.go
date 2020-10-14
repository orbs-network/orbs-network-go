// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/management"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestManagementFileProvider_GeneratePath(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		const url = "x1"
		cfg := newConfig(42, url)
		fileProvider := NewFileProvider(cfg, parent.Logger)
		path := fileProvider.generatePath(0)
		pathWithRef := fileProvider.generatePath(100)
		require.Equal(t, url, path)
		require.Equal(t, url+"/100", pathWithRef)
	})
}

func TestManagementFileProvider_NoMatchVc(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		with.Context(func(ctx context.Context) {
			cfg := newConfig(42, "")
			fileProvider := NewFileProvider(cfg, parent.Logger)
			_, err := fileProvider.parseData([]byte(`{
	"CurrentRefTime": 3, 
	"PageStartRefTime": 0, 
	"PageEndRefTime": 2, 
	"VirtualChains": { 
		"44": { 
		}
	}
}`), false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "could not find current vc in data")
		})
	})
}

func TestManagementFileProvider_BadCurrentInPage(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		with.Context(func(ctx context.Context) {
			cfg := newConfig(42, "")
			fileProvider := NewFileProvider(cfg, parent.Logger)
			_, err := fileProvider.parseData([]byte(`{
	"CurrentRefTime": 3, 
	"PageStartRefTime": 0, 
	"PageEndRefTime": 2, 
	"VirtualChains": { 
		"42": { 
		}
	}
}`), false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "data: CurrentRefTime (3) ")

			_, err = fileProvider.parseData([]byte(`{
	"CurrentRefTime": 2, 
	"PageStartRefTime": 3, 
	"PageEndRefTime": 2, 
	"VirtualChains": { 
		"42": { 
		}
	}
}`), false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "data: CurrentRefTime (2) ")

			_, err = fileProvider.parseData([]byte(`{
	"CurrentRefTime": 4, 
	"PageStartRefTime": 2, 
	"PageEndRefTime": 5, 
	"VirtualChains": { 
		"42": { 
		}
	}
}`), true)
			require.Error(t, err)
			require.Contains(t, err.Error(), "historic data : CurrentRefTime (4) ")

			_, err = fileProvider.parseData([]byte(`{
	"CurrentRefTime": 4, 
	"PageStartRefTime": 2, 
	"PageEndRefTime": 1, 
	"VirtualChains": { 
		"42": { 
		}
	}
}`), true)
			require.Error(t, err)
			require.Contains(t, err.Error(), "historic data : CurrentRefTime (4) ")
		})
	})
}

func TestManagementFileProvider_ReadFile(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			topologyFilePath := filepath.Join(config.GetCurrentSourceFileDirPath(), "_data", "good.json")
			cfg := newConfig(42, topologyFilePath)
			fileProvider := NewFileProvider(cfg, parent.Logger)
			expectFileProviderToReadCorrectly(t, ctx, fileProvider)
		})
	})
}

func TestManagementFileProvider_ReadUrl(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			const url = "https://raw.githubusercontent.com/orbs-network/management-service/master/static-tests/management-vc-file.json"
			cfg := newConfig(42, url)
			fileProvider := NewFileProvider(cfg, parent.Logger)
			expectFileProviderToReadCorrectly(t, ctx, fileProvider)
		})
	})
}

func TestManagementFileProvider_parseTopology(t *testing.T) {
	_, encodingErr := parseTopology([]topologyNode{{OrbsAddress: "ZZZZZ"}})
	require.EqualError(t, encodingErr, "cannot translate topology node address from hex ZZZZZ: encoding/hex: invalid byte: U+005A 'Z'")

	_, portErr := parseTopology([]topologyNode{{OrbsAddress: "ffff", Ip: "1.2.3.4", Port: 10000000}})
	require.EqualError(t, portErr, "topology node port 10000000 needs to be 1024-65535 range")

	_, emptyErr := parseTopology([]topologyNode{{OrbsAddress: "ffff", Ip: "", Port: 2048}})
	require.EqualError(t, emptyErr, "empty ip address for node ffff")
}

func TestManagementFileProvider_parseCommittee(t *testing.T) {
	_, encodingErr := parseCommittees([]committeeEvent{ {RefTime: 4, Committee: []committee{{OrbsAddress: "ZZZZZ"}} }} )
	require.Error(t, encodingErr)
	require.Contains(t, encodingErr.Error(), "cannot decode committee node address hex ")

	_, weightErr := parseCommittees([]committeeEvent{ {RefTime: 4, Committee: []committee{{OrbsAddress: "ffff"}} }} )
	require.Error(t, weightErr)
	require.Contains(t, weightErr.Error(), "Weight of node")
}

func expectFileProviderToReadCorrectly(t *testing.T, ctx context.Context, fp management.Provider) {
	data, err := fp.Get(ctx, 0)
	require.NoError(t, err)
	require.EqualValues(t, data.CurrentReference, 1582616070)
	require.EqualValues(t, data.GenesisReference, 1582615603)
	require.Len(t, data.CurrentTopology, 4)
	requireTopologyToBeSameAsStatic(t, data.CurrentTopology)
	require.Len(t, data.Committees, 3)
	requireCommitteeToBeSameAsStatic(t, data.Committees)
}

func requireTopologyToBeSameAsStatic(t *testing.T, peers []*services.GossipPeer) {
	var staticTopology []*services.GossipPeer
	staticTopology = append(staticTopology, &services.GossipPeer{Address: testKeys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress(), Endpoint: "192.168.199.2", Port: 4400})
	staticTopology = append(staticTopology, &services.GossipPeer{Address: testKeys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress(), Endpoint: "192.168.199.3", Port: 4400})
	staticTopology = append(staticTopology, &services.GossipPeer{Address: testKeys.EcdsaSecp256K1KeyPairForTests(2).NodeAddress(), Endpoint: "192.168.199.4", Port: 4400})
	staticTopology = append(staticTopology, &services.GossipPeer{Address: testKeys.EcdsaSecp256K1KeyPairForTests(3).NodeAddress(), Endpoint: "192.168.199.5", Port: 4400})
	require.ElementsMatch(t, staticTopology, peers)
}

func requireCommitteeToBeSameAsStatic(t *testing.T, c []management.CommitteeTerm) {
	committee := []primitives.NodeAddress{testKeys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress(), testKeys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()}
	weights := []primitives.Weight{16578000, 828363}
	weights2 := []primitives.Weight{16578435, 828363}
	// notice order of ref from small to big.
	require.EqualValues(t, 1582613000, c[0].AsOfReference)
	require.ElementsMatch(t, committee, c[0].Members)
	require.ElementsMatch(t, weights, c[0].Weights)
	require.EqualValues(t, 1582614000, c[1].AsOfReference)
	require.ElementsMatch(t, committee, c[1].Members)
	require.ElementsMatch(t, weights2, c[1].Weights)
	require.EqualValues(t, 1582616000, c[2].AsOfReference)
	require.ElementsMatch(t, committee, c[2].Members)
	require.ElementsMatch(t, weights, c[2].Weights)
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
