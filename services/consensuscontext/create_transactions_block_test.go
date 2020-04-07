// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/Triggers"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func newHarnessWithConfigOnly(enableTriggers bool) *service {
	return &service{
		config: config.ForConsensusContextTests(enableTriggers),
	}
}

func requireTransactionToBeATriggerTransaction(t *testing.T, tx *protocol.SignedTransaction, cfg config.ConsensusContextConfig) {
	require.Empty(t, tx.Signature())
	require.Equal(t, cfg.MaximalProtocolVersionSupported(), tx.Transaction().ProtocolVersion())
	require.Equal(t, cfg.VirtualChainId(), tx.Transaction().VirtualChainId())
	require.Equal(t, primitives.ContractName(triggers_systemcontract.CONTRACT_NAME), tx.Transaction().ContractName())
	require.Equal(t, primitives.MethodName(triggers_systemcontract.METHOD_TRIGGER), tx.Transaction().MethodName())
	require.Empty(t, tx.Transaction().InputArgumentArray())
	require.Empty(t, tx.Transaction().Signer().Raw())
}

func TestConsensusContextCreateBlock_UpdateDoesntAddTriggerWhenDisabled(t *testing.T) {
	s := newHarnessWithConfigOnly(false)
	txs := []*protocol.SignedTransaction{builders.Transaction().Build()}
	outputTxs := s.updateTransactions(txs,1,0)
	require.Equal(t, len(txs), len(outputTxs), "should not add txs")
	require.EqualValues(t, txs[0], outputTxs[0], "should be same tx")
}

func TestConsensusContextCreateBlock_UpdateAddTriggerWhenEnabled(t *testing.T) {
	s := newHarnessWithConfigOnly(true)
	txs := []*protocol.SignedTransaction{builders.Transaction().Build()}
	outputTxs := s.updateTransactions(txs,1,6)
	require.Equal(t, len(txs)+1, len(outputTxs), "should not add txs")
	require.EqualValues(t, txs[0], outputTxs[0], "should be same tx")
	requireTransactionToBeATriggerTransaction(t, outputTxs[1], s.config)
}
