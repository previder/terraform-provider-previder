package kubernetes_cluster

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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
	"log"
	"reflect"
	"strings"
	"time"
)

const ResourceType = "previder_kubernetes_cluster"

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
			MarkdownDescription: "ID of the Kubernetes Cluster",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required: true,
		},
		"state": schema.StringAttribute{
			Computed: true,
		},
		"version": schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"vips": schema.ListAttribute{
			Required:    true,
			ElementType: types.StringType,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"endpoints": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
		"minimal_nodes": schema.Int64Attribute{
			Required: true,
		},
		"maximal_nodes": schema.Int64Attribute{
			Optional: true,
			Computed: true,
		},
		"auto_update": schema.BoolAttribute{
			Optional: true,
		},
		"auto_scale_enabled": schema.BoolAttribute{
			Optional: true,
			Computed: true,
		},
		"control_plane_cpu_cores": schema.Int64Attribute{
			Required: true,
		},
		"control_plane_memory_gb": schema.Int64Attribute{
			Required: true,
		},
		"control_plane_storage_gb": schema.Int64Attribute{
			Required: true,
		},
		"node_cpu_cores": schema.Int64Attribute{
			Required: true,
		},
		"node_memory_gb": schema.Int64Attribute{
			Required: true,
		},
		"node_storage_gb": schema.Int64Attribute{
			Required: true,
		},
		"compute_cluster": schema.StringAttribute{
			Required: true,
		},
		"cni": schema.StringAttribute{
			Optional: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"high_available_control_plane": schema.BoolAttribute{
			Required: true,
		},
		"network": schema.StringAttribute{
			Required: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"reference": schema.StringAttribute{
			Computed: true,
		},
		"kubeconfig": schema.StringAttribute{
			Computed: true,
		},
	}
}

func (r *resourceImpl) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceData

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster, err := r.client.KubernetesCluster.Get(data.Id.ValueString())

	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	populateResourceData(r.client, &data, cluster)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var create client.KubernetesClusterCreate

	create.Name = data.Name.ValueString()
	create.Version = data.Version.ValueString()
	create.AutoUpdate = data.AutoUpdate.ValueBool()
	create.AutoScaleEnabled = data.AutoScaleEnabled.ValueBool()
	create.MinimalNodes = int(data.MinimalNodes.ValueInt64())
	create.MaximalNodes = int(data.MaximalNodes.ValueInt64())
	create.ControlPlaneCpuCores = int(data.ControlPlaneCpuCores.ValueInt64())
	create.ControlPlaneMemoryGb = int(data.ControlPlaneMemoryGb.ValueInt64())
	create.ControlPlaneStorageGb = int(data.ControlPlaneStorageGb.ValueInt64())
	create.NodeCpuCores = int(data.NodeCpuCores.ValueInt64())
	create.NodeMemoryGb = int(data.NodeMemoryGb.ValueInt64())
	create.NodeStorageGb = int(data.NodeStorageGb.ValueInt64())
	create.ComputeCluster = data.ComputeCluster.ValueString()
	create.HighAvailableControlPlane = data.HighAvailableControlPlane.ValueBool()
	create.CNI = data.CNI.ValueString()
	create.Network = data.Network.ValueString()

	var createVips []string
	for _, v := range data.Vips {
		createVips = append(createVips, v.ValueString())
	}
	create.Vips = createVips

	var createEndPoints []string
	for _, v := range data.Endpoints {
		createEndPoints = append(createEndPoints, v.ValueString())
	}
	create.Endpoints = createEndPoints

	createdKubernetesCluster, err := r.client.KubernetesCluster.Create(create)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Kubernetes Cluster", fmt.Sprintf("An error occured during the create of a Kubernetes Cluster: %s", err.Error()))
		return
	}

	createdCluster, err := r.client.KubernetesCluster.Get(createdKubernetesCluster.Id)
	data.Id = types.StringValue(createdCluster.Id)

	if createdCluster == nil {
		resp.Diagnostics.AddError("Kubernetes Cluster not found in list", fmt.Sprintln("Cluster is not found"))
		return
	}

	if data.Id.IsNull() {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintln("An invalid (empty) id was returned after creation"))
		return
	}

	err = waitForKubernetesClusterState(r.client, data.Id, "READY")
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for Kubernetes Cluster (%s) to become ready: %s", data.Id, err))
		return
	}

	populateResourceData(r.client, &data, createdCluster)

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

	if data.CNI != plan.CNI || data.Network != plan.Network || !reflect.DeepEqual(data.Vips, plan.Vips) || !reflect.DeepEqual(data.Endpoints, plan.Endpoints) {
		resp.Diagnostics.AddError("Invalid updated fields", fmt.Sprintf("Fields cni,network,vips,endpoints cannot be updated after creation"))
		return
	}

	var update client.KubernetesClusterUpdate
	update.Name = plan.Name.ValueString()
	update.Version = plan.Version.ValueString()
	update.AutoUpdate = plan.AutoUpdate.ValueBool()
	update.AutoScaleEnabled = plan.AutoScaleEnabled.ValueBool()
	update.MinimalNodes = int(plan.MinimalNodes.ValueInt64())
	update.MaximalNodes = int(plan.MaximalNodes.ValueInt64())
	update.ControlPlaneCpuCores = int(plan.ControlPlaneCpuCores.ValueInt64())
	update.ControlPlaneMemoryGb = int(plan.ControlPlaneMemoryGb.ValueInt64())
	update.ControlPlaneStorageGb = int(plan.ControlPlaneStorageGb.ValueInt64())
	update.NodeCpuCores = int(plan.NodeCpuCores.ValueInt64())
	update.NodeMemoryGb = int(plan.NodeMemoryGb.ValueInt64())
	update.NodeStorageGb = int(plan.NodeStorageGb.ValueInt64())
	update.ComputeCluster = plan.ComputeCluster.ValueString()
	update.HighAvailableControlPlane = plan.HighAvailableControlPlane.ValueBool()

	log.Printf("Updating cluster %s", data.Id.ValueString())
	err = r.client.KubernetesCluster.Update(data.Id.ValueString(), update)

	if err != nil {
		resp.Diagnostics.AddError("Error while updating Kubernetes cluster", err.Error())
		return
	}

	err = waitForKubernetesClusterState(r.client, data.Id, "READY")
	if err != nil {
		resp.Diagnostics.AddError("Error while waiting for cluster to become ready", err.Error())
		return
	}

	var updatedCluster *client.KubernetesClusterExt

	updatedCluster, err = r.client.KubernetesCluster.Get(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes Cluster not found", fmt.Sprintln("Kubernetes Cluster is not found after matched in list"))
	}

	populateResourceData(r.client, &data, updatedCluster)

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

	err = waitForKubernetesClusterDeleted(r.client, data.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Kubernetes Cluster: %s", err.Error())
	}
}

func (r *resourceImpl) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resourceData

	var cluster, _ = r.client.KubernetesCluster.Get(req.ID)

	populateResourceData(r.client, &data, cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func waitForKubernetesClusterState(client *client.PreviderClient, id types.String, target string) error {
	log.Printf("[INFO] Waiting for Kubernetes cluster (%s) to have state %s", id, target)

	backoffOperation := func() error {
		cluster, err := client.KubernetesCluster.Get(id.ValueString())

		if err != nil {
			log.Printf("invalid Kubernetes Cluster id: %v", err)
			return errors.New(fmt.Sprintf("invalid kubernetes cluster id: %s", id))
		}
		if cluster.State != target {
			return errors.New(fmt.Sprintf("Waiting for cluster to become ready: %s (%s)", id, cluster.State))
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

func waitForKubernetesClusterDeleted(client *client.PreviderClient, id types.String) error {

	backoffOperation := func() error {
		cluster, err := client.KubernetesCluster.Get(id.ValueString())
		log.Printf("Fetching cluster: %v", id)
		if err != nil {
			log.Printf("Error while fetching a cluster")
			if strings.Contains(err.Error(), "404") &&
				strings.Contains(err.Error(), "not found") {
				return nil
			}
			log.Printf("Cluster still exists, but got an error: %v", err)
			return errors.New(fmt.Sprintf("cluster still exists, but got an error: %s", id.ValueString()))
		}
		if cluster.State == "PENDING_REMOVAL" {
			log.Printf("Waiting for cluster to be gone: %v", cluster.State)
			return errors.New(fmt.Sprintf("Waiting for cluster to be gone: %s", id.ValueString()))
		}
		return nil
	}
	log.Printf("Waiting for cluster deletion: %v", id)
	// Max waiting time is 20 mins
	backoffConfig := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second*10), 120)

	err := backoff.Retry(backoffOperation, backoffConfig)
	if err != nil {
		return err
	}

	return nil
}
