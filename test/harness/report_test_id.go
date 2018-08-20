package harness

import (
	"fmt"
	"testing"
)

func ReportTestId(t *testing.T, testId string) {
	if t.Failed() {
		fmt.Println("FAIL search snippet: grep _test-id="+testId, "test.out")
	}
}
