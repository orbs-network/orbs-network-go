package elections_systemcontract

import "github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"

// returns the addresses (20 bytes) of all elected validators joined to a single byte array
// if we have 5 elected, the output is a byte array of 5*20 = 100 bytes
func getElectedValidators() []byte {
	return _readResults()
}

func _readResults() []byte {
	return state.ReadBytes([]byte("ElectedValidators"))
}

func _writeResults(joinedAddresses []byte) {
	state.WriteBytes([]byte("ElectedValidators"), joinedAddresses)
}
