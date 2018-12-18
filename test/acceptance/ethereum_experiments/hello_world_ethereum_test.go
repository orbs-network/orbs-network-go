package ethereum_experiments

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
)

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

const code1 = `0x608060405234801561001057600080fd5b506104a4806100206000396000f300608060405260043610610041576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680634c6d262714610046575b600080fd5b34801561005257600080fd5b5061006d6004803603610068919081019061027d565b610083565b60405161007a9190610375565b60405180910390f35b6060600061008f6101af565b8391508173ffffffffffffffffffffffffffffffffffffffff1663c22fd45e6040518163ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401600060405180830381600087803b1580156100f657600080fd5b505af115801561010a573d6000803e3d6000fd5b505050506040513d6000823e3d601f19601f8201168201806040525061013391908101906102a6565b90507fd86ca04bba5b0f8c205270709b02733edc5794651f5eb97ead36ea76dcf0e6e981600001516040516101689190610353565b60405180910390a16040805190810160405280600281526020017f686900000000000000000000000000000000000000000000000000000000000081525092505050919050565b602060405190810160405280606081525090565b60006101cf8235610406565b905092915050565b600082601f83011215156101ea57600080fd5b81516101fd6101f8826103c4565b610397565b9150808252602083016020830185838301111561021957600080fd5b610224838284610426565b50505092915050565b60006020828403121561023f57600080fd5b6102496020610397565b9050600082015167ffffffffffffffff81111561026557600080fd5b610271848285016101d7565b60008301525092915050565b60006020828403121561028f57600080fd5b600061029d848285016101c3565b91505092915050565b6000602082840312156102b857600080fd5b600082015167ffffffffffffffff8111156102d257600080fd5b6102de8482850161022d565b91505092915050565b60006102f2826103fb565b808452610306816020860160208601610426565b61030f81610459565b602085010191505092915050565b6000610328826103f0565b80845261033c816020860160208601610426565b61034581610459565b602085010191505092915050565b6000602082019050818103600083015261036d818461031d565b905092915050565b6000602082019050818103600083015261038f81846102e7565b905092915050565b6000604051905081810181811067ffffffffffffffff821117156103ba57600080fd5b8060405250919050565b600067ffffffffffffffff8211156103db57600080fd5b601f19601f8301169050602081019050919050565b600081519050919050565b600081519050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60005b83811015610444578082015181840152602081019050610429565b83811115610453576000848401525b50505050565b6000601f19601f83011690509190505600a265627a7a7230582046ce1be3e13d98cd3dc5609938a00000fddd7029db15427c00b37b34716e52f16c6578706572696d656e74616cf50037`
const abi1 = `[
    {
      "constant": false,
      "inputs": [
        {
          "name": "helloWorld2Address",
          "type": "address"
        }
      ],
      "name": "sayHi",
      "outputs": [
        {
          "name": "",
          "type": "string"
        }
      ],
      "payable": false,
      "stateMutability": "nonpayable",
      "type": "function"
    }
  ]`
const eventAbi1 = `[
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": false,
          "name": "name",
          "type": "string"
        }
      ],
      "name": "BabyBorn",
      "type": "event"
    }
	]`

func TestHelloWorldOnEthereum(t *testing.T) {
	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)

	// deploy2

	txHw2 := types.NewContractCreation(0, big.NewInt(0), 300000, big.NewInt(1), common.FromHex(code2))
	txHw2, err := types.SignTx(txHw2, types.HomesteadSigner{}, testKey)
	require.NoError(t, err)

	backend.SendTransaction(context.Background(), txHw2)
	backend.Commit()

	contractHw2Address, err := bind.WaitDeployed(context.Background(), backend, txHw2)
	require.NoError(t, err)

	t.Log("deployed2", contractHw2Address.Hex())

	// deploy1

	tx := types.NewContractCreation(1, big.NewInt(0), 3000000, big.NewInt(1), common.FromHex(code1))
	tx, err = types.SignTx(tx, types.HomesteadSigner{}, testKey)
	require.NoError(t, err)

	backend.SendTransaction(context.Background(), tx)
	backend.Commit()

	contractAddress, err := bind.WaitDeployed(context.Background(), backend, tx)
	require.NoError(t, err)

	t.Log("deployed1", contractAddress.Hex())

	// send tx

	parsedAbi, err := abi.JSON(strings.NewReader(abi1))
	require.NoError(t, err)

	input, err := parsedAbi.Pack("sayHi", contractHw2Address)
	require.NoError(t, err)

	tx2 := types.NewTransaction(2, contractAddress, big.NewInt(0), 300000, big.NewInt(1), input)
	tx2, err = types.SignTx(tx2, types.HomesteadSigner{}, testKey)
	require.NoError(t, err)

	backend.SendTransaction(context.Background(), tx2)
	backend.Commit()

	receipt, err := bind.WaitMined(context.Background(), backend, tx2)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	t.Log("txhash", receipt.TxHash.Hex())

	require.Equal(t, 1, len(receipt.Logs))
	parsedEventAbi, err := abi.JSON(strings.NewReader(eventAbi1))
	require.NoError(t, err)
	output, err := parsedEventAbi.Events["BabyBorn"].Inputs.UnpackValues(receipt.Logs[0].Data)
	require.NoError(t, err)

	t.Log("log", output)
}
