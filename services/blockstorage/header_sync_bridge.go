// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)


func (s *Service) HandleHeaderAvailabilityResponse(ctx context.Context, input *gossiptopics.HeaderAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	if s.headerSync != nil {
		s.headerSync.HandleHeaderAvailabilityResponse(ctx, input)
	}
	return nil, nil
}

func (s *Service) HandleHeaderSyncResponse(ctx context.Context, input *gossiptopics.HeaderSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	if s.headerSync != nil {
		s.headerSync.HandleBlockSyncResponse(ctx, input)
	}
	return nil, nil
}
