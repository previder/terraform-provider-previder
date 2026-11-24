package virtual_network

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
)

const ResourceType = "previder_virtual_network"

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
			MarkdownDescription: "ID of the virtual network",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required: true,
		},
		"type": schema.StringAttribute{
			Required: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"group": schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"address_pools": schema.MapNestedAttribute{
			MarkdownDescription: "Map of address pools on the virtual network. The map keys are Terraform-only stable identifiers and are not sent to the Previder API",
			Optional:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: addressPoolSchemaAttributes(),
			},
		},
	}
}

func (r *resourceImpl) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateAddressPools(plan.AddressPools); err != nil {
		resp.Diagnostics.AddError("Error creating virtual network", err.Error())
		return
	}

	task, err := r.client.VirtualNetwork.Create(&client.VirtualNetworkUpdate{
		Name:  plan.Name.ValueString(),
		Type:  plan.Type.ValueString(),
		Group: plan.Group.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating virtual network", fmt.Sprintf("Creating virtual network %s: %s", plan.Name, err))
		return
	}

	data := plan
	data.Id = types.StringValue(task.VirtualNetwork)
	data.AddressPools = nil
	if data.Group.IsUnknown() {
		data.Group = types.StringNull()
	}
	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}()

	if _, err := r.client.Task.WaitFor(task.Id, 5*time.Minute); err != nil {
		resp.Diagnostics.AddError("Error creating virtual network", fmt.Sprintf("Waiting for virtual network %s: %s", data.Id, err))
		return
	}
	network, err := waitForVirtualNetworkReady(ctx, r.client.VirtualNetwork, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating virtual network", fmt.Sprintf("Waiting for virtual network %s to become ready: %s", data.Id, err))
		return
	}
	populateResourceData(&data, network)

	data.AddressPools, err = syncAddressPools(r.client.VirtualNetwork, data.Id.ValueString(), plan.AddressPools, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating address pools", fmt.Sprintf("Virtual network %s: %s", data.Id, err))
	}
}

func (r *resourceImpl) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	network, err := r.client.VirtualNetwork.Get(data.Id.ValueString())
	if err != nil {
		var apiErr *client.ApiError
		if errors.As(err, &apiErr) && apiErr.Code == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading virtual network", fmt.Sprintf("Reading virtual network %s: %s", data.Id, err))
		return
	}

	data.AddressPools, err = refreshAddressPools(r.client.VirtualNetwork, data.Id.ValueString(), data.AddressPools)
	if err != nil {
		resp.Diagnostics.AddError("Error reading address pools", fmt.Sprintf("Virtual network %s: %s", data.Id, err))
		return
	}
	populateResourceData(&data, network)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var prior, plan resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := validateAddressPools(plan.AddressPools); err != nil {
		resp.Diagnostics.AddError("Error updating virtual network", err.Error())
		return
	}

	networkID := prior.Id.ValueString()
	task, err := r.client.VirtualNetwork.Update(networkID, &client.VirtualNetworkUpdate{
		Name: plan.Name.ValueString(),
		Type: plan.Type.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating virtual network", fmt.Sprintf("Updating virtual network %s: %s", networkID, err))
		return
	}
	if _, err := r.client.Task.WaitFor(task.Id, 5*time.Minute); err != nil {
		resp.Diagnostics.AddError("Error updating virtual network", fmt.Sprintf("Waiting for virtual network %s: %s", networkID, err))
		return
	}

	network, err := r.client.VirtualNetwork.Get(networkID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading virtual network after update", fmt.Sprintf("Reading virtual network %s: %s", networkID, err))
		return
	}

	data := plan
	populateResourceData(&data, network)
	data.AddressPools, err = syncAddressPools(r.client.VirtualNetwork, networkID, plan.AddressPools, prior.AddressPools)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating address pools", fmt.Sprintf("Virtual network %s: %s", networkID, err))
	}
}

func (r *resourceImpl) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Destroy the Virtual Networks
	task, err := r.client.VirtualNetwork.Delete(state.Id.ValueString())

	// Handle remotely destroyed Virtual Networks
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			resp.Diagnostics.AddError("Virtual network not found", fmt.Sprintf("Virtual network is not found: %s", state.Name))
		}
		resp.Diagnostics.AddError("Virtual network not deleted", fmt.Sprintf("Virtual network is not deleted: %s %s", state.Name, err))
		return
	}

	_, err = r.client.Task.WaitFor(task.Id, 30*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Virtual network not deleted", fmt.Sprintf("Virtual network is not deleted: %s", err.Error()))
		return
	}

}

func (r *resourceImpl) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resourceData

	network, err := r.client.VirtualNetwork.Get(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing virtual network", fmt.Sprintf("Reading virtual network %s: %s", req.ID, err))
		return
	}
	populateResourceData(&data, network)

	pools, err := r.client.VirtualNetwork.ListAddressPools(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing address pools", fmt.Sprintf("Reading address pools for virtual network %s: %s", req.ID, err))
		return
	}
	if len(*pools) > 0 {
		data.AddressPools = make(map[string]resourceDataAddressPool, len(*pools))
	}
	for _, pool := range *pools {
		if _, exists := data.AddressPools[pool.Id]; pool.Id == "" || exists {
			resp.Diagnostics.AddError("Error importing address pools", fmt.Sprintf("Virtual network %s returned a missing or duplicate address pool ID %q.", req.ID, pool.Id))
			return
		}
		data.AddressPools[pool.Id] = flattenAddressPool(pool, resourceDataAddressPool{})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func waitForVirtualNetworkReady(ctx context.Context, service client.VirtualNetworkService, id string) (*client.VirtualNetwork, error) {
	retry := backoff.WithMaxRetries(backoff.NewConstantBackOff(5*time.Second), 30)
	return backoff.RetryWithData(func() (*client.VirtualNetwork, error) {
		network, err := service.Get(id)
		if err != nil {
			return nil, err
		}
		if network.State != client.VirtualNetworkStateReady {
			return nil, fmt.Errorf("virtual network state is %s", network.State)
		}
		return network, nil
	}, backoff.WithContext(retry, ctx))
}
