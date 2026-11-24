package secure_vault

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
)

type secureVaultResourceData struct {
	Id                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Group                types.String `tfsdk:"group"`
	State                types.String `tfsdk:"state"`
	TerminationProtected types.Bool   `tfsdk:"termination_protected"`
	Token                types.String `tfsdk:"token"`
}

func populateSecureVaultData(data *secureVaultResourceData, in *client.SecureVaultExt) {
	data.Id = types.StringValue(in.Id)
	data.Name = types.StringValue(in.Name)
	data.State = types.StringValue(in.State)
	data.TerminationProtected = types.BoolValue(in.TerminationProtected)

	// Keep name-based and ID-based configurations in the same form on refresh.
	switch {
	case data.Group.ValueString() != "" && !util.IsValidObjectId(data.Group.ValueString()) && in.GroupName != "":
		data.Group = types.StringValue(in.GroupName)
	case in.Group != "":
		data.Group = types.StringValue(in.Group)
	case in.GroupName != "":
		data.Group = types.StringValue(in.GroupName)
	default:
		data.Group = types.StringNull()
	}
}
