package presentation

import "time"

// @import { Configuration } from "github.com/spice-framework/spice/annotation/core"

// ServerSettings contains safe Petclinic HTTP server defaults.
//
// @Configuration(prefix="petclinic.server")
type ServerSettings struct {
	Address           string        `spice:"address,default=127.0.0.1:8080,env=SPICE_PETCLINIC_ADDRESS"`
	ReadHeaderTimeout time.Duration `spice:"read-header-timeout,default=5s"`
	ReadTimeout       time.Duration `spice:"read-timeout,default=15s"`
	WriteTimeout      time.Duration `spice:"write-timeout,default=15s"`
	IdleTimeout       time.Duration `spice:"idle-timeout,default=60s"`
}
