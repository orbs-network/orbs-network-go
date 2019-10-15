// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/services/processor/javascript"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
)

type config struct {
	path string
}

func (c *config) ProcessorPluginPath() string {
	return c.path
}

func (c *config) VirtualChainId() primitives.VirtualChainId {
	return 42
}

func TestService(t *testing.T) {
	logger := log.DefaultTestingLogger(t)
	processor := javascript.NewJavaScriptProcessor(logger, &config{
		"./dummy_plugin.bin",
	})

	mockVm := &services.MockVirtualMachine{}
	processor.RegisterContractSdkCallHandler(mockVm)

	mockVm.When("HandleSdkCall", mock.Any, mock.Any).Return(nil).Times(1)

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
