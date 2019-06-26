package provider

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/previder/previder-go-sdk/client"
	"log"
	"strconv"
	"strings"
	"time"
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
							Optional: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"label": {
							Type:     schema.TypeString,
							Required: true,
						},
						"uuid": {
							Type:     schema.TypeString,
							Computed: true,
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
							Type:     schema.TypeInt,
							Computed: true,
						},
						"network": {
							Type:     schema.TypeString,
							Required: true,
						},
						"connected": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"primary": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"ipv4_address": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
							Computed: true,
						},
						"ipv6_address": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"mac_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"label": {
							Type:     schema.TypeString,
							Optional: true,
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
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
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
		vm.Memory = uint64(attr.(int))
	}
	if attr, ok := d.GetOk("user_data"); ok {
		vm.UserData = attr.(string)
	}
	if attr, ok := d.GetOk("provisioning_type"); ok {
		vm.ProvisioningType = attr.(string)
	}
	if attr, ok := d.GetOk("template"); ok {
		vm.Template = attr.(string)
	}
	if attr, ok := d.GetOk("cluster"); ok {
		vm.ComputeCluster = attr.(string)
	}
	if attr, ok := d.GetOk("tags"); ok {
		tfTags := attr.(*schema.Set).List()
		tags := make([]string, len(tfTags))
		for i, tag := range tfTags {
			tags[i] = tag.(string)
		}
		vm.Tags = tags
	}

	if networkInterfaceList, ok := d.GetOk("network_interface"); ok {
		vm.NetworkInterfaces = make([]client.NetworkInterface, len(networkInterfaceList.([]interface{})))
		for i, vL := range networkInterfaceList.([]interface{}) {
			networkInterface := vL.(map[string]interface{})
			if v, ok := networkInterface["network"].(string); ok && v != "" {
				vm.NetworkInterfaces[i].Network = v
			}
			if v, ok := networkInterface["connected"].(bool); ok {
				vm.NetworkInterfaces[i].Connected = v
			} else {
				vm.NetworkInterfaces[i].Connected = true
			}

			if v, ok := networkInterface["label"].(string); ok && v != "" {
				vm.NetworkInterfaces[i].Label = v
			}

			if _, ok := networkInterface["primary"].(bool); ok {
				_ = d.Set("primaryNetworkInterfaceIdx", i)
			}
		}
	} else {
		var networkInterface client.NetworkInterface
		networkInterface.Network = defaultNetworkName
		networkInterface.Connected = true
		vm.NetworkInterfaces = []client.NetworkInterface{networkInterface}
	}

	if disks, ok := d.GetOk("disk"); ok {
		vm.Disks = make([]client.Disk, len(disks.([]interface{})))
		for i, vL := range disks.([]interface{}) {
			disk := vL.(map[string]interface{})

			if v, ok := disk["id"].(string); ok && len(v) > 0 {
				vm.Disks[i].Id = new(int)
				*vm.Disks[i].Id, _ = strconv.Atoi(v)
			}

			if v, ok := disk["size"].(int); ok && v > 0 {
				vm.Disks[i].Size = uint64(v)
			}

			if v, ok := disk["label"].(string); ok && len(v) > 0 {
				vm.Disks[i].Label = v
			}
		}
	}

	task, err := c.VirtualMachine.Create(&vm)
	if err != nil {
		return fmt.Errorf(
			"Error while creating VirtualMachine (%s): %s", d.Id(), err)
	}

	_, _ = c.Task.WaitFor(task.Id, 5*time.Minute)
	d.SetId(task.VirtualMachine)

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

	vmJson, _ := json.Marshal(vm)
	log.Printf("[DEBUG] Virtual Server read configuration: %s", string(vmJson))

	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("invalid VirtualMachine id: %v", err)
	}

	_ = d.Set("name", vm.Name)
	_ = d.Set("memory", vm.Memory)
	_ = d.Set("cpucores", vm.CpuCores)
	_ = d.Set("template", vm.Template)
	_ = d.Set("cluster", vm.ComputeCluster)
	_ = d.Set("state", vm.State)
	_ = d.Set("termination_protection", vm.TerminationProtectionEnabled)
	_ = d.Set("initial_password", vm.InitialPassword)

	var primaryNetworkInterfaceIdx = 0
	if idx, ok := d.GetOk("primaryNetworkInterfaceIdx"); ok {
		primaryNetworkInterfaceIdx = idx.(int)
	}

	disks := make([]map[string]interface{}, len(vm.Disks))

	for vmIdx, vmDisk := range vm.Disks {
		if tfDisks, ok := d.GetOk("disk"); ok {
			for _, tfDisk := range tfDisks.([]interface{}) {
				disk := tfDisk.(map[string]interface{})
				if disk["uuid"] == vmDisk.Uuid || disk["label"] == vmDisk.Label {
					disks[vmIdx] = make(map[string]interface{})
					disks[vmIdx]["id"] = strconv.Itoa(*vmDisk.Id)
					disks[vmIdx]["size"] = vmDisk.Size
					disks[vmIdx]["uuid"] = vmDisk.Uuid
					disks[vmIdx]["label"] = vmDisk.Label
				}
			}
		}
	}
	err = d.Set("disk", disks)

	networkInterfaces := make([]map[string]interface{}, len(vm.NetworkInterfaces))
	for i, v := range vm.NetworkInterfaces {
		networkInterfaces[i] = make(map[string]interface{})
		networkInterfaces[i]["id"] = i
		networkInterfaces[i]["network"] = v.Network
		networkInterfaces[i]["mac_address"] = v.MacAddress
		networkInterfaces[i]["label"] = v.Label
		networkInterfaces[i]["connected"] = v.Connected

		ipv4Address := make([]string, 0)
		ipv6Address := make([]string, 0)
		for _, a := range v.AssignedAddresses {
			if strings.Contains(a, ":") {
				ipv6Address = append(ipv6Address, a)
			} else {
				ipv4Address = append(ipv4Address, a)
			}
		}

		networkInterfaces[i]["ipv4_address"] = ipv4Address
		networkInterfaces[i]["ipv6_address"] = ipv6Address

		if primaryNetworkInterfaceIdx == i {
			networkInterfaces[i]["primary"] = true

			d.Set("ipv4_address", networkInterfaces[0]["ipv4_address"].([]string)[0])
			d.SetConnInfo(map[string]string{
				"type": "ssh",
				"host": networkInterfaces[0]["ipv4_address"].([]string)[0],
			})
		}
	}

	err = d.Set("network_interface", networkInterfaces)
	log.Printf("Err: %s", err)

	return nil
}

func resourcePreviderVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	var machineHasShutdown = false

	vm, err := c.VirtualMachine.Get(d.Id())
	if err != nil {
		return err
	}

	if d.HasChange("cluster") {
		vm.ComputeCluster = d.Get("cluster").(string)
	}

	if d.HasChange("cpucores") || d.HasChange("memory") || d.HasChange("name") {
		_ = gracefullyShutdownVirtualMachine(d, meta)
		machineHasShutdown = true

		// End V1 compatibility
		vm.CpuCores = d.Get("cpucores").(int)
		vm.Memory = d.Get("memory").(uint64)
		vm.Name = d.Get("name").(string)

		task, err := c.VirtualMachine.Update(d.Id(), vm)

		if err != nil {
			return fmt.Errorf(
				"Error waiting for update of cpuCores or memory of VirtualMachine (%s): %s", d.Id(), err)
		}

		_, _ = c.Task.WaitFor(task.Id, 5*time.Minute)

	}

	if d.HasChange("disk") {
		_, newDisks := d.GetChange("disk")
		// Added and changed disks
		for _, disk := range newDisks.([]interface{}) {
			found := false
			tfDisk := disk.(map[string]interface{})
			for _, vmDisk := range vm.Disks {
				if tfDisk["uuid"] == vmDisk.Uuid {
					vm.Disks[*vmDisk.Id].Size = uint64(tfDisk["size"].(int))
					vm.Disks[*vmDisk.Id].Label = tfDisk["label"].(string)
					found = true
				}
			}

			// If the disk was not found in API data, then its a new disk.
			if !found {
				d := client.Disk{}
				d.Size = uint64(tfDisk["size"].(int))
				d.Label = tfDisk["label"].(string)
				vm.Disks = append(vm.Disks, d)
			}
		}

		// Removed disks
		for _, vmDisk := range vm.Disks {
			found := false
			for _, disk := range newDisks.([]interface{}) {
				tfDisk := disk.(map[string]interface{})
				if vmDisk.Uuid == tfDisk["uuid"] {
					found = true
				}

			}

			if !found {
				i := *vmDisk.Id
				vm.Disks = append(vm.Disks[:i], vm.Disks[i+1:]...)
			}
		}
	}

	if d.HasChange("network_interface") {
		_, newNetworkInterfaces := d.GetChange("network_interface")
		for _, nic := range newNetworkInterfaces.([]interface{}) {
			tfNic := nic.(map[string]interface{})
			found := false
			for _, vmNic := range vm.NetworkInterfaces {
				if vmNic.MacAddress == tfNic["mac_address"] || vmNic.Label == tfNic["label"] {
					found = true
				}
			}

			if !found {
				n := client.NetworkInterface{}
				n.Network = tfNic["network"].(string)
				n.Primary = tfNic["primary"].(bool)
				n.Connected = tfNic["connected"].(bool)
				n.Label = tfNic["label"].(string)
				vm.NetworkInterfaces = append(vm.NetworkInterfaces, n)
			}
		}
	}

	log.Printf("[INFO] Updating VirtualMachine: %s", d.Id())
	task, err := c.VirtualMachine.Update(vm.Id, vm)

	if err != nil {
		return fmt.Errorf(
			"Error while updating VirtualMachine (%s): %s", d.Id(), err)
	}

	_, _ = c.Task.WaitFor(task.Id, 5*time.Minute)
	if machineHasShutdown == true {
		//	c.VirtualMachine.Control(d.Id(), client.VmActionPowerOn)
	}

	return resourcePreviderVirtualMachineRead(d, meta)

}

func resourcePreviderVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.BaseClient)

	_ = resourcePreviderVirtualMachineRead(d, meta)

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

	c.Task.WaitFor(task.Id, 30*time.Minute)

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
		log.Printf("invalid VirtualMachine id: %v", err)
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

	c.Task.WaitFor(task.Id, 5*time.Minute)

	_, err = WaitForVirtualMachineAttribute(d, "POWEREDOFF", []string{""}, "state", meta)
	if err != nil {
		fmt.Sprintf(
			"Error waiting for VirtualMachine (%s) to shutdown: %s", d.Id(), err.Error())
	}

	return nil

}
