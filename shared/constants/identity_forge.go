package constants

// PermissionArea is one of the areas a role's access is set by. Every permission code belongs to exactly one area.
type PermissionArea string

const (
	// PermissionAreaCatalog covers items, categories, product lines, properties and units.
	PermissionAreaCatalog PermissionArea = "catalog"
	// PermissionAreaSales covers customers, contacts, sales orders, pricing and sales reps.
	PermissionAreaSales PermissionArea = "sales"
	// PermissionAreaPurchasing covers suppliers, purchase orders and receiving.
	PermissionAreaPurchasing PermissionArea = "purchasing"
	// PermissionAreaInventory covers inventory, locations, picks, shipments and carriers.
	PermissionAreaInventory PermissionArea = "inventory"
	// PermissionAreaProduction covers production runs, batches, scanning, work centers and machines.
	PermissionAreaProduction PermissionArea = "production"
	// PermissionAreaFinance covers invoices, payments, payment terms and costs.
	PermissionAreaFinance PermissionArea = "finance"
	// PermissionAreaSettings covers the team, roles, API keys, integrations, logs and account settings.
	PermissionAreaSettings PermissionArea = "settings"
)

// PermissionAreas lists every area in the order roles show them.
func PermissionAreas() []PermissionArea {
	return []PermissionArea{
		PermissionAreaCatalog, PermissionAreaSales, PermissionAreaPurchasing, PermissionAreaInventory,
		PermissionAreaProduction, PermissionAreaFinance, PermissionAreaSettings,
	}
}

func (a PermissionArea) IsValid() bool {
	switch a {
	case PermissionAreaCatalog, PermissionAreaSales, PermissionAreaPurchasing, PermissionAreaInventory,
		PermissionAreaProduction, PermissionAreaFinance, PermissionAreaSettings:
		return true
	default:
		return false
	}
}

func (a PermissionArea) EnumValues() []string {
	out := make([]string, 0, 7)
	for _, v := range PermissionAreas() {
		out = append(out, string(v))
	}
	return out
}

// AccessLevel is how much a role can do in one area. Each level includes the ones below it.
type AccessLevel string

const (
	// AccessLevelNone grants nothing in the area beyond the permissions every member holds.
	AccessLevelNone AccessLevel = "none"
	// AccessLevelRead lists and retrieves.
	AccessLevelRead AccessLevel = "read"
	// AccessLevelEdit adds create and update.
	AccessLevelEdit AccessLevel = "edit"
	// AccessLevelFull adds delete and the area's sensitive actions.
	AccessLevelFull AccessLevel = "full"
)

// AccessLevels lists every level from least to most access.
func AccessLevels() []AccessLevel {
	return []AccessLevel{AccessLevelNone, AccessLevelRead, AccessLevelEdit, AccessLevelFull}
}

func (l AccessLevel) IsValid() bool {
	switch l {
	case AccessLevelNone, AccessLevelRead, AccessLevelEdit, AccessLevelFull:
		return true
	default:
		return false
	}
}

func (l AccessLevel) EnumValues() []string {
	return []string{string(AccessLevelNone), string(AccessLevelRead), string(AccessLevelEdit), string(AccessLevelFull)}
}

// Rank orders levels: none 0, read 1, edit 2, full 3. An unknown level ranks as none.
func (l AccessLevel) Rank() int {
	switch l {
	case AccessLevelRead:
		return 1
	case AccessLevelEdit:
		return 2
	case AccessLevelFull:
		return 3
	default:
		return 0
	}
}

// RoleTemplate is a starting point for a new role. A role remembers the template it was created from, but later
// changes to the template don't reach it.
type RoleTemplate string

const (
	RoleTemplateOwner                RoleTemplate = "owner"
	RoleTemplateAdmin                RoleTemplate = "admin"
	RoleTemplateSales                RoleTemplate = "sales"
	RoleTemplateCustomerService      RoleTemplate = "customer_service"
	RoleTemplatePurchasing           RoleTemplate = "purchasing"
	RoleTemplateWarehouse            RoleTemplate = "warehouse"
	RoleTemplateProductionSupervisor RoleTemplate = "production_supervisor"
	RoleTemplateOperator             RoleTemplate = "operator"
	RoleTemplateAccounting           RoleTemplate = "accounting"
	RoleTemplateReadOnly             RoleTemplate = "read_only"
)

func (t RoleTemplate) IsValid() bool {
	switch t {
	case RoleTemplateOwner, RoleTemplateAdmin, RoleTemplateSales, RoleTemplateCustomerService, RoleTemplatePurchasing,
		RoleTemplateWarehouse, RoleTemplateProductionSupervisor, RoleTemplateOperator, RoleTemplateAccounting,
		RoleTemplateReadOnly:
		return true
	default:
		return false
	}
}

func (t RoleTemplate) EnumValues() []string {
	return []string{
		string(RoleTemplateOwner), string(RoleTemplateAdmin), string(RoleTemplateSales), string(RoleTemplateCustomerService),
		string(RoleTemplatePurchasing), string(RoleTemplateWarehouse), string(RoleTemplateProductionSupervisor),
		string(RoleTemplateOperator), string(RoleTemplateAccounting), string(RoleTemplateReadOnly),
	}
}

// PortalAccess is whether an account user can sign in: never invited, invited but not yet signed in, or signed in.
type PortalAccess string

const (
	PortalAccessNone    PortalAccess = "none"
	PortalAccessInvited PortalAccess = "invited"
	PortalAccessActive  PortalAccess = "active"
)

func (p PortalAccess) IsValid() bool {
	switch p {
	case PortalAccessNone, PortalAccessInvited, PortalAccessActive:
		return true
	default:
		return false
	}
}

func (p PortalAccess) EnumValues() []string {
	return []string{string(PortalAccessNone), string(PortalAccessInvited), string(PortalAccessActive)}
}

// CommissionEligibility is whether an account user can be named as a sales rep.
type CommissionEligibility string

const (
	CommissionEligibilityEligible   CommissionEligibility = "eligible"
	CommissionEligibilityIneligible CommissionEligibility = "ineligible"
)

func (c CommissionEligibility) IsValid() bool {
	return c == CommissionEligibilityEligible || c == CommissionEligibilityIneligible
}

func (c CommissionEligibility) EnumValues() []string {
	return []string{string(CommissionEligibilityEligible), string(CommissionEligibilityIneligible)}
}

// Bool is the stored flag for this value.
func (c CommissionEligibility) Bool() bool { return c == CommissionEligibilityEligible }

// CommissionEligibilityOf is the value for a stored flag.
func CommissionEligibilityOf(eligible bool) CommissionEligibility {
	if eligible {
		return CommissionEligibilityEligible
	}
	return CommissionEligibilityIneligible
}

func (a *PermissionArea) StringPtr() *string {
	if a == nil {
		return nil
	}
	s := string(*a)
	return &s
}

func (l *AccessLevel) StringPtr() *string {
	if l == nil {
		return nil
	}
	s := string(*l)
	return &s
}

func (t *RoleTemplate) StringPtr() *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}

func (p *PortalAccess) StringPtr() *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func (c *CommissionEligibility) StringPtr() *string {
	if c == nil {
		return nil
	}
	s := string(*c)
	return &s
}
