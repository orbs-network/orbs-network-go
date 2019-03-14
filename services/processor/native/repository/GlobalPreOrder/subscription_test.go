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
	s := &subscriptionData{id: 17, plan: planName, startTime: time.Now().Add(-1 * time.Hour), tokensPaid: big.NewInt(int64(plans[planName] - 1))}
	require.Error(t, s.validate(17, time.Now()), "validate did not fail on underfunded plan")
}

func TestSubscriptionData_Validate_FailsOnFutureStartTime(t *testing.T) {
	s := &subscriptionData{id: 17, startTime: time.Now().Add(1 * time.Hour)}
	require.Error(t, s.validate(17, time.Now()), "validate did not fail on future start time")
}

func TestSubscriptionData_Validate_SucceedsOnValidSubscription(t *testing.T) {
	planName := "B0"
	s := &subscriptionData{id: 17, plan: planName, startTime: time.Now().Add(-1 * time.Hour), tokensPaid: big.NewInt(int64(plans[planName]))}

	require.NoError(t, s.validate(17, time.Now()), "validate failed on a valid subscription")
}
