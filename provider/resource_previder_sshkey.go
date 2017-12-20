package provider

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/previder/previder-go-sdk"
)

func resourcePreviderSSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourcePreviderSSHKeyCreate,
		Read:   resourcePreviderSSHKeyRead,
		Delete: resourcePreviderSSHKeyDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"rid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"public_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePreviderSSHKeyCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	// Build up our creation options
	var create client.SSHKeyCreate
	create.Name = d.Get("name").(string)
	create.PublicKey = d.Get("public_key").(string)

	log.Printf("[DEBUG] SSH key create configuration: %#v", create)
	sshKey, err := c.SSHKey.Create(&create)

	if err != nil {
		return fmt.Errorf("Error creating SSH key: %s", err)
	}

	d.SetId(sshKey.Id)
	log.Printf("[INFO] SSH Key: %d", sshKey.Id)

	return resourcePreviderSSHKeyRead(d, meta)
}

func resourcePreviderSSHKeyRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	log.Printf("[INFO] Retreiving SSH key with ID: %s", d.Id())
	sshKey, err := c.SSHKey.Get(d.Id())

	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving SSH key: %s", err.Error())
	}

	d.Set("name", sshKey.Name)
	d.Set("fingerprint", sshKey.Fingerprint)
	d.Set("public_key", sshKey.PublicKey)

	return nil
}

func resourcePreviderSSHKeyDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	log.Printf("[INFO] Deleting SSH key: %s", d.Id())
	err := c.SSHKey.Delete(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting SSH key: %s", err)
	}

	d.SetId("")
	return nil
}
