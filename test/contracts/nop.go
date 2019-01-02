package contracts

const NOP_SOURCE_CODE = `
package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
)

var PUBLIC = sdk.Export()
`

func SourceCodeForNop() []byte {
	return []byte(NOP_SOURCE_CODE)
}
