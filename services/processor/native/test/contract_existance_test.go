package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCallWithUnknownContractFails(t *testing.T) {
	h := newHarness()
	input := processCallInput().WithUnknownContract().Build()
	h.expectSdkCallMadeWithServiceCallMethod(deployments.CONTRACT.Name, deployments.METHOD_GET_CODE.Name, builders.MethodArgumentsArray(string(input.ContractName)), errors.New("code not found error"))

	_, err := h.service.ProcessCall(input)
	require.Error(t, err, "call should fail")

	h.verifySdkCallMade(t)
}

func TestGetContractInfoWithUnknownContractFails(t *testing.T) {
	h := newHarness()
	input := getContractInfoInput().WithUnknownContract().Build()
	h.expectSdkCallMadeWithServiceCallMethod(deployments.CONTRACT.Name, deployments.METHOD_GET_CODE.Name, builders.MethodArgumentsArray(string(input.ContractName)), errors.New("code not found error"))

	_, err := h.service.GetContractInfo(input)
	require.Error(t, err, "GetContractInfo should fail")

	h.verifySdkCallMade(t)
}
