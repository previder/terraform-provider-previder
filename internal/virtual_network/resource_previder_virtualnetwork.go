package virtual_network

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
	"log"
	"strings"
	"time"
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
			MarkdownDescription: "ID of the virtual server",
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
	}
}

func (r *resourceImpl) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var create client.VirtualNetworkUpdate
	var plan, data resourceData

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	create.Name = plan.Name.ValueString()
	create.Type = plan.Type.ValueString()
	create.Group = plan.Group.ValueString()

	task, err := r.client.VirtualNetwork.Create(&create)
	if err != nil {
		resp.Diagnostics.AddError("Error while creating Virtual Server", fmt.Sprintf("Error while creating Virtual Network (%s): %s", plan.Name.ValueString(), err))
		return
	}

	_, _ = r.client.Task.WaitFor(task.Id, 5*time.Minute)

	network, err := r.client.VirtualNetwork.Get(task.VirtualNetwork)
	if err != nil {
		resp.Diagnostics.AddError("Virtual Network could not be found after creation", fmt.Sprintf("Error while creating Virtual Network (%s): %s", plan.Name.ValueString(), err))
		return
	}

	data.Id = types.StringValue(task.VirtualNetwork)
	err = waitForVirtualNetworkState(*r.client, data.Id.ValueString(), client.VirtualNetworkStateReady)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for Virtual Network (%s) to become ready: %s", data.Id, err))
		return
	}

	populateResourceData(&data, network, &plan)

	log.Printf("Searching for ID %s", data.Id.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state, data resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Retrieve the Virtual Network properties for updating the state
	network, err := r.client.VirtualNetwork.Get(state.Id.ValueString())

	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			resp.Diagnostics.AddError("Virtual network not found", fmt.Sprintf("Error while getting Virtual network: %s", data.Id))
			return
		}
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error while fetching Virtual Network (%s): %s", data.Id, err))
		return
	}

	populateResourceData(&data, network, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan, data resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var vm *client.VirtualNetwork
	update := client.VirtualNetworkUpdate{}

	vm, err := r.client.VirtualNetwork.Get(state.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Virtual server not found", fmt.Sprintf("Error while getting Virtual Server: %s", state.Id))
		return
	}

	update.Name = plan.Name.ValueString()
	update.Type = plan.Type.ValueString()

	task, err := r.client.VirtualNetwork.Update(state.Id.ValueString(), &update)

	if err != nil {
		resp.Diagnostics.AddError("Error updating Virtual Network", fmt.Sprintf("Virtual Network has not been updated %s: %s", state.Name, err.Error()))
		return
	}
	_, _ = r.client.Task.WaitFor(task.Id, 5*time.Minute)

	vm, err = r.client.VirtualNetwork.Get(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Virtual Network could not be found after update", fmt.Sprintf("Error while updating Virtual Network (%s): %s", plan.Name.ValueString(), err))
		return
	}

	populateResourceData(&data, vm, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

	var network, _ = r.client.VirtualNetwork.Get(req.ID)

	populateResourceData(&data, network, nil)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func waitForVirtualNetworkState(client client.PreviderClient, id string, target string) error {

	backoffOperation := func() error {
		network, err := client.VirtualNetwork.Get(id)

		if err != nil {
			return errors.New(fmt.Sprintf("invalid Virtual Network id: %s", id))
		}
		if network.State != target {
			return errors.New(fmt.Sprintf("Waiting for Virtual Network to become ready: %s", id))
		}
		return nil
	}
	// Max waiting time is 10 mins
	backoffConfig := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second*5), 30)

	err := backoff.Retry(backoffOperation, backoffConfig)
	if err != nil {
		return err
	}

	return nil
}
