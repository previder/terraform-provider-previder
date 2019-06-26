package provider

import (
	"github.com/previder/previder-go-sdk/client"
	"log"
)

const (
	defaultClusterName  = "express"
	defaultTemplateName = "ubuntu-18.04"
	defaultNetworkName  = "Public WAN"
)

type Config struct {
	Token string
	Url   string
}

func (c *Config) Client() (*client.BaseClient, error) {
	d, err := client.New(&client.ClientOptions{Token: c.Token, BaseUrl: c.Url})
	if err != nil {
		log.Printf("[ERROR] ERROR")
		return nil, err
	}
	log.Printf("[INFO] Previder Client configured")

	return d, nil
}
