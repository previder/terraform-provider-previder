package kubernetes_cluster

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
)

const SecureVaultResourceType = "previder_kubernetes_secure_vault"

var _ resource.Resource = (*secureVaultResourceImpl)(nil)
var _ resource.ResourceWithConfigure = (*secureVaultResourceImpl)(nil)
var _ resource.ResourceWithImportState = (*secureVaultResourceImpl)(nil)

type secureVaultResourceImpl struct {
	client *client.PreviderClient
}

func (r *secureVaultResourceImpl) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = SecureVaultResourceType
}

func NewSecureVaultResource() resource.Resource {
	return &secureVaultResourceImpl{}
}

func (r *secureVaultResourceImpl) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	var newDiags diag.Diagnostics
	r.client, newDiags = util.ConfigureClient(req.ProviderData)
	resp.Diagnostics.Append(newDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *secureVaultResourceImpl) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema.Attributes = map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "ID of the secure vault",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required: true,
		},
		"group": schema.StringAttribute{
			MarkdownDescription: "ID or name of the group this secure vault belongs to",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"state": schema.StringAttribute{
			Computed: true,
		},
		"termination_protected": schema.BoolAttribute{
			MarkdownDescription: "Protect the secure vault from being deleted",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
		"token": schema.StringAttribute{
			MarkdownDescription: "Bootstrap token returned by the API at creation and stored in Terraform state",
			Computed:            true,
			Sensitive:           true,
		},
	}
}

func (r *secureVaultResourceImpl) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, data secureVaultResourceData

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	create := client.KubernetesSecureVaultCreate{
		Name:                 plan.Name.ValueString(),
		Group:                plan.Group.ValueString(),
		TerminationProtected: plan.TerminationProtected.ValueBool(),
	}

	response, err := r.client.KubernetesCluster.CreateSecureVault(create)
	if err != nil {
		resp.Diagnostics.AddError("Error creating secure vault", fmt.Sprintf("Error while creating secure vault (%s): %s", plan.Name.ValueString(), err))
		return
	}

	data = plan
	data.Id = types.StringValue(response.Id)
	data.Token = types.StringValue(response.Token)
	data.State = types.StringNull()
	if data.Group.IsUnknown() {
		data.Group = types.StringNull()
	}
	if data.TerminationProtected.IsUnknown() {
		data.TerminationProtected = types.BoolValue(create.TerminationProtected)
	}
	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}()

	vault, err := r.client.KubernetesCluster.GetSecureVault(response.Id)
	if err != nil {
		resp.Diagnostics.AddError("Secure vault not found after creation", fmt.Sprintf("Error while getting secure vault %s: %s", response.Id, err))
		return
	}

	populateSecureVaultData(&data, vault)
}

func (r *secureVaultResourceImpl) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data secureVaultResourceData

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vault, err := r.client.KubernetesCluster.GetSecureVault(data.Id.ValueString())
	if err != nil {
		var apiErr *client.ApiError
		if errors.As(err, &apiErr) && apiErr.Code == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading secure vault", fmt.Sprintf("Error while getting secure vault %s: %s", data.Id, err))
		return
	}

	populateSecureVaultData(&data, vault)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *secureVaultResourceImpl) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan, data secureVaultResourceData

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := client.KubernetesSecureVaultCreate{
		Name:                 plan.Name.ValueString(),
		Group:                plan.Group.ValueString(),
		TerminationProtected: plan.TerminationProtected.ValueBool(),
	}

	err := r.client.KubernetesCluster.UpdateSecureVault(state.Id.ValueString(), update)
	if err != nil {
		resp.Diagnostics.AddError("Error updating secure vault", fmt.Sprintf("Error while updating secure vault %s: %s", state.Id, err))
		return
	}

	vault, err := r.client.KubernetesCluster.GetSecureVault(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Secure vault not found after update", fmt.Sprintf("Error while getting secure vault %s: %s", state.Id, err))
		return
	}

	data = plan
	data.Token = state.Token
	populateSecureVaultData(&data, vault)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *secureVaultResourceImpl) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secureVaultResourceData

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.KubernetesCluster.DeleteSecureVault(state.Id.ValueString())
	if err != nil {
		var apiErr *client.ApiError
		if !(errors.As(err, &apiErr) && apiErr.Code == 404) {
			resp.Diagnostics.AddError("Error deleting secure vault", fmt.Sprintf("Error while deleting secure vault %s: %s", state.Id, err))
		}
	}
}

func (r *secureVaultResourceImpl) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data secureVaultResourceData

	vault, err := r.client.KubernetesCluster.GetSecureVault(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading secure vault", fmt.Sprintf("Error while getting secure vault %s: %s", req.ID, err))
		return
	}

	populateSecureVaultData(&data, vault)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
