package provider

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"log"
)

// Provider returns a schema.Provider for Previder.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The token key for API operations.",
			},
			"url": {
				Type:        schema.TypeString,
				Default:     "https://portal.previder.nl/api/",
				Optional:    true,
				Description: "The API endpoint URL.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"previder_virtualmachine": resourcePreviderVirtualMachine(),
			"previder_virtualnetwork": resourcePreviderVirtualNetwork(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	log.Printf("[INFO] Previder Client configured")
	config := Config{
		Token: d.Get("token").(string),
		Url:   d.Get("url").(string),
	}
	return config.Client()
}
