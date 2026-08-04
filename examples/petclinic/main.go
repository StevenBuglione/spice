package main

import (
	"os"

	spiceapp "github.com/StevenBuglione/spice/examples/petclinic/internal/spicegen/petclinic"
	_ "github.com/StevenBuglione/spice/examples/petclinic/memory"
	_ "github.com/StevenBuglione/spice/examples/petclinic/model"
	_ "github.com/StevenBuglione/spice/examples/petclinic/owner"
	_ "github.com/StevenBuglione/spice/examples/petclinic/presentation"
	_ "github.com/StevenBuglione/spice/examples/petclinic/system"
	_ "github.com/StevenBuglione/spice/examples/petclinic/vet"
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
