// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package globalpreorder_systemcontract

import (
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
	"time"
)

func TestSubscriptionData_Validate_FailsOnVirtualChainIdMismatch(t *testing.T) {
	s := &subscriptionData{id: 17}
	require.Error(t, s.validate(42, time.Now()), "validate did not fail on mismatching virtual chain id")
}

func TestSubscriptionData_Validate_FailsOnUnrecognizedPlan(t *testing.T) {
	s := &subscriptionData{id: 17, plan: "foo"}
	require.Error(t, s.validate(17, time.Now()), "validate did not fail on bad plan name")
}

func TestSubscriptionData_Validate_FailsOnUnderfundedPlan(t *testing.T) {
	planName := "B0"
	s := &subscriptionData{id: 17, plan: planName, startTime: time.Now().Add(-1 * time.Hour), tokensPaidInOrbs: planCostsInOrbs[planName] - 1}
	require.Error(t, s.validate(17, time.Now()), "validate did not fail on underfunded plan")
}

func TestSubscriptionData_Validate_FailsOnFutureStartTime(t *testing.T) {
	s := &subscriptionData{id: 17, startTime: time.Now().Add(1 * time.Hour)}
	require.Error(t, s.validate(17, time.Now()), "validate did not fail on future start time")
}

func TestSubscriptionData_Validate_SucceedsOnValidSubscription(t *testing.T) {
	planName := "B0"
	s := &subscriptionData{id: 17, plan: planName, startTime: time.Now().Add(-1 * time.Hour), tokensPaidInOrbs: planCostsInOrbs[planName]}

	require.NoError(t, s.validate(17, time.Now()), "validate failed on a valid subscription")
}

func TestSatoshiToOrbs(t *testing.T) {
	t.Log(satoshiFactor)
	require.EqualValues(t, 17, _satoshiToOrbs(new(big.Int).Mul(big.NewInt(17), big.NewInt(1000000000000000000))))
}
