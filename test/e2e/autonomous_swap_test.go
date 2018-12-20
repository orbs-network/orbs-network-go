package e2e

import (
	"bufio"
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-client-sdk-go/orbsclient"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/ASBEthereum"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/ERC20Proxy"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"math/big"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAutonomousSwap_EthereumToOrbs(t *testing.T) {

	h := newHarness()
	lt := time.Now()
	printTestTime(t, "started", &lt)

	h.waitUntilTransactionPoolIsReady(t)
	printTestTime(t, "first block committed", &lt)
	d := newAutonomousSwapDriver(h)

	etherAmountBefore := big.NewInt(200)
	amountToTransfer := big.NewInt(55)

	d.generateOrbsAccount(t)

	//	d.deployTokenContractToEthereum(t)
	//	d.generateEthereumAccountAndAssignFunds(t, etherAmountBefore)

	//d.deployAutonomousSwapBridgeToEthereum(t)
	//d.deployASBToEthereumUsingTruffle(t)
	d.deployToEthereumUsingTruffle(t)
	d.etherGetAsbAddress(t)
	d.bindOrbsAutonomousSwapBridgeToEthereum(t)

	d.generateEthereumAccountAndAssignFunds(t, etherAmountBefore)
	d.approveTransferInEthereumTokenContract(t, amountToTransfer)
	transferOutTxHash := d.transferOutFromEthereum(t, amountToTransfer)
	t.Log("Eth tx hash", transferOutTxHash)

	// TODO v1 deploy causes who is owner - important for both.
	d.transferInToOrbs(t, transferOutTxHash)

	balanceAfterTransfer := d.getBalanceInOrbs(t)
	require.EqualValues(t, amountToTransfer.Uint64(), balanceAfterTransfer, "wrong amount of tokens in orbs")

	etherBalanceAfterTransfer := d.getBalanceInEthereum(t)
	require.EqualValues(t, etherAmountBefore.Sub(etherAmountBefore, amountToTransfer).Uint64(), etherBalanceAfterTransfer, "wrong amount of tokens left in ethereum")
}

type ethconfig struct{}

func (e *ethconfig) EthereumEndpoint() string {
	return getConfig().ethereumEndpoint
}

func newAutonomousSwapDriver(h *harness) *driver {
	ethereum := adapter.NewEthereumRpcConnection(&ethconfig{}, log.GetLogger())
	key, err := crypto.HexToECDSA("f2ce3a9eddde6e5d996f6fe7c1882960b0e8ee8d799e0ef608276b8de4dc7f19")
	if err != nil {
		panic(err)
	}
	opts := bind.NewKeyedTransactor(key)
	opts.GasLimit = 1234567890
	//opts.GasPrice = big.NewInt(1)

	return &driver{
		harness:                  h,
		ethereum:                 ethereum,
		addressInEthereum:        opts,
		orbsASBContractName:      asb_ether.CONTRACT_NAME,
		orbsContractOwnerAddress: testKeys.Ed25519KeyPairForTests(5),
	}
}

type driver struct {
	ethereum *adapter.EthereumRpcConnection

	orbsContractOwnerAddress *keys.Ed25519KeyPair
	orbsASBContractName      string
	orbsUser                 *orbsclient.OrbsAccount
	orbsUserAddress          [20]byte
	orbsUserKeyPair          *keys.Ed25519KeyPair

	addressInEthereum *bind.TransactOpts // we use a single address for both the "admin" stuff like deploying the contracts and as our swapping user, so as to simplify setup - otherwise we'll need to create two PKs in the simulator

	erc20contract   *bind.BoundContract
	erc20address    *common.Address
	ethASBAddress   *common.Address
	ethASBAddresHex string
	ethASBContract  *bind.BoundContract
	harness         *harness
}

// orbs side funcs
func (d *driver) generateOrbsAccount(t *testing.T) {
	orbsUser, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")

	copy(d.orbsUserAddress[:], orbsUser.RawAddress)
	d.orbsUser = orbsUser
	d.orbsUserKeyPair = keys.NewEd25519KeyPair(orbsUser.PublicKey, orbsUser.PrivateKey)
}

func (d *driver) generateOrbsFunds(t *testing.T, amount *big.Int) {
	response, _, err := d.harness.sendTransaction(d.orbsContractOwnerAddress, erc20proxy.CONTRACT_NAME, "mint", d.orbsUser.RawAddress, amount.Uint64())
	requireSuccess(t, err, response, "mint transaction")
}

func (d *driver) getBalanceInOrbs(t *testing.T) uint64 {
	address, err := hexutil.Decode("0x3fced656aCBd6700cE7d546f6EFDCDd482D8142a")
	response, err := d.harness.callMethod(d.orbsContractOwnerAddress, erc20proxy.CONTRACT_NAME, "balanceOf", address)
	require.NoError(t, err, "failed sending  to Orbs")
	require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult, "failed getting balance in Orbs")
	return response.OutputArguments[0].(uint64)
}

func (d *driver) approveTransferInOrbsTokenContract(ctx context.Context, t *testing.T, amount *big.Int) {
	response, _, err := d.harness.sendTransaction(d.orbsContractOwnerAddress, erc20proxy.CONTRACT_NAME, "approve", d.addressInEthereum.From, amount.Uint64())
	requireSuccess(t, err, response, "approve transaction")
}

func (d *driver) bindOrbsAutonomousSwapBridgeToEthereum(t *testing.T) {
	response, _, err := d.harness.sendTransaction(d.orbsContractOwnerAddress, d.orbsASBContractName, "setAsbAddr", d.ethASBAddresHex /*d.ethASBAddress.Hex()*/)
	requireSuccess(t, err, response, "setAsbAddr transaction")
}

func (d *driver) transferInToOrbs(t *testing.T, transferOutTxHash string) {
	response, _, err := d.harness.sendTransaction(d.orbsContractOwnerAddress, d.orbsASBContractName, "transferIn", transferOutTxHash)
	requireSuccess(t, err, response, "transferIn in Orbs")
}

func (d *driver) transferOutFromOrbs(ctx context.Context, t *testing.T, amount *big.Int) {
	response, _, err := d.harness.sendTransaction(d.orbsContractOwnerAddress, d.orbsASBContractName, "transferOut", d.addressInEthereum.From.Bytes(), amount.Uint64())
	requireSuccess(t, err, response, "transferOut in Orbs")
}

// Ethereum scripts
func (d *driver) deployToEthereumUsingTruffle(t *testing.T) {
	path := "../../vendor/github.com/orbs-network/orbs-federation/asb"

	cmd := exec.Command("truffle", "compile")
	cmd.Dir = path
	err := cmd.Run()
	require.NoError(t, err, "could not compile")

	cmd = exec.Command("truffle", "migrate")
	cmd.Dir = path
	err = cmd.Run()
	require.NoError(t, err, "could not run deploy script")
}

func (d *driver) etherGetAsbAddress(t *testing.T) {
	output := runTruffleCommand(t, "getasbaddr.js")
	bf := bufio.NewReader(strings.NewReader(string(output)))
	bf.ReadLine()
	bf.ReadLine()
	bytes, _, err := bf.ReadLine()

	require.NoError(t, err, "could not parse address")
	d.ethASBAddresHex = string(bytes)
}

func (d *driver) generateEthereumAccountAndAssignFunds(t *testing.T, amount *big.Int) {
	runTruffleCommand(t, "assign.js")
	//ethContractUserAuth := d.addressInEthereum
	//// we don't REALLY care who is the user we transfer from, so for simplicity's sake we use the same mega-user defined when simulator is created
	//_, err := d.erc20contract.Transact(d.addressInEthereum, "assign", ethContractUserAuth.From /*address of user*/, amount)
	//// generate token in source address
	//require.NoError(t, err, "could not assign token to sender")
}

func (d *driver) getBalanceInEthereum(t *testing.T) uint64 {
	output := runTruffleCommand(t, "getBalance.js")
	bf := bufio.NewReader(strings.NewReader(string(output)))
	bf.ReadLine()
	bf.ReadLine()
	bytes, _, err := bf.ReadLine()
	i, err := strconv.Atoi(string(bytes))
	require.NoError(t, err, "not a number")
	return uint64(i)
	//ethContractUserAuth := d.addressInEthereum
	//// we don't REALLY care who is the user we transfer from, so for simplicity's sake we use the same mega-user defined when simulator is created
	//var (
	//	ret0 = new(*big.Int)
	//)
	//result := ret0
	//err := d.erc20contract.Call(nil, result, "balanceOf", ethContractUserAuth.From /*address of user*/)
	//// generate token in source address
	//require.NoError(t, err, "could not get token balance of user")
	//return (*result).Uint64()
}

func (d *driver) approveTransferInEthereumTokenContract(t *testing.T, amount *big.Int) {
	runTruffleCommand(t, "approve.js")
	//tx, err := d.erc20contract.Transact(d.addressInEthereum, "approve", d.ethASBAddress, amount)
	//require.NoError(t, err, "could not approve transfer")
	//receipt, err := d.ethereum.Receipt(tx.Hash())
	//require.NoError(t, err, "could not get receipt")
	//require.EqualValues(t, types.ReceiptStatusSuccessful, receipt.Status, "call to approve on tet in Ethereum failed")

}

func (d *driver) transferOutFromEthereum(t *testing.T, amount *big.Int) string {
	output := runTruffleCommand(t, "transferOut.js")
	bf := bufio.NewReader(strings.NewReader(string(output)))
	bf.ReadLine()
	bf.ReadLine()
	bytes, _, _ := bf.ReadLine()
	//transferOutTx, err := d.ethASBContract.Transact(d.addressInEthereum, "transferOut", d.orbsUserAddress, amount)
	//require.NoError(t, err, "could not transfer out")
	//
	//receipt, err := d.ethereum.Receipt(transferOutTx.Hash())
	//require.NoError(t, err, "could not get receipt")
	//require.EqualValues(t, types.ReceiptStatusSuccessful, receipt.Status, "call to transferOut on ASB in Ethereum failed")
	t.Log(string(bytes))
	return string(bytes)
}

func requireSuccess(t *testing.T, err error, response *codec.SendTransactionResponse, description string) {
	require.NoError(t, err, "failed sending "+description)
	require.EqualValues(t, string(codec.EXECUTION_RESULT_SUCCESS), string(response.ExecutionResult), description+" execution failed")
}

func runTruffleCommand(t *testing.T, script string) []byte {
	cmd := exec.Command("truffle", "exec", script)
	path := "../../vendor/github.com/orbs-network/orbs-federation/asb"
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "could run command %s output is %s", script, string(output))
	return output
}
