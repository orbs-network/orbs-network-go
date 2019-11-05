// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
//
// +build !javascript

package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) getProcessor(processorType protocol.ProcessorType) (services.Processor, error) {
	switch processorType {
	case protocol.PROCESSOR_TYPE_NATIVE:
		return s.processors[protocol.PROCESSOR_TYPE_NATIVE], nil
	default:
		return nil, errors.Errorf("_Deployments.getInfo contract returned unknown processor type: %s", processorType)
	}
}
