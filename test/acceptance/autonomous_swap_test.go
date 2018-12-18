package acceptance

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-client-sdk-go/orbsclient"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/ASBEthereum"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/ERC20Proxy"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestTransferFromEthereumToOrbs(t *testing.T) {
	harness.Network(t).
		Start(func(ctx context.Context, network harness.TestNetworkDriver) {
			d := newAutonomousSwapDriver(network)

			etherAmountBefore := big.NewInt(20)
			amountToTransfer := big.NewInt(17)

			d.generateOrbsAccount(t)

			d.deployTokenContractToEthereum(t)
			d.generateEthereumAccountAndAssignFunds(t, etherAmountBefore)

			d.deployAutonomousSwapBridgeToEthereum(t)
			d.bindOrbsAutonomousSwapBridgeToEthereum(ctx, t)

			d.approveTransferInEthereumTokenContract(t, amountToTransfer)
			transferOutTxHash := d.transferOutFromEthereum(t, amountToTransfer)
			t.Log("Eth tx hash", transferOutTxHash)

			// TODO v1 deploy causes who is owner - important for both.
			d.transferInToOrbs(ctx, t, transferOutTxHash)

			balanceAfterTransfer := d.getBalanceInOrbs(ctx, t)
			require.EqualValues(t, amountToTransfer.Uint64(), balanceAfterTransfer, "wrong amount of tokens in orbs")

			etherBalanceAfterTransfer := d.getBalanceInEthereum(t)
			require.EqualValues(t, etherAmountBefore.Sub(etherAmountBefore, amountToTransfer).Uint64(), etherBalanceAfterTransfer, "wrong amount of tokens left in ethereum")

		})
}

func TestTransferFromOrbsToEthereum(t *testing.T) {
	harness.Network(t).
		Start(func(ctx context.Context, network harness.TestNetworkDriver) {
			d := newAutonomousSwapDriver(network)

			etherAmount := big.NewInt(3)
			amount := big.NewInt(17)

			d.generateOrbsAccount(t)
			d.generateOrbsFunds(ctx, t, amount)

			d.deployTokenContractToEthereum(t)
			d.generateEthereumAccountAndAssignFunds(t, etherAmount)

			d.deployAutonomousSwapBridgeToEthereum(t)
			d.bindOrbsAutonomousSwapBridgeToEthereum(ctx, t)

			// TODO v1 deploy causes who is owner - important for both.
			d.transferOutFromOrbs(ctx, t, amount)

			//d.transferInToEthereum(ctx, t)

			//balanceAfterTransfer := d.getBalanceInEthereum(t)
			//require.EqualValues(t, etherAmount.Add(etherAmount, amount).Uint64(), balanceAfterTransfer, "wrong amount")

			orbsBalanceAfterTransfer := d.getBalanceInOrbs(ctx, t)
			require.EqualValues(t, 0, orbsBalanceAfterTransfer, "wrong amount")
		})
}

func newAutonomousSwapDriver(networkDriver harness.TestNetworkDriver) *driver {
	simulator := networkDriver.EthereumSimulator()
	return &driver{
		network:                  networkDriver,
		simulator:                simulator,
		addressInEthereum:        simulator.GetAuth(),
		orbsASBContractName:      asb_ether.CONTRACT_NAME,
		orbsContractOwnerAddress: testKeys.Ed25519KeyPairForTests(5),
	}
}

type driver struct {
	network   harness.TestNetworkDriver
	simulator *adapter.EthereumSimulator

	orbsContractOwnerAddress *keys.Ed25519KeyPair
	orbsASBContractName      string
	orbsUser                 *orbsclient.OrbsAccount
	orbsUserAddress          [20]byte
	orbsUserKeyPair          *keys.Ed25519KeyPair

	addressInEthereum *bind.TransactOpts // we use a single address for both the "admin" stuff like deploying the contracts and as our swapping user, so as to simplify setup - otherwise we'll need to create two PKs in the simulator

	erc20contract  *bind.BoundContract
	erc20address   *common.Address
	ethASBAddress  *common.Address
	ethASBContract *bind.BoundContract
}

// orbs side funcs
func (d *driver) generateOrbsAccount(t *testing.T) {
	orbsUser, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")

	copy(d.orbsUserAddress[:], orbsUser.RawAddress)
	d.orbsUser = orbsUser
	d.orbsUserKeyPair = keys.NewEd25519KeyPair(orbsUser.PublicKey, orbsUser.PrivateKey)
}

func (d *driver) generateOrbsFunds(ctx context.Context, t *testing.T, amount *big.Int) {
	response, txHash := d.network.SendTransaction(ctx, builders.Transaction().
		WithMethod(primitives.ContractName(erc20proxy.CONTRACT_NAME), "mint").
		WithEd25519Signer(d.orbsContractOwnerAddress).
		WithArgs(d.orbsUser.RawAddress, amount.Uint64()).
		Builder(), 0)
	d.network.WaitForTransactionInState(ctx, txHash)
	test.RequireSuccess(t, response, "failed setting minting tokens at orbs")
}

func (d *driver) getBalanceInOrbs(ctx context.Context, t *testing.T) uint64 {
	balanceResponse := d.network.CallMethod(ctx, builders.Transaction().
		WithEd25519Signer(d.orbsContractOwnerAddress).
		WithMethod(primitives.ContractName(erc20proxy.CONTRACT_NAME), "balanceOf").
		WithArgs(d.orbsUser.RawAddress).
		Builder().Transaction, 0)
	require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS, balanceResponse.CallMethodResult())
	// check that the tokens got there.
	outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(balanceResponse)
	value := outputArgsIterator.NextArguments().Uint64Value()
	return value
}

func (d *driver) approveTransferInOrbsTokenContract(ctx context.Context, t *testing.T, amount *big.Int) {
	response, txHash := d.network.SendTransaction(ctx, builders.Transaction().
		WithEd25519Signer(d.orbsUserKeyPair).
		WithMethod(primitives.ContractName(erc20proxy.CONTRACT_NAME), "approve").
		WithArgs(d.addressInEthereum.From, amount.Uint64()).
		Builder(), 0)
	d.network.WaitForTransactionInState(ctx, txHash)
	test.RequireSuccess(t, response, "failed approve transfer in orbs")
}

func (d *driver) bindOrbsAutonomousSwapBridgeToEthereum(ctx context.Context, t *testing.T) {
	response, txHash := d.network.SendTransaction(ctx, builders.Transaction().
		WithMethod(primitives.ContractName(d.orbsASBContractName), "setAsbAddr").
		WithEd25519Signer(d.orbsContractOwnerAddress).
		WithArgs(d.ethASBAddress.Hex()).
		Builder(), 0)
	d.network.WaitForTransactionInState(ctx, txHash)
	test.RequireSuccess(t, response, "failed setting asb address")
}

func (d *driver) transferInToOrbs(ctx context.Context, t *testing.T, transferOutTxHash string) {
	response, txHash := d.network.SendTransaction(ctx, builders.Transaction().
		WithMethod(primitives.ContractName(d.orbsASBContractName), "transferIn").
		WithEd25519Signer(d.orbsContractOwnerAddress).
		WithArgs(transferOutTxHash).
		Builder(), 0)
	d.network.WaitForTransactionInState(ctx, txHash)
	test.RequireSuccess(t, response, "failed transferIn in orbs")
}

func (d *driver) transferOutFromOrbs(ctx context.Context, t *testing.T, amount *big.Int) {
	response, txHash := d.network.SendTransaction(ctx, builders.Transaction().
		WithMethod(primitives.ContractName(d.orbsASBContractName), "transferOut").
		WithEd25519Signer(d.orbsUserKeyPair).
		WithArgs(d.addressInEthereum.From.Bytes(), amount.Uint64()).
		Builder(), 0)
	d.network.WaitForTransactionInState(ctx, txHash)

	t.Log(response.StringTransactionReceipt())
	t.Log(builders.PackedArgumentArrayDecode(response.TransactionReceipt().RawOutputArgumentArrayWithHeader()))
	test.RequireSuccess(t, response, "failed transfer out in orbs")
}

// Ethereum related funcs
func (d *driver) deployAutonomousSwapBridgeToEthereum(t *testing.T) {
	fakeFederation := common.BigToAddress(big.NewInt(1700))

	verifierAddress, _, err := d.simulator.DeployEthereumContract(d.addressInEthereum, verifierABI, verifierByteCode, fakeFederation)
	require.NoError(t, err, "could not deploy verifier to Ethereum")
	d.simulator.Commit()

	ethAsbAddress, ethAsbContract, err := d.simulator.DeployEthereumContract(d.addressInEthereum, asbABI, asbByteCode, uint32(0), uint64(builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID), //TODO CHANGE IN LEoNId,
		d.orbsASBContractName, d.erc20address, fakeFederation, verifierAddress)
	require.NoError(t, err, "could not deploy asb to Ethereum")
	d.simulator.Commit()
	d.ethASBAddress = ethAsbAddress
	d.ethASBContract = ethAsbContract
}

func (d *driver) deployTokenContractToEthereum(t *testing.T) {
	ethTetAddress, ethTetContract, err := d.simulator.DeployEthereumContract(d.addressInEthereum, tetABI, tetByteCode)
	require.NoError(t, err, "could not deploy erc token to Ethereum")
	d.simulator.Commit()
	d.erc20contract = ethTetContract
	d.erc20address = ethTetAddress
}

func (d *driver) generateEthereumAccountAndAssignFunds(t *testing.T, amount *big.Int) {
	ethContractUserAuth := d.addressInEthereum
	// we don't REALLY care who is the user we transfer from, so for simplicity's sake we use the same mega-user defined when simulator is created
	_, err := d.erc20contract.Transact(d.addressInEthereum, "assign", ethContractUserAuth.From /*address of user*/, amount)
	// generate token in source address
	require.NoError(t, err, "could not assign token to sender")
	d.simulator.Commit()
}

func (d *driver) getBalanceInEthereum(t *testing.T) uint64 {
	ethContractUserAuth := d.addressInEthereum
	// we don't REALLY care who is the user we transfer from, so for simplicity's sake we use the same mega-user defined when simulator is created
	var (
		ret0 = new(*big.Int)
	)
	result := ret0
	err := d.erc20contract.Call(nil, result, "balanceOf", ethContractUserAuth.From /*address of user*/)
	// generate token in source address
	require.NoError(t, err, "could not get token balance of user")
	d.simulator.Commit()
	return (*result).Uint64()
}

func (d *driver) approveTransferInEthereumTokenContract(t *testing.T, amount *big.Int) {
	_, err := d.erc20contract.Transact(d.addressInEthereum, "approve", d.ethASBAddress, amount)
	require.NoError(t, err, "could not approve transfer")
	d.simulator.Commit()
}

func (d *driver) transferOutFromEthereum(t *testing.T, amount *big.Int) string {
	transferOutTx, err := d.ethASBContract.Transact(d.addressInEthereum, "transferOut", d.orbsUserAddress, amount)
	require.NoError(t, err, "could not transfer out")
	d.simulator.Commit()

	receipt, err := d.simulator.Receipt(transferOutTx.Hash())
	require.NoError(t, err, "could not get receipt")

	require.EqualValues(t, types.ReceiptStatusSuccessful, receipt.Status, "call to transferOut on ASB in Ethereum failed")

	return transferOutTx.Hash().Hex()
}

const tetABI = `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"addedValue","type":"uint256"}],"name":"increaseAllowance","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"subtractedValue","type":"uint256"}],"name":"decreaseAllowance","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"constant":false,"inputs":[{"name":"_account","type":"address"},{"name":"_value","type":"uint256"}],"name":"assign","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
const tetByteCode = "0x608060405234801561001057600080fd5b506111aa806100206000396000f300608060405260043610610099576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063095ea7b31461009e57806318160ddd1461010357806323b872dd1461012e57806339509351146101b357806370a0823114610218578063a457c2d71461026f578063a9059cbb146102d4578063be76048814610339578063dd62ed3e14610386575b600080fd5b3480156100aa57600080fd5b506100e9600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506103fd565b604051808215151515815260200191505060405180910390f35b34801561010f57600080fd5b5061011861052a565b6040518082815260200191505060405180910390f35b34801561013a57600080fd5b50610199600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610534565b604051808215151515815260200191505060405180910390f35b3480156101bf57600080fd5b506101fe600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506106e6565b604051808215151515815260200191505060405180910390f35b34801561022457600080fd5b50610259600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061091d565b6040518082815260200191505060405180910390f35b34801561027b57600080fd5b506102ba600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610965565b604051808215151515815260200191505060405180910390f35b3480156102e057600080fd5b5061031f600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610b9c565b604051808215151515815260200191505060405180910390f35b34801561034557600080fd5b50610384600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610bb3565b005b34801561039257600080fd5b506103e7600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610bd3565b6040518082815260200191505060405180910390f35b60008073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415151561043a57600080fd5b81600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040518082815260200191505060405180910390a36001905092915050565b6000600254905090565b6000600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205482111515156105c157600080fd5b61065082600160008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610c5a90919063ffffffff16565b600160008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506106db848484610c7b565b600190509392505050565b60008073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415151561072357600080fd5b6107b282600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610e9490919063ffffffff16565b600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546040518082815260200191505060405180910390a36001905092915050565b60008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b60008073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16141515156109a257600080fd5b610a3182600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610c5a90919063ffffffff16565b600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546040518082815260200191505060405180910390a36001905092915050565b6000610ba9338484610c7b565b6001905092915050565b610bc582610bc08461091d565b610eb5565b610bcf8282611040565b5050565b6000600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b600080838311151515610c6c57600080fd5b82840390508091505092915050565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548111151515610cc857600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1614151515610d0457600080fd5b610d55816000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610c5a90919063ffffffff16565b6000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550610de8816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610e9490919063ffffffff16565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a3505050565b6000808284019050838110151515610eab57600080fd5b8091505092915050565b60008273ffffffffffffffffffffffffffffffffffffffff1614151515610edb57600080fd5b6000808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548111151515610f2857600080fd5b610f3d81600254610c5a90919063ffffffff16565b600281905550610f94816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610c5a90919063ffffffff16565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a35050565b60008273ffffffffffffffffffffffffffffffffffffffff161415151561106657600080fd5b61107b81600254610e9490919063ffffffff16565b6002819055506110d2816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610e9490919063ffffffff16565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a350505600a165627a7a7230582094609698b7c76dff80d05dcaa6cb6f31fb8f800a5026ee62cf6e4f71bba041a40029"

//const asbABI = `[{"constant":true,"inputs":[],"name":"orbsASBContractName","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"federation","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"renounceOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"isOwner","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"spentOrbsTuids","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"virtualChainId","outputs":[{"name":"","type":"uint64"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"tuidCounter","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"networkType","outputs":[{"name":"","type":"uint32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"token","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"VERSION","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_networkType","type":"uint32"},{"name":"_virtualChainId","type":"uint64"},{"name":"_orbsASBContractName","type":"string"},{"name":"_token","type":"address"},{"name":"_federation","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"bytes20"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredOut","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"bytes20"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredIn","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"constant":false,"inputs":[{"name":"_to","type":"bytes20"},{"name":"_value","type":"uint256"}],"name":"transferOut","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
//const asbByteCode = "0x6080604052600060045534801561001557600080fd5b50604051610f86380380610f868339810180604052810190808051906020019092919080519060200190929190805182019291906020018051906020019092919080519060200190929190505050336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3600083511115156101be576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260298152602001807f4f7262732041534220636f6e7472616374206e616d65206d757374206e6f742081526020017f626520656d70747921000000000000000000000000000000000000000000000081525060400191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1614151515610263576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260148152602001807f546f6b656e206d757374206e6f7420626520302100000000000000000000000081525060200191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614151515610308576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260198152602001807f46656465726174696f6e206d757374206e6f742062652030210000000000000081525060200191505060405180910390fd5b84600060146101000a81548163ffffffff021916908363ffffffff16021790555083600060186101000a81548167ffffffffffffffff021916908367ffffffffffffffff16021790555082600190805190602001906103689291906103f5565b5081600260006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555080600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505050505061049a565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061043657805160ff1916838001178555610464565b82800160010185558215610464579182015b82811115610463578251825591602001919060010190610448565b5b5090506104719190610475565b5090565b61049791905b8082111561049357600081600090555060010161047b565b5090565b90565b610add806104a96000396000f3006080604052600436106100c5576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806333d7fafd146100ca5780635c45428c1461011057806366c86119146101a0578063715018a6146101f75780638da5cb5b1461020e5780638f32d59b14610265578063923aebf014610294578063bd19dffb146102d9578063e1d5c25514610318578063f2fde38b14610343578063f3762c1114610386578063fc0c546a146103bd578063ffa1ad7414610414575b600080fd5b3480156100d657600080fd5b5061010e60048036038101908080356bffffffffffffffffffffffff191690602001909291908035906020019092919050505061043f565b005b34801561011c57600080fd5b506101256106e0565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561016557808201518184015260208101905061014a565b50505050905090810190601f1680156101925780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b3480156101ac57600080fd5b506101b561077e565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561020357600080fd5b5061020c6107a4565b005b34801561021a57600080fd5b50610223610876565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561027157600080fd5b5061027a61089f565b604051808215151515815260200191505060405180910390f35b3480156102a057600080fd5b506102bf600480360381019080803590602001909291905050506108f6565b604051808215151515815260200191505060405180910390f35b3480156102e557600080fd5b506102ee610916565b604051808267ffffffffffffffff1667ffffffffffffffff16815260200191505060405180910390f35b34801561032457600080fd5b5061032d610930565b6040518082815260200191505060405180910390f35b34801561034f57600080fd5b50610384600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610936565b005b34801561039257600080fd5b5061039b610955565b604051808263ffffffff1663ffffffff16815260200191505060405180910390f35b3480156103c957600080fd5b506103d261096b565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561042057600080fd5b50610429610991565b6040518082815260200191505060405180910390f35b6000811115156104b7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601d8152602001807f56616c7565206d7573742062652067726561746572207468616e20302100000081525060200191505060405180910390fd5b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166323b872dd3330846040518463ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019350505050602060405180830381600087803b1580156105b057600080fd5b505af11580156105c4573d6000803e3d6000fd5b505050506040513d60208110156105da57600080fd5b8101908080519060200190929190505050151561065f576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260178152602001807f496e73756666696369656e7420616c6c6f77616e63652100000000000000000081525060200191505060405180910390fd5b610675600160045461099690919063ffffffff16565b600481905550816bffffffffffffffffffffffff19163373ffffffffffffffffffffffffffffffffffffffff166004547fc7d2da8a0df0279cb4e0a81f2975445675cc6527c94016791d29977a1fa0f251846040518082815260200191505060405180910390a45050565b60018054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156107765780601f1061074b57610100808354040283529160200191610776565b820191906000526020600020905b81548152906001019060200180831161075957829003601f168201915b505050505081565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6107ac61089f565b15156107b757600080fd5b600073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a360008060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614905090565b60056020528060005260406000206000915054906101000a900460ff1681565b600060189054906101000a900467ffffffffffffffff1681565b60045481565b61093e61089f565b151561094957600080fd5b610952816109b7565b50565b600060149054906101000a900463ffffffff1681565b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600181565b60008082840190508381101515156109ad57600080fd5b8091505092915050565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515156109f357600080fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505600a165627a7a72305820dc57f244d775b10898e9eb6a77cb0e0d0c0e0c9a6f66c39ac55d2442a523faaa0029"

const asbABI = `[{"constant":true,"inputs":[],"name":"verifier","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"orbsASBContractName","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"federation","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"renounceOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"isOwner","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"spentOrbsTuids","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"virtualChainId","outputs":[{"name":"","type":"uint64"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"tuidCounter","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"networkType","outputs":[{"name":"","type":"uint32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"token","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"VERSION","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_networkType","type":"uint32"},{"name":"_virtualChainId","type":"uint64"},{"name":"_orbsASBContractName","type":"string"},{"name":"_token","type":"address"},{"name":"_federation","type":"address"},{"name":"_verifier","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"bytes20"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredOut","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"bytes20"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredIn","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"constant":false,"inputs":[{"name":"_to","type":"bytes20"},{"name":"_value","type":"uint256"}],"name":"transferOut","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_resultsBlockHeader","type":"bytes"},{"name":"_resultsBlockProof","type":"bytes"},{"name":"_transactionReceipt","type":"bytes"},{"name":"_transactionReceiptProof","type":"bytes32[]"}],"name":"transferIn","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_verifier","type":"address"}],"name":"setAutonomousSwapProofVerifier","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
const asbByteCode = "0x608060405260006005553480156200001657600080fd5b50604051620029ad380380620029ad83398101806040526200003c9190810190620005e6565b336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a36000845111151562000141576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401620001389062000797565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1614151515620001b6576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401620001ad90620007fd565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16141515156200022b576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016200022290620007db565b60405180910390fd5b620002458162000336640100000000026401000000009004565b85600060146101000a81548163ffffffff021916908363ffffffff16021790555084600060186101000a81548167ffffffffffffffff021916908367ffffffffffffffff1602179055508360019080519060200190620002a79291906200046b565b5082600260006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555081600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050505050505062000930565b6200034f62000414640100000000026401000000009004565b15156200035b57600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614151515620003d0576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401620003c790620007b9565b60405180910390fd5b80600460006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614905090565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f10620004ae57805160ff1916838001178555620004df565b82800160010185558215620004df579182015b82811115620004de578251825591602001919060010190620004c1565b5b509050620004ee9190620004f2565b5090565b6200051791905b8082111562000513576000816000905550600101620004f9565b5090565b90565b60006200052882516200089a565b905092915050565b60006200053e8251620008ae565b905092915050565b6000620005548251620008c2565b905092915050565b600082601f83011215156200057057600080fd5b81516200058762000581826200084d565b6200081f565b91508082526020830160208301858383011115620005a457600080fd5b620005b1838284620008fa565b50505092915050565b6000620005c88251620008d6565b905092915050565b6000620005de8251620008e6565b905092915050565b60008060008060008060c087890312156200060057600080fd5b60006200061089828a01620005ba565b96505060206200062389828a01620005d0565b955050604087015167ffffffffffffffff8111156200064157600080fd5b6200064f89828a016200055c565b94505060606200066289828a0162000530565b93505060806200067589828a0162000546565b92505060a06200068889828a016200051a565b9150509295509295509295565b6000602982527f4f7262732041534220636f6e7472616374206e616d65206d757374206e6f742060208301527f626520656d7074792100000000000000000000000000000000000000000000006040830152606082019050919050565b6000601782527f5665726966696572206d757374206e6f742062652030210000000000000000006020830152604082019050919050565b6000601982527f46656465726174696f6e206d757374206e6f74206265203021000000000000006020830152604082019050919050565b6000601482527f546f6b656e206d757374206e6f742062652030210000000000000000000000006020830152604082019050919050565b60006020820190508181036000830152620007b28162000695565b9050919050565b60006020820190508181036000830152620007d481620006f2565b9050919050565b60006020820190508181036000830152620007f68162000729565b9050919050565b60006020820190508181036000830152620008188162000760565b9050919050565b6000604051905081810181811067ffffffffffffffff821117156200084357600080fd5b8060405250919050565b600067ffffffffffffffff8211156200086557600080fd5b601f19601f8301169050602081019050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000620008a7826200087a565b9050919050565b6000620008bb826200087a565b9050919050565b6000620008cf826200087a565b9050919050565b600063ffffffff82169050919050565b600067ffffffffffffffff82169050919050565b60005b838110156200091a578082015181840152602081019050620008fd565b838111156200092a576000848401525b50505050565b61206d80620009406000396000f3006080604052600436106100e6576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680632b7ac3f3146100eb57806333d7fafd146101165780635c45428c1461013f57806366c861191461016a578063715018a6146101955780638da5cb5b146101ac5780638f32d59b146101d7578063923aebf014610202578063aebac8801461023f578063bd19dffb14610268578063d2b43fee14610293578063e1d5c255146102bc578063f2fde38b146102e7578063f3762c1114610310578063fc0c546a1461033b578063ffa1ad7414610366575b600080fd5b3480156100f757600080fd5b50610100610391565b60405161010d9190611b8c565b60405180910390f35b34801561012257600080fd5b5061013d6004803603610138919081019061153f565b6103b7565b005b34801561014b57600080fd5b50610154610695565b6040516101619190611bdd565b60405180910390f35b34801561017657600080fd5b5061017f610733565b60405161018c9190611bc2565b60405180910390f35b3480156101a157600080fd5b506101aa610759565b005b3480156101b857600080fd5b506101c161082b565b6040516101ce9190611a7a565b60405180910390f35b3480156101e357600080fd5b506101ec610854565b6040516101f99190611af5565b60405180910390f35b34801561020e57600080fd5b50610229600480360361022491908101906116a8565b6108ab565b6040516102369190611af5565b60405180910390f35b34801561024b57600080fd5b506102666004803603610261919081019061163e565b6108cb565b005b34801561027457600080fd5b5061027d610994565b60405161028a9190611d75565b60405180910390f35b34801561029f57600080fd5b506102ba60048036036102b5919081019061157b565b6109ae565b005b3480156102c857600080fd5b506102d1610f36565b6040516102de9190611d3f565b60405180910390f35b3480156102f357600080fd5b5061030e600480360361030991908101906114ed565b610f3c565b005b34801561031c57600080fd5b50610325610f5b565b6040516103329190611d5a565b60405180910390f35b34801561034757600080fd5b50610350610f71565b60405161035d9190611ba7565b60405180910390f35b34801561037257600080fd5b5061037b610f97565b6040516103889190611d3f565b60405180910390f35b600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166361cd4172836040518263ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040161042e9190611b10565b602060405180830381600087803b15801561044857600080fd5b505af115801561045c573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052506104809190810190611516565b15156104c1576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016104b890611bff565b60405180910390fd5b600081111515610506576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016104fd90611c7f565b60405180910390fd5b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166323b872dd3330846040518463ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040161058193929190611a95565b602060405180830381600087803b15801561059b57600080fd5b505af11580156105af573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052506105d39190810190611516565b1515610614576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161060b90611c5f565b60405180910390fd5b61062a6001600554610f9c90919063ffffffff16565b600581905550816bffffffffffffffffffffffff19163373ffffffffffffffffffffffffffffffffffffffff166005547fc7d2da8a0df0279cb4e0a81f2975445675cc6527c94016791d29977a1fa0f251846040516106899190611d3f565b60405180910390a45050565b60018054600181600116156101000203166002900480601f01602080910402602001604051908101604052809291908181526020018280546001816001161561010002031660029004801561072b5780601f106107005761010080835404028352916020019161072b565b820191906000526020600020905b81548152906001019060200180831161070e57829003601f168201915b505050505081565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b610761610854565b151561076c57600080fd5b600073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a360008060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614905090565b60066020528060005260406000206000915054906101000a900460ff1681565b6108d3610854565b15156108de57600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614151515610950576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161094790611c3f565b60405180910390fd5b80600460006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b600060189054906101000a900467ffffffffffffffff1681565b6109b66111a5565b600460009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a7c53a75868686866040518563ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401610a339493929190611b2b565b600060405180830381600087803b158015610a4d57600080fd5b505af1158015610a61573d6000803e3d6000fd5b505050506040513d6000823e3d601f19601f82011682018060405250610a8a9190810190611667565b9050600073ffffffffffffffffffffffffffffffffffffffff16816080015173ffffffffffffffffffffffffffffffffffffffff1614151515610b02576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610af990611cbf565b60405180910390fd5b60008160a00151111515610b4b576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610b4290611c7f565b60405180910390fd5b806000015163ffffffff16600060149054906101000a900463ffffffff1663ffffffff16141515610bb1576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610ba890611cdf565b60405180910390fd5b806020015167ffffffffffffffff16600060189054906101000a900467ffffffffffffffff1667ffffffffffffffff16141515610c23576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610c1a90611c1f565b60405180910390fd5b610cd4816040015160018054600181600116156101000203166002900480601f016020809104026020016040519081016040528092919081815260200182805460018160011615610100020316600290048015610cc15780601f10610c9657610100808354040283529160200191610cc1565b820191906000526020600020905b815481529060010190602001808311610ca457829003601f168201915b5050505050610fbd90919063ffffffff16565b1515610d15576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610d0c90611cff565b60405180910390fd5b600660008260c00151815260200190815260200160002060009054906101000a900460ff16151515610d7c576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610d7390611c9f565b60405180910390fd5b6001600660008360c00151815260200190815260200160002060006101000a81548160ff021916908315150217905550600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a9059cbb82608001518360a001516040518363ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401610e2d929190611acc565b602060405180830381600087803b158015610e4757600080fd5b505af1158015610e5b573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250610e7f9190810190611516565b1515610ec0576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610eb790611d1f565b60405180910390fd5b806080015173ffffffffffffffffffffffffffffffffffffffff1681606001516bffffffffffffffffffffffff19168260c001517f0e1884a0d68b0d4801a22895bf83fe72c3ae24c9cf5b2dec620e192dd225b8a88460a00151604051610f279190611d3f565b60405180910390a45050505050565b60055481565b610f44610854565b1515610f4f57600080fd5b610f58816110ab565b50565b600060149054906101000a900463ffffffff1681565b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600181565b6000808284019050838110151515610fb357600080fd5b8091505092915050565b600081518351141515610fd357600090506110a5565b816040518082805190602001908083835b6020831015156110095780518252602082019150602081019050602083039250610fe4565b6001836020036101000a038019825116818451168082178552505050505050905001915050604051809103902060001916836040518082805190602001908083835b602083101515611070578051825260208201915060208101905060208303925061104b565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040518091039020600019161490505b92915050565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515156110e757600080fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b60e060405190810160405280600063ffffffff168152602001600067ffffffffffffffff1681526020016060815260200160006bffffffffffffffffffffffff19168152602001600073ffffffffffffffffffffffffffffffffffffffff16815260200160008152602001600081525090565b60006112248235611f08565b905092915050565b60006112388251611f08565b905092915050565b600082601f830112151561125357600080fd5b813561126661126182611dbd565b611d90565b9150818183526020840193506020810190508385602084028201111561128b57600080fd5b60005b838110156112bb57816112a18882611301565b84526020840193506020830192505060018101905061128e565b5050505092915050565b60006112d18251611f28565b905092915050565b60006112e58235611f34565b905092915050565b60006112f98251611f34565b905092915050565b600061130d8235611f60565b905092915050565b600082601f830112151561132857600080fd5b813561133b61133682611de5565b611d90565b9150808252602083016020830185838301111561135757600080fd5b611362838284611fe0565b50505092915050565b60006113778235611f6a565b905092915050565b600082601f830112151561139257600080fd5b81516113a56113a082611e11565b611d90565b915080825260208301602083018583830111156113c157600080fd5b6113cc838284611fef565b50505092915050565b600060e082840312156113e757600080fd5b6113f160e0611d90565b90506000611401848285016114c5565b6000830152506020611415848285016114d9565b602083015250604082015167ffffffffffffffff81111561143557600080fd5b6114418482850161137f565b6040830152506060611455848285016112ed565b60608301525060806114698482850161122c565b60808301525060a061147d848285016114b1565b60a08301525060c0611491848285016114b1565b60c08301525092915050565b60006114a98235611f7c565b905092915050565b60006114bd8251611f7c565b905092915050565b60006114d18251611f86565b905092915050565b60006114e58251611f96565b905092915050565b6000602082840312156114ff57600080fd5b600061150d84828501611218565b91505092915050565b60006020828403121561152857600080fd5b6000611536848285016112c5565b91505092915050565b6000806040838503121561155257600080fd5b6000611560858286016112d9565b92505060206115718582860161149d565b9150509250929050565b6000806000806080858703121561159157600080fd5b600085013567ffffffffffffffff8111156115ab57600080fd5b6115b787828801611315565b945050602085013567ffffffffffffffff8111156115d457600080fd5b6115e087828801611315565b935050604085013567ffffffffffffffff8111156115fd57600080fd5b61160987828801611315565b925050606085013567ffffffffffffffff81111561162657600080fd5b61163287828801611240565b91505092959194509250565b60006020828403121561165057600080fd5b600061165e8482850161136b565b91505092915050565b60006020828403121561167957600080fd5b600082015167ffffffffffffffff81111561169357600080fd5b61169f848285016113d5565b91505092915050565b6000602082840312156116ba57600080fd5b60006116c88482850161149d565b91505092915050565b6116da81611e78565b82525050565b60006116eb82611e4a565b8084526020840193506116fd83611e3d565b60005b8281101561172f57611713868351611759565b61171c82611e6b565b9150602086019550600181019050611700565b50849250505092915050565b61174481611e98565b82525050565b61175381611ea4565b82525050565b61176281611ed0565b82525050565b600061177382611e55565b808452611787816020860160208601611fef565b61179081612022565b602085010191505092915050565b6117a781611faa565b82525050565b6117b681611fbc565b82525050565b6117c581611fce565b82525050565b60006117d682611e60565b8084526117ea816020860160208601611fef565b6117f381612022565b602085010191505092915050565b6000601882527f4f726273206164647265737320697320696e76616c69642100000000000000006020830152604082019050919050565b6000601b82527f496e636f7272656374207669727475616c20636861696e2049442100000000006020830152604082019050919050565b6000601782527f5665726966696572206d757374206e6f742062652030210000000000000000006020830152604082019050919050565b6000601782527f496e73756666696369656e7420616c6c6f77616e6365210000000000000000006020830152604082019050919050565b6000601d82527f56616c7565206d7573742062652067726561746572207468616e2030210000006020830152604082019050919050565b6000601782527f545549442077617320616c7265616479207370656e74210000000000000000006020830152604082019050919050565b6000601f82527f44657374696e6174696f6e20616464726573732063616e2774206265203021006020830152604082019050919050565b6000601782527f496e636f7272656374206e6574776f726b2074797065210000000000000000006020830152604082019050919050565b6000602182527f496e636f7272656374204f7262732041534220636f6e7472616374206e616d6560208301527f21000000000000000000000000000000000000000000000000000000000000006040830152606082019050919050565b6000601382527f496e73756666696369656e742066756e647321000000000000000000000000006020830152604082019050919050565b611a5681611eda565b82525050565b611a6581611ee4565b82525050565b611a7481611ef4565b82525050565b6000602082019050611a8f60008301846116d1565b92915050565b6000606082019050611aaa60008301866116d1565b611ab760208301856116d1565b611ac46040830184611a4d565b949350505050565b6000604082019050611ae160008301856116d1565b611aee6020830184611a4d565b9392505050565b6000602082019050611b0a600083018461173b565b92915050565b6000602082019050611b25600083018461174a565b92915050565b60006080820190508181036000830152611b458187611768565b90508181036020830152611b598186611768565b90508181036040830152611b6d8185611768565b90508181036060830152611b8181846116e0565b905095945050505050565b6000602082019050611ba1600083018461179e565b92915050565b6000602082019050611bbc60008301846117ad565b92915050565b6000602082019050611bd760008301846117bc565b92915050565b60006020820190508181036000830152611bf781846117cb565b905092915050565b60006020820190508181036000830152611c1881611801565b9050919050565b60006020820190508181036000830152611c3881611838565b9050919050565b60006020820190508181036000830152611c588161186f565b9050919050565b60006020820190508181036000830152611c78816118a6565b9050919050565b60006020820190508181036000830152611c98816118dd565b9050919050565b60006020820190508181036000830152611cb881611914565b9050919050565b60006020820190508181036000830152611cd88161194b565b9050919050565b60006020820190508181036000830152611cf881611982565b9050919050565b60006020820190508181036000830152611d18816119b9565b9050919050565b60006020820190508181036000830152611d3881611a16565b9050919050565b6000602082019050611d546000830184611a4d565b92915050565b6000602082019050611d6f6000830184611a5c565b92915050565b6000602082019050611d8a6000830184611a6b565b92915050565b6000604051905081810181811067ffffffffffffffff82111715611db357600080fd5b8060405250919050565b600067ffffffffffffffff821115611dd457600080fd5b602082029050602081019050919050565b600067ffffffffffffffff821115611dfc57600080fd5b601f19601f8301169050602081019050919050565b600067ffffffffffffffff821115611e2857600080fd5b601f19601f8301169050602081019050919050565b6000602082019050919050565b600081519050919050565b600081519050919050565b600081519050919050565b6000602082019050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60008115159050919050565b60007fffffffffffffffffffffffffffffffffffffffff00000000000000000000000082169050919050565b6000819050919050565b6000819050919050565b600063ffffffff82169050919050565b600067ffffffffffffffff82169050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60008115159050919050565b60007fffffffffffffffffffffffffffffffffffffffff00000000000000000000000082169050919050565b6000819050919050565b6000611f7582611e78565b9050919050565b6000819050919050565b600063ffffffff82169050919050565b600067ffffffffffffffff82169050919050565b6000611fb582611e78565b9050919050565b6000611fc782611e78565b9050919050565b6000611fd982611e78565b9050919050565b82818337600083830152505050565b60005b8381101561200d578082015181840152602081019050611ff2565b8381111561201c576000848401525b50505050565b6000601f19601f83011690509190505600a265627a7a723058203a060a72adc9e4fa81fd78a34f3f84017551db3328cdd42c0a82a6e969f96f346c6578706572696d656e74616cf50037"

const verifierABI = ` [
    {
      "constant": true,
      "inputs": [],
      "name": "EXECUTION_RESULT_SUCCESS",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "MAX_SIGNATURES",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "BLOCKREFMESSAGE_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "ORBS_ADDRESS_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "SIGNATURE_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "BLOCKHASH_OFFSET",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "SHA256_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "EXECUTION_RESULT_OFFSET",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "federation",
      "outputs": [
        {
          "name": "",
          "type": "address"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "UINT32_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "ORBS_PROTOCOL_VERSION",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "NODE_PK_SIG_NESTING_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "UINT64_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "TRANSFERRED_OUT",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "ADDRESS_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "UINT256_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "ONEOF_NESTING_SIZE",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [],
      "name": "VERSION",
      "outputs": [
        {
          "name": "",
          "type": "uint256"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "name": "_federation",
          "type": "address"
        }
      ],
      "payable": false,
      "stateMutability": "nonpayable",
      "type": "constructor"
    },
    {
      "constant": true,
      "inputs": [
        {
          "name": "_resultsBlockHeader",
          "type": "bytes"
        },
        {
          "name": "_resultsBlockProof",
          "type": "bytes"
        },
        {
          "name": "_transactionReceipt",
          "type": "bytes"
        },
        {
          "name": "_transactionReceiptProof",
          "type": "bytes32[]"
        }
      ],
      "name": "processProof",
      "outputs": [
        {
          "name": "transferInEvent",
          "type": "string"
        }
      ],
      "payable": false,
      "stateMutability": "view",
      "type": "function"
    },
    {
      "constant": true,
      "inputs": [
        {
          "name": "_address",
          "type": "bytes20"
        }
      ],
      "name": "isOrbsAddressValid",
      "outputs": [
        {
          "name": "",
          "type": "bool"
        }
      ],
      "payable": false,
      "stateMutability": "pure",
      "type": "function"
    }
  ]`
const verifierByteCode = "0x608060405234801561001057600080fd5b5060405160208061088283398101806040528101908080519060200190929190505050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515156100d8576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260198152602001807f46656465726174696f6e206d757374206e6f742062652030210000000000000081525060200191505060405180910390fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505061075a806101286000396000f300608060405260043610610111576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806212117d14610116578063037a379614610141578063047ee7e81461016c57806326413cd014610197578063308ff0c9146101c2578063397d6e17146101ed578063550bed71146102185780635680b8aa1461024357806361cd41721461026e57806366c86119146102c2578063690ba608146103195780636d2c93ac146103445780638d9ecc661461036f5780639c3db9a51461039a578063a7c53a75146103c5578063b11954b314610576578063c4c3c17e146105a1578063eea7f0ab146105cc578063f18a25ce146105f7578063ffa1ad7414610622575b600080fd5b34801561012257600080fd5b5061012b61064d565b6040518082815260200191505060405180910390f35b34801561014d57600080fd5b50610156610652565b6040518082815260200191505060405180910390f35b34801561017857600080fd5b50610181610657565b6040518082815260200191505060405180910390f35b3480156101a357600080fd5b506101ac61065c565b6040518082815260200191505060405180910390f35b3480156101ce57600080fd5b506101d7610661565b6040518082815260200191505060405180910390f35b3480156101f957600080fd5b50610202610666565b6040518082815260200191505060405180910390f35b34801561022457600080fd5b5061022d61066b565b6040518082815260200191505060405180910390f35b34801561024f57600080fd5b50610258610670565b6040518082815260200191505060405180910390f35b34801561027a57600080fd5b506102a860048036038101908080356bffffffffffffffffffffffff19169060200190929190505050610675565b604051808215151515815260200191505060405180910390f35b3480156102ce57600080fd5b506102d76106bf565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561032557600080fd5b5061032e6106e4565b6040518082815260200191505060405180910390f35b34801561035057600080fd5b506103596106e9565b6040518082815260200191505060405180910390f35b34801561037b57600080fd5b506103846106ee565b6040518082815260200191505060405180910390f35b3480156103a657600080fd5b506103af6106f3565b6040518082815260200191505060405180910390f35b3480156103d157600080fd5b506104fb600480360381019080803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080602002602001604051908101604052809392919081815260200183836020028082843782019150505050505091929192905050506106f8565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561053b578082015181840152602081019050610520565b50505050905090810190601f1680156105685780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561058257600080fd5b5061058b610715565b6040518082815260200191505060405180910390f35b3480156105ad57600080fd5b506105b661071a565b6040518082815260200191505060405180910390f35b3480156105d857600080fd5b506105e161071f565b6040518082815260200191505060405180910390f35b34801561060357600080fd5b5061060c610724565b6040518082815260200191505060405180910390f35b34801561062e57600080fd5b50610637610729565b6040518082815260200191505060405180910390f35b600181565b606481565b603481565b601481565b604181565b601481565b602081565b602481565b6000806c01000000000000000000000000026bffffffffffffffffffffffff1916826bffffffffffffffffffffffff191614156106b557600090506106ba565b600190505b919050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600481565b600281565b600481565b600881565b606060206040519081016040528060008152509050949350505050565b600181565b601481565b602081565b600c81565b6001815600a165627a7a7230582094ec3ba0712d2514c135671ecc0b9487c56e09ba8efc78a976a3a6977bf80f620029"

const npragma = "0x608060405234801561001057600080fd5b5060405160208061088283398101806040528101908080519060200190929190505050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515156100d8576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260198152602001807f46656465726174696f6e206d757374206e6f742062652030210000000000000081525060200191505060405180910390fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505061075a806101286000396000f300608060405260043610610111576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806212117d14610116578063037a379614610141578063047ee7e81461016c57806326413cd014610197578063308ff0c9146101c2578063397d6e17146101ed578063550bed71146102185780635680b8aa1461024357806361cd41721461026e57806366c86119146102c2578063690ba608146103195780636d2c93ac146103445780638d9ecc661461036f5780639c3db9a51461039a578063a7c53a75146103c5578063b11954b314610576578063c4c3c17e146105a1578063eea7f0ab146105cc578063f18a25ce146105f7578063ffa1ad7414610622575b600080fd5b34801561012257600080fd5b5061012b61064d565b6040518082815260200191505060405180910390f35b34801561014d57600080fd5b50610156610652565b6040518082815260200191505060405180910390f35b34801561017857600080fd5b50610181610657565b6040518082815260200191505060405180910390f35b3480156101a357600080fd5b506101ac61065c565b6040518082815260200191505060405180910390f35b3480156101ce57600080fd5b506101d7610661565b6040518082815260200191505060405180910390f35b3480156101f957600080fd5b50610202610666565b6040518082815260200191505060405180910390f35b34801561022457600080fd5b5061022d61066b565b6040518082815260200191505060405180910390f35b34801561024f57600080fd5b50610258610670565b6040518082815260200191505060405180910390f35b34801561027a57600080fd5b506102a860048036038101908080356bffffffffffffffffffffffff19169060200190929190505050610675565b604051808215151515815260200191505060405180910390f35b3480156102ce57600080fd5b506102d76106bf565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561032557600080fd5b5061032e6106e4565b6040518082815260200191505060405180910390f35b34801561035057600080fd5b506103596106e9565b6040518082815260200191505060405180910390f35b34801561037b57600080fd5b506103846106ee565b6040518082815260200191505060405180910390f35b3480156103a657600080fd5b506103af6106f3565b6040518082815260200191505060405180910390f35b3480156103d157600080fd5b506104fb600480360381019080803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080602002602001604051908101604052809392919081815260200183836020028082843782019150505050505091929192905050506106f8565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561053b578082015181840152602081019050610520565b50505050905090810190601f1680156105685780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561058257600080fd5b5061058b610715565b6040518082815260200191505060405180910390f35b3480156105ad57600080fd5b506105b661071a565b6040518082815260200191505060405180910390f35b3480156105d857600080fd5b506105e161071f565b6040518082815260200191505060405180910390f35b34801561060357600080fd5b5061060c610724565b6040518082815260200191505060405180910390f35b34801561062e57600080fd5b50610637610729565b6040518082815260200191505060405180910390f35b600181565b606481565b603481565b601481565b604181565b601481565b602081565b602481565b6000806c01000000000000000000000000026bffffffffffffffffffffffff1916826bffffffffffffffffffffffff191614156106b557600090506106ba565b600190505b919050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600481565b600281565b600481565b600881565b606060206040519081016040528060008152509050949350505050565b600181565b601481565b602081565b600c81565b6001815600a165627a7a7230582094ec3ba0712d2514c135671ecc0b9487c56e09ba8efc78a976a3a6977bf80f620029"
const ypragma = "0x608060405234801561001057600080fd5b5060405160208061088283398101806040528101908080519060200190929190505050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515156100d8576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260198152602001807f46656465726174696f6e206d757374206e6f742062652030210000000000000081525060200191505060405180910390fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505061075a806101286000396000f300608060405260043610610111576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806212117d14610116578063037a379614610141578063047ee7e81461016c57806326413cd014610197578063308ff0c9146101c2578063397d6e17146101ed578063550bed71146102185780635680b8aa1461024357806361cd41721461026e57806366c86119146102c2578063690ba608146103195780636d2c93ac146103445780638d9ecc661461036f5780639c3db9a51461039a578063a7c53a75146103c5578063b11954b314610576578063c4c3c17e146105a1578063eea7f0ab146105cc578063f18a25ce146105f7578063ffa1ad7414610622575b600080fd5b34801561012257600080fd5b5061012b61064d565b6040518082815260200191505060405180910390f35b34801561014d57600080fd5b50610156610652565b6040518082815260200191505060405180910390f35b34801561017857600080fd5b50610181610657565b6040518082815260200191505060405180910390f35b3480156101a357600080fd5b506101ac61065c565b6040518082815260200191505060405180910390f35b3480156101ce57600080fd5b506101d7610661565b6040518082815260200191505060405180910390f35b3480156101f957600080fd5b50610202610666565b6040518082815260200191505060405180910390f35b34801561022457600080fd5b5061022d61066b565b6040518082815260200191505060405180910390f35b34801561024f57600080fd5b50610258610670565b6040518082815260200191505060405180910390f35b34801561027a57600080fd5b506102a860048036038101908080356bffffffffffffffffffffffff19169060200190929190505050610675565b604051808215151515815260200191505060405180910390f35b3480156102ce57600080fd5b506102d76106bf565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561032557600080fd5b5061032e6106e4565b6040518082815260200191505060405180910390f35b34801561035057600080fd5b506103596106e9565b6040518082815260200191505060405180910390f35b34801561037b57600080fd5b506103846106ee565b6040518082815260200191505060405180910390f35b3480156103a657600080fd5b506103af6106f3565b6040518082815260200191505060405180910390f35b3480156103d157600080fd5b506104fb600480360381019080803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080601f0160208091040260200160405190810160405280939291908181526020018383808284378201915050505050509192919290803590602001908201803590602001908080602002602001604051908101604052809392919081815260200183836020028082843782019150505050505091929192905050506106f8565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561053b578082015181840152602081019050610520565b50505050905090810190601f1680156105685780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561058257600080fd5b5061058b610715565b6040518082815260200191505060405180910390f35b3480156105ad57600080fd5b506105b661071a565b6040518082815260200191505060405180910390f35b3480156105d857600080fd5b506105e161071f565b6040518082815260200191505060405180910390f35b34801561060357600080fd5b5061060c610724565b6040518082815260200191505060405180910390f35b34801561062e57600080fd5b50610637610729565b6040518082815260200191505060405180910390f35b600181565b606481565b603481565b601481565b604181565b601481565b602081565b602481565b6000806c01000000000000000000000000026bffffffffffffffffffffffff1916826bffffffffffffffffffffffff191614156106b557600090506106ba565b600190505b919050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600481565b600281565b600481565b600881565b606060206040519081016040528060008152509050949350505050565b600181565b601481565b602081565b600c81565b6001815600a165627a7a72305820cb8439ca8a63f21876be4cff618e1fb08e2eedf6ef9a44851870657abc637a020029"
