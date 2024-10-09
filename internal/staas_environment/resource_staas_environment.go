package staas_environment

import (
	"context"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
	"github.com/previder/terraform-provider-previder/internal/util/validators"
	"log"
	"net"
	"reflect"
	"strings"
	"time"
)

const ResourceType = "previder_staas_environment"

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
		"windows": schema.BoolAttribute{
			Optional: true,
		},
		// Maps are always ordered by key by Terraform
		"volumes": schema.MapNestedAttribute{
			Required: true,
			Validators: []validator.Map{
				validators.MinMax(0, 16),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"name": schema.StringAttribute{
						Required: true,
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
					"allowed_ips_ro": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"allowed_ips_rw": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"synchronous_environment_id": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
					"synchronous_environment_name": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
				},
			},
		},
		"networks": schema.MapNestedAttribute{
			Required: true,
			Validators: []validator.Map{
				validators.MinMax(0, 16),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"network_id": schema.StringAttribute{
						Required: true,
					},
					"network_name": schema.StringAttribute{
						Computed: true,
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
					"ip_addresses": schema.ListAttribute{
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
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

	populateResourceData(ctx, &data, environment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.checkNetworks(data.Networks)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, volume := range data.Volumes {
		for _, roCidr := range volume.AllowedIpsRo {
			resp.Diagnostics.Append(r.checkCidr(roCidr)...)
		}
		for _, rwCidr := range volume.AllowedIpsRw {
			resp.Diagnostics.Append(r.checkCidr(rwCidr)...)
		}
	}
	for _, network := range data.Networks {
		resp.Diagnostics.Append(r.checkCidr(network.Cidr)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var create client.STaaSEnvironmentCreate

	create.Name = data.Name.ValueString()
	create.Type = data.Type.ValueString()
	create.Cluster = data.Cluster.ValueString()
	create.Windows = data.Windows.ValueBool()

	createdEnvironmentReference, err := r.client.STaaSEnvironment.Create(create)
	if err != nil {
		resp.Diagnostics.AddError("Error creating STaaS Environment", fmt.Sprintf("An error occured during the create of a STaaS Environment: %s", err.Error()))
		return
	}

	createdEnvironment, err := r.client.STaaSEnvironment.Get(createdEnvironmentReference.Id)
	if err == nil {
		resp.Diagnostics.AddError("STaaS Environment not found after creation", "Environment is not found")
		return
	}
	data.Id = types.StringValue(createdEnvironment.Id)

	if data.Id.IsNull() {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintln("An invalid (empty) id was returned after creation"))
		return
	}

	err = waitForSTaaSEnvironmentState(r.client, data.Id, "READY")
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for STaaS environment (%s) to become ready: %s", data.Id, err))
		return
	}

	for _, v := range data.Volumes {
		var volumeCreate client.STaaSVolumeCreate
		volumeCreate.Name = v.Name.ValueString()
		volumeCreate.SizeMb = int(v.SizeMb.ValueInt64())
		volumeCreate.Type = v.Type.ValueString()

		var createAllowedIpsRo []string
		for _, a := range v.AllowedIpsRo {
			createAllowedIpsRo = append(createAllowedIpsRo, a.ValueString())
		}

		var createAllowedIpsRw []string
		for _, a := range v.AllowedIpsRw {
			createAllowedIpsRw = append(createAllowedIpsRw, a.ValueString())
		}

		volumeCreate.AllowedIpsRo = createAllowedIpsRo
		volumeCreate.AllowedIpsRw = createAllowedIpsRw

		err = r.client.STaaSEnvironment.CreateVolume(createdEnvironment.Id, volumeCreate)

		createdEnvironment, err = r.client.STaaSEnvironment.Get(createdEnvironment.Id)

		var volumeId = ""
		for _, b := range createdEnvironment.Volumes {
			log.Printf("[INFO] Found volumes in environment (%s) %s %s", b.Id, b.Name, b.State)

			if types.StringValue(b.Name) == v.Name {
				volumeId = b.Id
			}
		}

		if volumeId == "" {
			resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for volume (%s) to become ready: %s", data.Id, err))
			return
		}

		err = waitForSTaaSVolumeState(r.client, data.Id, volumeId, "READY")
	}

	for _, n := range data.Networks {
		var networkCreate client.STaaSNetworkCreate
		networkCreate.Network = n.NetworkId.ValueString()
		networkCreate.Cidr = n.Cidr.ValueString()

		err = r.client.STaaSEnvironment.CreateNetwork(createdEnvironment.Id, networkCreate)

		createdEnvironment, err = r.client.STaaSEnvironment.Get(createdEnvironment.Id)

		var networkId = ""
		for _, b := range createdEnvironment.Networks {
			log.Printf("[INFO] Found network in environment (%s) %s %s", b.Id, b.NetworkName, b.State)

			if types.StringValue(b.NetworkId) == n.NetworkId {
				networkId = b.Id
			}
		}

		if networkId == "" {
			resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for network (%s) to become ready: %s", data.Id, err))
			return
		}

		err = waitForSTaaSNetworkState(r.client, data.Id, networkId, []string{"READY", "SYNCED"})
	}

	createdEnvironment, err = r.client.STaaSEnvironment.Get(createdEnvironment.Id)

	populateResourceData(ctx, &data, createdEnvironment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.STaaSEnvironment.Get(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid STaaS environment", fmt.Sprintf("STaaS environment with ID %s not found", plan.Id))
		return
	}

	resp.Diagnostics.Append(r.checkNetworks(plan.Networks)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, volume := range plan.Volumes {
		for _, roCidr := range volume.AllowedIpsRo {
			resp.Diagnostics.Append(r.checkCidr(roCidr)...)
		}
		for _, rwCidr := range volume.AllowedIpsRw {
			resp.Diagnostics.Append(r.checkCidr(rwCidr)...)
		}
	}
	for _, network := range plan.Networks {
		resp.Diagnostics.Append(r.checkCidr(network.Cidr)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Name.ValueString() != state.Name.ValueString() || plan.Windows.ValueBool() != state.Windows.ValueBool() {
		var update client.STaaSEnvironmentUpdate
		update.Name = plan.Name.ValueString()
		update.Windows = plan.Windows.ValueBool()

		log.Printf("Updating environment %s", plan.Id.ValueString())

		err = r.client.STaaSEnvironment.Update(plan.Id.ValueString(), update)
	}

	// First go over plan volumes, and check if volumes exists state and values match
	for _, planVolume := range plan.Volumes {
		var found = false
		for _, stateVolume := range state.Volumes {

			log.Printf("Checking ID %s %s", planVolume.Id, stateVolume.Id)
			if planVolume.Name == stateVolume.Name {
				found = true
				var changed = false
				if planVolume.SizeMb != stateVolume.SizeMb {
					changed = true
				}
				if !reflect.DeepEqual(planVolume.AllowedIpsRo, stateVolume.AllowedIpsRo) {
					changed = true
				}

				if !reflect.DeepEqual(planVolume.AllowedIpsRw, stateVolume.AllowedIpsRw) {
					changed = true
				}

				if changed {
					var update client.STaaSVolumeUpdate
					update.Name = planVolume.Name.ValueString()
					update.SizeMb = int(planVolume.SizeMb.ValueInt64())
					update.Type = planVolume.Type.ValueString()
					var updateAllowedIpsRo []string
					for _, a := range planVolume.AllowedIpsRo {
						updateAllowedIpsRo = append(updateAllowedIpsRo, a.ValueString())
					}

					var updateAllowedIpsRw []string
					for _, a := range planVolume.AllowedIpsRw {
						updateAllowedIpsRw = append(updateAllowedIpsRw, a.ValueString())
					}

					update.AllowedIpsRo = updateAllowedIpsRo
					update.AllowedIpsRw = updateAllowedIpsRw

					err = r.client.STaaSEnvironment.UpdateVolume(plan.Id.ValueString(), stateVolume.Id.ValueString(), update)

					err = waitForSTaaSVolumeState(r.client, plan.Id, stateVolume.Id.ValueString(), "READY")
				}
			}

			if !found {
				var volumeCreate client.STaaSVolumeCreate
				volumeCreate.Name = planVolume.Name.ValueString()
				volumeCreate.SizeMb = int(planVolume.SizeMb.ValueInt64())
				volumeCreate.Type = planVolume.Type.ValueString()

				var createAllowedIpsRo []string
				for _, a := range planVolume.AllowedIpsRo {
					createAllowedIpsRo = append(createAllowedIpsRo, a.ValueString())
				}

				var createAllowedIpsRw []string
				for _, a := range planVolume.AllowedIpsRw {
					createAllowedIpsRw = append(createAllowedIpsRw, a.ValueString())
				}

				volumeCreate.AllowedIpsRo = createAllowedIpsRo
				volumeCreate.AllowedIpsRw = createAllowedIpsRw

				err = r.client.STaaSEnvironment.CreateVolume(plan.Id.ValueString(), volumeCreate)

				var environment, err2 = r.client.STaaSEnvironment.Get(plan.Id.ValueString())

				if err2 != nil {
					log.Printf("[INFO] Environment not found (%s) %s %s", plan.Id, plan.Name, plan.State)
				}
				var volumeId = ""
				for _, b := range environment.Volumes {
					log.Printf("[INFO] Found volumes in environment (%s) %s %s", b.Id, b.Name, b.State)

					if types.StringValue(b.Name) == planVolume.Name {
						volumeId = b.Id
					}
				}

				if volumeId == "" {
					resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for STaaS environment (%s) to become ready: %s", plan.Id, err))
					return
				}

				err = waitForSTaaSVolumeState(r.client, plan.Id, volumeId, "READY")
			}
		}
	}

	// Then go over volumes in state and check if any volumes were removed from plan
	// Pay attention that removed volumes can have state GRACE_TERMINATED.
	for _, stateVolume := range state.Volumes {
		if stateVolume.State.ValueString() == "GRACE_TERMINATED" {
			continue
		}

		var found = false
		for _, planVolume := range plan.Volumes {
			if stateVolume.Name == planVolume.Name {
				found = true
			}
		}

		if !found {
			// stateVolume has not been found in plan.Volumes, so it has been removed.
			var deleteVolume client.STaaSVolumeDelete
			deleteVolume.Force = true
			err = r.client.STaaSEnvironment.DeleteVolume(plan.Id.ValueString(), stateVolume.Id.ValueString(), deleteVolume)

			err = waitForSTaaSVolumeState(r.client, plan.Id, stateVolume.Id.ValueString(), "GRACE_TERMINATED")
		}
	}

	// Go over plan networks, and check if they are in state
	for _, planNetwork := range plan.Networks {
		var found = false
		for _, stateNetwork := range state.Networks {
			if planNetwork.NetworkId == stateNetwork.NetworkId {
				found = true
			}
		}

		if !found {
			var networkCreate client.STaaSNetworkCreate
			networkCreate.Network = planNetwork.NetworkId.ValueString()
			networkCreate.Cidr = planNetwork.Cidr.ValueString()

			err = r.client.STaaSEnvironment.CreateNetwork(plan.Id.ValueString(), networkCreate)

			createdEnvironment, _ := r.client.STaaSEnvironment.Get(plan.Id.ValueString())

			var networkId = ""
			for _, b := range createdEnvironment.Networks {
				log.Printf("[INFO] Found network in environment (%s) %s %s", b.Id, b.NetworkName, b.State)

				if types.StringValue(b.NetworkId) == planNetwork.NetworkId {
					networkId = b.Id
				}
			}

			if networkId == "" {
				resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for network (%s) to become ready: %s", plan.Id, err))
				return
			}

			err = waitForSTaaSNetworkState(r.client, plan.Id, networkId, []string{"READY", "SYNCED"})
		}
	}

	// Then go over state network, and delete networks that are not in plan networks
	for _, stateNetwork := range state.Networks {
		var found = false
		for _, planNetwork := range plan.Networks {
			if planNetwork.Id.ValueString() == stateNetwork.Id.ValueString() {
				found = true
			}
		}

		if !found {
			err = r.client.STaaSEnvironment.DeleteNetwork(plan.Id.ValueString(), stateNetwork.Id.ValueString())

			err = waitForSTaaSNetworkDeleted(r.client, plan.Id, stateNetwork.Id)
		}
	}

	if err != nil {
		resp.Diagnostics.AddError("Error while updating STaaS environment", err.Error())
		return
	}

	err = waitForSTaaSEnvironmentState(r.client, plan.Id, "READY")
	if err != nil {
		resp.Diagnostics.AddError("Error while waiting for environment to become ready", err.Error())
		return
	}

	var updatedEnvironment *client.STaaSEnvironmentExt

	updatedEnvironment, err = r.client.STaaSEnvironment.Get(plan.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("STaaS environment not found", fmt.Sprintln("STaaS environment is not found after matched in list"))
	}

	populateResourceData(ctx, &plan, updatedEnvironment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r *resourceImpl) checkNetworks(networks map[string]resourceDataNetwork) diag.Diagnostics {
	var newDiags diag.Diagnostics
	for key, n := range networks {
		networkResponse, err := r.client.VirtualNetwork.Get(n.NetworkId.ValueString())
		if err != nil {
			newDiags.AddError("Error in STaaS Environment", fmt.Sprintf("Network %s does not exists: %s", key, err.Error()))
		}
		if networkResponse.Type != "VLAN" {
			newDiags.AddError("Error in STaaS Environment", fmt.Sprintf("Network %s is not of type VLAN", key))
		}
	}

	return newDiags
}

func (r *resourceImpl) checkCidr(cidr types.String) diag.Diagnostics {
	var newDiags diag.Diagnostics

	_, _, err := net.ParseCIDR(cidr.ValueString())
	if err != nil {
		newDiags.AddError("Error  in STaaS environment", fmt.Sprintf("Invalid CIDR %s", cidr))
	}

	return newDiags
}

func (r *resourceImpl) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceData

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting STaaS environment: %s", data.Id)
	var deleteEnvironment client.STaaSEnvironmentDelete
	deleteEnvironment.Force = true

	err := r.client.STaaSEnvironment.Delete(data.Id.ValueString(), deleteEnvironment)

	err = waitForSTaaSEnvironmentDeleted(r.client, data.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting STaaS environment: %s", err.Error())
	}
}

func (r *resourceImpl) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resourceData

	var environment, _ = r.client.STaaSEnvironment.Get(req.ID)

	populateResourceData(ctx, &data, environment)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func waitForSTaaSEnvironmentState(client *client.PreviderClient, id types.String, target string) error {
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

func waitForSTaaSVolumeState(client *client.PreviderClient, id types.String, volumeId string, target string) error {
	log.Printf("[INFO] Waiting for STaaSVolume (%s) to have state %s", id, target)

	backoffOperation := func() error {
		cluster, err := client.STaaSEnvironment.Get(id.ValueString())

		if err != nil {
			log.Printf("invalid STaaS Environment id: %v", err)
			return errors.New(fmt.Sprintf("invalid staas volume id: %s", id))
		}

		for _, volume := range cluster.Volumes {
			if volume.Id == volumeId && volume.State != target {
				return errors.New(fmt.Sprintf("Waiting for volume %s to become ready: %s (%s)", volumeId, id, cluster.State))
			}
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

func waitForSTaaSNetworkState(client *client.PreviderClient, id types.String, networkId string, target []string) error {
	log.Printf("[INFO] Waiting for STaaSNetwork (%s) to have state %s", id, target)

	backoffOperation := func() error {
		cluster, err := client.STaaSEnvironment.Get(id.ValueString())

		if err != nil {
			log.Printf("invalid STaaS Environment id: %v", err)
			return errors.New(fmt.Sprintf("invalid staas environment id: %s", id))
		}

		for _, network := range cluster.Networks {
			if network.Id == networkId && !contains(target, network.State) {
				return errors.New(fmt.Sprintf("Waiting for network state %s to become : %s (%s)", networkId, id, cluster.State))
			}
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func waitForSTaaSEnvironmentDeleted(client *client.PreviderClient, id types.String) error {
	backoffOperation := func() error {
		cluster, err := client.STaaSEnvironment.Get(id.ValueString())
		log.Printf("Fetching environment: %v", id)
		if err != nil {
			log.Printf("Error while fetching a environment")
			if strings.Contains(err.Error(), "404") &&
				strings.Contains(err.Error(), "not found") {
				return nil
			}
			log.Printf("Environment still exists, but got an error: %v", err)
			return errors.New(fmt.Sprintf("environment still exists, but got an error: %s", id.ValueString()))
		}
		if cluster.State == "FORCE_REMOVAL" {
			log.Printf("Waiting for environment to be gone: %v", cluster.State)
			return errors.New(fmt.Sprintf("Waiting for environment to be gone: %s", id.ValueString()))
		}
		return nil
	}
	log.Printf("Waiting for environment deletion: %v", id)
	// Max waiting time is 20 mins
	backoffConfig := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second*10), 120)

	err := backoff.Retry(backoffOperation, backoffConfig)
	if err != nil {
		return err
	}

	return nil
}

func waitForSTaaSNetworkDeleted(client *client.PreviderClient, id types.String, networkId types.String) error {
	backoffOperation := func() error {
		cluster, err := client.STaaSEnvironment.Get(id.ValueString())
		log.Printf("Fetching environment: %v", id)
		if err != nil {
			log.Printf("Error while fetching a environment")
			if strings.Contains(err.Error(), "404") &&
				strings.Contains(err.Error(), "not found") {
				return nil
			}
			log.Printf("Environment still exists, but got an error: %v", err)
			return errors.New(fmt.Sprintf("environment still exists, but got an error: %s", id.ValueString()))
		}
		for _, n := range cluster.Networks {
			if n.Id == networkId.ValueString() {
				if n.State == "PENDING_REMOVAL" {
					log.Printf("Waiting for network to be gone: %v", cluster.State)
					return errors.New(fmt.Sprintf("Waiting for network to be gone: %s", networkId.ValueString()))
				}
			}
		}

		return nil
	}
	log.Printf("Waiting for network deletion: %v", id)
	// Max waiting time is 20 mins
	backoffConfig := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second*10), 120)

	err := backoff.Retry(backoffOperation, backoffConfig)
	if err != nil {
		return err
	}

	return nil
}
