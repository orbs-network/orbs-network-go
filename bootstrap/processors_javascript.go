// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
//
// +build javascript

package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/processor/javascript"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
)

func getProcessors(nativeCompiler adapter.Compiler, nodeConfig config.NodeConfig, logger log.Logger, metricRegistry metric.Registry) map[protocol.ProcessorType]services.Processor {
	processors := make(map[protocol.ProcessorType]services.Processor)
	processors[protocol.PROCESSOR_TYPE_NATIVE] = native.NewNativeProcessor(nativeCompiler, nodeConfig, logger, metricRegistry)
	processors[protocol.PROCESSOR_TYPE_JAVASCRIPT] = javascript.NewJavaScriptProcessor(logger, nodeConfig)

	return processors
}
