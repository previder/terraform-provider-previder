package virtual_firewall

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
)

type resourceData struct {
	Id                   types.String                   `tfsdk:"id"`
	Name                 types.String                   `tfsdk:"name"`
	Type                 types.String                   `tfsdk:"type"`
	TypeName             types.String                   `tfsdk:"type_name"`
	Group                types.String                   `tfsdk:"group"`
	GroupName            types.String                   `tfsdk:"group_name"`
	Network              types.String                   `tfsdk:"network"`
	NetworkName          types.String                   `tfsdk:"network_name"`
	LanAddress           types.String                   `tfsdk:"lan_address"`
	WanAddress           types.List                     `tfsdk:"wan_address"`
	DhcpEnabled          types.Bool                     `tfsdk:"dhcp_enabled"`
	DhcpRangeStart       types.String                   `tfsdk:"dhcp_range_start"`
	DhcpRangeEnd         types.String                   `tfsdk:"dhcp_range_end"`
	LocalDomainName      types.String                   `tfsdk:"local_domain_name"`
	DnsEnabled           types.Bool                     `tfsdk:"dns_enabled"`
	Nameservers          types.List                     `tfsdk:"nameservers"`
	TerminationProtected types.Bool                     `tfsdk:"termination_protected"`
	IcmpWanEnabled       types.Bool                     `tfsdk:"icmp_wan_enabled"`
	IcmpLanEnabled       types.Bool                     `tfsdk:"icmp_lan_enabled"`
	State                types.String                   `tfsdk:"state"`
	NatRules             map[string]resourceDataNatRule `tfsdk:"nat_rules"`
}

type resourceDataNatRule struct {
	Id             types.String `tfsdk:"id"`
	Description    types.String `tfsdk:"description"`
	Port           types.Int32  `tfsdk:"port"`
	Protocol       types.String `tfsdk:"protocol"`
	Active         types.Bool   `tfsdk:"active"`
	Source         types.String `tfsdk:"source"`
	NatDestination types.String `tfsdk:"nat_destination"`
	NatPort        types.Int32  `tfsdk:"nat_port"`
}

func populateResourceData(data *resourceData, in *client.VirtualFirewallExt, inNatRules *[]client.VirtualFirewallNatRule, plan *resourceData) diag.Diagnostics {
	var diags diag.Diagnostics
	var newDiags diag.Diagnostics

	if plan == nil {
		plan = &resourceData{}
	}

	data.Id = types.StringValue(in.Id)
	data.Name = types.StringValue(in.Name)
	data.Type = types.StringValue(in.TypeLabel)
	data.TypeName = types.StringValue(in.TypeName)
	if util.IsValidObjectId(plan.Group.ValueString()) {
		data.Group = types.StringValue(in.Group)
	} else {
		data.Group = types.StringValue(in.GroupName)
	}
	data.GroupName = types.StringValue(in.GroupName)
	if util.IsValidObjectId(plan.Network.ValueString()) {
		data.Network = types.StringValue(in.Network)
	} else {
		data.Network = types.StringValue(in.NetworkName)
	}
	data.NetworkName = types.StringValue(in.NetworkName)
	data.LanAddress = types.StringValue(in.LanAddress)
	var readWanAddress []attr.Value
	for _, wanAddress := range in.WanAddress {
		readWanAddress = append(readWanAddress, types.StringValue(wanAddress))
	}
	readWanAddressValue, _ := types.ListValueFrom(context.Background(), types.StringType, readWanAddress)
	data.WanAddress = readWanAddressValue

	data.DhcpEnabled = types.BoolValue(in.DhcpEnabled)
	data.DhcpRangeStart = types.StringValue(in.DhcpRangeStart)
	data.DhcpRangeEnd = types.StringValue(in.DhcpRangeEnd)
	data.LocalDomainName = types.StringValue(in.LocalDomainName)
	data.DnsEnabled = types.BoolValue(in.DnsEnabled)
	var readNameservers []attr.Value
	for _, nameserver := range in.Nameservers {
		readNameservers = append(readNameservers, types.StringValue(nameserver))
	}
	readNameserversValue, _ := types.ListValueFrom(context.Background(), types.StringType, readNameservers)
	data.Nameservers = readNameserversValue

	data.TerminationProtected = types.BoolValue(in.TerminationProtected)
	data.IcmpWanEnabled = types.BoolValue(in.IcmpWanEnabled)
	data.IcmpLanEnabled = types.BoolValue(in.IcmpLanEnabled)
	data.State = types.StringValue(in.State)

	var readNatRules = make(map[string]resourceDataNatRule)
	for _, natRule := range *inNatRules {
		readNatRules[natRule.Description] = resourceDataNatRule{
			Id:             types.StringValue(natRule.Id),
			Description:    types.StringValue(natRule.Description),
			Active:         types.BoolValue(natRule.Active),
			Source:         types.StringValue(natRule.Source),
			Port:           types.Int32Value(int32(natRule.Port)),
			Protocol:       types.StringValue(natRule.Protocol),
			NatDestination: types.StringValue(natRule.NatDestination),
			NatPort:        types.Int32Value(int32(natRule.NatPort)),
		}
	}
	data.NatRules = readNatRules

	diags.Append(newDiags...)

	return diags
}
