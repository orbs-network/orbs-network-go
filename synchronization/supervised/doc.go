// Package supervised provides basic supervision abilities for running goroutines,
// namely making sure that panics are not swallowed and that long-running goroutines
// are restarted if they crash
package supervised

