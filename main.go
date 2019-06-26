package main

import (
	"github.com/hashicorp/terraform/plugin"
	"gitlab.previder.net/devops/terraform-provider-previder/provider"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.Provider,
	})
}
