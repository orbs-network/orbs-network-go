package javascript

import (
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/services/processor"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type defaultProcessor struct {
}

func (defaultProcessor) ProcessMethodCall(executionContextId primitives.ExecutionContextId, code string, methodName primitives.MethodName, args *protocol.ArgumentArray) (contractOutputArgs *protocol.ArgumentArray, contractOutputErr error, err error) {
	return nil, nil, errors.New("JS processor is not implemented")
}

func DefaultWorker(handler sdkContext.SdkHandler) processor.StatelessProcessor {
	return &defaultProcessor{}
}
