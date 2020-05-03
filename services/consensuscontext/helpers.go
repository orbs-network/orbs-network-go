// Copyright 2020 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
)

func (s *service) prevReferenceOrGenesis(ctx context.Context, blockHeight primitives.BlockHeight, prevBlockReferenceTime primitives.TimestampSeconds) (primitives.TimestampSeconds, error) {
	if blockHeight == 1 { // genesis block
		reference, err := s.management.GetGenesisReference(ctx, &services.GetGenesisReferenceInput{})
		if err != nil {
			s.logger.Error("management.GetGenesisReference should not return error", log.Error(err))
			return 0, err
		}
		if reference.GenesisReference > reference.CurrentReference {
			return 0, errors.Errorf("failed genesis time reference (%d) cannot be after current time reference (%d)", reference.GenesisReference, reference.CurrentReference)
		}
		prevBlockReferenceTime = reference.GenesisReference
	}
	return prevBlockReferenceTime, nil
}

func toAddresses(input *protocol.Argument) (addresses []primitives.NodeAddress) {
	itr := input.BytesArrayValueIterator()
	for itr.HasNext() {
		addresses = append(addresses, itr.NextBytes())
	}
	return
}

func (s *service) printTxHash(logger log.Logger, txBlock *protocol.TransactionsBlockContainer) {
	for _, tx := range txBlock.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		logger.Info("transaction entered transactions block", log.String("flow", "checkpoint"), logfields.Transaction(txHash), logfields.BlockHeight(txBlock.Header.BlockHeight()))
	}
}
