package test

import (
	"fmt"
	"github.com/onsi/gomega/types"
	"github.com/orbs-network/go-mock"
)

func ExecuteAsPlanned() types.GomegaMatcher {
	return &verifiedMockMatcher{}
}

type verifiedMockMatcher struct {
	verifyError error
}

func (matcher *verifiedMockMatcher) Match(actual interface{}) (success bool, err error) {
	mock, ok := actual.(mock.HasVerify)
	if !ok {
		return false, fmt.Errorf("ExecuteAsPlanned matcher expects a go-mock mock")
	}
	success, matcher.verifyError = mock.Verify()
	return
}

func (matcher *verifiedMockMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected mock to execute as planned\n\t%s", matcher.verifyError.Error())
}

func (matcher *verifiedMockMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected mock to execute not according to plan\n\t%s", matcher.verifyError.Error())
}
