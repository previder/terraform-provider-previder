package staas_environment

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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
	"github.com/previder/terraform-provider-previder/internal/util/validators"
	"log"
	"time"
)

const ResourceType = "previder_staas_environment"

var _ resource.Resource = (*resourceImpl)(nil)
var _ resource.ResourceWithConfigure = (*resourceImpl)(nil)
var _ resource.ResourceWithImportState = (*resourceImpl)(nil)

type resourceImpl struct {
	client *client.BaseClient
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
			MarkdownDescription: "ID of the STaas Environment",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required: true,
		},
		"cluster": schema.StringAttribute{
			Required: true,
		},
		"state": schema.StringAttribute{
			Computed: true,
		},
		"type": schema.StringAttribute{
			Required: true,
		},
		"windows": schema.StringAttribute{
			Optional: true,
		},
		// Maps are always ordered by key by Terraform
		"volume": schema.MapNestedAttribute{
			Required: true,
			Validators: []validator.Map{
				validators.MinMax(1, 1),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"size_mb": schema.Int64Attribute{
						Required: true,
					},
					"type": schema.StringAttribute{
						Required: true,
					},
					"state": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"allowed_ips_ro": schema.StringAttribute{
						Optional: true,
					},
					"allowed_ips_rw": schema.StringAttribute{
						Optional: true,
					},
				},
			},
		},
		"network": schema.MapNestedAttribute{
			Required: true,
			Validators: []validator.Map{
				validators.MinMax(1, 1),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"network": schema.StringAttribute{
						Required: true,
					},
					"cidr": schema.StringAttribute{
						Required: true,
					},
					"state": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r *resourceImpl) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceData

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	environment, err := r.client.STaaSEnvironment.Get(data.Id.ValueString())

	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	populateResourceData(r.client, &data, environment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var create client.STaaSEnvironmentCreate

	create.Name = data.Name.ValueString()
	create.Type = data.Type.ValueString()
	create.Cluster = data.Cluster.ValueString()

	if !data.Volume.Name.IsNull() {
		create.Volume.Name = data.Volume.Name.ValueString()
		create.Volume.SizeMb = int(data.Volume.SizeMb.ValueInt64())
		create.Volume.Type = data.Volume.Type.ValueString()

		create.Volume.AllowedIpsRo = data.Volume.AllowedIpsRo.ValueString()
		create.Volume.AllowedIpsRw = data.Volume.AllowedIpsRw.ValueString()
	}

	err := r.client.STaaSEnvironment.Create(create)
	if err != nil {
		resp.Diagnostics.AddError("Error creating STaaS Environment", fmt.Sprintf("An error occured during the create of a STaaS Environment: %s", err.Error()))
		return
	}

	// Get id from response in the future, for now, get list and then fetch id for this entry
	var pageRequest client.PageRequest
	pageRequest.Size = 10
	pageRequest.Query = create.Name
	pageRequest.Page = 0
	var _, result, err2 = r.client.STaaSEnvironment.Page(pageRequest)

	var createdEnvironment *client.STaaSEnvironmentExt
	if err2 != nil {
		return
	}
	for _, item := range *result {
		if item.Name == create.Name {
			createdEnvironment, err2 = r.client.STaaSEnvironment.Get(item.Id)
			data.Id = types.StringValue(item.Id)
			if err2 != nil {
				resp.Diagnostics.AddError("STaaS Environment not found", fmt.Sprintln("STaaS Environment is not found after matched in list"))
			}
		}
	}
	if createdEnvironment == nil {
		resp.Diagnostics.AddError("STaaS Environment not found in list", fmt.Sprintln("Environment is not found"))
		return
	}

	if data.Id.IsNull() {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintln("An invalid (empty) id was returned after creation"))
		return
	}

	err = waitForSTaaSEnvironmentState(r.client, data.Id, "READY")
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for Kubernetes Cluster (%s) to become ready: %s", data.Id, err))
		return
	}

	populateResourceData(r.client, &data, createdEnvironment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, plan resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.KubernetesCluster.Get(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid kubernetes cluster", fmt.Sprintf("Kubernetes Cluster with ID %s not found", data.Id))
		return
	}

	var update client.STaaSEnvironmentUpdate
	update.Name = plan.Name.ValueString()
	update.Windows = plan.Windows.ValueBool()

	log.Printf("Updating cluster %s", data.Id.ValueString())
	err = r.client.STaaSEnvironment.Update(data.Id.ValueString(), update)

	if err != nil {
		resp.Diagnostics.AddError("Error while updating Kubernetes cluster", err.Error())
		return
	}

	err = waitForSTaaSEnvironmentState(r.client, data.Id, "READY")
	if err != nil {
		resp.Diagnostics.AddError("Error while waiting for environment to become ready", err.Error())
		return
	}

	var updatedEnvironment *client.STaaSEnvironmentExt

	updatedEnvironment, err = r.client.STaaSEnvironment.Get(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("STaaS environment not found", fmt.Sprintln("STaaS environment is not found after matched in list"))
	}

	populateResourceData(r.client, &data, updatedEnvironment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *resourceImpl) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceData

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting Kubernetes Cluster: %s", data.Id)

	err := r.client.KubernetesCluster.Delete(data.Id.ValueString())

	err = waitForSTaaSEnvironmentDeleted(r.client, data.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Kubernetes Cluster: %s", err.Error())
	}
}

func (r *resourceImpl) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resourceData

	var environment, _ = r.client.STaaSEnvironment.Get(req.ID)

	populateResourceData(r.client, &data, environment)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func waitForSTaaSEnvironmentState(client *client.BaseClient, id types.String, target string) error {
	log.Printf("[INFO] Waiting for STaaSEnvironment (%s) to have state %s", id, target)

	backoffOperation := func() error {
		cluster, err := client.STaaSEnvironment.Get(id.ValueString())

		if err != nil {
			log.Printf("invalid STaaS Environment id: %v", err)
			return errors.New(fmt.Sprintf("invalid staas environment id: %s", id))
		}
		if cluster.State != target {
			return errors.New(fmt.Sprintf("Waiting for environment to become ready: %s (%s)", id, cluster.State))
		}
		return nil
	}
	// Max waiting time is 30 mins (should be max 10 mins for smaller clusters)
	backoffConfig := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second*10), 180)

	err := backoff.Retry(backoffOperation, backoffConfig)
	if err != nil {
		return err
	}

	return nil
}

func waitForSTaaSEnvironmentDeleted(client *client.BaseClient, id types.String) error {
	// TODO: force removal of staas environment is not implmeneted yet. Empty environments will be removed after 24h.
	return nil
	//backoffOperation := func() error {
	//	cluster, err := client.STaaSEnvironment.Get(id.ValueString())
	//	log.Printf("Fetching environment: %v", id)
	//	if err != nil {
	//		log.Printf("Error while fetching a environment")
	//		if strings.Contains(err.Error(), "404") &&
	//			strings.Contains(err.Error(), "not found") {
	//			return nil
	//		}
	//		log.Printf("Environment still exists, but got an error: %v", err)
	//		return errors.New(fmt.Sprintf("environment still exists, but got an error: %s", id.ValueString()))
	//	}
	//	if cluster.State == "PENDING_REMOVAL" {
	//		log.Printf("Waiting for environment to be gone: %v", cluster.State)
	//		return errors.New(fmt.Sprintf("Waiting for environment to be gone: %s", id.ValueString()))
	//	}
	//	return nil
	//}
	//log.Printf("Waiting for environment deletion: %v", id)
	//// Max waiting time is 20 mins
	//backoffConfig := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second*10), 120)
	//
	//err := backoff.Retry(backoffOperation, backoffConfig)
	//if err != nil {
	//	return err
	//}
	//
	//return nil
}
