package contract

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"strings"
)

const EmitEventAbi = `
[
    {
      "inputs": [],
      "payable": false,
      "stateMutability": "nonpayable",
      "type": "constructor"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "name": "tuid",
          "type": "uint256"
        },
        {
          "indexed": true,
          "name": "ethAddress",
          "type": "address"
        },
        {
          "indexed": true,
          "name": "orbsAddress",
          "type": "bytes20"
        },
        {
          "indexed": false,
          "name": "value",
          "type": "uint256"
        }
      ],
      "name": "TransferredOut",
      "type": "event"
    },
    {
      "constant": false,
      "inputs": [
        {
          "name": "tuid",
          "type": "uint256"
        },
        {
          "name": "ethAddress",
          "type": "address"
        },
        {
          "name": "orbsAddress",
          "type": "bytes20"
        },
        {
          "name": "value",
          "type": "uint256"
        }
      ],
      "name": "transferOut",
      "outputs": [],
      "payable": false,
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ]`

const EmitEventBin = "0x608060405234801561001057600080fd5b50610147806100206000396000f300608060405260043610610041576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680632e45ac0614610046575b600080fd5b34801561005257600080fd5b506100b460048036038101908080359060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080356bffffffffffffffffffffffff19169060200190929190803590602001909291905050506100b6565b005b816bffffffffffffffffffffffff19168373ffffffffffffffffffffffffffffffffffffffff16857fc7d2da8a0df0279cb4e0a81f2975445675cc6527c94016791d29977a1fa0f251846040518082815260200191505060405180910390a4505050505600a165627a7a723058209b9bcb73251f6dbe1f0b171c71ffb48e65bd3cfa242833261e3f29248caf97a30029"

func DeployEmitEvent(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, error) {
	parsed, err := abi.JSON(strings.NewReader(EmitEventAbi))
	if err != nil {
		return common.Address{}, nil, err
	}
	address, tx, _, err := bind.DeployContract(auth, parsed, common.FromHex(EmitEventBin), backend)
	if err != nil {
		return common.Address{}, nil, err
	}
	return address, tx, nil
}
