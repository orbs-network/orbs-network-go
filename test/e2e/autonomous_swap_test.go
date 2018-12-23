package e2e

import (
	"bufio"
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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

	etherAmountBefore := 150
	amountToTransfer := 65

	d.generateOrbsAccount(t)

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
	require.EqualValues(t, amountToTransfer, balanceAfterTransfer, "wrong amount of tokens in orbs")
	t.Log(balanceAfterTransfer)
	etherBalanceAfterTransfer := d.getBalanceInEthereum(t)
	require.EqualValues(t, etherAmountBefore - amountToTransfer, etherBalanceAfterTransfer, "wrong amount of tokens left in ethereum")
	t.Log(etherBalanceAfterTransfer)
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
		ethUserAddressHex:        "0x44AA79091FAD956d12086C5Ee782DDf3A8124549",
		orbsASBContractName:      asb_ether.CONTRACT_NAME,
		orbsContractOwnerAddress: testKeys.Ed25519KeyPairForTests(5),
	}
}

type driver struct {
	ethereum *adapter.EthereumRpcConnection

	orbsContractOwnerAddress *keys.Ed25519KeyPair
	orbsASBContractName      string
	orbsUser                 *orbsclient.OrbsAccount
	orbsUserAddressHex       string

	addressInEthereum *bind.TransactOpts // we use a single address for both the "admin" stuff like deploying the contracts and as our swapping user, so as to simplify setup - otherwise we'll need to create two PKs in the simulator
	ethASBAddressHex string
	ethUserAddressHex string

	harness         *harness
}

// orbs side funcs
func (d *driver) generateOrbsAccount(t *testing.T) {
	orbsUser, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")

	d.orbsUserAddressHex = hexutil.Encode(orbsUser.RawAddress)
	d.orbsUser = orbsUser
}

func (d *driver) generateOrbsFunds(t *testing.T, amount *big.Int) {
	response, _, err := d.harness.sendTransaction(d.orbsContractOwnerAddress, erc20proxy.CONTRACT_NAME, "mint", d.orbsUser.RawAddress, amount.Uint64())
	requireSuccess(t, err, response, "mint transaction")
}

func (d *driver) getBalanceInOrbs(t *testing.T) uint64 {
	response, err := d.harness.callMethod(d.orbsContractOwnerAddress, erc20proxy.CONTRACT_NAME, "balanceOf", d.orbsUser.RawAddress)
	require.NoError(t, err, "failed sending  to Orbs")
	require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult, "failed getting balance in Orbs")
	return response.OutputArguments[0].(uint64)
}

func (d *driver) approveTransferInOrbsTokenContract(ctx context.Context, t *testing.T, amount *big.Int) {
	response, _, err := d.harness.sendTransaction(d.orbsContractOwnerAddress, erc20proxy.CONTRACT_NAME, "approve", d.addressInEthereum.From, amount.Uint64())
	requireSuccess(t, err, response, "approve transaction")
}

func (d *driver) bindOrbsAutonomousSwapBridgeToEthereum(t *testing.T) {
	response, _, err := d.harness.sendTransaction(d.orbsContractOwnerAddress, d.orbsASBContractName, "setAsbAddr", d.ethASBAddressHex /*d.ethASBAddress.Hex()*/)
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

	cmd = exec.Command("truffle", "migrate", "--reset")
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
	d.ethASBAddressHex = string(bytes)
}

func (d *driver) generateEthereumAccountAndAssignFunds(t *testing.T, amount int) {
	output := runTruffleCommand(t, "assign.js", d.ethUserAddressHex, strconv.Itoa(amount))
	t.Log("YYY" + string(output))
}

func (d *driver) getBalanceInEthereum(t *testing.T) int {
	output := runTruffleCommand(t, "getBalance.js", d.ethUserAddressHex)
	bf := bufio.NewReader(strings.NewReader(string(output)))
	bf.ReadLine()
	bf.ReadLine()
	bytes, _, err := bf.ReadLine()
	i, err := strconv.Atoi(string(bytes))
	require.NoError(t, err, "not a number")
	return i
}

func (d *driver) approveTransferInEthereumTokenContract(t *testing.T, amount int) {
	runTruffleCommand(t, "approve.js", d.ethUserAddressHex, strconv.Itoa(amount))
}

func (d *driver) transferOutFromEthereum(t *testing.T, amount int) string {
	output := runTruffleCommand(t, "transferOut.js", d.ethUserAddressHex, d.orbsUserAddressHex, strconv.Itoa(amount))
	bf := bufio.NewReader(strings.NewReader(string(output)))
	bf.ReadLine()
	bf.ReadLine()
	bytes, _, _ := bf.ReadLine()
	require.True(t, len(bytes) > 50, "missing a legal tx hash")
	return string(bytes)
}

func requireSuccess(t *testing.T, err error, response *codec.SendTransactionResponse, description string) {
	require.NoError(t, err, "failed sending "+description)
	require.EqualValues(t, string(codec.EXECUTION_RESULT_SUCCESS), string(response.ExecutionResult), description+" execution failed")
}

func runTruffleCommand(t *testing.T, script string, args ...string) []byte {
	cmd := exec.Command("truffle", append([]string{"exec", script}, args...)...)
	path := "../../vendor/github.com/orbs-network/orbs-federation/asb"
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "could run command %s output is %s", script, string(output))
	return output
}
