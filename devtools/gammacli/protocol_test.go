package gammacli

import (
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestJsonMarshallUnmarshallMethodArgument(t *testing.T) {
	arg := &JSONMethodArgument{
		Name:  "arg1",
		Type:  "uint64",
		Value: uint64(42324),
	}

	jsonBytes, err := json.Marshal(arg)
	require.NoError(t, err, "failed to marshall JSONMethodArgument to json")
	t.Log(string(jsonBytes))

	unMarshalledArg := &JSONMethodArgument{}
	require.NoError(t, json.Unmarshal(jsonBytes, unMarshalledArg), "failed to unmarshall json to JSONMethodArgument")

	unMarshalledArg.Value = uint64(unMarshalledArg.Value.(float64))

	test.RequireCmpEqual(t, arg, unMarshalledArg, "unmarshalled arg is different than original")
}

func TestJsonMarshallUnmarshallSendTransactionRequest(t *testing.T) {
	arg := JSONMethodArgument{
		Name:  "arg1",
		Type:  "string",
		Value: "this is a test string",
	}

	req := &JSONTransaction{
		ContractName: "contract",
		MethodName:   "method",
		Arguments:    []JSONMethodArgument{arg},
	}

	jsonBytes, err := json.Marshal(&req)
	require.NoError(t, err, "failed to marshall SignedTransactionRequest to json")
	t.Log(string(jsonBytes))

	unMarshalledReq := &JSONTransaction{}
	require.NoError(t, json.Unmarshal(jsonBytes, unMarshalledReq), "failed to unmarshall json to SignedTransactionRequest")

	test.RequireCmpEqual(t, req, unMarshalledReq, "unmarshalled arg is different than original")
}
