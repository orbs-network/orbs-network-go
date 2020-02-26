// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package file

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestFileTopology_ReadFile(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			topologyFilePath := filepath.Join(config.GetCurrentSourceFileDirPath(), "_data", "good-topology.json")
			cfg := newTopologyConfig(42, topologyFilePath)
			topologyProvider := NewTopologyProvider(cfg, parent.Logger)
			err := topologyProvider.UpdateTopology(ctx)
			require.NoError(t, err)
			require.Len(t, topologyProvider.GetTopology(ctx), 4)
		})
	})
}

func TestFileTopology_ReadUrl(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			const url = "https://gist.githubusercontent.com/noambergIL/c4dd1472af977c7378459cf98f06e0fa/raw/6fd5ab45c319951c03c4524e69ea292f774b783c/topology.json"
			cfg := newTopologyConfig(42, url)
			topologyProvider := NewTopologyProvider(cfg, parent.Logger)
			err := topologyProvider.UpdateTopology(ctx)
			require.NoError(t, err)
			require.Len(t, topologyProvider.GetTopology(ctx), 3)
		})
	})
}

type topologyConfig struct {
	vcId primitives.VirtualChainId
	path string
}

func newTopologyConfig(vcId primitives.VirtualChainId, path string) *topologyConfig {
	return &topologyConfig{
		vcId: vcId,
		path: path,
	}
}

func (tc *topologyConfig) VirtualChainId() primitives.VirtualChainId {
	return tc.vcId
}

func (tc *topologyConfig) GossipTopologyFilePath() string {
	return tc.path
}
