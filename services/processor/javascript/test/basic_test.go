package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/services/processor/javascript"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestService(t *testing.T) {
	logger := log.DefaultTestingLogger(t)
	processor := javascript.NewJavaScriptProcessor(logger)

	mockVm := &services.MockVirtualMachine{}
	processor.RegisterContractSdkCallHandler(mockVm)

	mockVm.When("HandleSdkCall", mock.Any, mock.Any, mock.Any, mock.Any)

	out, err := processor.ProcessCall(context.TODO(), &services.ProcessCallInput{
		ContextId:              []byte("test"),
		ContractName:           "Hello",
		MethodName:             "hello",
		AccessScope:            protocol.ACCESS_SCOPE_READ_WRITE,
		CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
		InputArgumentArray:     protocol.ArgumentArrayReader(nil),
	})

	require.NoError(t, err)
	fmt.Println(out)
}
