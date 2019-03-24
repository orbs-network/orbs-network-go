// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"math/big"
	"strings"
)

type DeployingEthereumConnection interface {
	EthereumConnection
	DeploySimpleStorageContract(auth *bind.TransactOpts, stringValue string) ([]byte, error)
	DeployEthereumContract(auth *bind.TransactOpts, abijson string, bytecode string, params ...interface{}) (*common.Address, *bind.BoundContract, error)
}

// this is a helper for integration test, not used in production code
func (c *connectorCommon) DeploySimpleStorageContract(auth *bind.TransactOpts, stringValue string) ([]byte, error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	address, _, _, err := contract.DeploySimpleStorage(auth, client, big.NewInt(42), stringValue)
	return address.Bytes(), err
}

// this is a helper for integration test, not used in production code
func (c *connectorCommon) DeployEthereumContract(auth *bind.TransactOpts, abijson string, bytecode string, params ...interface{}) (*common.Address, *bind.BoundContract, error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, nil, err
	}

	// deploy
	parsedAbi, err := abi.JSON(strings.NewReader(abijson))
	if err != nil {
		return nil, nil, err
	}
	address, _, contract, err := bind.DeployContract(auth, parsedAbi, common.FromHex(bytecode), client, params...)
	if err != nil {
		return nil, nil, err
	}

	return &address, contract, nil
}

func (c *connectorCommon) DeployEthereumContractManually(ctx context.Context, auth *bind.TransactOpts, abijson string, bytecode string, params ...interface{}) (*common.Address, error) {

	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	parsedAbi, err := abi.JSON(strings.NewReader(abijson))
	if err != nil {
		return nil, err
	}

	input, err := parsedAbi.Pack("", params...)
	if err != nil {
		return nil, err
	}

	data := append(common.FromHex(bytecode), input...)

	nonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account nonce: %s", err)
	}

	rawTx := types.NewContractCreation(nonce, big.NewInt(0), 300000000, big.NewInt(1), data)
	signedTx, err := auth.Signer(types.HomesteadSigner{}, auth.From, rawTx)
	if err != nil {
		return nil, err
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, err
	}

	contractAddress, err := bind.WaitDeployed(ctx, client, signedTx)
	if err != nil {
		return nil, err
	}

	return &contractAddress, nil
}
