// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

// TODO POSV2 REFTIME MGMT -> replace with mock from spec
type managementMock struct {
	ref primitives.TimestampSeconds
	gen primitives.TimestampSeconds
}

func (m *managementMock) GetCurrentReference(ctx context.Context) primitives.TimestampSeconds {
	return m.ref
}

func (m *managementMock) GetGenesisReference(ctx context.Context) primitives.TimestampSeconds {
	return m.gen
}

func (m *managementMock) GetProtocolVersion(ctx context.Context, reference primitives.TimestampSeconds) primitives.ProtocolVersion {
	return 0
}

func (m *managementMock) setGenesis(gen primitives.TimestampSeconds) {
	m.gen = gen
}

func newHarnessWithManagement(ref primitives.TimestampSeconds, gen primitives.TimestampSeconds) *service {
	return &service{
		management: &managementMock{ref: ref, gen: gen},
	}
}

func TestFixRefForGenesis_IsGenesisAndGoodValues(t *testing.T) {
	with.Context(func(ctx context.Context) {
		currRef := primitives.TimestampSeconds(5000)
		prevRef := currRef - 10
		genesis := currRef - 100
		s := newHarnessWithManagement(currRef, genesis)
		actualRef, err := s.fixPrevReferenceTimeIfGenesis(ctx, 1, prevRef)
		require.NoError(t, err, "should not error")
		require.Equal(t, genesis, actualRef, "should return the management genesis reference")
	})
}

func TestFixRefForGenesis_IsGenesisAndBadValues(t *testing.T) {
	with.Context(func(ctx context.Context) {
		currRef := primitives.TimestampSeconds(5000)
		prevRef := currRef - 10
		genesis := currRef + 1
		s := newHarnessWithManagement(currRef, genesis)
		actualRef, err := s.fixPrevReferenceTimeIfGenesis(ctx, 1, prevRef)
		require.Error(t, err, "should not error")
		require.Equal(t, primitives.TimestampSeconds(0), actualRef, "should return 0")
	})
}

func TestFixRefForGenesis_NotGenesisAndGoodValues(t *testing.T) {
	with.Context(func(ctx context.Context) {
		currRef := primitives.TimestampSeconds(5000)
		prevRef := currRef - 10
		genesis := currRef - 100
		s := newHarnessWithManagement(currRef, genesis)
		actualRef, err := s.fixPrevReferenceTimeIfGenesis(ctx, 2, prevRef)
		require.NoError(t, err, "should not error")
		require.Equal(t, prevRef, actualRef, "should return the management genesis reference")
	})
}

func TestFixRefForGenesis_NotGenesisAndBadValues_Ignore(t *testing.T) {
	with.Context(func(ctx context.Context) {
		currRef := primitives.TimestampSeconds(5000)
		prevRef := currRef - 10
		genesis := currRef + 1
		s := newHarnessWithManagement(currRef, genesis)
		actualRef, err := s.fixPrevReferenceTimeIfGenesis(ctx, 2, prevRef)
		require.NoError(t, err, "should not error")
		require.Equal(t, prevRef, actualRef, "should return the management genesis reference")
	})
}

