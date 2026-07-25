// Package app defines the typed components in the generated hello-world application.
//
// @Module
package app

// Hello marks Server as the root of the generated example application.
//
// @Application
func Hello(*Server) {
	panic("Spice application marker bodies are never executed")
}
