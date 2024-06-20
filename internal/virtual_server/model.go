package virtual_server

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"net"
)

type resourceData struct {
	Id                    types.String                            `tfsdk:"id"`
	Name                  types.String                            `tfsdk:"name"`
	Group                 types.String                            `tfsdk:"group"`
	ComputeCluster        types.String                            `tfsdk:"compute_cluster"`
	CpuCores              types.Int64                             `tfsdk:"cpu_cores"`
	CpuSockets            types.Int64                             `tfsdk:"cpu_sockets"`
	Memory                types.Int64                             `tfsdk:"memory"`
	Template              types.String                            `tfsdk:"template"`
	GuestId               types.String                            `tfsdk:"guest_id"`
	Source                types.String                            `tfsdk:"source"`
	State                 types.String                            `tfsdk:"state"`
	Tags                  []types.String                          `tfsdk:"tags"`
	Disks                 map[string]resourceDataDisk             `tfsdk:"disks"`
	NetworkInterfaces     map[string]resourceDataNetworkInterface `tfsdk:"network_interfaces"`
	TerminationProtection types.Bool                              `tfsdk:"termination_protection"`
	UserData              types.String                            `tfsdk:"user_data"`
	ProvisioningType      types.String                            `tfsdk:"provisioning_type"`
	InitialPassword       types.String                            `tfsdk:"initial_password"`
}

type resourceDataDisk struct {
	Id    types.String `tfsdk:"id"`
	Size  types.Int64  `tfsdk:"size"`
	Uuid  types.String `tfsdk:"uuid"`
	Label types.String `tfsdk:"label"`
}

type resourceDataNetworkInterface struct {
	Id                  types.String `tfsdk:"id"`
	Network             types.String `tfsdk:"network"`
	Connected           types.Bool   `tfsdk:"connected"`
	Label               types.String `tfsdk:"label"`
	IPv4Address         types.String `tfsdk:"ipv4_address"`
	IPv6Address         types.String `tfsdk:"ipv6_address"`
	MACAddress          types.String `tfsdk:"mac_address"`
	AssignedAddresses   types.List   `tfsdk:"assigned_addresses"`
	DiscoveredAddresses types.List   `tfsdk:"discovered_addresses"`
	Type                types.String `tfsdk:"type"`
}

func populateResourceData(ctx context.Context, data *resourceData, in *client.VirtualMachineExt) diag.Diagnostics {
	var diags diag.Diagnostics
	var newDiags diag.Diagnostics

	data.Id = types.StringValue(in.Id)
	data.Name = types.StringValue(in.Name)
	data.Group = types.StringValue(in.Group)
	data.ComputeCluster = types.StringValue(in.ComputeCluster)
	data.CpuCores = types.Int64Value(int64(in.CpuCores))
	data.Memory = types.Int64Value(int64(in.Memory))
	if len(in.Template) == 0 {
		data.Template = types.StringNull()
	} else {
		data.Template = types.StringValue(in.Template)
	}
	if len(in.GuestId) == 0 {
		data.GuestId = types.StringNull()
	} else {
		data.GuestId = types.StringValue(in.GuestId)
	}
	data.InitialPassword = types.StringValue(in.InitialPassword)

	data.State = types.StringValue(in.State)
	var readTags = make([]types.String, len(in.Tags))
	for i, v := range in.Tags {
		readTags[i] = types.StringValue(v)
	}
	data.Tags = readTags

	var readDisks = make(map[string]resourceDataDisk)
	for _, v := range in.Disks {
		readDisks[v.Label] = resourceDataDisk{
			Id:    types.StringValue(v.Id),
			Size:  types.Int64Value(int64(v.Size)),
			Label: types.StringValue(v.Label),
			Uuid:  types.StringValue(v.Uuid),
		}
	}
	data.Disks = readDisks

	var readNetworkInterfaces = make(map[string]resourceDataNetworkInterface)
	for _, v := range in.NetworkInterfaces {
		readNetworkInterface := resourceDataNetworkInterface{
			Id:        types.StringValue(v.Id),
			Network:   types.StringValue(v.Network),
			Label:     types.StringValue(v.Label),
			Type:      types.StringValue(v.Type),
			Connected: types.BoolValue(v.Connected),
		}

		var readAssignedAddresses []types.String
		for _, av := range v.AssignedAddresses {
			readAssignedAddresses = append(readAssignedAddresses, types.StringValue(av))
			if ip := net.ParseIP(av); ip != nil {
				if ip.To4() != nil && readNetworkInterface.IPv4Address.IsNull() {
					readNetworkInterface.IPv4Address = types.StringValue(ip.String())
				}
				if ip.To4() == nil && readNetworkInterface.IPv6Address.IsNull() {
					readNetworkInterface.IPv6Address = types.StringValue(ip.String())
				}
			}
		}
		readNetworkInterface.AssignedAddresses, _ = types.ListValueFrom(ctx, types.StringType, readAssignedAddresses)

		var readDiscoveredAddresses []types.String
		for _, av := range v.DiscoveredAddresses {
			readDiscoveredAddresses = append(readDiscoveredAddresses, types.StringValue(av))
		}
		readNetworkInterface.DiscoveredAddresses, _ = types.ListValueFrom(ctx, types.StringType, readDiscoveredAddresses)
		readNetworkInterface.MACAddress = types.StringValue(v.MacAddress)
		readNetworkInterfaces[v.Label] = readNetworkInterface

	}
	data.NetworkInterfaces = readNetworkInterfaces

	data.TerminationProtection = types.BoolValue(in.TerminationProtectionEnabled)

	diags.Append(newDiags...)

	return diags
}
