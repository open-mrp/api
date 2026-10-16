package roleep

import (
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/field"
)

// RoleAccessInput sets a role's access level per area. Areas left out keep their level (on create: the template's,
// else `none`).
type RoleAccessInput struct {
	// Items, categories, product lines, properties and units: `none`, `read`, `edit` or `full`.
	Catalog field.Optional[constants.AccessLevel] `json:"catalog,omitzero"`
	// Customers, contacts, sales orders, pricing and sales reps.
	Sales field.Optional[constants.AccessLevel] `json:"sales,omitzero"`
	// Suppliers, purchase orders and receiving.
	Purchasing field.Optional[constants.AccessLevel] `json:"purchasing,omitzero"`
	// Inventory, locations, picks, shipments and carriers.
	Inventory field.Optional[constants.AccessLevel] `json:"inventory,omitzero"`
	// Production, batches, scanning, work centers and machines.
	Production field.Optional[constants.AccessLevel] `json:"production,omitzero"`
	// Invoices, payments, payment terms and costs.
	Finance field.Optional[constants.AccessLevel] `json:"finance,omitzero"`
	// Your team, roles, API keys, integrations, logs and account settings.
	Settings field.Optional[constants.AccessLevel] `json:"settings,omitzero"`
}

// levels is the areas the input sets.
func (in *RoleAccessInput) levels() map[string]string {
	if in == nil {
		return nil
	}
	out := map[string]string{}
	for area, v := range map[constants.PermissionArea]field.Optional[constants.AccessLevel]{
		constants.PermissionAreaCatalog: in.Catalog, constants.PermissionAreaSales: in.Sales,
		constants.PermissionAreaPurchasing: in.Purchasing, constants.PermissionAreaInventory: in.Inventory,
		constants.PermissionAreaProduction: in.Production, constants.PermissionAreaFinance: in.Finance,
		constants.PermissionAreaSettings: in.Settings,
	} {
		if l, ok := v.Value(); ok {
			out[string(area)] = string(l)
		}
	}
	return out
}

func (in *RoleAccessInput) roleAccess() types.RoleAccess {
	out := types.RoleAccess{}
	for k, v := range in.levels() {
		out[constants.PermissionArea(k)] = constants.AccessLevel(v)
	}
	return out
}
