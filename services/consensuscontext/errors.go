// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import "errors"

var ErrMismatchedProtocolVersion = errors.New("ErrMismatchedProtocolVersion")
var ErrMismatchedVirtualChainID = errors.New("ErrMismatchedVirtualChainID")
var ErrMismatchedBlockHeight = errors.New("ErrMismatchedBlockHeight")
var ErrMismatchedPrevBlockHash = errors.New("ErrMismatchedPrevBlockHash")
var ErrInvalidBlockTimestamp = errors.New("ErrInvalidBlockTimestamp")

var ErrFailedTransactionOrdering = errors.New("ErrFailedTransactionOrdering")

var ErrMismatchedTxRxBlockHeight = errors.New("ErrMismatchedTxRxBlockHeight mismatched block height between transactions and results")
var ErrMismatchedTxRxTimestamps = errors.New("ErrMismatchedTxRxTimestamps mismatched timestamp between transactions and results")
var ErrMismatchedTxHashPtrToActualTxBlock = errors.New("ErrMismatchedTxHashPtrToActualTxBlock mismatched tx block hash ptr to actual tx block hash")

var ErrGetStateHash = errors.New("ErrGetStateHash failed in GetStateHash() so cannot retrieve pre-execution state diff merkleRoot from previous block")
var ErrMismatchedPreExecutionStateMerkleRoot = errors.New("ErrMismatchedPreExecutionStateMerkleRoot pre-execution state diff merkleRoot is different between results block header and extracted from state storage for previous block")
var ErrProcessTransactionSet = errors.New("ErrProcessTransactionSet failed in ProcessTransactionSet()")
