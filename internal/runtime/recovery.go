// Package runtime provides the internal runtime for GoStrike.
// This file contains panic recovery utilities.
package runtime

import (
	"fmt"
	"runtime/debug"
)

// logPanic is set by bridge package to enable panic logging
var logPanic func(context string, panicVal interface{}, stack string)

// SetPanicLogger sets the panic logging function
func SetPanicLogger(fn func(context string, panicVal interface{}, stack string)) {
	logPanic = fn
}

func logPanicError(context string, panicVal interface{}, stack string) {
	if logPanic != nil {
		logPanic(context, panicVal, stack)
	} else {
		fmt.Printf("[PANIC] %s: %v\n%s\n", context, panicVal, stack)
	}
}

// RecoverPanic recovers from a panic and logs the error
// Should be called via defer at the start of goroutines/callbacks
func RecoverPanic(context string) {
	if r := recover(); r != nil {
		stack := string(debug.Stack())
		logPanicError(context, r, stack)
	}
}

// SafeCall calls a function with panic recovery
// Returns true if the function completed without panicking
func SafeCall(context string, fn func()) bool {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logPanicError(context, r, stack)
		}
	}()
	fn()
	return true
}

// SafeCallWithResult calls a function with panic recovery and returns its result
// If a panic occurs, returns the default value
func SafeCallWithResult[T any](context string, defaultVal T, fn func() T) T {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logPanicError(context, r, stack)
		}
	}()
	return fn()
}

// SafeCallWithError calls a function with panic recovery
// If a panic occurs, returns an error
func SafeCallWithError(context string, fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			logPanicError(context, r, stack)
		}
	}()
	return fn()
}
