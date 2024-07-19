package previder

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"log"
)

type Config struct {
	Token      types.String `tfsdk:"token"`
	Url        types.String `tfsdk:"url"`
	CustomerId types.String `tfsdk:"customer"`
}

func (c *Config) Client() (baseClient *client.BaseClient) {
	var url = "https://portal.previder.nl/api/"
	if !c.Url.IsNull() && c.Url.ValueString() != "" {
		url = c.Url.ValueString()
	}
	var customerId = ""
	if !c.CustomerId.IsNull() && c.CustomerId.ValueString() != "" {
		customerId = c.CustomerId.ValueString()
	}

	d, err := client.New(&client.ClientOptions{Token: c.Token.ValueString(), BaseUrl: url, CustomerId: customerId})

	if err != nil {
		log.Printf("[ERROR] ERROR")
		return nil
	}

	return d
}
