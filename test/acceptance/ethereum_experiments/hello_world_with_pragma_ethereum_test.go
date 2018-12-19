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

const code2 = `0x608060405234801561001057600080fd5b5061033b806100206000396000f30060806040526004361061004c576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680630c49c36c14610051578063c22fd45e1461007c575b600080fd5b34801561005d57600080fd5b506100666100a7565b6040516100739190610243565b60405180910390f35b34801561008857600080fd5b50610091610119565b60405161009e9190610285565b60405180910390f35b60607fd86ca04bba5b0f8c205270709b02733edc5794651f5eb97ead36ea76dcf0e6e96040516100d690610265565b60405180910390a16040805190810160405280600281526020017f6869000000000000000000000000000000000000000000000000000000000000815250905090565b610121610162565b6040805190810160405280600981526020017f73756e666c6f7765720000000000000000000000000000000000000000000000815250816000018190525090565b602060405190810160405280606081525090565b6000610181826102b2565b8084526101958160208601602086016102bd565b61019e816102f0565b602085010191505092915050565b60006101b7826102a7565b8084526101cb8160208601602086016102bd565b6101d4816102f0565b602085010191505092915050565b6000600582527f6d69747a690000000000000000000000000000000000000000000000000000006020830152604082019050919050565b6000602083016000830151848203600086015261023682826101ac565b9150508091505092915050565b6000602082019050818103600083015261025d8184610176565b905092915050565b6000602082019050818103600083015261027e816101e2565b9050919050565b6000602082019050818103600083015261029f8184610219565b905092915050565b600081519050919050565b600081519050919050565b60005b838110156102db5780820151818401526020810190506102c0565b838111156102ea576000848401525b50505050565b6000601f19601f83011690509190505600a265627a7a723058209e6b4308ecf716d734759ae6a344fc9858a32fb3e5e05a330256b96ba0070c8d6c6578706572696d656e74616cf50037`
const abi2 = `[
    {
      "constant": false,
      "inputs": [],
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
const eventAbi2 = `[
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

func TestHelloWorldWithPragmaOnEthereum(t *testing.T) {
	backend := backends.NewSimulatedBackend(
		core.GenesisAlloc{
			crypto.PubkeyToAddress(testKey.PublicKey): {Balance: big.NewInt(10000000000)},
		}, 10000000,
	)

	// deploy

	tx := types.NewContractCreation(0, big.NewInt(0), 300000, big.NewInt(1), common.FromHex(code2))
	tx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	require.NoError(t, err)

	backend.SendTransaction(context.Background(), tx)
	backend.Commit()

	contractAddress, err := bind.WaitDeployed(context.Background(), backend, tx)
	require.NoError(t, err)

	t.Log("deployed", contractAddress.Hex())

	// send tx

	parsedAbi, err := abi.JSON(strings.NewReader(abi2))
	require.NoError(t, err)

	input, err := parsedAbi.Pack("sayHi")
	require.NoError(t, err)

	tx2 := types.NewTransaction(1, contractAddress, big.NewInt(0), 300000, big.NewInt(1), input)
	tx2, err = types.SignTx(tx2, types.HomesteadSigner{}, testKey)
	require.NoError(t, err)

	backend.SendTransaction(context.Background(), tx2)
	backend.Commit()

	receipt, err := bind.WaitMined(context.Background(), backend, tx2)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	t.Log("txhash", receipt.TxHash.Hex())

	require.Equal(t, 1, len(receipt.Logs))
	parsedEventAbi, err := abi.JSON(strings.NewReader(eventAbi2))
	require.NoError(t, err)
	output, err := parsedEventAbi.Events["BabyBorn"].Inputs.UnpackValues(receipt.Logs[0].Data)
	require.NoError(t, err)

	t.Log("log", output)
}
