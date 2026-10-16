package types

import "github.com/open-mrp/api/shared/constants"

// RoleTemplateSpec is a starting point for a role: a name, what it's for, and its access and grants.
type RoleTemplateSpec struct {
	Code        constants.RoleTemplate
	Name        string
	Description string
	Access      RoleAccess
	Grants      []string
}

func everyArea(level constants.AccessLevel) RoleAccess {
	out := RoleAccess{}
	for _, area := range constants.PermissionAreas() {
		out[area] = level
	}
	return out
}

func withAccess(base RoleAccess, overrides RoleAccess) RoleAccess {
	out := everyArea(constants.AccessLevelNone)
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

var roleTemplates = []RoleTemplateSpec{
	{
		Code: constants.RoleTemplateOwner, Name: "Owner",
		Description: "Everything, including billing, API keys and integrations.",
		Access:      everyArea(constants.AccessLevelFull),
	},
	{
		Code: constants.RoleTemplateAdmin, Name: "Admin",
		Description: "Everything, including your team, roles and settings.",
		Access:      everyArea(constants.AccessLevelFull),
	},
	{
		Code: constants.RoleTemplateSales, Name: "Sales",
		Description: "Works customers and their orders; reads the catalog and stock.",
		Access: withAccess(nil, RoleAccess{
			constants.PermissionAreaCatalog: constants.AccessLevelRead, constants.PermissionAreaSales: constants.AccessLevelEdit,
			constants.PermissionAreaInventory: constants.AccessLevelRead,
		}),
	},
	{
		Code: constants.RoleTemplateCustomerService, Name: "Customer service",
		Description: "Answers customers: orders, shipments and invoices.",
		Access: withAccess(nil, RoleAccess{
			constants.PermissionAreaCatalog: constants.AccessLevelRead, constants.PermissionAreaSales: constants.AccessLevelEdit,
			constants.PermissionAreaInventory: constants.AccessLevelRead, constants.PermissionAreaProduction: constants.AccessLevelRead,
			constants.PermissionAreaFinance: constants.AccessLevelRead,
		}),
	},
	{
		Code: constants.RoleTemplatePurchasing, Name: "Purchasing",
		Description: "Buys from suppliers and receives what arrives.",
		Access: withAccess(nil, RoleAccess{
			constants.PermissionAreaCatalog: constants.AccessLevelRead, constants.PermissionAreaPurchasing: constants.AccessLevelFull,
			constants.PermissionAreaInventory: constants.AccessLevelRead,
		}),
	},
	{
		Code: constants.RoleTemplateWarehouse, Name: "Warehouse",
		Description: "Moves stock, picks, packs and ships, and receives deliveries.",
		Access: withAccess(nil, RoleAccess{
			constants.PermissionAreaCatalog: constants.AccessLevelRead, constants.PermissionAreaSales: constants.AccessLevelRead,
			constants.PermissionAreaPurchasing: constants.AccessLevelRead, constants.PermissionAreaInventory: constants.AccessLevelEdit,
		}),
		Grants: []string{"receiving:create", "receiving:update"},
	},
	{
		Code: constants.RoleTemplateProductionSupervisor, Name: "Production supervisor",
		Description: "Runs the shop floor: jobs, batches, work centers and machines.",
		Access: withAccess(nil, RoleAccess{
			constants.PermissionAreaCatalog: constants.AccessLevelRead, constants.PermissionAreaSales: constants.AccessLevelRead,
			constants.PermissionAreaInventory: constants.AccessLevelEdit, constants.PermissionAreaProduction: constants.AccessLevelFull,
		}),
	},
	{
		Code: constants.RoleTemplateOperator, Name: "Operator",
		Description: "Works at a station: scans batches and records downtime.",
		Access: withAccess(nil, RoleAccess{
			constants.PermissionAreaCatalog: constants.AccessLevelRead, constants.PermissionAreaInventory: constants.AccessLevelRead,
			constants.PermissionAreaProduction: constants.AccessLevelRead,
		}),
		Grants: []string{"batches:create", "batches:update", "machine_downtime:create", "machine_downtime:update"},
	},
	{
		Code: constants.RoleTemplateAccounting, Name: "Accounting",
		Description: "Invoices, payments and costs; reads orders on both sides.",
		Access: withAccess(nil, RoleAccess{
			constants.PermissionAreaCatalog: constants.AccessLevelRead, constants.PermissionAreaSales: constants.AccessLevelRead,
			constants.PermissionAreaPurchasing: constants.AccessLevelRead, constants.PermissionAreaFinance: constants.AccessLevelFull,
		}),
	},
	{
		Code: constants.RoleTemplateReadOnly, Name: "Read only",
		Description: "Sees everything outside settings and changes nothing.",
		Access: withAccess(everyArea(constants.AccessLevelRead), RoleAccess{
			constants.PermissionAreaSettings: constants.AccessLevelNone,
		}),
	},
}

// RoleTemplates lists every role template in catalog order. Callers must not modify it.
func RoleTemplates() []RoleTemplateSpec { return roleTemplates }

// RoleTemplateByCode looks a template up by its code.
func RoleTemplateByCode(code constants.RoleTemplate) (RoleTemplateSpec, bool) {
	for _, t := range roleTemplates {
		if t.Code == code {
			return t, true
		}
	}
	return RoleTemplateSpec{}, false
}
