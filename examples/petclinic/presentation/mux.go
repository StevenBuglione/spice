// Package presentation owns Petclinic's HTTP presentation infrastructure.
package presentation

import "net/http"

// @import { Bean, Fallback } from "github.com/StevenBuglione/spice/annotation/core"

// NewMux returns an instance-owned Go 1.22 pattern-aware HTTP mux with
// immutable embedded assets.
//
// @Bean
// @Fallback
func NewMux() (*http.ServeMux, error) {
	mux := http.NewServeMux()
	assets, err := staticHandler()
	if err != nil {
		return nil, err
	}
	mux.Handle("GET /resources/", http.StripPrefix("/resources/", assets))
	return mux, nil
}
