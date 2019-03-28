// Copyright 2019 the orbs-ethereum-contracts authors
// This file is part of the orbs-ethereum-contracts library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package elections_systemcontract

import "math/big"

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Elections"
const METHOD_GET_ELECTED_VALIDATORS = "getElectedValidatorsOrbsAddress"

// parameters
var DELEGATION_NAME = "Delegate"
var DELEGATION_BY_TRANSFER_NAME = "Transfer"
var DELEGATION_BY_TRANSFER_VALUE = big.NewInt(70000000000000000)
var ETHEREUM_STAKE_FACTOR = big.NewInt(1000000000000000000)
var VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS = uint64(545)
var VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS = uint64(45500)
var ELECTION_PERIOD_LENGTH_IN_BLOCKS = uint64(20000)
var TRANSITION_PERIOD_LENGTH_IN_BLOCKS = uint64(1)
var FIRST_ELECTION_BLOCK = uint64(7528900)
var MAX_ELECTED_VALIDATORS = 22
var MIN_ELECTED_VALIDATORS = 7
var VOTE_OUT_WEIGHT_PERCENT = uint64(70)

func _init() {
}
