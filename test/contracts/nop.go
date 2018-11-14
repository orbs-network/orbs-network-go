package contracts

const NOP_SOURCE_CODE = `
package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
)

var EXPORTS = sdk.Export()

func _init() {
}
`

func SourceCodeForNop() []byte {
	return []byte(NOP_SOURCE_CODE)
}
