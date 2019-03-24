// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

const Passed = "--- PASS:"
const Failed = "--- FAIL:"
const PanicOhNo = "panic: oh no"
const BeforeLoggerCreated = "before logger is created"
const LoggedWithLogger = "logged using the logger"
const ErrorWithLogger = "error logged using the logger"
const BeforeCallPanic = "about to call theFunctionThrowingThePanic"
const AfterCallPanic = "after call to theFunctionThrowingThePanic"
const BeforeLoggerError = "about to call logger.Error"
const AfterLoggerError = "after call to logger.Error"
const ParentScopeBeforeTest = "this is in parent before the sub test"
const ParentScopeAfterTest = "this is parent after the sub test"
const MustShow = "this is supposed to show even if the test fails"
const MustNotShow = "this is not supposed to show when the test fails"
