// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetBlock_GetBlockStorageOk(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, t, 1*time.Second, 1*time.Minute)

		now := time.Now()
		blockPair := builders.BlockPair().WithBlockCreated(now).WithHeight(8).Build()
		harness.prepareGetBlock(blockPair, nil)
		result, err := harness.papi.GetBlock(ctx, &services.GetBlockInput{
			ClientRequest: (&client.GetBlockRequestBuilder{
				BlockHeight:     8,
				ProtocolVersion: builders.DEFAULT_TEST_PROTOCOL_VERSION,
				VirtualChainId:  builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "get block returned nil instead of object")
		require.Equal(t, protocol.REQUEST_STATUS_COMPLETED, result.ClientResponse.RequestResult().RequestStatus(), "got wrong status")
		require.Equal(t, blockPair.TransactionsBlock.Header.BlockHeight(), result.ClientResponse.RequestResult().BlockHeight(), "got wrong block height")
		require.Equal(t, blockPair.TransactionsBlock.Header.Timestamp(), result.ClientResponse.RequestResult().BlockTimestamp(), "got wrong timestamp")
		require.NotNil(t, result.ClientResponse.TransactionsBlockHeader(), "got empty tx block header")
		// other fields checked in unit test
	})
}

func TestGetBlock_GetBlockStorageFail(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, t, 1*time.Second, 1*time.Minute)

		harness.getBlockFails()
		result, err := harness.papi.GetBlock(ctx, &services.GetBlockInput{
			ClientRequest: (&client.GetBlockRequestBuilder{
				BlockHeight:     8,
				ProtocolVersion: builders.DEFAULT_TEST_PROTOCOL_VERSION,
				VirtualChainId:  builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.Error(t, err, "error did not happened when it should")
		require.NotNil(t, result, "get block returned nil instead of object")
		require.Equal(t, protocol.REQUEST_STATUS_SYSTEM_ERROR, result.ClientResponse.RequestResult().RequestStatus(), "got wrong status")
		require.EqualValues(t, 0, result.ClientResponse.RequestResult().BlockHeight(), "got wrong block height")
		require.EqualValues(t, 0, result.ClientResponse.RequestResult().BlockTimestamp(), "got wrong status")
		require.Equal(t, 0, len(result.ClientResponse.TransactionsBlockHeader().Raw()), "TransactionsBlockHeader should be empty")
	})
}

func TestGetBlock_GetBlockStorageNoRecord(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, t, 1*time.Second, 1*time.Minute)

		now := time.Now()
		lastCommitedPair := builders.BlockPair().WithBlockCreated(now).WithHeight(8).Build()
		harness.prepareGetBlock(nil, lastCommitedPair)
		result, err := harness.papi.GetBlock(ctx, &services.GetBlockInput{
			ClientRequest: (&client.GetBlockRequestBuilder{
				BlockHeight:     1000,
				ProtocolVersion: builders.DEFAULT_TEST_PROTOCOL_VERSION,
				VirtualChainId:  builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "get block returned nil instead of object")
		require.Equal(t, protocol.REQUEST_STATUS_NOT_FOUND, result.ClientResponse.RequestResult().RequestStatus(), "got wrong status")
		require.Equal(t, lastCommitedPair.TransactionsBlock.Header.BlockHeight(), result.ClientResponse.RequestResult().BlockHeight(), "got wrong block height")
		require.Equal(t, lastCommitedPair.TransactionsBlock.Header.Timestamp(), result.ClientResponse.RequestResult().BlockTimestamp(), "got wrong timestamp")
		require.Equal(t, 0, len(result.ClientResponse.TransactionsBlockHeader().Raw()), "TransactionsBlockHeader should be empty")
	})
}

func TestGetBlock_GetBlockStorageNoRecordThenFailsToGetLast(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, t, 1*time.Second, 1*time.Minute)

		harness.prepareGetBlock(nil, nil)
		result, err := harness.papi.GetBlock(ctx, &services.GetBlockInput{
			ClientRequest: (&client.GetBlockRequestBuilder{
				BlockHeight:     1000,
				ProtocolVersion: builders.DEFAULT_TEST_PROTOCOL_VERSION,
				VirtualChainId:  builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.Error(t, err, "error happened when it should not")
		require.NotNil(t, result, "get block returned nil instead of object")
		require.Equal(t, protocol.REQUEST_STATUS_SYSTEM_ERROR, result.ClientResponse.RequestResult().RequestStatus(), "got wrong status")
		require.EqualValues(t, 0, result.ClientResponse.RequestResult().BlockHeight(), "got wrong block height")
		require.EqualValues(t, 0, result.ClientResponse.RequestResult().BlockTimestamp(), "got wrong status")
		require.Equal(t, 0, len(result.ClientResponse.TransactionsBlockHeader().Raw()), "TransactionsBlockHeader should be empty")
	})
}

func TestGetBlock_RequestBlockZero(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, t, 1*time.Second, 1*time.Minute)

		now := time.Now()
		lastCommitedPair := builders.BlockPair().WithBlockCreated(now).WithHeight(8).Build()
		harness.prepareGetLastBlock(lastCommitedPair)
		result, err := harness.papi.GetBlock(ctx, &services.GetBlockInput{
			ClientRequest: (&client.GetBlockRequestBuilder{
				BlockHeight:     0,
				ProtocolVersion: builders.DEFAULT_TEST_PROTOCOL_VERSION,
				VirtualChainId:  builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
			}).Build(),
		})

		harness.verifyMocks(t) // contract test

		// value test
		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, result, "get block returned nil instead of object")
		require.Equal(t, protocol.REQUEST_STATUS_BAD_REQUEST, result.ClientResponse.RequestResult().RequestStatus(), "got wrong status")
		require.Equal(t, lastCommitedPair.TransactionsBlock.Header.BlockHeight(), result.ClientResponse.RequestResult().BlockHeight(), "got wrong block height")
		require.Equal(t, lastCommitedPair.TransactionsBlock.Header.Timestamp(), result.ClientResponse.RequestResult().BlockTimestamp(), "got wrong timestamp")
		require.Equal(t, 0, len(result.ClientResponse.TransactionsBlockHeader().Raw()), "TransactionsBlockHeader should be empty")
	})
}
