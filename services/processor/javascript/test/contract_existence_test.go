// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCall_WithUnknownContractFails(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger, "")
			input := processCallInput().WithUnknownContract().Build()
			h.expectSdkCallMadeWithServiceCallMethod(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_CODE, builders.ArgumentsArray(string(input.ContractName)), nil, errors.New("code not found error"))

			_, err := h.service.ProcessCall(ctx, input)
			require.EqualError(t, err, "code not found error")

			h.verifySdkCallMade(t)
		})
	})
}
