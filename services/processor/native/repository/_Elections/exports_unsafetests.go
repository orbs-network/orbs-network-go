// +build unsafetests

package elections_systemcontract

import "github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"

var PUBLIC = sdk.Export(getElectedValidators, unsafetests_setElectedValidators)

func unsafetests_setElectedValidators(joinedAddresses []byte) {
	_writeResults(joinedAddresses)
}
