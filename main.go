package main

import (
	"context"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/previder/terraform-provider-previder/previder"
	"log"
)

func main() {
	ctx := context.Background()

	providerFactory, err := previder.GetMuxedProvider(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf6server.ServeOpt

	err = tf6server.Serve(
		"registry.terraform.io/previder/previder",
		providerFactory,
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
