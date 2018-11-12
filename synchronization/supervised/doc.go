// Package supervised provides basic supervision abilities for running goroutines,
// namely making sure that panics are not swallowed and that long-running goroutines
// are restarted if they crash
// Run go tools (build/test) with "-tags norecover" to disable panic recovery (useful for debugging/testing)
package supervised
