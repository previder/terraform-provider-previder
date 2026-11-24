package virtual_network

import (
	"errors"
	"fmt"
	"maps"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/attr"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/previder/previder-go-sdk/client"
	"github.com/previder/terraform-provider-previder/internal/util/sorters"
)

type resourceDataAddressPool struct {
	Id          types.String `tfsdk:"id"`
	Start       types.String `tfsdk:"start"`
	End         types.String `tfsdk:"end"`
	Mask        types.String `tfsdk:"mask"`
	Gateway     types.String `tfsdk:"gateway"`
	Type        types.String `tfsdk:"type"`
	NameServers types.List   `tfsdk:"nameservers"`
}

func addressPoolSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"start": schema.StringAttribute{
			Required: true,
		},
		"end": schema.StringAttribute{
			Required: true,
		},
		"mask": schema.StringAttribute{
			Required: true,
		},
		"gateway": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},
		"type": schema.StringAttribute{
			Required: true,
		},
		"nameservers": schema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Computed:    true,
		},
	}
}

func validateAddressPools(pools map[string]resourceDataAddressPool) error {
	for _, key := range sorters.SortMapKeys(pools) {
		pool := pools[key]
		if net.ParseIP(pool.Start.ValueString()) == nil {
			return fmt.Errorf("address pool %q has an invalid start address", key)
		}
		if net.ParseIP(pool.End.ValueString()) == nil {
			return fmt.Errorf("address pool %q has an invalid end address", key)
		}
		if net.ParseIP(pool.Mask.ValueString()) == nil {
			return fmt.Errorf("address pool %q has an invalid mask", key)
		}
		if gateway := pool.Gateway.ValueString(); gateway != "" && net.ParseIP(gateway) == nil {
			return fmt.Errorf("address pool %q has an invalid gateway", key)
		}
		for _, nameserver := range pool.NameServers.Elements() {
			if net.ParseIP(nameserver.(types.String).ValueString()) == nil {
				return fmt.Errorf("address pool %q has an invalid nameserver", key)
			}
		}
	}
	return nil
}

// refreshAddressPools only follows IDs already managed by this resource.
func refreshAddressPools(service client.VirtualNetworkService, networkID string, state map[string]resourceDataAddressPool) (map[string]resourceDataAddressPool, error) {
	if len(state) == 0 {
		return state, nil
	}

	pools, err := service.ListAddressPools(networkID)
	if err != nil {
		return nil, err
	}
	remoteByID := make(map[string]client.AddressPool, len(*pools))
	for _, pool := range *pools {
		remoteByID[pool.Id] = pool
	}

	refreshed := make(map[string]resourceDataAddressPool, len(state))
	for key, prior := range state {
		id := prior.Id.ValueString()
		if pool, found := remoteByID[id]; id != "" && found {
			refreshed[key] = flattenAddressPool(pool, prior)
		}
	}
	return refreshed, nil
}

// syncAddressPools returns the state of all successful changes, including when
// a later operation fails. The caller saves that state before returning an error.
func syncAddressPools(service client.VirtualNetworkService, networkID string, plan, prior map[string]resourceDataAddressPool) (map[string]resourceDataAddressPool, error) {
	current := maps.Clone(prior)
	if current == nil && plan != nil {
		current = make(map[string]resourceDataAddressPool, len(plan))
	}

	for _, key := range sorters.SortMapKeys(plan) {
		desired := plan[key]
		previous := prior[key]
		poolID := previous.Id.ValueString()
		if poolID != "" {
			if desired.Gateway.IsNull() || desired.Gateway.IsUnknown() {
				desired.Gateway = previous.Gateway
			}
			if desired.NameServers.IsNull() || desired.NameServers.IsUnknown() {
				desired.NameServers = previous.NameServers
			}
			if desired.equal(previous) {
				continue
			}
		}

		input := expandAddressPool(desired)
		var remote *client.AddressPool
		var err error
		if poolID == "" {
			remote, err = service.CreateAddressPool(networkID, &input)
		} else {
			remote, err = service.UpdateAddressPool(networkID, poolID, &input)
		}
		if err != nil {
			return current, fmt.Errorf("writing address pool %q: %w", key, err)
		}
		if poolID == "" {
			poolID = remote.Id
			if poolID == "" {
				return current, fmt.Errorf("creating address pool %q returned no ID", key)
			}
		}
		desired.Id = types.StringValue(poolID)
		if remote.Start == "" || remote.End == "" || remote.Mask == "" || remote.Type == "" {
			if desired.Gateway.IsUnknown() {
				desired.Gateway = types.StringNull()
			}
			if desired.NameServers.IsUnknown() {
				desired.NameServers = types.ListNull(types.StringType)
			}
			current[key] = desired
			remote, err = service.GetAddressPool(networkID, poolID)
			if err != nil {
				return current, fmt.Errorf("reading address pool %q after write: %w", key, err)
			}
		}
		remote.Id = poolID
		current[key] = flattenAddressPool(*remote, desired)
	}

	for _, key := range sorters.SortMapKeys(prior) {
		if _, kept := plan[key]; kept {
			continue
		}
		if poolID := prior[key].Id.ValueString(); poolID != "" {
			err := service.DeleteAddressPool(networkID, poolID)
			var apiErr *client.ApiError
			if err != nil && !(errors.As(err, &apiErr) && apiErr.Code == 404) {
				return current, fmt.Errorf("deleting address pool %q: %w", key, err)
			}
		}
		delete(current, key)
	}
	if plan == nil {
		return nil, nil
	}
	return current, nil
}

func (pool resourceDataAddressPool) equal(other resourceDataAddressPool) bool {
	return pool.Start.Equal(other.Start) &&
		pool.End.Equal(other.End) &&
		pool.Mask.Equal(other.Mask) &&
		pool.Gateway.Equal(other.Gateway) &&
		pool.Type.Equal(other.Type) &&
		pool.NameServers.Equal(other.NameServers)
}

func expandAddressPool(pool resourceDataAddressPool) client.AddressPoolUpdate {
	update := client.AddressPoolUpdate{
		Start:   pool.Start.ValueString(),
		End:     pool.End.ValueString(),
		Mask:    pool.Mask.ValueString(),
		Gateway: pool.Gateway.ValueString(),
		Type:    pool.Type.ValueString(),
	}
	for _, nameserver := range pool.NameServers.Elements() {
		update.NameServers = append(update.NameServers, nameserver.(types.String).ValueString())
	}
	return update
}

func flattenAddressPool(remote client.AddressPool, prior resourceDataAddressPool) resourceDataAddressPool {
	nameservers := types.ListNull(types.StringType)
	priorNameservers := prior.NameServers.Elements()
	if remote.NameServers != nil || (!prior.NameServers.IsNull() && !prior.NameServers.IsUnknown() && len(priorNameservers) == 0) {
		values := make([]attr.Value, 0, len(remote.NameServers))
		for i, ip := range remote.NameServers {
			previous := types.StringNull()
			if i < len(priorNameservers) {
				previous = priorNameservers[i].(types.String)
			}
			values = append(values, preserveIP(ip, previous))
		}
		nameservers = types.ListValueMust(types.StringType, values)
	}

	return resourceDataAddressPool{
		Id:          types.StringValue(remote.Id),
		Start:       preserveIP(remote.Start, prior.Start),
		End:         preserveIP(remote.End, prior.End),
		Mask:        preserveIP(remote.Mask, prior.Mask),
		Gateway:     preserveIP(remote.Gateway, prior.Gateway),
		Type:        types.StringValue(remote.Type),
		NameServers: nameservers,
	}
}

func preserveIP(remote string, prior types.String) types.String {
	if parsed := net.ParseIP(remote); parsed != nil && parsed.Equal(net.ParseIP(prior.ValueString())) {
		return prior
	}
	return types.StringValue(remote)
}
