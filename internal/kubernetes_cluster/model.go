package kubernetes_cluster

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
)

type resourceData struct {
	Id                        types.String   `tfsdk:"id"`
	Name                      types.String   `tfsdk:"name"`
	State                     types.String   `tfsdk:"state"`
	Version                   types.String   `tfsdk:"version"`
	Vips                      []types.String `tfsdk:"vips"`
	Endpoints                 []types.String `tfsdk:"endpoints"`
	MinimalNodes              types.Int64    `tfsdk:"minimal_nodes"`
	MaximalNodes              types.Int64    `tfsdk:"maximal_nodes"`
	AutoUpdate                types.Bool     `tfsdk:"auto_update"`
	AutoScaleEnabled          types.Bool     `tfsdk:"auto_scale_enabled"`
	ControlPlaneCpuCores      types.Int64    `tfsdk:"control_plane_cpu_cores"`
	ControlPlaneMemoryGb      types.Int64    `tfsdk:"control_plane_memory_gb"`
	ControlPlaneStorageGb     types.Int64    `tfsdk:"control_plane_storage_gb"`
	NodeCpuCores              types.Int64    `tfsdk:"node_cpu_cores"`
	NodeMemoryGb              types.Int64    `tfsdk:"node_memory_gb"`
	NodeStorageGb             types.Int64    `tfsdk:"node_storage_gb"`
	ComputeCluster            types.String   `tfsdk:"compute_cluster"`
	CNI                       types.String   `tfsdk:"cni"`
	HighAvailableControlPlane types.Bool     `tfsdk:"high_available_control_plane"`
	Network                   types.String   `tfsdk:"network"`
	Reference                 types.String   `tfsdk:"reference"`
	KubeConfig                types.String   `tfsdk:"kubeconfig"`
}

func populateResourceData(client *client.PreviderClient, data *resourceData, in *client.KubernetesClusterExt, plan *resourceData) diag.Diagnostics {
	var diags diag.Diagnostics
	var newDiags diag.Diagnostics

	if plan == nil {
		plan = &resourceData{}
	}

	data.Id = types.StringValue(in.Id)
	data.Name = types.StringValue(in.Name)
	data.State = types.StringValue(in.State)
	data.Version = types.StringValue(in.Version)
	var vips []types.String
	for _, v := range in.Vips {
		vips = append(vips, types.StringValue(v))
	}
	data.Vips = vips

	var endpoints []types.String
	for _, v := range in.Endpoints {
		endpoints = append(endpoints, types.StringValue(v))
	}

	data.Endpoints = endpoints
	data.MinimalNodes = types.Int64Value(int64(in.MinimalNodes))
	data.MaximalNodes = types.Int64Value(int64(in.MaximalNodes))
	data.AutoUpdate = types.BoolValue(in.AutoUpdate)
	data.AutoScaleEnabled = types.BoolValue(in.AutoScaleEnabled)
	data.ControlPlaneCpuCores = types.Int64Value(int64(in.ControlPlaneCpuCores))
	data.ControlPlaneMemoryGb = types.Int64Value(int64(in.ControlPlaneMemoryGb))
	data.ControlPlaneStorageGb = types.Int64Value(int64(in.ControlPlaneStorageGb))
	data.NodeCpuCores = types.Int64Value(int64(in.NodeCpuCores))
	data.NodeMemoryGb = types.Int64Value(int64(in.NodeMemoryGb))
	data.NodeStorageGb = types.Int64Value(int64(in.NodeStorageGb))
	data.ComputeCluster = types.StringValue(in.ComputeCluster)
	data.CNI = types.StringValue(in.CNI)
	data.HighAvailableControlPlane = types.BoolValue(in.HighAvailableControlPlane)
	if util.IsValidObjectId(plan.Network.ValueString()) {
		data.Network = types.StringValue(in.Network)
	} else {
		data.Network = plan.Network
	}
	data.Reference = types.StringValue(in.Reference)

	address := in.Vips[0]
	if len(in.Endpoints) > 0 {
		address = in.Endpoints[0]
	}
	kubeConfigResponse, err := client.KubernetesCluster.GetKubeConfig(data.Id.ValueString(), address)
	if err != nil {
		data.KubeConfig = types.StringNull()
	} else {
		data.KubeConfig = types.StringValue(kubeConfigResponse.Config)
	}

	diags.Append(newDiags...)

	return diags
}
