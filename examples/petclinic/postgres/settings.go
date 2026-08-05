package postgres

// @import { Configuration } from "github.com/spice-framework/spice/annotation/core"

// Settings configures the PostgreSQL Petclinic application target.
//
// URL is intentionally required and redacted. Local development must opt in
// to disabled TLS explicitly; production defaults remain secure.
//
// @Configuration(prefix="petclinic.datasource")
type Settings struct {
	URL           string `spice:"url,required,env=SPICE_PETCLINIC_POSTGRES_URL,secret"`
	AllowInsecure bool   `spice:"allow-insecure,default=false,env=SPICE_PETCLINIC_POSTGRES_ALLOW_INSECURE"`
}
