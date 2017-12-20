package provider

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/previder/previder-go-sdk"
)

func resourcePreviderDNSRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourcePreviderDNSRecordCreate,
		Read:   resourcePreviderDNSRecordRead,
		Delete: resourcePreviderDNSRecordDelete,
		Exists: resourcePreviderDNSRecordExists,

		Schema: map[string]*schema.Schema{
			"zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ttl": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"prio": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"records": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				ForceNew: true,
				Set:      schema.HashString,
			},
		},
	}
}

func resourcePreviderDNSRecordCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	update := new(client.DomainZoneUpdate)
	update.Remove = make([]client.DomainRecord, 0)
	update.Update = make([]client.DomainRecord, 0)

	records := d.Get("records").(*schema.Set).List()
	update.Add = make([]client.DomainRecord, len(records))

	for i, content := range records {
		record := new(client.DomainRecord)
		record.Name = d.Get("name").(string)
		record.Type = d.Get("type").(string)
		record.Ttl = d.Get("ttl").(int)
		record.Prio = d.Get("prio").(int)
		record.Content = content.(string)
		update.Add[i] = *record
	}

	err := c.DNSRecord.Update(d.Get("zone").(string), update)
	d.SetId(d.Get("name").(string) + "-" + d.Get("type").(string))

	return err
}

func resourcePreviderDNSRecordRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	zone, err := c.DNSRecord.Get(d.Get("zone").(string))
	if err != nil {
		return err
	}

	content := make([]string, 0)
	for _, record := range zone.Records {
		if d.Get("type").(string) == record.Type && d.Get("name").(string) == record.Name {
			d.Set("ttl", record.Ttl)
			d.Set("prio", record.Prio)
			d.Set("type", record.Type)
			d.Set("name", record.Name)
			content = append(content, record.Content)

		}
	}

	d.Set("records", content)
	d.SetId(d.Get("name").(string) + "-" + d.Get("type").(string))

	return nil
}

func resourcePreviderDNSRecordDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	zone, err := c.DNSRecord.Get(d.Get("zone").(string))
	if err != nil {
		return err
	}

	update := new(client.DomainZoneUpdate)
	update.Add = make([]client.DomainRecord, 0)
	update.Update = make([]client.DomainRecord, 0)
	update.Remove = make([]client.DomainRecord, 0)
	for _, record := range zone.Records {
		if d.Get("type").(string) == record.Type && d.Get("name").(string) == record.Name {
			update.Remove = append(update.Remove, record)
		}
	}

	return c.DNSRecord.Update(d.Get("zone").(string), update)
}

func resourcePreviderDNSRecordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	c := meta.(*client.BaseClient)

	zone, err := c.DNSRecord.Get(d.Get("zone").(string))
	if err != nil {
		return false, err
	}

	for _, record := range zone.Records {
		if d.Get("type").(string) == record.Type && d.Get("name").(string) == record.Name {
			return true, nil
		}
	}

	return false, nil
}
