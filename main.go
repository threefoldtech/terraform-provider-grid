package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider"
	"github.com/threefoldtech/terraform-provider-grid/pkg/state"
)

// Run "go generate" to format example terraform files and generate the docs for the registry/website

// If you do not have terraform installed, you can remove the formatting command, but its suggested to
// ensure the documentation is formatted properly.
//go:generate terraform fmt -recursive ./examples/

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary
	version string = "dev"

	// goreleaser can also pass the specific commit if you want
	// commit  string = ""
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	db, err := state.NewLocalStateDB(state.TypeFile)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = db.Load()
	if err != nil {
		log.Fatal(err.Error())
	}

	/** 1- determine network for user
		2- use provider for network user wants

		Ex: network = main , call mainnet.provider.new , (make provider outside internal)

		function *schema.Provider -->
	**/
	// mainnet.

	var providerFunc func() *schema.Provider //contain provider function
	// if ()

	// providerFunc, sub := provider.New(version, db.GetState())
	// if sub != nil {
	// 	defer sub.Close()
	// }
	opts := &plugin.ServeOpts{ProviderFunc: providerFunc}

	if debugMode {
		// TODO: update this string with the full name of your provider as used in your configs
		err := plugin.Debug(context.Background(), "registry.terraform.io/hashicorp/scaffolding", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
	err = db.Save()
	if err != nil {
		log.Fatal(err.Error())
	}
}
