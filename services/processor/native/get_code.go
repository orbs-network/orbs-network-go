package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type contractInstance struct {
	publicMethods map[string]interface{}
	systemMethods map[string]interface{}
}

func extractMethodName(fullPackageName string) string {
	parts := strings.Split(fullPackageName, ".")
	if len(parts) == 0 {
		return ""
	} else {
		return parts[len(parts)-1]
	}
}

func initializeContractInstance(contractInfo *sdkContext.ContractInfo) (*contractInstance, error) {
	res := &contractInstance{
		publicMethods: make(map[string]interface{}),
		systemMethods: make(map[string]interface{}),
	}
	for _, method := range contractInfo.PublicMethods {
		v := reflect.ValueOf(method)
		if v.Kind() != reflect.Func {
			return nil, errors.New("public method is not a valid func")
		}
		name := extractMethodName(runtime.FuncForPC(v.Pointer()).Name())
		res.publicMethods[name] = method
	}
	for _, method := range contractInfo.SystemMethods {
		v := reflect.ValueOf(method)
		if v.Kind() != reflect.Func {
			return nil, errors.New("system method is not a valid func")
		}
		name := extractMethodName(runtime.FuncForPC(v.Pointer()).Name())
		res.systemMethods[name] = method
	}
	return res, nil
}

func initializePreBuiltContractInstances() map[string]*contractInstance {
	res := make(map[string]*contractInstance)
	for contractName, contractInfo := range repository.PreBuiltContracts {
		instance, err := initializeContractInstance(contractInfo)
		if err == nil {
			res[contractName] = instance
		}
	}
	return res
}

func (s *service) retrieveContractInfo(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	// 1. try pre-built repository
	contractInfo, found := repository.PreBuiltContracts[contractName]
	if found {
		return contractInfo, nil
	}

	// 2. try deployed artifact cache (if already compiled)
	contractInfo = s.getDeployedContractInfoFromCache(contractName)
	if contractInfo != nil {
		return contractInfo, nil
	}

	// 3. try deployable code from state (if not yet compiled)
	return s.retrieveDeployedContractInfoFromState(ctx, executionContextId, contractName)
}

func (s *service) retrieveDeployedContractInfoFromState(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	start := time.Now()

	codeBytes, err := s.callGetCodeOfDeploymentSystemContract(ctx, executionContextId, contractName)
	if err != nil {
		return nil, err
	}

	code, err := sanitizeDeployedSourceCode(string(codeBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "source code for contract '%s' failed security sandbox audit", contractName)
	}

	// TODO: replace with given wrapped given context
	ctx, cancel := context.WithTimeout(context.Background(), adapter.MAX_COMPILATION_TIME)
	defer cancel()

	newContractInfo, err := s.compiler.Compile(ctx, code)
	if err != nil {
		return nil, errors.Wrapf(err, "compilation of deployable contract '%s' failed", contractName)
	}
	if newContractInfo == nil {
		return nil, errors.Errorf("compilation and load of deployable contract '%s' did not return a valid symbol", contractName)
	}

	instance, err := initializeContractInstance(newContractInfo)
	if err != nil {
		return nil, errors.Errorf("instance initialization of deployable contract '%s' failed", contractName)
	}
	s.addContractInstance(contractName, instance)
	s.addDeployedContractInfoToCache(contractName, newContractInfo) // must add after instance to avoid race (when somebody RunsMethod at same time)

	s.logger.Info("compiled and loaded deployable contract successfully", log.String("contract", contractName))

	s.metrics.deployedContracts.Inc()
	s.metrics.contractCompilationTime.RecordSince(start)
	// only want to log meter on success (so this line is not under defer)

	return newContractInfo, nil
}

func (s *service) callGetCodeOfDeploymentSystemContract(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) ([]byte, error) {
	systemContractName := primitives.ContractName(deployments_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(deployments_systemcontract.METHOD_GET_CODE)

	output, err := s.sdkHandler.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_SERVICE,
		MethodName:    "callMethod",
		InputArguments: []*protocol.MethodArgument{
			(&protocol.MethodArgumentBuilder{
				Name:        "serviceName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(systemContractName),
			}).Build(),
			(&protocol.MethodArgumentBuilder{
				Name:        "methodName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(systemMethodName),
			}).Build(),
			(&protocol.MethodArgumentBuilder{
				Name:       "inputArgs",
				Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: argsToMethodArgumentArray(string(contractName)).Raw(),
			}).Build(),
		},
		PermissionScope: protocol.PERMISSION_SCOPE_SYSTEM,
	})
	if err != nil {
		return nil, err
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		return nil, errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	methodArgumentArray := protocol.MethodArgumentArrayReader(output.OutputArguments[0].BytesValue())
	argIterator := methodArgumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return nil, errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeBytesValue() {
		return nil, errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	return arg0.BytesValue(), nil
}
