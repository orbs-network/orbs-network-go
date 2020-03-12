// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

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
			topologyFilePath := filepath.Join(config.GetCurrentSourceFileDirPath(), "_data", "good.json")
			cfg := newConfig(42, topologyFilePath)
			fileProvider := NewFileProvider(cfg, parent.Logger)
			ref, topology, committees, err := fileProvider.Get(ctx)
			require.NoError(t, err)
			require.EqualValues(t, ref, 1582616070)
			require.Len(t, topology, 4)
			require.Len(t, committees, 2)
		})
	})
}

func TestFileTopology_ReadUrl(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			const url = "https://gist.githubusercontent.com/noambergIL/8131667fda382905e1c3997c7522a9c3/raw/edb958635c0ff2783c0447cb0322988ba71b0214/management.json"
			cfg := newConfig(42, url)
			fileProvider := NewFileProvider(cfg, parent.Logger)
			ref, topology, committees, err := fileProvider.Get(ctx)
			require.NoError(t, err)
			require.EqualValues(t, ref, 1582616070)
			require.Len(t, topology, 4)
			require.Len(t, committees, 3)
		})
	})
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
