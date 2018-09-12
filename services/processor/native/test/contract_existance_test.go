package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCallWithUnknownContractFails(t *testing.T) {
	h := newHarness()
	h.expectSdkCallMadeWithServiceCallMethod(deployments.CONTRACT.Name, deployments.METHOD_GET_CODE.Name, errors.New("code not found error"))

	input := processCallInput().WithUnknownContract().Build()
	_, err := h.service.ProcessCall(input)
	require.Error(t, err, "call should fail")

	h.verifySdkCallMade(t)
}

func TestGetContractInfoWithUnknownContractFails(t *testing.T) {
	h := newHarness()
	h.expectSdkCallMadeWithServiceCallMethod(deployments.CONTRACT.Name, deployments.METHOD_GET_CODE.Name, errors.New("code not found error"))

	input := getContractInfoInput().WithUnknownContract().Build()
	_, err := h.service.GetContractInfo(input)
	require.Error(t, err, "GetContractInfo should fail")

	h.verifySdkCallMade(t)
}
