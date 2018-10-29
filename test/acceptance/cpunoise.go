// +build cpunoise

package acceptance

import "github.com/orbs-network/orbs-network-go/test"

func init() {
	test.StartCpuSchedulingJitter()
}
