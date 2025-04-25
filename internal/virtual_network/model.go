package virtual_network

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
)

type resourceData struct {
	Id    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Type  types.String `tfsdk:"type"`
	Group types.String `tfsdk:"group"`
}

func populateResourceData(data *resourceData, in *client.VirtualNetwork, plan *resourceData) diag.Diagnostics {
	var diags diag.Diagnostics
	var newDiags diag.Diagnostics

	if plan == nil {
		plan = &resourceData{}
	}

	data.Id = types.StringValue(in.Id)
	data.Name = types.StringValue(in.Name)
	data.Type = types.StringValue(in.Type)
	if util.IsValidObjectId(plan.Group.ValueString()) {
		data.Group = types.StringValue(in.Group)
	} else {
		data.Group = types.StringValue(in.GroupName)
	}

	diags.Append(newDiags...)

	return diags
}
