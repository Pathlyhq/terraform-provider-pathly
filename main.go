// Provider Terraform pour Pathly.
//
// Compilation locale, puis déclaration dans ~/.terraformrc via un
// dev_overrides : voir README.md.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/pathlyhq/terraform-provider-pathly/internal/provider"
)

// version est remplacée à la compilation : -ldflags "-X main.version=1.0.0".
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "attendre un débogueur avant de servir le provider")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		// Adresse de registre : elle doit correspondre au bloc
		// required_providers des configurations clientes.
		Address: "registry.terraform.io/pathlyhq/pathly",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
