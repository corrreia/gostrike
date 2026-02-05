// Package main is the entry point for the GoStrike c-shared library.
// This file is built with -buildmode=c-shared to create libgostrike_go.so
package main

import "C"

// Import the bridge package which exports all CGO functions
import (
	// Bridge exports all CGO functions to C++
	_ "github.com/corrreia/gostrike/internal/bridge"

	// Import core modules (modules register themselves via init())
	_ "github.com/corrreia/gostrike/internal/modules/http"

	// Import example plugin (plugins register themselves via init())
	_ "github.com/corrreia/gostrike/plugins/example"
)

// main is required for c-shared build mode but is never called
func main() {}
