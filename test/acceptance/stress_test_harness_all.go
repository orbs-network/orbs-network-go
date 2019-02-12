// +build !memoryleak

package acceptance

// as we are using a build flag, and we want to avoid logging in the stress test
// as the harness will cache them because of t.Log, we have this conditional compilation for creating the harness
func getStressTestHarness() *networkHarnessBuilder {
	return newHarness()
}
