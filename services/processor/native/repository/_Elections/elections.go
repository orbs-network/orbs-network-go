package elections_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Elections"
const METHOD_GET_ELECTED_VALIDATORS = "getElectedValidators"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(getElectedValidators)
