package staas_environment

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
)

type resourceData struct {
	Id       types.String                   `tfsdk:"id"`
	Name     types.String                   `tfsdk:"name"`
	State    types.String                   `tfsdk:"state"`
	Windows  types.Bool                     `tfsdk:"windows"`
	Cluster  types.String                   `tfsdk:"cluster"`
	Type     types.String                   `tfsdk:"type"`
	Volumes  map[string]resourceDataVolume  `tfsdk:"volumes"`
	Networks map[string]resourceDataNetwork `tfsdk:"networks"`
}

type resourceDataVolume struct {
	Id                         types.String   `tfsdk:"id"`
	Name                       types.String   `tfsdk:"name"`
	SizeMb                     types.Int64    `tfsdk:"size_mb"`
	State                      types.String   `tfsdk:"state"`
	Type                       types.String   `tfsdk:"type"`
	AllowedIpsRo               []types.String `tfsdk:"allowed_ips_ro"`
	AllowedIpsRw               []types.String `tfsdk:"allowed_ips_rw"`
	SynchronousEnvironmentId   types.String   `tfsdk:"synchronous_environment_id"`
	SynchronousEnvironmentName types.String   `tfsdk:"synchronous_environment_name"`
}

type resourceDataNetwork struct {
	Id          types.String `tfsdk:"id"`
	State       types.String `tfsdk:"state"`
	NetworkId   types.String `tfsdk:"network_id"`
	NetworkName types.String `tfsdk:"network_name"`
	IpAddresses types.List   `tfsdk:"ip_addresses"`
	Cidr        types.String `tfsdk:"cidr"`
}

func populateResourceData(ctx context.Context, client *client.BaseClient, data *resourceData, in *client.STaaSEnvironmentExt) diag.Diagnostics {
	var diags diag.Diagnostics
	var newDiags diag.Diagnostics

	data.Id = types.StringValue(in.Id)
	data.Name = types.StringValue(in.Name)
	data.State = types.StringValue(in.State)
	data.Windows = types.BoolValue(in.Windows)

	var readVolumes = make(map[string]resourceDataVolume)
	for _, v := range in.Volumes {
		volume := resourceDataVolume{
			Id:                       types.StringValue(v.Id),
			Name:                     types.StringValue(v.Name),
			SizeMb:                   types.Int64Value(int64(v.SizeMb)),
			State:                    types.StringValue(v.State),
			Type:                     types.StringValue(v.Type),
			SynchronousEnvironmentId: types.StringValue(v.SynchronousEnvironmentId),
		}

		var readAllowedIpsRo = make([]types.String, len(v.AllowedIpsRo))
		for i, a := range v.AllowedIpsRo {
			readAllowedIpsRo[i] = types.StringValue(a)
		}
		volume.AllowedIpsRo = readAllowedIpsRo

		var readAllowedIpsRw = make([]types.String, len(v.AllowedIpsRw))
		for i, a := range v.AllowedIpsRw {
			readAllowedIpsRw[i] = types.StringValue(a)
		}
		volume.AllowedIpsRw = readAllowedIpsRw

		readVolumes[v.Name] = volume
	}

	data.Volumes = readVolumes

	var readNetworks = make(map[string]resourceDataNetwork)
	for _, v := range in.Networks {
		network := resourceDataNetwork{
			Id:          types.StringValue(v.Id),
			NetworkId:   types.StringValue(v.NetworkId),
			NetworkName: types.StringValue(v.NetworkName),
			State:       types.StringValue(v.State),
			Cidr:        types.StringValue(v.Cidr),
		}

		network.IpAddresses, _ = types.ListValueFrom(ctx, types.StringType, v.IpAddresses)

		readNetworks[v.NetworkId] = network
	}

	data.Networks = readNetworks
	diags.Append(newDiags...)

	return diags
}
