// Package presentation owns Petclinic's HTTP presentation infrastructure.
package presentation

import "net/http"

// @import { Bean, Fallback } from "github.com/StevenBuglione/spice/annotation/core"

// NewMux returns an instance-owned Go 1.22 pattern-aware HTTP mux.
//
// @Bean
// @Fallback
func NewMux() *http.ServeMux {
	return http.NewServeMux()
}
