package util

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/previder/previder-go-sdk/client"
	"log"
)

func ConfigureClient(providerData any) (*client.BaseClient, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	if providerData == nil {
		return nil, diagnostics
	}

	baseClient, ok := providerData.(*client.BaseClient)
	if !ok {
		log.Println("Got the data")
		diagnostics.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *client.BaseClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil, diagnostics
	}

	log.Printf("Trying to fetch API information")
	result, err := baseClient.ApiInfo()
	if err != nil {
		diagnostics.AddError(
			"Invalid client",
			fmt.Sprintf("API could not be queried"),
		)
		return nil, diagnostics
	}

	log.Printf("API version %s", result.Version)

	return baseClient, diagnostics
}