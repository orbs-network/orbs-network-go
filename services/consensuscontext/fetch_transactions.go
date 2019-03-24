// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) fetchTransactions(ctx context.Context, currentBlockHeight primitives.BlockHeight, prevBlockTimestamp primitives.TimestampNano, maxNumberOfTransactions uint32) (*services.GetTransactionsForOrderingOutput, error) {

	input := &services.GetTransactionsForOrderingInput{
		MaxTransactionsSetSizeKb: 0, // TODO(v1): either fill in or delete from spec
		MaxNumberOfTransactions:  maxNumberOfTransactions,
		CurrentBlockHeight:       currentBlockHeight,
		PrevBlockTimestamp:       prevBlockTimestamp,
	}

	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(ctx, input)
	if err != nil {
		return nil, err
	}

	return proposedTransactions, nil
}
