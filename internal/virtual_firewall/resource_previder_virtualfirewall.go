package virtual_firewall

import (
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util"
	"log"
	"net"
	"strings"
	"time"

	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

const ResourceType = "previder_virtual_firewall"

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
			MarkdownDescription: "ID of the Virtual Firewall",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required: true,
		},
		"type": schema.StringAttribute{
			Required: true,
		},
		"type_name": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"lan_address": schema.StringAttribute{
			Required: true,
		},
		"wan_address": schema.ListAttribute{
			ElementType: types.StringType,
			Computed:    true,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
		"group": schema.StringAttribute{
			Optional: true,
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"group_name": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"state": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"network": schema.StringAttribute{
			Required: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"network_name": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"dhcp_enabled": schema.BoolAttribute{
			Required: true,
		},
		"dhcp_range_start": schema.StringAttribute{
			Optional: true,
		},
		"dhcp_range_end": schema.StringAttribute{
			Optional: true,
		},
		"local_domain_name": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString("int"),
		},
		"dns_enabled": schema.BoolAttribute{
			Required: true,
		},
		"nameservers": schema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
		},
		"termination_protected": schema.BoolAttribute{
			Optional: true,
		},
		"icmp_wan_enabled": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(true),
		},
		"icmp_lan_enabled": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(true),
		},
		"nat_rules": schema.MapNestedAttribute{
			Optional: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"source": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
					"port": schema.Int32Attribute{
						Required: true,
					},
					"protocol": schema.StringAttribute{
						Required: true,
					},
					"active": schema.BoolAttribute{
						Required: true,
					},
					"description": schema.StringAttribute{
						Computed: true,
					},
					"nat_destination": schema.StringAttribute{
						Required: true,
					},
					"nat_port": schema.Int32Attribute{
						Required: true,
					},
				},
			},
		},
	}
}

func (r *resourceImpl) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceData

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	virtualFirewall, err := r.client.VirtualFirewall.Get(data.Id.ValueString())

	if err != nil {
		if err.(*client.ApiError).Code == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	rules, err := r.getAllNatRules(virtualFirewall.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error while updating Virtual Firewall", err.Error())
		return
	}

	populateResourceData(&data, virtualFirewall, rules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.validateNatRules(data.NatRules)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Virtual Firewall", err.Error())
		return
	}

	var create client.VirtualFirewallCreate

	create.Name = data.Name.ValueString()
	create.Type = data.Type.ValueString()
	create.Network = data.Network.ValueString()

	create.Group = data.Group.ValueString()
	create.LanAddress = data.LanAddress.ValueString()
	_, network, err := net.ParseCIDR(create.LanAddress)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Virtual Firewall", "The Lan Address is not a valid CIDR")
		return
	}
	create.DhcpEnabled = data.DhcpEnabled.ValueBool()
	if create.DhcpEnabled {
		create.DhcpRangeStart = net.ParseIP(data.DhcpRangeStart.ValueString())
		if !network.Contains(create.DhcpRangeStart) {
			resp.Diagnostics.AddError("Error creating Virtual Firewall", fmt.Sprintf("The DHCP range start is not in the LAN subnet %v-%v", network.String(), create.DhcpRangeStart))
			return
		}
		create.DhcpRangeEnd = net.ParseIP(data.DhcpRangeEnd.ValueString())
		if !network.Contains(create.DhcpRangeEnd) {
			resp.Diagnostics.AddError("Error creating Virtual Firewall", "The DHCP range end is not in the LAN subnet")
			return
		}
		create.LocalDomainName = data.LocalDomainName.ValueString()
	}
	create.DnsEnabled = data.DnsEnabled.ValueBool()
	if create.DnsEnabled {
		var createNameservers []net.IP
		var dataNameservers []types.String
		data.Nameservers.ElementsAs(ctx, &dataNameservers, false)
		for _, nameserver := range dataNameservers {
			createNameservers = append(createNameservers, net.ParseIP(nameserver.ValueString()))
		}
		create.Nameservers = createNameservers
	}

	create.TerminationProtected = data.TerminationProtected.ValueBool()
	create.IcmpWanEnabled = data.IcmpWanEnabled.ValueBool()
	create.IcmpLanEnabled = data.IcmpLanEnabled.ValueBool()

	createdFirewallReference, err := r.client.VirtualFirewall.Create(create)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Virtual Firewall", fmt.Sprintf("An error occured during the create of a Virtual Firewall: %s", err.Error()))
		return
	}
	data.Id = types.StringValue(createdFirewallReference.Id)

	if data.Id.IsNull() {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintln("An invalid (empty) id was returned after creation"))
		return
	}

	err = waitForVirtualFirewallState(r.client, data.Id, "READY")
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Error waiting for Virtual Firewall (%s) to become ready: %s", data.Id, err))
		return
	}

	err = r.processNatRules(data.Id.ValueString(), data.NatRules, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error while updating Virtual Firewall NAT rules", err.Error())
		return
	}

	createdFirewall, err := r.client.VirtualFirewall.Get(createdFirewallReference.Id)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("ID from creation not found (%s): %s", data.Id, err))
		return
	}

	rules, err := r.getAllNatRules(createdFirewall.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error while updating Virtual Firewall", err.Error())
		return
	}

	populateResourceData(&data, createdFirewall, rules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *resourceImpl) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, plan resourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.VirtualFirewall.Get(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Virtual Firewall", fmt.Sprintf("Virtual Firewall with ID %s not found", data.Id))
		return
	}

	if data.Network != plan.Network {
		resp.Diagnostics.AddError("Invalid updated fields", fmt.Sprintf("Fields network cannot be updated after creation"))
		return
	}

	err = r.validateNatRules(plan.NatRules)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Virtual Firewall", err.Error())
		return
	}

	var update client.VirtualFirewallUpdate
	update.Name = plan.Name.ValueString()
	update.Group = plan.Group.ValueString()
	update.Network = plan.Network.ValueString()
	update.LanAddress = plan.LanAddress.ValueString()
	_, network, err := net.ParseCIDR(update.LanAddress)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Virtual Firewall", "The Lan Address is not a valid CIDR")
		return
	}
	update.DhcpEnabled = plan.DhcpEnabled.ValueBool()
	if update.DhcpEnabled {
		update.DhcpRangeStart = net.ParseIP(plan.DhcpRangeStart.ValueString())
		if !network.Contains(update.DhcpRangeStart) {
			resp.Diagnostics.AddError("Error updating Virtual Firewall", "The DHCP range start is not in the LAN subnet")
			return
		}
		update.DhcpRangeEnd = net.ParseIP(plan.DhcpRangeEnd.ValueString())
		if !network.Contains(update.DhcpRangeEnd) {
			resp.Diagnostics.AddError("Error updating Virtual Firewall", "The DHCP range end is not in the LAN subnet")
			return
		}
		update.LocalDomainName = plan.LocalDomainName.ValueString()
	}

	update.DnsEnabled = plan.DnsEnabled.ValueBool()
	if update.DnsEnabled {
		var createNameservers []net.IP
		var dataNameservers []types.String
		plan.Nameservers.ElementsAs(ctx, &dataNameservers, false)
		for _, nameserver := range dataNameservers {
			createNameservers = append(createNameservers, net.ParseIP(nameserver.ValueString()))
		}
		update.Nameservers = createNameservers
	}

	update.TerminationProtected = plan.TerminationProtected.ValueBool()
	update.IcmpWanEnabled = plan.IcmpWanEnabled.ValueBool()
	update.IcmpLanEnabled = plan.IcmpLanEnabled.ValueBool()

	log.Printf("Updating Virtual Firewall %s", data.Id.ValueString())
	err = r.client.VirtualFirewall.Update(data.Id.ValueString(), update)

	if err != nil {
		resp.Diagnostics.AddError("Error while updating Virtual Firewall", err.Error())
		return
	}

	err = waitForVirtualFirewallState(r.client, data.Id, "READY")
	if err != nil {
		resp.Diagnostics.AddError("Error while waiting for Virtual Firewall to become ready", err.Error())
		return
	}

	err = r.processNatRules(data.Id.ValueString(), plan.NatRules, &data)
	if err != nil {
		resp.Diagnostics.AddError("Error while updating Virtual Firewall NAT rules", err.Error())
		return
	}

	var updatedFirewall *client.VirtualFirewallExt

	updatedFirewall, err = r.client.VirtualFirewall.Get(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Virtual Firewall not found", fmt.Sprintln("Virtual Firewall is not found after matched in list"))
	}

	rules, err := r.getAllNatRules(updatedFirewall.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error while updating Virtual Firewall", err.Error())
		return
	}

	populateResourceData(&data, updatedFirewall, rules)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *resourceImpl) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceData

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting Virtual Firewall: %s", data.Id)

	err := r.client.VirtualFirewall.Delete(data.Id.ValueString())

	err = waitForVirtualFirewallDeleted(r.client, data.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting Virtual Firewall: %s", err.Error())
	}
}

func (r *resourceImpl) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data resourceData

	var virtualFirewall, _ = r.client.VirtualFirewall.Get(req.ID)

	rules, err := r.getAllNatRules(virtualFirewall.Id)
	if err != nil {
		return
	}

	populateResourceData(&data, virtualFirewall, rules)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *resourceImpl) validateNatRules(dataNatRules map[string]resourceDataNatRule) error {
	for _, rule := range dataNatRules {
		if ok := net.ParseIP(rule.NatDestination.ValueString()); ok == nil {
			return errors.New(fmt.Sprintf("invalid destination IP %v", rule.NatDestination.ValueString()))
		}
		if rule.Source.ValueString() != "" {
			_, _, err := net.ParseCIDR(rule.Source.ValueString())
			if err != nil {
				return errors.New(fmt.Sprintf("invalid source CIDR %v", rule.Source.ValueString()))
			}
		}
		if rule.Port.ValueInt32() < 1 || rule.Port.ValueInt32() > 65535 {
			return errors.New(fmt.Sprintf("invalid port %v", rule.Port.ValueInt32()))
		}
		if rule.NatPort.ValueInt32() < 1 || rule.NatPort.ValueInt32() > 65535 {
			return errors.New(fmt.Sprintf("invalid port %v", rule.NatPort.ValueInt32()))
		}
		if rule.Protocol.ValueString() != "TCP" && rule.Protocol.ValueString() != "UDP" {
			return errors.New(fmt.Sprintf("invalid protocol %v, allowed values are TCP or UDP", rule.Protocol.ValueString()))
		}
	}
	return nil
}

func (r *resourceImpl) processNatRules(firewallId string, dataNatRules map[string]resourceDataNatRule, existingData *resourceData) error {
	for key, rule := range dataNatRules {
		updateNatRule := client.VirtualFirewallNatRuleCreate{
			Description:    key,
			Active:         rule.Active.ValueBool(),
			Source:         rule.Source.ValueString(),
			Protocol:       rule.Protocol.ValueString(),
			Port:           int(rule.Port.ValueInt32()),
			NatPort:        int(rule.NatPort.ValueInt32()),
			NatDestination: rule.NatDestination.ValueString(),
		}
		var ruleId string
		if existingData != nil {
			if existingRule, ok := existingData.NatRules[key]; ok {
				ruleId = existingRule.Id.ValueString()
			}
		}
		if ruleId != "" {
			// Existing rule
			err := r.client.VirtualFirewall.UpdateNatRule(firewallId, ruleId, updateNatRule)
			if err != nil {
				return err
			}
		} else {
			// New rule
			_, err := r.client.VirtualFirewall.CreateNatRule(firewallId, updateNatRule)
			if err != nil {
				return err
			}
		}
	}
	// Cleanup old rules
	if existingData != nil {
		currentRules, err := r.getAllNatRules(firewallId)
		if err != nil {
			return err
		}
		for _, rule := range *currentRules {
			if _, ok := dataNatRules[rule.Description]; !ok {
				// Should remove old rule
				err := r.client.VirtualFirewall.DeleteNatRule(firewallId, rule.Id)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *resourceImpl) getAllNatRules(id string) (*[]client.VirtualFirewallNatRule, error) {
	var page client.PageRequest
	page.Size = 100
	page.Page = 0
	page.Sort = "+description"
	page.Query = ""

	_, rules, err := r.client.VirtualFirewall.PageNatRules(id, page)
	if err != nil {
		return nil, err
	}

	return rules, nil
}

func waitForVirtualFirewallState(client *client.PreviderClient, id types.String, target string) error {
	log.Printf("[INFO] Waiting for Virtual Firewall (%s) to have state %s", id, target)

	backoffOperation := func() error {
		cluster, err := client.VirtualFirewall.Get(id.ValueString())

		if err != nil {
			log.Printf("invalid Virtual Firewall id: %v", err)
			return errors.New(fmt.Sprintf("invalid Virtual Firewall id: %s", id))
		}
		if cluster.State != target {
			return errors.New(fmt.Sprintf("Waiting for Virtual Firewall to become ready: %s (%s)", id, cluster.State))
		}
		return nil
	}
	// Max waiting time is 30 mins (should be max 10 mins for smaller clusters)
	backoffConfig := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 180)

	err := backoff.Retry(backoffOperation, backoffConfig)
	if err != nil {
		return err
	}

	return nil
}

func waitForVirtualFirewallDeleted(client *client.PreviderClient, id types.String) error {
	backoffOperation := func() error {
		cluster, err := client.VirtualFirewall.Get(id.ValueString())
		log.Printf("Fetching Virtual Firewall: %v", id)
		if err != nil {
			log.Printf("Error while fetching a Virtual Firewall")
			if strings.Contains(err.Error(), "404") &&
				strings.Contains(err.Error(), "not found") {
				return nil
			}
			log.Printf("Virtual Firewall still exists, but got an error: %v", err)
			return errors.New(fmt.Sprintf("Virtual Firewall still exists, but got an error: %s", id.ValueString()))
		}
		if cluster.State == "PENDING_REMOVAL" {
			log.Printf("Waiting for Virtual Firewall to be gone: %v", cluster.State)
			return errors.New(fmt.Sprintf("Waiting for Virtual Firewall to be gone: %s", id.ValueString()))
		}
		return nil
	}
	log.Printf("Waiting for Virtual Firewall deletion: %v", id)
	// Max waiting time is 20 mins
	backoffConfig := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second*10), 120)

	err := backoff.Retry(backoffOperation, backoffConfig)
	if err != nil {
		return err
	}

	return nil
}
