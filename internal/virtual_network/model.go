package virtual_network

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
)

type resourceData struct {
	Id    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Type  types.String `tfsdk:"type"`
	Group types.String `tfsdk:"group"`
}

func populateResourceData(ctx context.Context, data *resourceData, in *client.VirtualNetwork) diag.Diagnostics {
	var diags diag.Diagnostics
	var newDiags diag.Diagnostics

	data.Id = types.StringValue(in.Id)
	data.Name = types.StringValue(in.Name)
	data.Type = types.StringValue(in.Type)
	data.Group = types.StringValue(in.Group)

	diags.Append(newDiags...)

	return diags
}
