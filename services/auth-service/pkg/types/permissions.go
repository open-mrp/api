package types

type PermissionDomain string

const (
	PermissionDomainAgents                        PermissionDomain = "agents"
	PermissionDomainAgentRuns                     PermissionDomain = "agent_runs"
	PermissionDomainAgentMemories                 PermissionDomain = "agent_memories"
	PermissionDomainAlerts                        PermissionDomain = "alerts"
	PermissionDomainAccount                       PermissionDomain = "self"
	PermissionDomainDeliveries                    PermissionDomain = "deliveries"
	PermissionDomainLocations                     PermissionDomain = "locations"
	PermissionDomainSettlements                   PermissionDomain = "settlements"
	PermissionDomainTransactions                  PermissionDomain = "transactions"
	PermissionDomainBatches                       PermissionDomain = "batches"
	PermissionDomainCarriers                      PermissionDomain = "carriers"
	PermissionDomainCustomerGroups                PermissionDomain = "customer_groups"
	PermissionDomainCustomers                     PermissionDomain = "customers"
	PermissionDomainCustomerUsers                 PermissionDomain = "contacts"
	PermissionDomainDepartmentPicks               PermissionDomain = "department_picks"
	PermissionDomainDepartments                   PermissionDomain = "departments"
	PermissionDomainDiscounts                     PermissionDomain = "discounts"
	PermissionDomainEdiLocations                  PermissionDomain = "edi_locations"
	PermissionDomainEdiRuns                       PermissionDomain = "edi_runs"
	PermissionDomainEmailLogs                     PermissionDomain = "email_logs"
	PermissionDomainErrorLogs                     PermissionDomain = "error_logs"
	PermissionDomainProducts                      PermissionDomain = "products"
	PermissionDomainInventory                     PermissionDomain = "inventory"
	PermissionDomainInventoryChangeLogs           PermissionDomain = "inventory_change_logs"
	PermissionDomainInventoryLogs                 PermissionDomain = "inventory_logs"
	PermissionDomainInvoices                      PermissionDomain = "invoices"
	PermissionDomainCategories                    PermissionDomain = "item_categories"
	PermissionDomainMachines                      PermissionDomain = "machines"
	PermissionDomainMachineDowntime               PermissionDomain = "machine_downtime"
	PermissionDomainProductionSchedules           PermissionDomain = "production_schedules"
	PermissionDomainDemandOverrides               PermissionDomain = "demand_overrides"
	PermissionDomainMaterials                     PermissionDomain = "materials"
	PermissionDomainOrganization                  PermissionDomain = "accounts"
	PermissionDomainPaymentTerms                  PermissionDomain = "payment_terms"
	PermissionDomainPermissions                   PermissionDomain = "permissions"
	PermissionDomainParts                         PermissionDomain = "parts"
	PermissionDomainPicks                         PermissionDomain = "picks"
	PermissionDomainReceivingOrders               PermissionDomain = "receiving_orders"
	PermissionDomainProductGroups                 PermissionDomain = "product_groups"
	PermissionDomainItems                         PermissionDomain = "items"
	PermissionDomainProductionRuns                PermissionDomain = "production_runs"
	PermissionDomainProductionStepTransformations PermissionDomain = "production_step_transformations"
	PermissionDomainProductionSteps               PermissionDomain = "production_steps"
	PermissionDomainProductLines                  PermissionDomain = "product_lines"
	PermissionDomainProductVariations             PermissionDomain = "product_variations"
	PermissionDomainProperties                    PermissionDomain = "properties"
	PermissionDomainPurchaseOrders                PermissionDomain = "purchase_orders"
	PermissionDomainSuppliers                     PermissionDomain = "suppliers"
	PermissionDomainReceiving                     PermissionDomain = "receiving"
	PermissionDomainProductLineAccess             PermissionDomain = "relevant_products"
	PermissionDomainRoles                         PermissionDomain = "roles"
	PermissionDomainSalesOrders                   PermissionDomain = "sales_orders"
	PermissionDomainTerritories                   PermissionDomain = "sales_rep_territories"
	PermissionDomainSalesTargets                  PermissionDomain = "sales_targets"
	PermissionDomainScanningStations              PermissionDomain = "scanners"
	PermissionDomainScanningErrorLogs             PermissionDomain = "scanning_error_logs"
	PermissionDomainShifts                        PermissionDomain = "shifts"
	PermissionDomainShipments                     PermissionDomain = "shipments"
	PermissionDomainShippingCases                 PermissionDomain = "shipping_cases"
	PermissionDomainShippingTerms                 PermissionDomain = "shipping_terms"
	PermissionDomainSupplies                      PermissionDomain = "supplies"
	PermissionDomainSystemProperties              PermissionDomain = "system_properties"
	PermissionDomainTeamUsers                     PermissionDomain = "team"
	PermissionDomainUnits                         PermissionDomain = "units"
	PermissionDomainUnitGroups                    PermissionDomain = "unit_groups"
	PermissionDomainRequestLogs                   PermissionDomain = "request_logs"
	PermissionDomainAuditEvents                   PermissionDomain = "audit_events"
	PermissionDomainAPIKeys                       PermissionDomain = "api_keys"
	PermissionDomainSandbox                       PermissionDomain = "sandboxes"
	PermissionDomainAddresses                     PermissionDomain = "addresses"
	PermissionDomainIntegrations                  PermissionDomain = "integrations"
	PermissionDomainAdjustmentTypes               PermissionDomain = "adjustment_types"
	PermissionDomainPriorities                    PermissionDomain = "priorities"
	PermissionDomainProductTypes                  PermissionDomain = "product_types"
	PermissionDomainJobs                          PermissionDomain = "jobs"
	PermissionDomainMessaging                     PermissionDomain = "messaging"
)

type Action string

const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

// Permission is a single required permission: a domain paired with an action. Declaring permissions with these typed constants (instead of raw "<domain>:<action>" strings) is checked by the compiler, preventing typos.
type Permission struct {
	Domain PermissionDomain
	Action Action
}

// AnyOfPermissions lists permissions where holding any one satisfies the requirement. API gateway endpoints declare this shape on RequiredPermissions: the coarse gate rejects callers who hold none of the listed permissions; handlers may still apply finer per-resource checks (for example unified search skips types the caller cannot read).
type AnyOfPermissions []Permission

// String renders the permission in canonical "<domain>:<action>" form.
func (p Permission) String() string {
	return string(p.Domain) + ":" + string(p.Action)
}

// AllPermissionDomains lists every permission domain the platform recognizes, so a role's `<domain>:<action>` grant can be checked against a real domain instead of persisting an unusable one. Kept beside the constants it enumerates: adding a domain without adding it here makes that domain unassignable through the roles API.
func AllPermissionDomains() []PermissionDomain {
	return []PermissionDomain{
		PermissionDomainAgents,
		PermissionDomainAgentRuns,
		PermissionDomainAgentMemories,
		PermissionDomainAlerts,
		PermissionDomainAccount,
		PermissionDomainDeliveries,
		PermissionDomainLocations,
		PermissionDomainSettlements,
		PermissionDomainTransactions,
		PermissionDomainBatches,
		PermissionDomainCarriers,
		PermissionDomainCustomerGroups,
		PermissionDomainCustomers,
		PermissionDomainCustomerUsers,
		PermissionDomainDepartmentPicks,
		PermissionDomainDepartments,
		PermissionDomainDiscounts,
		PermissionDomainEdiLocations,
		PermissionDomainEdiRuns,
		PermissionDomainEmailLogs,
		PermissionDomainErrorLogs,
		PermissionDomainProducts,
		PermissionDomainInventory,
		PermissionDomainInventoryChangeLogs,
		PermissionDomainInventoryLogs,
		PermissionDomainInvoices,
		PermissionDomainCategories,
		PermissionDomainMachines,
		PermissionDomainMachineDowntime,
		PermissionDomainProductionSchedules,
		PermissionDomainDemandOverrides,
		PermissionDomainMaterials,
		PermissionDomainOrganization,
		PermissionDomainPaymentTerms,
		PermissionDomainPermissions,
		PermissionDomainParts,
		PermissionDomainPicks,
		PermissionDomainReceivingOrders,
		PermissionDomainProductGroups,
		PermissionDomainItems,
		PermissionDomainProductionRuns,
		PermissionDomainProductionStepTransformations,
		PermissionDomainProductionSteps,
		PermissionDomainProductLines,
		PermissionDomainProductVariations,
		PermissionDomainProperties,
		PermissionDomainPurchaseOrders,
		PermissionDomainSuppliers,
		PermissionDomainReceiving,
		PermissionDomainProductLineAccess,
		PermissionDomainRoles,
		PermissionDomainSalesOrders,
		PermissionDomainTerritories,
		PermissionDomainSalesTargets,
		PermissionDomainScanningStations,
		PermissionDomainScanningErrorLogs,
		PermissionDomainShifts,
		PermissionDomainShipments,
		PermissionDomainShippingCases,
		PermissionDomainShippingTerms,
		PermissionDomainSupplies,
		PermissionDomainSystemProperties,
		PermissionDomainTeamUsers,
		PermissionDomainUnits,
		PermissionDomainUnitGroups,
		PermissionDomainRequestLogs,
		PermissionDomainAuditEvents,
		PermissionDomainAPIKeys,
		PermissionDomainSandbox,
		PermissionDomainAddresses,
		PermissionDomainIntegrations,
		PermissionDomainAdjustmentTypes,
		PermissionDomainPriorities,
		PermissionDomainProductTypes,
		PermissionDomainJobs,
		PermissionDomainMessaging,
	}
}

// IsValid reports whether the domain is one the platform recognizes.
func (d PermissionDomain) IsValid() bool {
	for _, known := range AllPermissionDomains() {
		if d == known {
			return true
		}
	}
	return false
}
