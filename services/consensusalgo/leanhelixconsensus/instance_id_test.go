// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"encoding/binary"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCalcInstanceIdWithValidValues(t *testing.T) {
	networkType := protocol.SignerNetworkType(binary.LittleEndian.Uint16([]byte{1, 2}))
	virtualChainId := primitives.VirtualChainId(binary.LittleEndian.Uint32([]byte{3, 4, 5, 6}))
	expected := lhprimitives.InstanceId(binary.LittleEndian.Uint64([]byte{0, 0, 1, 2, 3, 4, 5, 6}))
	actual := CalcInstanceId(networkType, virtualChainId)
	require.Equal(t, expected, actual)
}

func TestCalcInstanceIdWithEmptyNetworkType(t *testing.T) {
	networkType := protocol.SignerNetworkType(0)
	virtualChainId := primitives.VirtualChainId(binary.LittleEndian.Uint32([]byte{2, 3, 4, 5}))
	expected := lhprimitives.InstanceId(binary.LittleEndian.Uint64([]byte{0, 0, 0, 0, 2, 3, 4, 5}))
	actual := CalcInstanceId(networkType, virtualChainId)
	require.Equal(t, expected, actual)

}
