package virtual_network

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
)

type resourceData struct {
	Id           types.String                       `tfsdk:"id"`
	Name         types.String                       `tfsdk:"name"`
	Type         types.String                       `tfsdk:"type"`
	Group        types.String                       `tfsdk:"group"`
	AddressPools map[string]resourceDataAddressPool `tfsdk:"address_pools"`
}

func populateResourceData(data *resourceData, network *client.VirtualNetwork) {
	data.Id = types.StringValue(network.Id)
	data.Name = types.StringValue(network.Name)
	data.Type = types.StringValue(network.Type)
	if util.IsValidObjectId(data.Group.ValueString()) {
		data.Group = types.StringValue(network.Group)
	} else {
		data.Group = types.StringValue(network.GroupName)
	}
}
