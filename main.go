package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/previder/terraform-provider-previder/provider"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.Provider,
	})
}
