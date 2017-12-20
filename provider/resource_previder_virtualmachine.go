package provider

import (
	"fmt"
	"log"
	"strings"
	"time"

	"encoding/json"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/previder/previder-go-sdk"
)

func resourcePreviderVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourcePreviderVirtualMachineCreate,
		Read:   resourcePreviderVirtualMachineRead,
		Update: resourcePreviderVirtualMachineUpdate,
		Delete: resourcePreviderVirtualMachineDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cluster": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"memory": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"cpucores": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"disk": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 8,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"network_interface": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 8,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"network": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ipv4_address": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"ipv6_address": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
					},
				},
			},
			"template": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, o, n string, d *schema.ResourceData) bool {
					oTmpl := strings.SplitN(o, " ", 2)
					nTmpl := strings.SplitN(n, " ", 2)
					if len(nTmpl) == 1 && oTmpl[0] == nTmpl[0] {
						return true
					}
					return o == n
				},
			},
			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"termination_protection": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"first_ipv4_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"provisioning_type": {
				Type:     schema.TypeString,
				Computed: false,
				Optional: true,
			},
			"initial_password": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePreviderVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	// Set default values if not set
	if _, ok := d.GetOk("cluster"); !ok {
		d.Set("cluster", defaultClusterName)
	}
	if _, ok := d.GetOk("template"); !ok {
		d.Set("template", defaultTemplateName)
	}

	var vm client.VirtualMachineCreate

	if attr, ok := d.GetOk("name"); ok {
		vm.Name = attr.(string)
	}
	if attr, ok := d.GetOk("cpucores"); ok {
		vm.CpuCores = attr.(int)
	}
	if attr, ok := d.GetOk("memory"); ok {
		vm.MemoryMb = attr.(int)
	}
	if attr, ok := d.GetOk("user_data"); ok {
		vm.UserData = attr.(string)
	}
	if attr, ok := d.GetOk("provisioning_type"); ok {
		vm.ProvisioningType = attr.(string)
	}
	if vL, ok := d.GetOk("network_interface"); ok {
		vm.NetworkInterfaces = make([]client.NetworkInterface, len(vL.([]interface{})))
		for i, vL := range vL.([]interface{}) {
			networkInterface := vL.(map[string]interface{})
			if v, ok := networkInterface["network"].(string); ok && v != "" {
				network, err := c.VirtualNetwork.Get(v)
				if err != nil {
					return err
				}
				vm.NetworkInterfaces[i].Network = *network
			}
		}
	} else {
		network, err := c.VirtualNetwork.Get(defaultNetworkName)
		if err != nil {
			return err
		}
		var networkInterface client.NetworkInterface
		networkInterface.Network = *network
		vm.NetworkInterfaces = []client.NetworkInterface{networkInterface}
	}

	if vL, ok := d.GetOk("disk"); ok {
		vm.VirtualDisks = make([]client.VirtualDisk, len(vL.([]interface{})))
		for i, vL := range vL.([]interface{}) {
			disk := vL.(map[string]interface{})
			if v, ok := disk["size"].(int); ok && v > 0 {
				vm.VirtualDisks[i].DiskSizeMb = v
			}
		}
	}

	if attr, ok := d.GetOk("cluster"); ok {
		computeCluster, err := c.VirtualMachine.ComputeClusterGet(attr.(string))
		if err != nil {
			return err
		}
		vm.ComputeCluster = *computeCluster
	}

	if attr, ok := d.GetOk("template"); ok {
		template, err := c.VirtualMachine.VirtualMachineTemplateGet(attr.(string))
		if err != nil {
			return err
		}
		vm.Template = *template
	}

	vmJson, _ := json.Marshal(vm)
	log.Printf("[DEBUG] Virtual Server create configuration: %s", string(vmJson))

	task, err := c.VirtualMachine.Create(&vm)
	if err != nil {
		return fmt.Errorf(
			"Error while creating VirtualMachine (%s): %s", d.Id(), err)
	}

	c.Task.WaitForTask(task, 5*time.Minute)
	d.SetId(task.ConfigurationItem.Id)

	_, err = WaitForVirtualMachineAttribute(d, "POWEREDON", []string{"NEW", "DEPLOYING"}, "state", meta)
	if err != nil {
		return fmt.Errorf(
			"Error waiting for VirtualMachine (%s) to become ready: %s", d.Id(), err)
	}

	return resourcePreviderVirtualMachineRead(d, meta)
}

func resourcePreviderVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	// Retrieve the VirtualMachine properties for updating the state
	vm, err := c.VirtualMachine.Get(d.Id())

	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("invalid VirtualMachine id: %v", err)
	}

	d.Set("name", vm.Name)
	d.Set("memory", vm.MemoryMb)
	d.Set("cpucores", vm.CpuCores)
	d.Set("template", vm.Template.Name)
	d.Set("cluster", vm.ComputeCluster.Name)
	d.Set("state", vm.State)
	d.Set("termination_protection", vm.TerminationProtection)
	d.Set("initial_password", vm.InitialPassword)

	disks := make([]map[string]interface{}, len(vm.VirtualDisks))
	for i, v := range vm.VirtualDisks {
		disks[i] = make(map[string]interface{})
		disks[i]["id"] = v.Id
		disks[i]["size"] = v.DiskSizeMb
	}
	d.Set("disk", disks)

	networkInterfaces := make([]map[string]interface{}, len(vm.NetworkInterfaces))
	for i, v := range vm.NetworkInterfaces {
		networkInterfaces[i] = make(map[string]interface{})
		networkInterfaces[i]["id"] = v.Id
		networkInterfaces[i]["network"] = v.Network.Name
		ipv4_address := make([]string, 0)
		ipv6_address := make([]string, 0)
		for _, a := range v.AddressAssignments {
			if a.Type == "IPV4" {
				ipv4_address = append(ipv4_address, a.Address)
			} else if a.Type == "IPV6" {
				ipv6_address = append(ipv6_address, a.Address)
			}
		}
		networkInterfaces[i]["ipv4_address"] = ipv4_address
		networkInterfaces[i]["ipv6_address"] = ipv6_address
	}
	d.Set("network_interface", networkInterfaces)

	if vm.NetworkInterfaces[0].FirstIPv4AddressAssignment.Address != "" {
		d.Set("ipv4_address", vm.NetworkInterfaces[0].FirstIPv4AddressAssignment.Address)
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": vm.NetworkInterfaces[0].FirstIPv4AddressAssignment.Address,
		})
	}

	return nil
}

func resourcePreviderVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	var machineHasShutdown bool = false

	vm, err := c.VirtualMachine.Get(d.Id())
	if err != nil {
		return err
	}

	if d.HasChange("cpucores") || d.HasChange("memory") || d.HasChange("name") {
		gracefullyShutdownVirtualMachine(d, meta)
		machineHasShutdown = true

		// V1 compatibility
		if attr, ok := d.GetOk("cluster"); ok {
			d.Set("cluster", attr)
		} else {
			d.Set("cluster", vm.ComputeCluster.Name)
		}
		computeCluster, err := c.VirtualMachine.ComputeClusterGet(d.Get("cluster").(string))

		if err != nil {
			return err
		}

		var update client.VirtualMachineUpdate
		update.ComputeCluster = *computeCluster
		update.Hostname = vm.Hostname

		// End V1 compatibility

		update.CpuCores = d.Get("cpucores").(int)
		update.MemoryMb = d.Get("memory").(int)
		update.Name = d.Get("name").(string)

		task, err := c.VirtualMachine.Update(d.Id(), &update)

		if err != nil {
			return fmt.Errorf(
				"Error waiting for update of cpuCores or memory of VirtualMachine (%s): %s", d.Id(), err)
		}
		c.Task.WaitForTask(task, 5*time.Minute)

	}

	if d.HasChange("disk") {
		oldDisks, newDisks := d.GetChange("disk")

		// Removed disks
		for _, oldDiskRaw := range oldDisks.([]interface{}) {
			oldDisk := oldDiskRaw.(map[string]interface{})
			found := false
			for _, newDiskRaw := range newDisks.([]interface{}) {
				newDisk := newDiskRaw.(map[string]interface{})
				if newDisk["id"] == oldDisk["id"] {
					found = true
				}
			}
			if !found {
				c.VirtualMachine.DeleteDisk(d.Id(), oldDisk["id"].(string))
			}
		}

		// Added disks
		for _, newDiskRaw := range newDisks.([]interface{}) {
			newDisk := newDiskRaw.(map[string]interface{})
			found := false
			for _, oldDiskRaw := range oldDisks.([]interface{}) {
				oldDisk := oldDiskRaw.(map[string]interface{})
				if newDisk["id"] == oldDisk["id"] {
					found = true
				}
			}
			if !found {
				var virtualDisk client.VirtualDisk
				virtualDisk.DiskSizeMb = newDisk["size"].(int)
				c.VirtualMachine.CreateDisk(d.Id(), &virtualDisk)
			}
		}

	}

	if d.HasChange("network_interface") {
		oldNetworkInterfaces, newNetworkInterfaces := d.GetChange("network_interface")

		// Removed network interfaces
		for _, oldNetworkInterfaceRaw := range oldNetworkInterfaces.([]interface{}) {
			oldNetworkInterface := oldNetworkInterfaceRaw.(map[string]interface{})
			found := false
			for _, newNetworkInterfaceRaw := range newNetworkInterfaces.([]interface{}) {
				newNetworkInterface := newNetworkInterfaceRaw.(map[string]interface{})
				if newNetworkInterface["id"] == oldNetworkInterface["id"] {
					found = true
				}
			}
			if !found {
				c.VirtualMachine.DeleteNetworkInterface(d.Id(), oldNetworkInterface["id"].(string))
			}
		}

		// Added network interfaces
		for _, newNetworkInterfaceRaw := range newNetworkInterfaces.([]interface{}) {
			newNetworkInterface := newNetworkInterfaceRaw.(map[string]interface{})
			found := false
			for _, oldNetworkInterfaceRaw := range oldNetworkInterfaces.([]interface{}) {
				oldNetworkInterface := oldNetworkInterfaceRaw.(map[string]interface{})
				if newNetworkInterface["id"] == oldNetworkInterface["id"] {
					found = true
				}
			}
			if !found {
				var virtualNetworkInterface client.NetworkInterface
				network, err := c.VirtualNetwork.Get(newNetworkInterface["network"].(string))
				if err != nil {
					return err
				}
				virtualNetworkInterface.Network = *network
				c.VirtualMachine.CreateNetworkInterface(d.Id(), &virtualNetworkInterface)
			}
		}

	}

	if machineHasShutdown == true {
		c.VirtualMachine.Control(d.Id(), client.VmActionPowerOn)
	}

	return resourcePreviderVirtualMachineRead(d, meta)
}

func resourcePreviderVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	resourcePreviderVirtualMachineRead(d, meta)

	log.Printf("[INFO] Deleting VirtualMachine: %s", d.Id())

	if d.Get("termination_protection").(bool) == true {
		log.Println("[INFO] VirtualMachine is locked, skipping status check and retrying")
		return nil
	}

	// Destroy the VirtualMachine
	task, err := c.VirtualMachine.Delete(d.Id())

	// Handle remotely destroyed VirtualMachines
	if err != nil && strings.Contains(err.Error(), "404 Not Found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error deleting VirtualMachine: %s", err)
	}

	c.Task.WaitForTask(task, 30*time.Minute)

	log.Printf("[INFO] Deletion task ID: %s", task.Id)

	return nil
}

func WaitForVirtualMachineAttribute(
	d *schema.ResourceData, target string, pending []string, attribute string, meta interface{}) (interface{}, error) {
	log.Printf(
		"[INFO] Waiting for VirtualMachine (%s) to have %s of %s",
		d.Id(), attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending:        pending,
		Target:         []string{target},
		Refresh:        newVirtualMachineStateRefreshFunc(d, target, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	return stateConf.WaitForState()
}

func newVirtualMachineStateRefreshFunc(
	d *schema.ResourceData, target string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		state := getVirtualMachineStatus(d, meta)

		if state == target {
			return d, state, nil
		}

		return nil, "", nil
	}
}

func getVirtualMachineStatus(d *schema.ResourceData, meta interface{}) string {
	c := meta.(*client.BaseClient)
	vm, err := c.VirtualMachine.Get(d.Id())

	if err != nil {
		fmt.Errorf("invalid VirtualMachine id: %v", err)
		return "UNKNOWN"
	}

	return vm.State
}

func gracefullyShutdownVirtualMachine(d *schema.ResourceData, meta interface{}) error {

	if getVirtualMachineStatus(d, meta) != "POWEREDON" {
		return nil
	}

	c := meta.(*client.BaseClient)
	task, err := c.VirtualMachine.Control(d.Id(), client.VmActionShutdown)
	if err != nil {
		fmt.Sprintf(
			"Error received while shutting down (%s) to shutdown: %s", d.Id(), err.Error())
		task, err = c.VirtualMachine.Control(d.Id(), client.VmActionPowerOff)
	}

	c.Task.WaitForTask(task, 5*time.Minute)

	_, err = WaitForVirtualMachineAttribute(d, "POWEREDOFF", []string{""}, "state", meta)
	if err != nil {
		fmt.Sprintf(
			"Error waiting for VirtualMachine (%s) to shutdown: %s", d.Id(), err.Error())
	}

	return nil

}
