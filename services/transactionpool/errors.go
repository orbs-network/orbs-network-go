// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type ErrTransactionRejected struct {
	TransactionStatus protocol.TransactionStatus
	Expected          *log.Field
	Actual            *log.Field
}

func (e *ErrTransactionRejected) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Expected != nil && e.Actual != nil {
		return fmt.Sprintf("transaction rejected: %s (expected %s but got %s)", e.TransactionStatus, e.Expected.Value(), e.Actual.Value())
	} else {
		return fmt.Sprintf("transaction rejected: %s", e.TransactionStatus)
	}
}
