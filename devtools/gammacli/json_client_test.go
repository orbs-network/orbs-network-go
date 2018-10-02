package gammacli

import (
	"encoding/hex"
	"fmt"
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

func matchJSONContentToMatchInMemBuffVersion(t *testing.T, argMemBuffType protocol.MethodArgumentType, jsonArgument *JSONMethodArgument, memBufferPreBuiltArgument *protocol.MethodArgumentBuilder) {
	valueMessage := "Argument of type %s value was not converted properly"

	require.EqualValues(t, jsonArgument.Name, memBufferPreBuiltArgument.Name, "argument name was not converted properly")
	require.EqualValues(t, argMemBuffType, memBufferPreBuiltArgument.Type, "argument type was not converted properly")

	switch argMemBuffType {
	case protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE:
		require.EqualValues(t, jsonArgument.Value, memBufferPreBuiltArgument.StringValue, fmt.Sprintf(valueMessage, "string"))
	case protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE:
		require.EqualValues(t, jsonArgument.Value, memBufferPreBuiltArgument.Uint32Value, fmt.Sprintf(valueMessage, "uint32"))
	case protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE:
		require.EqualValues(t, jsonArgument.Value, memBufferPreBuiltArgument.Uint64Value, fmt.Sprintf(valueMessage, "uint64"))
	case protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE:
		decodedHex, decodeHexError := hex.DecodeString(jsonArgument.Value.(string))
		require.NoError(t, decodeHexError, "Expecting no problems when decoding hex string from argument")
		require.EqualValues(t, decodedHex, memBufferPreBuiltArgument.BytesValue, fmt.Sprintf(valueMessage, "bytes"))
	}
}

func matchJSONContentToMatchInMemBuffBuiltVersion(t *testing.T, argMemBuffType protocol.MethodArgumentType, jsonArgument *JSONMethodArgument, memBufferArgument *protocol.MethodArgument) {
	valueMessage := "Argument of type %s value was not converted properly"

	require.EqualValues(t, jsonArgument.Name, memBufferArgument.Name(), "argument name was not converted properly")
	require.EqualValues(t, argMemBuffType, memBufferArgument.Type(), "argument type was not converted properly")

	switch argMemBuffType {
	case protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE:
		require.EqualValues(t, jsonArgument.Value, memBufferArgument.StringValue(), fmt.Sprintf(valueMessage, "string"))
	case protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE:
		require.EqualValues(t, jsonArgument.Value, memBufferArgument.Uint32Value(), fmt.Sprintf(valueMessage, "uint32"))
	case protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE:
		require.EqualValues(t, jsonArgument.Value, memBufferArgument.Uint64Value(), fmt.Sprintf(valueMessage, "uint64"))
	case protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE:
		decodedHex, decodeHexError := hex.DecodeString(jsonArgument.Value.(string))
		require.NoError(t, decodeHexError, "Expecting no problems when decoding hex string from argument")
		require.EqualValues(t, decodedHex, memBufferArgument.BytesValue(), fmt.Sprintf(valueMessage, "bytes"))
	}
}

func TestConvertJSONMethodArgumentsToMemBuff(t *testing.T) {
	argString := JSONMethodArgument{
		Name:  "argString",
		Type:  "string",
		Value: "foo",
	}

	argUInt64 := JSONMethodArgument{
		Name:  "argInt64",
		Type:  "uint64",
		Value: float64(291288), // We choose float64 here since that's the realistic case of when we receive a JSON converted
		// To a Go struct as it's taking numbers always into float64 by default.
	}

	argUInt32 := JSONMethodArgument{
		Name:  "argInt32",
		Type:  "uint32",
		Value: float64(1234),
	}

	hexString := "74686973206973206120776f6e64657266756c20686578207465737421" // this is a wonderful hex test!

	argBytes := JSONMethodArgument{
		Name:  "argBytes",
		Type:  "bytes",
		Value: hexString,
	}

	input := []JSONMethodArgument{argString, argUInt64, argUInt32, argBytes}

	result, err := convertJSONMethodArgumentsToMemBuff(input)
	require.NoError(t, err, "Expected no error from convertJSONMethodArgumentsToMemBuff()")
	require.Len(t, input, len(result), "Expecting the same amount of arguments back after the operation is done")

	for _, arg := range result {
		require.IsType(t, &protocol.MethodArgumentBuilder{}, arg)
	}

	matchJSONContentToMatchInMemBuffVersion(t, protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, &argString, result[0])
	matchJSONContentToMatchInMemBuffVersion(t, protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, &argUInt64, result[1])
	matchJSONContentToMatchInMemBuffVersion(t, protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE, &argUInt32, result[2])
	matchJSONContentToMatchInMemBuffVersion(t, protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE, &argBytes, result[3])
}

func TestConvertTransactionWithArgumentsPostTransactionBuild(t *testing.T) {
	argString := JSONMethodArgument{
		Name:  "argString",
		Type:  "string",
		Value: "foo",
	}

	argUInt64 := JSONMethodArgument{
		Name:  "argInt64",
		Type:  "uint64",
		Value: float64(291288), // We choose float64 here since that's the realistic case of when we receive a JSON converted
		// To a Go struct as it's taking numbers always into float64 by default.
	}

	argUInt32 := JSONMethodArgument{
		Name:  "argInt32",
		Type:  "uint32",
		Value: float64(1234),
	}

	hexString := "74686973206973206120776f6e64657266756c20686578207465737421" // this is a wonderful hex test!

	argBytes := JSONMethodArgument{
		Name:  "argBytes",
		Type:  "bytes",
		Value: hexString,
	}

	req := &JSONTransaction{
		ContractName: "contract",
		MethodName:   "method",
		Arguments:    []JSONMethodArgument{argString, argUInt64, argUInt32, argBytes},
	}

	txb, err := ConvertJSONTransactionToMemBuff(req)
	require.NoError(t, err, "Expected no error from convertTransaction")
	tx := txb.Build()

	require.True(t, tx.IsValid(), "created binary is invalid")
	require.Equal(t, primitives.ProtocolVersion(1), tx.ProtocolVersion(), "protocol version mismatch")
	require.EqualValues(t, req.ContractName, tx.ContractName(), "contract name was not converted properly")
	require.EqualValues(t, req.MethodName, tx.MethodName(), "method name was not converted properly")
	require.Len(t, req.Arguments, 4, "argument slice was not converted properly")

	inputArgsIterator := builders.TransactionInputArgumentsParse(tx)
	builtStringArgument := inputArgsIterator.NextArguments()

	matchJSONContentToMatchInMemBuffBuiltVersion(t, protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, &argString, builtStringArgument)

	builtUInt64Argument := inputArgsIterator.NextArguments()
	matchJSONContentToMatchInMemBuffBuiltVersion(t, protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, &argUInt64, builtUInt64Argument)

	builtUInt32Argument := inputArgsIterator.NextArguments()
	matchJSONContentToMatchInMemBuffBuiltVersion(t, protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE, &argUInt32, builtUInt32Argument)

	builtBytesArgument := inputArgsIterator.NextArguments()
	matchJSONContentToMatchInMemBuffBuiltVersion(t, protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE, &argBytes, builtBytesArgument)
}

func TestConvertAndSignTransaction(t *testing.T) {
	keyPair := keys.Ed25519KeyPairForTests(1)

	arg := JSONMethodArgument{
		Name:  "arg1",
		Type:  "string",
		Value: "foo",
	}

	req := &JSONTransaction{
		ContractName: "contract",
		MethodName:   "method",
		Arguments:    []JSONMethodArgument{arg},
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
	require.EqualValues(t, "uint64", actualArg.Type, "argument type mismatched")
	require.EqualValues(t, uint64(200), actualArg.Value, "argument value mismatched")
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
		CallMethodResult:    protocol.EXECUTION_RESULT_SUCCESS,
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
	require.EqualValues(t, "string", actualArg.Type, "argument type mismatched")
	require.EqualValues(t, "bar", actualArg.Value, "argument value mismatched")
}

//TODO dedup from virtual machine (extract to crypto package?)
func verifyEd25519Signer(signedTransaction *protocol.SignedTransaction) bool {
	signerPublicKey := signedTransaction.Transaction().Signer().Eddsa().SignerPublicKey()
	txHash := digest.CalcTxHash(signedTransaction.Transaction())
	return signature.VerifyEd25519(signerPublicKey, txHash, signedTransaction.Signature())
}
