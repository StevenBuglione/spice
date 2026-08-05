package main

import (
	"os"

	spiceapp "github.com/spice-framework/spice/examples/petclinic/internal/spicegen/petclinic"
	_ "github.com/spice-framework/spice/examples/petclinic/memory"
	_ "github.com/spice-framework/spice/examples/petclinic/model"
	_ "github.com/spice-framework/spice/examples/petclinic/owner"
	_ "github.com/spice-framework/spice/examples/petclinic/presentation"
	_ "github.com/spice-framework/spice/examples/petclinic/system"
	_ "github.com/spice-framework/spice/examples/petclinic/vet"
)

// @import { Application } from "github.com/spice-framework/spice/annotation/core"
// @import { Enable } from "github.com/spice-framework/spice/annotation/management"
// @import { Logging } from "github.com/spice-framework/spice/annotation/observability"

// @Application
// @Enable(expose=["health", "liveness", "readiness", "info", "metrics", "configprops", "modules"], access="loopback")
// @Logging
func main() {
	// This package-main boundary alone owns the process status.
	//nolint:forbidigo // Generated Main returns the portable exit code.
	os.Exit(spiceapp.Main(os.Args[1:]))
}
