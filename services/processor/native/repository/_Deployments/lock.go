package deployments_systemcontract

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"github.com/orbs-network/orbs-network-go/crypto/encoding"
)

func lockNativeDeployment() {
	currentOwner := _readNativeDeploymentOwner()
	if len(currentOwner) == 0 {
		_writeNativeDeploymentOwner(address.GetSignerAddress())
	} else {
		panic(fmt.Sprintf("current owner %s must unlockNativeDeployment first", encoding.EncodeHex(currentOwner)))
	}
}

func unlockNativeDeployment() {
	_validateNativeDeploymentLock()
	_writeNativeDeploymentOwner([]byte{})
}

func _validateNativeDeploymentLock() {
	currentOwner := _readNativeDeploymentOwner()
	if len(currentOwner) != 0 && !bytes.Equal(currentOwner, address.GetSignerAddress()) {
		panic(fmt.Sprintf("native deployment is locked to owner %s", encoding.EncodeHex(currentOwner)))
	}
}

func _readNativeDeploymentOwner() []byte {
	return state.ReadBytes([]byte("NativeDeploymentOwner"))
}

func _writeNativeDeploymentOwner(newOwnerAddress []byte) {
	state.WriteBytes([]byte("NativeDeploymentOwner"), newOwnerAddress)
}
