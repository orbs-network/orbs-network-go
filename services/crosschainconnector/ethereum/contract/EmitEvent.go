// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package contract

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
)

type EmitEvent struct {
	Tuid        *big.Int
	EthAddress  [20]byte
	OrbsAddress [20]byte
	Value       *big.Int
}

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
      "anonymous": false,
      "inputs": [
        {
          "indexed": false,
          "name": "foo",
          "type": "string"
        }
      ],
      "name": "AnotherEvent",
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

const EmitEventBin = "0x608060405234801561001057600080fd5b506101af806100206000396000f300608060405260043610610041576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680632e45ac0614610046575b600080fd5b34801561005257600080fd5b506100b460048036038101908080359060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080356bffffffffffffffffffffffff19169060200190929190803590602001909291905050506100b6565b005b816bffffffffffffffffffffffff19168373ffffffffffffffffffffffffffffffffffffffff16857fc7d2da8a0df0279cb4e0a81f2975445675cc6527c94016791d29977a1fa0f251846040518082815260200191505060405180910390a47f8713b19d1e5f3f108242b79454ec6f7abe65027d4752405572f3ff16dce7d22e6040518080602001828103825260038152602001807f626172000000000000000000000000000000000000000000000000000000000081525060200191505060405180910390a1505050505600a165627a7a72305820ee12b1f7e53db7a46d144829f66ab6cc1c9baf0f626e09050836df6e7a960f0b0029"

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
