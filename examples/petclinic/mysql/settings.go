package mysql

// @import { Configuration } from "github.com/StevenBuglione/spice/annotation/core"

// Settings defines the explicit MySQL profile configuration.
//
// @Configuration(prefix="petclinic.datasource")
type Settings struct {
	URL           string `spice:"url,required,env=SPICE_PETCLINIC_MYSQL_URL,secret"`
	AllowInsecure bool   `spice:"allow-insecure,default=false,env=SPICE_PETCLINIC_MYSQL_ALLOW_INSECURE"`
}
