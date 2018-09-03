package jsonapi

import (
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestJsonMarshallUnmarshallMethodArgument(t *testing.T) {

	arg := &MethodArgument{
		Name:        "arg1",
		Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
		Uint64Value: 1,
		Uint32Value: 2,
		StringValue: "3",
		BytesValue:  []byte("foobar"),
	}

	jsonBytes, err := json.Marshal(arg)
	require.NoError(t, err, "failed to marshall MethodArgument to json")
	t.Log(string(jsonBytes))

	unMarshalledArg := &MethodArgument{}
	require.NoError(t, json.Unmarshal(jsonBytes, unMarshalledArg), "failed to unmarshall json to MethodArgument")

	test.RequireCmpEqual(t, arg, unMarshalledArg, "unmarshalled arg is different than original")
}

func TestJsonMarshallUnmarshallSendTransactionRequest(t *testing.T) {

	arg := MethodArgument{
		Name:        "arg1",
		Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
		Uint64Value: 1,
		Uint32Value: 2,
		StringValue: "3",
		BytesValue:  []byte("foobar"),
	}

	req := &Transaction{
		ContractName: "contract",
		MethodName:   "method",
		Arguments:    []MethodArgument{arg},
	}

	jsonBytes, err := json.Marshal(&req)
	require.NoError(t, err, "failed to marshall SignedTransactionRequest to json")
	t.Log(string(jsonBytes))

	unMarshalledReq := &Transaction{}
	require.NoError(t, json.Unmarshal(jsonBytes, unMarshalledReq), "failed to unmarshall json to SignedTransactionRequest")

	test.RequireCmpEqual(t, req, unMarshalledReq, "unmarshalled arg is different than original")
}

func ExampleJsonApi() {
	arg := MethodArgument{
		Name:        "arg1",
		Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
		Uint64Value: 1,
		Uint32Value: 2,
		StringValue: "3",
		BytesValue:  []byte("foobar"),
	}

	req := &Transaction{
		ContractName: "contract",
		MethodName:   "method",
		Arguments:    []MethodArgument{arg},
	}

	jsonBytes, _ := json.Marshal(&req)
	fmt.Println(string(jsonBytes))
	// Output: {"ContractName":"contract","MethodName":"method","Arguments":[{"Name":"arg1","Type":2,"Uint32Value":2,"Uint64Value":1,"StringValue":"3","BytesValue":"Zm9vYmFy"}]}
}
