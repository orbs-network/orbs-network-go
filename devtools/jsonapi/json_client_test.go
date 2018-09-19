package jsonapi

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

//TODO test method argument conversion for uint32, uint64, byte array

func TestConvertTransaction(t *testing.T) {
	arg := MethodArgument{
		Name:        "arg1",
		Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
		StringValue: "foo",
	}

	req := &Transaction{
		ContractName: "contract",
		MethodName:   "method",
		Arguments:    []MethodArgument{arg},
	}

	txb := ConvertTransaction(req)
	tx := txb.Build()

	require.True(t, tx.IsValid(), "created binary is invalid")
	require.Equal(t, primitives.ProtocolVersion(1), tx.ProtocolVersion(), "protocol version mismatch")
	require.EqualValues(t, req.ContractName, tx.ContractName(), "contract name was not converted properly")
	require.EqualValues(t, req.MethodName, tx.MethodName(), "method name was not converted properly")
	require.Len(t, req.Arguments, 1, "argument slice was not converted properly")

	inputArgsIterator := builders.TransactionInputArgumentsParse(tx)
	arg1 := inputArgsIterator.NextArguments()
	require.EqualValues(t, req.Arguments[0].Name, arg1.Name(), "argument name was not converted properly")
	require.EqualValues(t, req.Arguments[0].Type, arg1.Type(), "argument type was not converted properly")
	require.EqualValues(t, req.Arguments[0].StringValue, arg1.StringValue(), "argument string value was not converted properly")
}

func TestConvertAndSignTransaction(t *testing.T) {
	keyPair := keys.Ed25519KeyPairForTests(1)

	arg := MethodArgument{
		Name:        "arg1",
		Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
		StringValue: "foo",
	}

	req := &Transaction{
		ContractName: "contract",
		MethodName:   "method",
		Arguments:    []MethodArgument{arg},
	}

	stxb, err := ConvertAndSignTransaction(req, keyPair)
	require.NoError(t, err, "got an unexpected error")
	stx := stxb.Build()

	require.True(t, stx.IsValid(), "created binary is invalid")
	require.True(t, verifyEd25519Signer(stx), "transaction was not signed properly")
}

func TestConvertSendTransactionOutput(t *testing.T) {
	outputArgumentArray := (&protocol.MethodArgumentArrayBuilder{
		Arguments: []*protocol.MethodArgumentBuilder{
			{
				Name:        "foo",
				Type:        protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE,
				Uint64Value: 200,
			},
		},
	}).Build()
	sto := (&client.SendTransactionResponseBuilder{
		BlockHeight:       4,
		BlockTimestamp:    primitives.TimestampNano(time.Now().UnixNano()),
		TransactionStatus: protocol.TRANSACTION_STATUS_COMMITTED,
		TransactionReceipt: &protocol.TransactionReceiptBuilder{
			Txhash:              []byte("foo"),
			ExecutionResult:     protocol.EXECUTION_RESULT_SUCCESS,
			OutputArgumentArray: outputArgumentArray.RawArgumentsArray(),
		},
	}).Build()

	out := ConvertSendTransactionOutput(sto)

	require.EqualValues(t, sto.BlockHeight(), out.BlockHeight, "block height mismatched")
	require.EqualValues(t, sto.BlockTimestamp(), out.BlockTimestamp, "block timestamp mismatched")
	require.EqualValues(t, sto.TransactionStatus(), out.TransactionStatus, "transaction status mismatched")
	require.EqualValues(t, sto.TransactionReceipt().Txhash(), out.TransactionReceipt.Txhash, "transaction hash mismatched")
	require.EqualValues(t, sto.TransactionReceipt().ExecutionResult(), out.TransactionReceipt.ExecutionResult, "execution result mismatched")
	require.Len(t, out.TransactionReceipt.OutputArguments, 1, "expected exactly 1 output argument")

	outputArgsIterator := builders.TransactionReceiptOutputArgumentsParse(sto.TransactionReceipt())
	expectedArg := outputArgsIterator.NextArguments()
	actualArg := out.TransactionReceipt.OutputArguments[0]
	require.EqualValues(t, expectedArg.Name(), actualArg.Name, "argument name mismatched")
	require.EqualValues(t, expectedArg.Type(), actualArg.Type, "argument type mismatched")
	require.EqualValues(t, expectedArg.Uint64Value(), actualArg.Uint64Value, "argument value mismatched")

}

func TestConvertCallMethodOutput(t *testing.T) {
	outputArgumentArray := (&protocol.MethodArgumentArrayBuilder{
		Arguments: []*protocol.MethodArgumentBuilder{
			{
				Name:        "foo",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: "bar",
			},
		},
	}).Build()
	cmo := (&client.CallMethodResponseBuilder{
		BlockHeight:         4,
		BlockTimestamp:      primitives.TimestampNano(time.Now().UnixNano()),
		CallMethodResult:          protocol.EXECUTION_RESULT_SUCCESS,
		OutputArgumentArray: outputArgumentArray.RawArgumentsArray(),
	}).Build()

	out := ConvertCallMethodOutput(cmo)

	require.EqualValues(t, cmo.BlockHeight(), out.BlockHeight, "block height mismatched")
	require.EqualValues(t, cmo.BlockTimestamp(), out.BlockTimestamp, "block timestamp mismatched")
	require.EqualValues(t, cmo.CallMethodResult(), out.CallResult, "call result mismatched")
	require.Len(t, out.OutputArguments, 1, "expected exactly 1 output argument")

	outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(cmo)
	expectedArg := outputArgsIterator.NextArguments()
	actualArg := out.OutputArguments[0]
	require.EqualValues(t, expectedArg.Name(), actualArg.Name, "argument name mismatched")
	require.EqualValues(t, expectedArg.Type(), actualArg.Type, "argument type mismatched")
	require.EqualValues(t, expectedArg.StringValue(), actualArg.StringValue, "argument value mismatched")
}

//TODO dedup from virtual machine (extract to crypto package?)
func verifyEd25519Signer(signedTransaction *protocol.SignedTransaction) bool {
	signerPublicKey := signedTransaction.Transaction().Signer().Eddsa().SignerPublicKey()
	txHash := digest.CalcTxHash(signedTransaction.Transaction())
	return signature.VerifyEd25519(signerPublicKey, txHash, signedTransaction.Signature())
}
