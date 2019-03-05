// +build unsafetests

package globalpreorder_systemcontract

import "github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"

var PUBLIC = sdk.Export(approve, refreshSubscription, unsafetests_setSubscriptionOk, unsafetests_setSubscriptionProblem)

func unsafetests_setSubscriptionOk() {
	_clearSubscriptionProblem()
}

func unsafetests_setSubscriptionProblem() {
	_writeSubscriptionProblem("subscription not paid")
}
