// +build cpunoise

package test

import "github.com/orbs-network/orbs-network-go/test"

func init() {
	test.StartCpuSchedulingJitter()
}
