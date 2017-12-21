package provider

import (
	"fmt"
	"log"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/previder/previder-go-sdk"
	"time"
)

func resourcePreviderVirtualNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourcePreviderVirtualNetworkCreate,
		Read:   resourcePreviderVirtualNetworkRead,
		Delete: resourcePreviderVirtualNetworkDelete,
		Update: resourcePreviderVirtualNetworkUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"address_pool": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip_start": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ip_end": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ip_netmask": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ip_gateway": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ip_nameserver1": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ip_nameserver2": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourcePreviderVirtualNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	// Build up our creation options
	var network client.VirtualNetworkCreate
	network.Name = d.Get("name").(string)
	network.Type = "IAN"

	log.Printf("[DEBUG] VirtualNetwork create configuration: %#v", network)
	task, err := c.VirtualNetwork.Create(network)

	if err != nil {
		return fmt.Errorf("Error creating VirtualNetwork: %s", err)
	}

	c.Task.WaitForTask(task, 5*time.Minute)
	log.Printf("[INFO] Virtual network %s created", network.Name)

	return resourcePreviderVirtualNetworkUpdate(d, meta)
}

func resourcePreviderVirtualNetworkRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	log.Printf("[INFO] Retrieving network with name: %s", d.Get("name").(string))
	virtualNetwork, err := c.VirtualNetwork.Get(d.Get("name").(string))
	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving VirtualNetwork: %s", err)
	}
	d.SetId(virtualNetwork.Id)
	d.Set("name", virtualNetwork.Name)
	d.Set("addressPool", virtualNetwork.AddressPool)

	return nil
}

func resourcePreviderVirtualNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	log.Printf("[INFO] Retrieving network with name: %s", d.Get("name").(string))
	virtualNetwork, err := c.VirtualNetwork.Get(d.Get("name").(string))
	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving VirtualNetwork: %s", err)
	}

	// Build up addressPool
	var update client.VirtualNetworkUpdate
	//
	if d.Get("address_pool").(map[string]interface{})["ip_start"] != nil {
		var addressPool client.AddressPool
		addressPool.Start = d.Get("address_pool").(map[string]interface{})["ip_start"].(string)
		addressPool.End = d.Get("address_pool").(map[string]interface{})["ip_end"].(string)
		addressPool.Mask = d.Get("address_pool").(map[string]interface{})["ip_netmask"].(string)
		if d.Get("address_pool").(map[string]interface{})["ip_gateway"] != nil {
			addressPool.Gateway = d.Get("address_pool").(map[string]interface{})["ip_gateway"].(string)
		}
		if d.Get("address_pool").(map[string]interface{})["ip_nameserver1"] != nil {
			addressPool.NameServers = append(addressPool.NameServers, d.Get("address_pool").(map[string]interface{})["ip_nameserver1"].(string))
		}

		if d.Get("address_pool").(map[string]interface{})["ip_nameserver2"] != nil {
			addressPool.NameServers = append(addressPool.NameServers, d.Get("address_pool").(map[string]interface{})["ip_nameserver2"].(string))
		}

		update.AddressPool = addressPool
	}

	update.Id = virtualNetwork.Id
	update.Name = virtualNetwork.Name
	update.PublicNet = virtualNetwork.PublicNet
	update.VlanId = virtualNetwork.VlanId
	update.Type = virtualNetwork.Type

	_, uerr := c.VirtualNetwork.Update(virtualNetwork.Id, update)
	//
	if uerr != nil {
		return fmt.Errorf("Error updating VirtualNetwork: %s", uerr)
	}
	//
	log.Printf("[INFO] Virtual network %s updated", update.Name)

	return resourcePreviderVirtualNetworkRead(d, meta)
}

func resourcePreviderVirtualNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	log.Printf("[INFO] Deleting VirtualNetwork: %s", d.Id())
	task, err := c.VirtualNetwork.Delete(d.Id())
	c.Task.WaitForTask(task, 5*time.Minute)

	if err != nil {
		return fmt.Errorf("Error deleting VirtualNetwork: %s", err)
	}
	d.SetId("")

	return nil
}