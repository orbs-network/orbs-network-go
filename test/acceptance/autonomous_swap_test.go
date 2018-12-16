package acceptance

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/orbs-network/orbs-client-sdk-go/orbsclient"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/ASBEthereum"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/OIP2"
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
	harness.
		Network(t).
		Start(func(ctx context.Context, network harness.TestNetworkDriver) {
			amount := big.NewInt(17)

			d := newDriver(network)

			d.generateOrbsAccount(t)

			d.deployERC20Contract(t)
			d.generateEthAccountAndAssignFunds(t, amount)

			d.deployAutonomousSwapBridge(t)
			d.bindAutonomousSwapBridges(ctx, t)

			d.approveTransferInTokenContract(t, amount)
			transferOutTxHash := d.transferOutFromEthereum(t, amount)

			// TODO v1 deploy causes who is owner - important for both.
			d.transferInToOrbs(ctx, t, transferOutTxHash)

			balanceAfterTransfer := d.getBalance(ctx, t)
			require.EqualValues(t, amount.Uint64(), balanceAfterTransfer, "wrong amount")
		})
}

func newDriver(networkDriver harness.TestNetworkDriver) *driver {
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
	network                  harness.TestNetworkDriver
	simulator                *adapter.EthereumSimulator
	orbsUser                 *orbsclient.OrbsAccount
	addressInEthereum        *bind.TransactOpts // we use a single address for both the "admin" stuff like deploying the contracts and as our swapping user, so as to simplify setup - otherwise we'll need to create two PKs in the simulator
	erc20contract            *bind.BoundContract
	erc20address             *common.Address
	orbsASBContractName      string
	ethASBAddress            *common.Address
	ethASBContract           *bind.BoundContract
	orbsContractOwnerAddress *keys.Ed25519KeyPair
}

func (d *driver) transferOutFromEthereum(t *testing.T, amount *big.Int) string {
	var orbsUserAddress [20]byte
	copy(orbsUserAddress[:], d.orbsUser.RawAddress)

	transferOutTx, err := d.ethASBContract.Transact(d.addressInEthereum, "transferOut", orbsUserAddress, amount)
	require.NoError(t, err, "could not transfer out")
	d.simulator.Commit()
	return transferOutTx.Hash().Hex()
}

func (d *driver) approveTransferInTokenContract(t *testing.T, amount *big.Int) {
	_, err := d.erc20contract.Transact(d.addressInEthereum, "approve", d.ethASBAddress, amount)
	require.NoError(t, err, "could not approve transfer")
	d.simulator.Commit()
}

func (d *driver) getBalance(ctx context.Context, t *testing.T) uint64 {
	balanceResponse := d.network.CallMethod(ctx, builders.Transaction().
		WithEd25519Signer(d.orbsContractOwnerAddress).
		WithMethod(primitives.ContractName(oip2.CONTRACT_NAME), "balanceOf").
		WithArgs(d.orbsUser.RawAddress).
		Builder().Transaction, 0)
	require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS, balanceResponse.CallMethodResult())
	// check that the tokens got there.
	outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(balanceResponse)
	value := outputArgsIterator.NextArguments().Uint64Value()
	return value
}

func (d *driver) transferInToOrbs(ctx context.Context, t *testing.T, transferOutTxHash string) {
	response, txHash := d.network.SendTransaction(ctx, builders.Transaction().
		WithMethod(primitives.ContractName(d.orbsASBContractName), "transferIn").
		WithEd25519Signer(d.orbsContractOwnerAddress).
		WithArgs(transferOutTxHash).
		Builder(), 0)
	d.network.WaitForTransactionInState(ctx, txHash)
	test.RequireSuccess(t, response, "failed setting asb address")
}

func (d *driver) bindAutonomousSwapBridges(ctx context.Context, t *testing.T) {
	response, txHash := d.network.SendTransaction(ctx, builders.Transaction().
		WithMethod(primitives.ContractName(d.orbsASBContractName), "setAsbAddr").
		WithEd25519Signer(d.orbsContractOwnerAddress).
		WithArgs(d.ethASBAddress.Hex()).
		Builder(), 0)
	d.network.WaitForTransactionInState(ctx, txHash)
	test.RequireSuccess(t, response, "failed setting asb address")
}

func (d *driver) generateOrbsAccount(t *testing.T) {
	orbsUser, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")
	var orbsUserAddress [20]byte
	copy(orbsUserAddress[:], orbsUser.RawAddress)

	d.orbsUser = orbsUser
}

func (d *driver) generateEthAccountAndAssignFunds(t *testing.T, amount *big.Int) {
	ethContractUserAuth := d.addressInEthereum
	// we don't REALLY care who is the user we transfer from, so for simplicity's sake we use the same mega-user defined when simulator is created
	_, err := d.erc20contract.Transact(d.addressInEthereum, "assign", ethContractUserAuth.From /*address of user*/, amount)
	// generate token in source address
	require.NoError(t, err, "could not assign token to sender")
	d.simulator.Commit()
}

// orbs side of the contract is automatically deployed so this only deploys to Ethereum
func (d *driver) deployAutonomousSwapBridge(t *testing.T) {
	fakeFederation := common.BigToAddress(big.NewInt(1700))

	ethAsbAddress, ethAsbContract, err := d.simulator.DeployEthereumContract(d.addressInEthereum, asbABI, asbByteCode, uint32(0), uint64(builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID), //TODO CHANGE IN LEoNId,
		d.orbsASBContractName, d.erc20address, fakeFederation)
	require.NoError(t, err, "could not deploy asb to Ethereum")
	d.simulator.Commit()
	d.ethASBAddress = ethAsbAddress
	d.ethASBContract = ethAsbContract
}

func (d *driver) deployERC20Contract(t *testing.T) {
	ethTetAddress, ethTetContract, err := d.simulator.DeployEthereumContract(d.addressInEthereum, tetABI, tetByteCode)
	require.NoError(t, err, "could not deploy erc token to Ethereum")
	d.simulator.Commit()
	d.erc20contract = ethTetContract
	d.erc20address = ethTetAddress
}

const tetABI = `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"addedValue","type":"uint256"}],"name":"increaseAllowance","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"subtractedValue","type":"uint256"}],"name":"decreaseAllowance","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"constant":false,"inputs":[{"name":"_account","type":"address"},{"name":"_value","type":"uint256"}],"name":"assign","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
const tetByteCode = "0x608060405234801561001057600080fd5b506111aa806100206000396000f300608060405260043610610099576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063095ea7b31461009e57806318160ddd1461010357806323b872dd1461012e57806339509351146101b357806370a0823114610218578063a457c2d71461026f578063a9059cbb146102d4578063be76048814610339578063dd62ed3e14610386575b600080fd5b3480156100aa57600080fd5b506100e9600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506103fd565b604051808215151515815260200191505060405180910390f35b34801561010f57600080fd5b5061011861052a565b6040518082815260200191505060405180910390f35b34801561013a57600080fd5b50610199600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610534565b604051808215151515815260200191505060405180910390f35b3480156101bf57600080fd5b506101fe600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803590602001909291905050506106e6565b604051808215151515815260200191505060405180910390f35b34801561022457600080fd5b50610259600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061091d565b6040518082815260200191505060405180910390f35b34801561027b57600080fd5b506102ba600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610965565b604051808215151515815260200191505060405180910390f35b3480156102e057600080fd5b5061031f600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610b9c565b604051808215151515815260200191505060405180910390f35b34801561034557600080fd5b50610384600480360381019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919080359060200190929190505050610bb3565b005b34801561039257600080fd5b506103e7600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610bd3565b6040518082815260200191505060405180910390f35b60008073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415151561043a57600080fd5b81600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040518082815260200191505060405180910390a36001905092915050565b6000600254905090565b6000600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000205482111515156105c157600080fd5b61065082600160008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610c5a90919063ffffffff16565b600160008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055506106db848484610c7b565b600190509392505050565b60008073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff161415151561072357600080fd5b6107b282600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610e9490919063ffffffff16565b600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546040518082815260200191505060405180910390a36001905092915050565b60008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b60008073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16141515156109a257600080fd5b610a3182600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610c5a90919063ffffffff16565b600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508273ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925600160003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020546040518082815260200191505060405180910390a36001905092915050565b6000610ba9338484610c7b565b6001905092915050565b610bc582610bc08461091d565b610eb5565b610bcf8282611040565b5050565b6000600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b600080838311151515610c6c57600080fd5b82840390508091505092915050565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548111151515610cc857600080fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1614151515610d0457600080fd5b610d55816000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610c5a90919063ffffffff16565b6000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550610de8816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610e9490919063ffffffff16565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a3505050565b6000808284019050838110151515610eab57600080fd5b8091505092915050565b60008273ffffffffffffffffffffffffffffffffffffffff1614151515610edb57600080fd5b6000808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020548111151515610f2857600080fd5b610f3d81600254610c5a90919063ffffffff16565b600281905550610f94816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610c5a90919063ffffffff16565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a35050565b60008273ffffffffffffffffffffffffffffffffffffffff161415151561106657600080fd5b61107b81600254610e9490919063ffffffff16565b6002819055506110d2816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054610e9490919063ffffffff16565b6000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040518082815260200191505060405180910390a350505600a165627a7a7230582094609698b7c76dff80d05dcaa6cb6f31fb8f800a5026ee62cf6e4f71bba041a40029"

const asbABI = `[{"constant":true,"inputs":[],"name":"orbsASBContractName","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"federation","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"renounceOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"isOwner","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"spentOrbsTuids","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"virtualChainId","outputs":[{"name":"","type":"uint64"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"tuidCounter","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"networkType","outputs":[{"name":"","type":"uint32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"token","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"VERSION","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_networkType","type":"uint32"},{"name":"_virtualChainId","type":"uint64"},{"name":"_orbsASBContractName","type":"string"},{"name":"_token","type":"address"},{"name":"_federation","type":"address"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"bytes20"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredOut","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tuid","type":"uint256"},{"indexed":true,"name":"from","type":"bytes20"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"TransferredIn","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"constant":false,"inputs":[{"name":"_to","type":"bytes20"},{"name":"_value","type":"uint256"}],"name":"transferOut","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
const asbByteCode = "0x6080604052600060045534801561001557600080fd5b50604051610f86380380610f868339810180604052810190808051906020019092919080519060200190929190805182019291906020018051906020019092919080519060200190929190505050336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3600083511115156101be576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260298152602001807f4f7262732041534220636f6e7472616374206e616d65206d757374206e6f742081526020017f626520656d70747921000000000000000000000000000000000000000000000081525060400191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1614151515610263576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260148152602001807f546f6b656e206d757374206e6f7420626520302100000000000000000000000081525060200191505060405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614151515610308576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260198152602001807f46656465726174696f6e206d757374206e6f742062652030210000000000000081525060200191505060405180910390fd5b84600060146101000a81548163ffffffff021916908363ffffffff16021790555083600060186101000a81548167ffffffffffffffff021916908367ffffffffffffffff16021790555082600190805190602001906103689291906103f5565b5081600260006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555080600360006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505050505061049a565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061043657805160ff1916838001178555610464565b82800160010185558215610464579182015b82811115610463578251825591602001919060010190610448565b5b5090506104719190610475565b5090565b61049791905b8082111561049357600081600090555060010161047b565b5090565b90565b610add806104a96000396000f3006080604052600436106100c5576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806333d7fafd146100ca5780635c45428c1461011057806366c86119146101a0578063715018a6146101f75780638da5cb5b1461020e5780638f32d59b14610265578063923aebf014610294578063bd19dffb146102d9578063e1d5c25514610318578063f2fde38b14610343578063f3762c1114610386578063fc0c546a146103bd578063ffa1ad7414610414575b600080fd5b3480156100d657600080fd5b5061010e60048036038101908080356bffffffffffffffffffffffff191690602001909291908035906020019092919050505061043f565b005b34801561011c57600080fd5b506101256106e0565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561016557808201518184015260208101905061014a565b50505050905090810190601f1680156101925780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b3480156101ac57600080fd5b506101b561077e565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561020357600080fd5b5061020c6107a4565b005b34801561021a57600080fd5b50610223610876565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561027157600080fd5b5061027a61089f565b604051808215151515815260200191505060405180910390f35b3480156102a057600080fd5b506102bf600480360381019080803590602001909291905050506108f6565b604051808215151515815260200191505060405180910390f35b3480156102e557600080fd5b506102ee610916565b604051808267ffffffffffffffff1667ffffffffffffffff16815260200191505060405180910390f35b34801561032457600080fd5b5061032d610930565b6040518082815260200191505060405180910390f35b34801561034f57600080fd5b50610384600480360381019080803573ffffffffffffffffffffffffffffffffffffffff169060200190929190505050610936565b005b34801561039257600080fd5b5061039b610955565b604051808263ffffffff1663ffffffff16815260200191505060405180910390f35b3480156103c957600080fd5b506103d261096b565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34801561042057600080fd5b50610429610991565b6040518082815260200191505060405180910390f35b6000811115156104b7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601d8152602001807f56616c7565206d7573742062652067726561746572207468616e20302100000081525060200191505060405180910390fd5b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166323b872dd3330846040518463ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020018281526020019350505050602060405180830381600087803b1580156105b057600080fd5b505af11580156105c4573d6000803e3d6000fd5b505050506040513d60208110156105da57600080fd5b8101908080519060200190929190505050151561065f576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260178152602001807f496e73756666696369656e7420616c6c6f77616e63652100000000000000000081525060200191505060405180910390fd5b610675600160045461099690919063ffffffff16565b600481905550816bffffffffffffffffffffffff19163373ffffffffffffffffffffffffffffffffffffffff166004547fc7d2da8a0df0279cb4e0a81f2975445675cc6527c94016791d29977a1fa0f251846040518082815260200191505060405180910390a45050565b60018054600181600116156101000203166002900480601f0160208091040260200160405190810160405280929190818152602001828054600181600116156101000203166002900480156107765780601f1061074b57610100808354040283529160200191610776565b820191906000526020600020905b81548152906001019060200180831161075957829003601f168201915b505050505081565b600360009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6107ac61089f565b15156107b757600080fd5b600073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a360008060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614905090565b60056020528060005260406000206000915054906101000a900460ff1681565b600060189054906101000a900467ffffffffffffffff1681565b60045481565b61093e61089f565b151561094957600080fd5b610952816109b7565b50565b600060149054906101000a900463ffffffff1681565b600260009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600181565b60008082840190508381101515156109ad57600080fd5b8091505092915050565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16141515156109f357600080fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550505600a165627a7a72305820dc57f244d775b10898e9eb6a77cb0e0d0c0e0c9a6f66c39ac55d2442a523faaa0029"
