package main

import (
	"os"

	spiceapp "github.com/StevenBuglione/spice/examples/petclinic/internal/spicegen/petclinic"
)

// @import { Application } from "github.com/StevenBuglione/spice/annotation/core"
// @import { Enable } from "github.com/StevenBuglione/spice/annotation/management"
// @import { Logging } from "github.com/StevenBuglione/spice/annotation/observability"

// @Application
// @Enable(expose=["health", "liveness", "readiness", "info", "metrics", "configprops", "modules"], access="loopback")
// @Logging
func main() {
	// This package-main boundary alone owns the process status.
	//nolint:forbidigo // Generated Main returns the portable exit code.
	os.Exit(spiceapp.Main(os.Args[1:]))
}
