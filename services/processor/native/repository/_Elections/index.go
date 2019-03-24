package elections_systemcontract

import "math/big"

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Elections"
const METHOD_GET_ELECTED_VALIDATORS = "getElectedValidatorsOrbsAddress"

// parameters
var DELEGATION_NAME = "Delegate"
var DELEGATION_BY_TRANSFER_NAME = "Transfer"
var DELEGATION_BY_TRANSFER_VALUE = big.NewInt(7)
var ETHEREUM_STAKE_FACTOR = big.NewInt(1000000000000000000)
var VOTE_MIRROR_PERIOD_LENGTH_IN_BLOCKS = uint64(480)
var VOTE_VALID_PERIOD_LENGTH_IN_BLOCKS = uint64(40320)
var ELECTION_PERIOD_LENGTH_IN_BLOCKS = uint64(15000)
var TRANSITION_PERIOD_LENGTH_IN_BLOCKS = uint64(1)
var FIRST_ELECTION_BLOCK = uint64(7467969)
var MAX_ELECTED_VALIDATORS = 22
var MIN_ELECTED_VALIDATORS = 7
var VOTE_OUT_WEIGHT_PERCENT = uint64(70)

func _init() {
}
