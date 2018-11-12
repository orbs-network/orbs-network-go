// +build cpunoise

package externalsync

import "github.com/orbs-network/orbs-network-go/test"

func init() {
	test.StartCpuSchedulingJitter()
}
