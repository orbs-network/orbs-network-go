package jsonapi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const METHOD_ARGUMENT_TYPE_UINT32 string = "uint32"
const METHOD_ARGUMENT_TYPE_UINT64 string = "uint64"
const METHOD_ARGUMENT_TYPE_STRING string = "string"
const METHOD_ARGUMENT_TYPE_BYTES string = "bytes"

func ConvertAndSignTransaction(tx *Transaction, keyPair *keys.Ed25519KeyPair) (*protocol.SignedTransactionBuilder, error) {
	transaction, err := ConvertTransaction(tx)
	if err != nil { // Something in the JSON is not valid so we exit with a non zero exit code.
		fmt.Println(err)
		os.Exit(1)
	}
	transaction.Signer = &protocol.SignerBuilder{
		Scheme: protocol.SIGNER_SCHEME_EDDSA, //TODO move to Transaction
		Eddsa: &protocol.EdDSA01SignerBuilder{
			NetworkType:     protocol.NETWORK_TYPE_TEST_NET, //TODO move to Transaction
			SignerPublicKey: primitives.Ed25519PublicKey(keyPair.PublicKey()),
		},
	}

	signedTransaction := &protocol.SignedTransactionBuilder{
		Transaction: transaction,
	}

	txHash := digest.CalcTxHash(transaction.Build())
	if sig, err := signature.SignEd25519(keyPair.PrivateKey(), txHash); err != nil {
		return nil, err
	} else {
		signedTransaction.Signature = sig
		return signedTransaction, nil
	}
}

func ConvertTransaction(tx *Transaction) (*protocol.TransactionBuilder, error) {
	var inputArguments []*protocol.MethodArgumentBuilder
	for _, arg := range tx.Arguments {
		switch arg.Type {
		case METHOD_ARGUMENT_TYPE_UINT32:
			inputArguments = append(inputArguments, &protocol.MethodArgumentBuilder{
				Name: arg.Name, Type: protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.Value.(uint32),
			})
		case METHOD_ARGUMENT_TYPE_UINT64:
			inputArguments = append(inputArguments, &protocol.MethodArgumentBuilder{
				Name: arg.Name, Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.Value.(uint64),
			})
		case METHOD_ARGUMENT_TYPE_STRING:
			inputArguments = append(inputArguments, &protocol.MethodArgumentBuilder{
				Name: arg.Name, Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.Value.(string),
			})
		case METHOD_ARGUMENT_TYPE_BYTES:
			argBytesValue, err := hex.DecodeString(arg.Value.(string))
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("Could not decode hex string for argument %s value", arg.Name))
			}

			inputArguments = append(inputArguments, &protocol.MethodArgumentBuilder{
				Name: arg.Name, Type: protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE, BytesValue: argBytesValue,
			})
		}
	}
	inputArgumentArray := (&protocol.MethodArgumentArrayBuilder{Arguments: inputArguments}).Build()

	return &protocol.TransactionBuilder{
		ProtocolVersion:    1,
		VirtualChainId:     builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID, //TODO move to Transaction
		ContractName:       primitives.ContractName(tx.ContractName),
		MethodName:         primitives.MethodName(tx.MethodName),
		Timestamp:          primitives.TimestampNano(time.Now().UnixNano()),
		InputArgumentArray: inputArgumentArray.RawArgumentsArray(),
	}, nil

}

func ConvertSendTransactionOutput(sto *client.SendTransactionResponse) *SendTransactionOutput {
	outputArgsIterator := builders.TransactionReceiptOutputArgumentsParse(sto.TransactionReceipt())
	var outputArguments []MethodArgument
	for iter := outputArgsIterator; iter.HasNext(); {
		arg := iter.NextArguments()
		methodArg := convertMethodArgument(arg)
		outputArguments = append(outputArguments, methodArg)
	}

	return &SendTransactionOutput{
		BlockHeight:       sto.BlockHeight(),
		BlockTimestamp:    sto.BlockTimestamp(),
		TransactionStatus: sto.TransactionStatus(),
		TransactionReceipt: TransactionReceipt{
			Txhash:          sto.TransactionReceipt().Txhash(),
			ExecutionResult: sto.TransactionReceipt().ExecutionResult(),
			OutputArguments: outputArguments,
		},
	}
}

func ConvertCallMethodOutput(cmo *client.CallMethodResponse) *CallMethodOutput {
	outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsDecode(cmo)
	var outputArguments []MethodArgument
	for iter := outputArgsIterator; iter.HasNext(); {
		arg := iter.NextArguments()
		methodArg := convertMethodArgument(arg)
		outputArguments = append(outputArguments, methodArg)
	}

	return &CallMethodOutput{
		BlockHeight:     cmo.BlockHeight(),
		BlockTimestamp:  cmo.BlockTimestamp(),
		CallResult:      cmo.CallResult(),
		OutputArguments: outputArguments,
	}
}

func convertMethodArgument(arg *protocol.MethodArgument) MethodArgument {
	methodArg := MethodArgument{
		Name: arg.Name(),
	}
	switch arg.Type() {
	case protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE:
		methodArg.Type = METHOD_ARGUMENT_TYPE_UINT64
		methodArg.Value = arg.Uint64Value()
	case protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE:
		methodArg.Type = METHOD_ARGUMENT_TYPE_UINT32
		methodArg.Value = arg.Uint32Value()
	case protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE:
		methodArg.Type = METHOD_ARGUMENT_TYPE_STRING
		methodArg.Value = arg.StringValue()
	case protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE:
		argValueEncodedToString := hex.EncodeToString(arg.BytesValue())
		methodArg.Type = METHOD_ARGUMENT_TYPE_BYTES
		methodArg.Value = argValueEncodedToString
	}
	return methodArg
}

func SendTransaction(transferJson *Transaction, keyPair *keys.Ed25519KeyPair, serverUrl string, logVerbose bool) (*SendTransactionOutput, error) {
	tx, err := ConvertAndSignTransaction(transferJson, keyPair)

	if logVerbose {
		log.GetLogger().Info("sending transaction", log.Stringable("transaction", tx.Build()))
	}

	sendTransactionRequest := (&client.SendTransactionRequestBuilder{SignedTransaction: tx}).Build()
	res, err := http.Post(serverUrl+"/api/send-transaction", "application/octet-stream", bytes.NewReader(sendTransactionRequest.Raw()))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got unexpected http status code %s", res.StatusCode)
	}

	readBytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	return ConvertSendTransactionOutput(client.SendTransactionResponseReader(readBytes)), err
}

func CallMethod(transferJson *Transaction, serverUrl string, logVerbose bool) (*CallMethodOutput, error) {
	tx, err := ConvertTransaction(transferJson)
	if err != nil { // The JSON we got is probably invalid so we exit
		fmt.Println(err)
		os.Exit(1)
	}

	if logVerbose {
		log.GetLogger().Info("calling method", log.Stringable("transaction", tx.Build()))
	}

	request := (&client.CallMethodRequestBuilder{Transaction: tx}).Build()
	res, err := http.Post(serverUrl+"/api/call-method", "application/octet-stream", bytes.NewReader(request.Raw()))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got unexpected http status code %s", res.StatusCode)
	}

	readBytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	return ConvertCallMethodOutput(client.CallMethodResponseReader(readBytes)), err
}
