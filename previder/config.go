package previder

import (
	"errors"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"os"
)

type Config struct {
	Token      types.String `tfsdk:"token"`
	Url        types.String `tfsdk:"url"`
	CustomerId types.String `tfsdk:"customer"`
}

func (c *Config) Client() (*client.PreviderClient, error) {
	var url = "https://portal.previder.nl/api/"
	if !c.Url.IsNull() && c.Url.ValueString() != "" {
		url = c.Url.ValueString()
	}
	if os.Getenv("PREVIDER_URL") != "" {
		url = os.Getenv("PREVIDER_URL")
	}
	if url == "" {
		return nil, errors.New("no Previder URL found")
	}

	var customerId = ""
	if !c.CustomerId.IsNull() && c.CustomerId.ValueString() != "" {
		customerId = c.CustomerId.ValueString()
	}
	var token = c.Token.ValueString()
	if token == "" {
		if os.Getenv("PREVIDER_TOKEN") != "" {
			token = os.Getenv("PREVIDER_TOKEN")
		} else {
			return nil, errors.New("no Previder token found")
		}
	}

	d, err := client.New(&client.ClientOptions{Token: token, BaseUrl: url, CustomerId: customerId})

	if err != nil {
		return nil, err
	}

	return d, nil
}
