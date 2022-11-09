package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/terraform-provider-grid/internal/provider"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
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
var (
	substrateConn   *subi.SubstrateExt
	providerNetwork = map[string]string{
		"dev": "https://registry.terraform.io/providers/threefoldtech/grid/1.5.7",
	}
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	network := determineSubstrateNetwork()

	subext, err := provider.SubstrateVersion[network](provider.SUBSTRATE_URL[network]).SubstrateExt()
	if err != nil {
		log.Fatal(err)
	}
	defer subext.Close()
	rmbSubstrate, err := substrate.NewManager(provider.SUBSTRATE_URL[network]).Substrate()
	if err != nil {
		log.Fatal(err)
	}
	defer rmbSubstrate.Close()
	opts := &plugin.ServeOpts{ProviderFunc: provider.New(version, subext, rmbSubstrate), ProviderAddr: providerNetwork[network]}

	if debugMode {
		// TODO: update this string with the full name of your provider as used in your configs
		err := plugin.Debug(context.Background(), "registry.terraform.io/hashicorp/scaffolding", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}

func determineSubstrateNetwork() string {
	network := os.Getenv("NETWORK")
	if network == "" {
		network = "dev"
	}
	if network != "dev" && network != "qa" && network != "test" && network != "main" {
		log.Fatal("network must be one of dev, qa, test, or main")
	}
	return network
}
