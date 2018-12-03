package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"math/big"
)

type DeployingEthereumConnection interface {
	EthereumConnection
	DeploySimpleStorageContract(auth *bind.TransactOpts, stringValue string) ([]byte, error)
	DeployEmitEvent(auth *bind.TransactOpts) ([]byte, error)
	SendTransaction(ctx context.Context, auth *bind.TransactOpts, address []byte, packedInput []byte) (txHash primitives.Uint256, err error)
}


// this is a helper for integration test, not used in production code
func (c *connectorCommon) DeploySimpleStorageContract(auth *bind.TransactOpts, stringValue string) ([]byte, error){
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	address, _, _, err := contract.DeploySimpleStorage(auth, client, big.NewInt(42), stringValue)
	return address.Bytes(), err
}

// this is a helper for integration test, not used in production code
func (c *connectorCommon) DeployEmitEvent(auth *bind.TransactOpts) ([]byte, error){
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	address, _, err := contract.DeployEmitEvent(auth, client)
	return address.Bytes(), err
}

// used only for tests
func (c *connectorCommon) SendTransaction(ctx context.Context, auth *bind.TransactOpts, address []byte, packedInput []byte) (txHash primitives.Uint256, err error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	contractAddress := common.BytesToAddress(address)

	nonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		return nil, err
	}

	unsignedTx := types.NewTransaction(nonce, contractAddress, common.Big0, 90000000000, common.Big0, packedInput)

	signer := types.HomesteadSigner{}
	signedTx, err := auth.Signer(signer, auth.From, unsignedTx)
	if err != nil {
		return nil, err
	}

	txHash = signedTx.Hash().Bytes()
	err = client.SendTransaction(ctx, signedTx)

	c.logger.Info("sent transaction", log.Stringable("txHash", txHash))

	return
}