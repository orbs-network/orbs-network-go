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
			ref, topology, committees, err := fileProvider.Update(ctx)
			require.NoError(t, err)
			require.Equal(t, ref, 1582616070)
			require.Len(t, topology, 4)
			require.Len(t, committees, 2)
		})
	})
}

func TestFileTopology_ReadUrl(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			const url = "https://gist.githubusercontent.com/noambergIL/c4dd1472af977c7378459cf98f06e0fa/raw/8488ea8f6125bdeb49f786ebcf8c448af8f473ed/topology.json"
			cfg := newConfig(42, url)
			fileProvider := NewFileProvider(cfg, parent.Logger)
			ref, topology, committees, err := fileProvider.Update(ctx)
			require.NoError(t, err)
			require.Equal(t, ref, 1582616070)
			require.Len(t, topology, 4)
			require.Len(t, committees, 4)
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
