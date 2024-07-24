package staas_environment

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"log"
)

type resourceData struct {
	Id      types.String        `tfsdk:"id"`
	Name    types.String        `tfsdk:"name"`
	State   types.String        `tfsdk:"state"`
	Windows types.Bool          `tfsdk:"windows"`
	Cluster types.String        `tfsdk:"cluster"`
	Type    types.String        `tfsdk:"type"`
	Volume  resourceDataVolume  `tfsdk:"volume"`
	Network resourceDataNetwork `tfsdk:"network"`
}

type resourceDataVolume struct {
	Id                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	SizeMb                   types.Int64  `tfsdk:"size_mb"`
	State                    types.String `tfsdk:"state"`
	Type                     types.String `tfsdk:"type"`
	AllowedIpsRo             types.String `tfsdk:"allowed_ips_ro"`
	AllowedIpsRw             types.String `tfsdk:"allowed_ips_rw"`
	SynchronousEnvironmentId types.String `tfsdk:"synchronous_environment_id"`
}

type resourceDataNetwork struct {
	Id      types.String `tfsdk:"id"`
	State   types.String `tfsdk:"state"`
	Network types.String `tfsdk:"network"`
}

func populateResourceData(client *client.BaseClient, data *resourceData, in *client.STaaSEnvironmentExt) diag.Diagnostics {
	var diags diag.Diagnostics
	var newDiags diag.Diagnostics

	data.Id = types.StringValue(in.Id)
	data.Name = types.StringValue(in.Name)
	data.State = types.StringValue(in.State)
	data.Windows = types.BoolValue(in.Windows)

	log.Println(data)

	diags.Append(newDiags...)

	return diags
}
