package test

const Failed = "--- FAIL:"
const BeforeLoggerCreated = "before logger is created"
const LoggedWithLogger = "logged using the logger"
const BeforeCallPanic = "about to call theFunctionThrowingThePanic"
const AfterCallPanic = "after call to theFunctionThrowingThePanic"
const ParentScopeBeforeTest = "this is in parent before the sub test"
const ParentScopeAfterTest = "this is parent after the sub test"
const MustShow = "this is supposed to show even if the test fails"
const MustNotShow = "this is not supposed to show when the test fails"
