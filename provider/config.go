package provider

import (
	"github.com/previder/previder-go-sdk"
	"log"
)

const (
	defaultClusterName  = "Express"
	defaultTemplateName = "CoreOS"
	defaultNetworkName  = "Public WAN"
)

type Config struct {
	Token string
	Url   string
}

func (c *Config) Client() (*client.BaseClient, error) {
	d := client.New(&client.ClientOptions{Token: c.Token, BaseUrl: c.Url})

	log.Printf("[INFO] Previder Client configured")

	return d, nil
}
