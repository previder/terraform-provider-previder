package virtual_server

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
	"github.com/previder/terraform-provider-previder/internal/util/validators"
	"strings"
	"time"
)

const ResourceType = "previder_virtual_server"

var _ resource.Resource = (*resourceImpl)(nil)
var _ resource.ResourceWithConfigure = (*resourceImpl)(nil)
var _ resource.ResourceWithImportState = (*resourceImpl)(nil)

type resourceImpl struct {
	client *client.PreviderClient
}

func (r *resourceImpl) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = ResourceType
}

func NewResource() resource.Resource {
	return &resourceImpl{}
}

func (r *resourceImpl) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var newDiags diag.Diagnostics
	r.client, newDiags = util.ConfigureClient(req.ProviderData)
	resp.Diagnostics.Append(newDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceImpl) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema.Attributes = map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "ID of the virtual server",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required: true,
		},
		"compute_cluster": schema.StringAttribute{
			Required: true,
		},
		"memory": schema.Int64Attribute{
			Required: true,
		},
		"cpu_cores": schema.Int64Attribute{
			Required: true,
		},
		"cpu_sockets": schema.Int64Attribute{
			Optional: true,
		},
		"group": schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		// Maps are always ordered by key by Terraform
		"disks": schema.MapNestedAttribute{
			Required: true,
			Validators: []validator.Map{
				validators.MinMax(1, 16),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"size": schema.Int64Attribute{
						Required: true,
					},
					"label": schema.StringAttribute{
						Computed: true,
					},
					"uuid": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
		// Maps are always ordered by key by Terraform
		"network_interfaces": schema.MapNestedAttribute{
			Required: true,
			Validators: []validator.Map{
				validators.MinMax(1, 8),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"network": schema.StringAttribute{
						Required: true,
					},
					"connected": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(true),
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"type": schema.StringAttribute{
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"assigned_addresses": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"discovered_addresses": schema.ListAttribute{
						ElementType: types.StringType,
						Computed:    true,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"ipv4_address": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"ipv6_address": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"mac_address": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"label": schema.StringAttribute{
						Computed: true,
					},
				},
			},
		},
		"template": schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"guest_id": schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"source": schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"user_data": schema.StringAttribute{
			Optional: true,
		},
		"termination_protection": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"state": schema.StringAttribute{
			Computed: true,
		},
		"provisioning_type": schema.StringAttribute{
			Computed: false,
			Optional: true,
		},
		"initial_password": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"tags": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
	}
}

func (r *resourceImpl) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var create client.VirtualMachineCreate
	var plan, data resourceData

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	create.Name = plan.Name.ValueString()
	create.CpuCores = int(plan.CpuCores.ValueInt64())
	create.Memory = uint64(plan.Memory.ValueInt64())
	create.Group = plan.Group.ValueString()
	create.UserData = plan.UserData.ValueString()
	create.ProvisioningType = plan.ProvisioningType.ValueString()
	create.PowerOnAfterClone = true

	if !validVirtualServerSource(plan) {
		resp.Diagnostics.AddError("Error while creating Virtual Server", fmt.Sprintf("Either template, guest_id or source has to be provided, only 1 value allowed"))
		return
	}

	var existingDisks []client.Disk
	if !plan.Template.IsNull() && plan.Template.ValueString() != "" {
		create.Template = plan.Template.ValueString()
	} else if !plan.GuestId.IsNull() && plan.GuestId.ValueString() != "" {
		create.GuestId = plan.GuestId.ValueString()
	} else if !plan.Source.IsNull() && plan.Source.ValueString() != "" {
		create.SourceVirtualMachine = plan.Source.ValueString()
		sourceVm, err := r.client.VirtualServer.Get(plan.Source.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Virtual server could not be found after creation", fmt.Sprintf("Error while creating VirtualMachine (%s): %s", plan.Name.ValueString(), err))
			return
		}
		for _, existingDisk := range sourceVm.Disks {
			existingDisks = append(existingDisks, existingDisk)
		}
	} else {
		resp.Diagnostics.AddError("Error while creating Virtual Server", fmt.Sprintf("Either template, guest_id or source has to be provided"))
		return
	}
	create.ComputeCluster = plan.ComputeCluster.ValueString()

	var createTags []string
	for _, tag := range plan.Tags {
		createTags = append(createTags, tag.ValueString())
	}
	create.Tags = createTags

	var networkInterfaceKeys []string
	for k := range plan.NetworkInterfaces {
		networkInterfaceKeys = append(networkInterfaceKeys, k)
	}
	var createNetworkInterfaces []client.NetworkInterface
	for k, v := range plan.NetworkInterfaces {
		plannedNetworkInterface := v
		createNetworkInterfaces = append(createNetworkInterfaces, client.NetworkInterface{
			Network:   plannedNetworkInterface.Network.ValueString(),
			Connected: plannedNetworkInterface.Connected.ValueBool(),
			Label:     k,
			Type:      plannedNetworkInterface.Type.ValueString(),
		})
	}
	create.NetworkInterfaces = createNetworkInterfaces

	var createDisks []client.Disk
	for k, v := range plan.Disks {
		plannedDisk := v
		var newDiskId string
		if len(existingDisks) > len(createDisks) {
			existingDisk := existingDisks[len(createDisks)]
			if plannedDisk.Size.ValueInt64() < int64(existingDisk.Size) {
				resp.Diagnostics.AddError("Disks cannot be smaller when cloning", fmt.Sprintf("Disk %s is smaller than source virtual server disk", k))
				return
			}
			newDiskId = existingDisk.Id
		}
		createDisks = append(createDisks, client.Disk{
			Id:    newDiskId,
			Size:  uint64(plannedDisk.Size.ValueInt64()),
			Label: k,
		})
	}

	create.Disks = createDisks

	task, err := r.client.VirtualServer.Create(&create)
	if err != nil {
		resp.Diagnostics.AddError("Error while creating Virtual Server", fmt.Sprintf("Error while creating VirtualMachine (%s): %s", plan.Name.ValueString(), err))
		return
	}

	_, _ = r.client.Task.WaitFor(task.Id, 5*time.Minute)

	vm, err := r.client.VirtualServer.Get(task.VirtualMachine)
	if err != nil {
		resp.Diagnostics.AddError("Virtual server could not be found after creation", fmt.Sprintf("Error while creating VirtualMachine (%s): %s", plan.Name.ValueString(), err))
		return
	}

	populateResourceData(ctx, &data, vm)

	data.Id = types.StringValue(task.VirtualMachine)
	if plan.Source.IsNull() || plan.Source.ValueString() == "" {
		data.Source = types.StringNull()
	} else {
		data.Source = plan.Source
	}

	if len(vm.Template) == 0 {
		// Guest ID or sourceVirtualMachine should stay poweredoff
		err = waitForVirtualServerState(r.client, data.Id.ValueString(), client.VmStatePoweredOff)
	} else {
		// Template should always power on
		err = waitForVirtualServerState(r.client, data.Id.ValueString(), client.VmStatePoweredOn)
	}

	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for Virtual Server (%s) to become ready: %s", data.Id, err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state, data resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Retrieve the VirtualMachine properties for updating the state
	vm, err := r.client.VirtualServer.Get(state.Id.ValueString())

	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			resp.Diagnostics.AddError("Server not found", fmt.Sprintf("Error while getting Virtual Server: %s", data.Id))
			return
		}
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error while fetching Virtual Server (%s): %s", data.Id, err))
		return
	}

	populateResourceData(ctx, &data, vm)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *resourceImpl) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan, data resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var machineHasShutdown = false
	var vm *client.VirtualMachineExt
	update := client.VirtualMachineUpdate{}

	vm, err := r.client.VirtualServer.Get(state.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Virtual server not found", fmt.Sprintf("Error while getting Virtual Server: %s", state.Id))
		return
	}

	update.Name = plan.Name.ValueString()
	update.ComputeCluster = plan.ComputeCluster.ValueString()
	update.Group = plan.Group.ValueString()

	if plan.CpuCores.ValueInt64() != state.CpuCores.ValueInt64() ||
		plan.Memory.ValueInt64() != state.Memory.ValueInt64() {
		resp.Diagnostics.AddWarning("Virtual server shutdown", fmt.Sprintf("Virtual server shutdown to alter cpu cores or memory quantity %s", state.Id))

		err := gracefullyShutdownVirtualMachine(r.client, &resp.Diagnostics, vm.Id)

		if err != nil {
			return
		}
		machineHasShutdown = true

	}
	update.CpuCores = int(plan.CpuCores.ValueInt64())
	update.Memory = uint64(plan.Memory.ValueInt64())

	updateTags := make([]string, len(plan.Tags))
	for i, tag := range plan.Tags {
		updateTags[i] = tag.ValueString()
	}
	update.Tags = updateTags

	var updateDisks []client.DiskUpdate

	for k, v := range plan.Disks {
		plannedDisk := v
		var diskId string
		if existingDisk, ok := state.Disks[k]; ok {
			diskId = existingDisk.Id.ValueString()
		}
		updateDisks = append(updateDisks, client.DiskUpdate{
			Id:    diskId,
			Size:  uint64(plannedDisk.Size.ValueInt64()),
			Label: k,
			Uuid:  plannedDisk.Uuid.String(),
		})
	}

	for _, existingDisk := range state.Disks {
		var found bool
		for _, plannedDiskData := range updateDisks {
			if plannedDiskData.Label == existingDisk.Label.ValueString() {
				found = true
			}
		}
		if !found {
			resp.Diagnostics.AddWarning("removing disk from virtual server", fmt.Sprintf("Removing disk with label %s", existingDisk.Label.ValueString()))

			updateDisks = append(updateDisks, client.DiskUpdate{
				Id:     existingDisk.Id.ValueString(),
				Size:   uint64(existingDisk.Size.ValueInt64()),
				Label:  existingDisk.Label.ValueString(),
				Uuid:   existingDisk.Uuid.String(),
				Delete: true,
			})
		}
	}

	update.Disks = updateDisks

	var updateNetworkInterfaces []client.NetworkInterfaceUpdate

	for k, v := range plan.NetworkInterfaces {
		plannedNetworkInterface := v
		var networkInterfaceId string
		if existingNetworkInterface, ok := state.NetworkInterfaces[k]; ok {
			networkInterfaceId = existingNetworkInterface.Id.ValueString()
		}
		updateNetworkInterfaces = append(updateNetworkInterfaces, client.NetworkInterfaceUpdate{
			Id:        networkInterfaceId,
			Network:   plannedNetworkInterface.Network.ValueString(),
			Label:     k,
			Connected: plannedNetworkInterface.Connected.ValueBool(),
		})
	}

	for _, existingNetworkInterface := range state.NetworkInterfaces {
		var found bool
		for _, plannedNetworkInterfaceData := range updateNetworkInterfaces {
			if plannedNetworkInterfaceData.Label == existingNetworkInterface.Label.ValueString() {
				found = true
			}
		}
		if !found {
			resp.Diagnostics.AddWarning("removing network interface from virtual server", fmt.Sprintf("Removing network interface with label %s", existingNetworkInterface.Label.ValueString()))

			updateNetworkInterfaces = append(updateNetworkInterfaces, client.NetworkInterfaceUpdate{
				Id:      existingNetworkInterface.Id.ValueString(),
				Network: existingNetworkInterface.Network.ValueString(),
				Label:   existingNetworkInterface.Label.ValueString(),
				Deleted: true,
			})
		}
	}

	update.NetworkInterfaces = updateNetworkInterfaces

	update.TerminationProtectionEnabled = plan.TerminationProtection.ValueBool()

	task, err := r.client.VirtualServer.Update(state.Id.ValueString(), &update)

	if err != nil {
		resp.Diagnostics.AddError("Error updating virtual server", fmt.Sprintf("Virtual server has not been updated %s: %s", state.Name, err.Error()))
		return
	}

	_, _ = r.client.Task.WaitFor(task.Id, 5*time.Minute)
	if machineHasShutdown == true {
		task, err = r.client.VirtualServer.Control(state.Id.ValueString(), client.VmActionPowerOn)
		_, err = r.client.Task.WaitFor(task.Id, 5*time.Minute)
		if err != nil {
			return
		}
		resp.Diagnostics.AddWarning("Virtual server powered on", fmt.Sprintf("Virtual server poweredon after altering cpu cores or memory quantity %s", state.Id))

		err = waitForVirtualServerState(r.client, state.Id.ValueString(), client.VmStatePoweredOn)
		if err != nil {
			resp.Diagnostics.AddError("Error while waiting for poweredon", fmt.Sprintf("Virtual Server is not poweredon: %s", err.Error()))
		}
	}

	vm, err = r.client.VirtualServer.Get(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Virtual server could not be found after update", fmt.Sprintf("Error while updating VirtualMachine (%s): %s", plan.Name.ValueString(), err))
		return
	}

	populateResourceData(ctx, &data, vm)
	data.Source = plan.Source
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *resourceImpl) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var vm, _ = r.client.VirtualServer.Get(state.Id.ValueString())

	if vm.TerminationProtectionEnabled == true {
		resp.Diagnostics.AddError("Virtual server not deleted", "Virtual Server is locked, skipping status check and retrying")
		return
	}

	// Destroy the Virtual Server
	task, err := r.client.VirtualServer.Delete(state.Id.ValueString())

	// Handle remotely destroyed Virtual Servers
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			resp.Diagnostics.AddError("Virtual server not found", fmt.Sprintf("Virtual Server is not found: %s", state.Name))

		}
		resp.Diagnostics.AddError("Virtual server not deleted", fmt.Sprintf("Virtual Server is not deleted: %s %s", state.Name, err))
		return
	}

	r.client.Task.WaitFor(task.Id, 30*time.Minute)

}

func (r *resourceImpl) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resourceData

	var vm, _ = r.client.VirtualServer.Get(req.ID)

	populateResourceData(ctx, &data, vm)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func validVirtualServerSource(data resourceData) bool {
	count := 0
	if !data.Template.IsNull() && data.Template.ValueString() != "" {
		count++
	}
	if !data.GuestId.IsNull() && data.GuestId.ValueString() != "" {
		count++
	}
	if !data.GuestId.IsNull() && data.Source.ValueString() != "" {
		count++
	}
	return count == 1
}

func waitForVirtualServerState(client *client.PreviderClient, id string, target string) error {

	backoffOperation := func() error {
		vm, err := client.VirtualServer.Get(id)

		if err != nil {
			return errors.New(fmt.Sprintf("invalid Virtual Server id: %s", id))
		}
		if vm.State != target {
			return errors.New(fmt.Sprintf("Waiting for Virtual Server to become ready: %s", id))
		}
		return nil
	}
	// Max waiting time is 10 mins
	backoffConfig := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second*5), 120)

	err := backoff.Retry(backoffOperation, backoffConfig)
	if err != nil {
		return err
	}

	return nil
}

func gracefullyShutdownVirtualMachine(baseClient *client.PreviderClient, diag *diag.Diagnostics, id string) error {

	vm, err := baseClient.VirtualServer.Get(id)
	if vm.State == client.VmStatePoweredOff {
		return nil
	}

	task, err := baseClient.VirtualServer.Control(vm.Id, client.VmActionShutdown)
	if err != nil {
		diag.AddError("Virtual server not shutting down", fmt.Sprintf("Virtual Server is not shutting down after shutdown command: %s", err.Error()))
		task, err = baseClient.VirtualServer.Control(vm.Id, client.VmActionPowerOff)
	}

	_, err = baseClient.Task.WaitFor(task.Id, 5*time.Minute)
	if err != nil {
		diag.AddError("Virtual server power off failed", fmt.Sprintf("Virtual Server is not powering off after poweroff command: %s", err.Error()))
		return err
	}

	err = waitForVirtualServerState(baseClient, id, client.VmStatePoweredOff)
	if err != nil {
		diag.AddError("Error while waiting for shutdown", fmt.Sprintf("Virtual Server is not shutting down after shutdown command: %s", err.Error()))
	}

	return nil

}
