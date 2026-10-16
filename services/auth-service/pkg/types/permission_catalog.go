package types

import (
	"sort"
	"strings"

	"github.com/open-mrp/api/shared/constants"
)

// CatalogPermission is one grantable permission code with the area and level it belongs to. A role holds it when its
// access in Area is at least Level; grants and revokes adjust that one code at a time.
type CatalogPermission struct {
	Code        string
	Name        string
	Description string
	Area        constants.PermissionArea
	Level       constants.AccessLevel
}

// domainSpec places a permission domain in the catalog.
type domainSpec struct {
	area constants.PermissionArea
	name string
	// about completes "…" in each action's description, e.g. "sales orders".
	about   string
	actions []Action
	// full lists the actions that need full access although their verb alone would need less: the area's sensitive
	// actions.
	full []Action
	// baseline marks a domain every member holds whatever the role's access: their own profile, messaging, notes and
	// alerts, and polling their own async jobs. Its codes sit at level none and can still be revoked.
	baseline []Action
}

var (
	crud     = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete}
	readOnly = []Action{ActionRead}
	readEdit = []Action{ActionRead, ActionUpdate}
)

// permissionDomainSpecs is the catalog source. Domain codes keep their stored names (departments are work centers,
// scanners are scanning stations, relevant_products is product-line access) because roles store them.
var permissionDomainSpecs = map[PermissionDomain]domainSpec{
	PermissionDomainItems:                         {area: constants.PermissionAreaCatalog, name: "Items", about: "items", actions: crud},
	PermissionDomainMaterials:                     {area: constants.PermissionAreaCatalog, name: "Materials", about: "materials", actions: crud},
	PermissionDomainParts:                         {area: constants.PermissionAreaCatalog, name: "Parts", about: "parts", actions: crud},
	PermissionDomainProducts:                      {area: constants.PermissionAreaCatalog, name: "Products", about: "products", actions: crud},
	PermissionDomainProductGroups:                 {area: constants.PermissionAreaCatalog, name: "Product groups", about: "product groups", actions: crud},
	PermissionDomainProductLines:                  {area: constants.PermissionAreaCatalog, name: "Product lines", about: "product lines", actions: crud},
	PermissionDomainProductVariations:             {area: constants.PermissionAreaCatalog, name: "Product variations", about: "product variations", actions: crud},
	PermissionDomainProductTypes:                  {area: constants.PermissionAreaCatalog, name: "Product types", about: "product types", actions: crud},
	PermissionDomainCategories:                    {area: constants.PermissionAreaCatalog, name: "Item categories", about: "item categories", actions: crud},
	PermissionDomainProperties:                    {area: constants.PermissionAreaCatalog, name: "Properties", about: "properties and attributes", actions: crud},
	PermissionDomainUnits:                         {area: constants.PermissionAreaCatalog, name: "Units", about: "units of measure", actions: crud},
	PermissionDomainUnitGroups:                    {area: constants.PermissionAreaCatalog, name: "Unit groups", about: "unit groups", actions: crud},
	PermissionDomainSupplies:                      {area: constants.PermissionAreaCatalog, name: "Supplies", about: "supplies", actions: crud},
	PermissionDomainCustomers:                     {area: constants.PermissionAreaSales, name: "Customers", about: "customers", actions: crud},
	PermissionDomainCustomerGroups:                {area: constants.PermissionAreaSales, name: "Account groups", about: "account groups", actions: crud},
	PermissionDomainCustomerUsers:                 {area: constants.PermissionAreaSales, name: "Contacts", about: "customer and supplier contacts", actions: crud},
	PermissionDomainAddresses:                     {area: constants.PermissionAreaSales, name: "Addresses", about: "addresses", actions: crud},
	PermissionDomainSalesOrders:                   {area: constants.PermissionAreaSales, name: "Sales orders", about: "sales orders", actions: crud},
	PermissionDomainDiscounts:                     {area: constants.PermissionAreaSales, name: "Pricing", about: "prices and discounts", actions: crud},
	PermissionDomainProductLineAccess:             {area: constants.PermissionAreaSales, name: "Product-line access", about: "which product lines customers can order", actions: crud},
	PermissionDomainTerritories:                   {area: constants.PermissionAreaSales, name: "Territories", about: "sales rep territories", actions: crud},
	PermissionDomainSalesTargets:                  {area: constants.PermissionAreaSales, name: "Sales targets", about: "sales targets", actions: crud},
	PermissionDomainPriorities:                    {area: constants.PermissionAreaSales, name: "Priorities", about: "order priorities", actions: readOnly},
	PermissionDomainSuppliers:                     {area: constants.PermissionAreaPurchasing, name: "Suppliers", about: "suppliers and supplier items", actions: crud},
	PermissionDomainPurchaseOrders:                {area: constants.PermissionAreaPurchasing, name: "Purchase orders", about: "purchase orders", actions: crud},
	PermissionDomainReceivingOrders:               {area: constants.PermissionAreaPurchasing, name: "Receiving orders", about: "receiving orders", actions: crud},
	PermissionDomainReceiving:                     {area: constants.PermissionAreaPurchasing, name: "Receiving", about: "receipts", actions: crud},
	PermissionDomainInventory:                     {area: constants.PermissionAreaInventory, name: "Inventory", about: "inventory", actions: crud},
	PermissionDomainInventoryChangeLogs:           {area: constants.PermissionAreaInventory, name: "Inventory change logs", about: "inventory change logs", actions: readOnly},
	PermissionDomainInventoryLogs:                 {area: constants.PermissionAreaInventory, name: "Inventory logs", about: "inventory logs", actions: readOnly},
	PermissionDomainLocations:                     {area: constants.PermissionAreaInventory, name: "Locations", about: "locations and sites", actions: crud},
	PermissionDomainAdjustmentTypes:               {area: constants.PermissionAreaInventory, name: "Adjustment types", about: "inventory adjustment types", actions: readOnly},
	PermissionDomainPicks:                         {area: constants.PermissionAreaInventory, name: "Picks", about: "picks", actions: crud},
	PermissionDomainDepartmentPicks:               {area: constants.PermissionAreaInventory, name: "Work center picks", about: "picks for work centers", actions: crud},
	PermissionDomainShipments:                     {area: constants.PermissionAreaInventory, name: "Shipments", about: "shipments", actions: crud},
	PermissionDomainShippingCases:                 {area: constants.PermissionAreaInventory, name: "Shipping cases", about: "shipping cases", actions: crud},
	PermissionDomainDeliveries:                    {area: constants.PermissionAreaInventory, name: "Deliveries", about: "deliveries", actions: crud},
	PermissionDomainCarriers:                      {area: constants.PermissionAreaInventory, name: "Carriers", about: "carriers and service levels", actions: crud},
	PermissionDomainShippingTerms:                 {area: constants.PermissionAreaInventory, name: "Shipping terms", about: "shipping terms", actions: crud},
	PermissionDomainProductionRuns:                {area: constants.PermissionAreaProduction, name: "Production runs", about: "production runs", actions: crud},
	PermissionDomainProductionSchedules:           {area: constants.PermissionAreaProduction, name: "Production schedules", about: "production schedules", actions: crud},
	PermissionDomainProductionSteps:               {area: constants.PermissionAreaProduction, name: "Production steps", about: "production steps and flows", actions: crud},
	PermissionDomainProductionStepTransformations: {area: constants.PermissionAreaProduction, name: "Step transformations", about: "production step transformations", actions: crud},
	PermissionDomainBatches:                       {area: constants.PermissionAreaProduction, name: "Batches", about: "batches and scans", actions: crud},
	PermissionDomainDepartments:                   {area: constants.PermissionAreaProduction, name: "Work centers", about: "work centers", actions: crud},
	PermissionDomainMachines:                      {area: constants.PermissionAreaProduction, name: "Machines", about: "machines", actions: crud},
	PermissionDomainMachineDowntime:               {area: constants.PermissionAreaProduction, name: "Machine downtime", about: "machine downtime", actions: crud},
	PermissionDomainScanningStations:              {area: constants.PermissionAreaProduction, name: "Scanning stations", about: "scanning stations", actions: crud},
	PermissionDomainScanningErrorLogs:             {area: constants.PermissionAreaProduction, name: "Scanning error logs", about: "scanning error logs", actions: readOnly},
	PermissionDomainShifts:                        {area: constants.PermissionAreaProduction, name: "Shifts", about: "shifts", actions: crud},
	PermissionDomainDemandOverrides:               {area: constants.PermissionAreaProduction, name: "Demand overrides", about: "demand overrides", actions: crud},
	PermissionDomainInvoices:                      {area: constants.PermissionAreaFinance, name: "Invoices", about: "invoices", actions: crud},
	PermissionDomainTransactions:                  {area: constants.PermissionAreaFinance, name: "Payments", about: "payments and other transactions", actions: crud},
	PermissionDomainSettlements:                   {area: constants.PermissionAreaFinance, name: "Settlements", about: "settlements", actions: crud},
	PermissionDomainPaymentTerms:                  {area: constants.PermissionAreaFinance, name: "Payment terms", about: "payment terms", actions: crud},
	PermissionDomainCosts:                         {area: constants.PermissionAreaFinance, name: "Costs", about: "costs and margins", actions: readOnly, full: readOnly},
	PermissionDomainTeamUsers:                     {area: constants.PermissionAreaSettings, name: "Team", about: "your team's users", actions: crud},
	PermissionDomainRoles:                         {area: constants.PermissionAreaSettings, name: "Roles", about: "roles", actions: crud, full: []Action{ActionCreate, ActionUpdate}},
	PermissionDomainPermissions:                   {area: constants.PermissionAreaSettings, name: "Permissions", about: "the permission catalog", actions: readOnly},
	PermissionDomainOrganization:                  {area: constants.PermissionAreaSettings, name: "Accounts", about: "account details", actions: crud},
	PermissionDomainAPIKeys:                       {area: constants.PermissionAreaSettings, name: "API keys", about: "API keys", actions: crud, full: crud},
	PermissionDomainIntegrations:                  {area: constants.PermissionAreaSettings, name: "Integrations", about: "integrations", actions: crud, full: crud},
	PermissionDomainBilling:                       {area: constants.PermissionAreaSettings, name: "Billing", about: "agent spend and the spending cap", actions: readEdit, full: readEdit},
	PermissionDomainSandbox:                       {area: constants.PermissionAreaSettings, name: "Sandboxes", about: "sandboxes", actions: crud},
	PermissionDomainSystemProperties:              {area: constants.PermissionAreaSettings, name: "System properties", about: "system properties", actions: readEdit},
	PermissionDomainAuditEvents:                   {area: constants.PermissionAreaSettings, name: "Audit events", about: "audit events", actions: readOnly},
	PermissionDomainRequestLogs:                   {area: constants.PermissionAreaSettings, name: "Request logs", about: "API request logs", actions: readOnly},
	PermissionDomainEmailLogs:                     {area: constants.PermissionAreaSettings, name: "Email logs", about: "email logs", actions: readOnly},
	PermissionDomainErrorLogs:                     {area: constants.PermissionAreaSettings, name: "Error logs", about: "error logs", actions: readOnly},
	PermissionDomainAgents:                        {area: constants.PermissionAreaSettings, name: "Agents", about: "agents", actions: crud},
	PermissionDomainAgentRuns:                     {area: constants.PermissionAreaSettings, name: "Agent runs", about: "agent runs", actions: crud},
	PermissionDomainAgentMemories:                 {area: constants.PermissionAreaSettings, name: "Agent memories", about: "agent memories", actions: crud},
	PermissionDomainAccount:                       {area: constants.PermissionAreaSettings, name: "Self", about: "your own profile", actions: readEdit, baseline: readEdit},
	PermissionDomainMessaging:                     {area: constants.PermissionAreaSettings, name: "Messaging", about: "conversations and messages", actions: crud, baseline: crud},
	PermissionDomainNotes:                         {area: constants.PermissionAreaSettings, name: "Notes", about: "notes", actions: crud, baseline: crud},
	PermissionDomainAlerts:                        {area: constants.PermissionAreaSettings, name: "Alerts", about: "alerts", actions: crud, baseline: crud},
	PermissionDomainJobs:                          {area: constants.PermissionAreaSettings, name: "Async jobs", about: "async jobs", actions: []Action{ActionRead, ActionDelete}, baseline: readOnly},
}

var actionVerbs = map[Action]string{
	ActionRead:   "List and view",
	ActionCreate: "Create",
	ActionUpdate: "Update",
	ActionDelete: "Delete",
}

var actionNames = map[Action]string{
	ActionRead:   "read",
	ActionCreate: "create",
	ActionUpdate: "update",
	ActionDelete: "delete",
}

func levelFor(spec domainSpec, action Action) constants.AccessLevel {
	if containsAction(spec.baseline, action) {
		return constants.AccessLevelNone
	}
	if containsAction(spec.full, action) {
		return constants.AccessLevelFull
	}
	switch action {
	case ActionRead:
		return constants.AccessLevelRead
	case ActionCreate, ActionUpdate:
		return constants.AccessLevelEdit
	default:
		return constants.AccessLevelFull
	}
}

func containsAction(actions []Action, a Action) bool {
	for _, x := range actions {
		if x == a {
			return true
		}
	}
	return false
}

var (
	catalog       []CatalogPermission
	catalogByCode map[string]CatalogPermission
)

func init() {
	for domain, spec := range permissionDomainSpecs {
		for _, action := range spec.actions {
			code := Permission{Domain: domain, Action: action}.String()
			catalog = append(catalog, CatalogPermission{
				Code:        code,
				Name:        spec.name + ": " + actionNames[action],
				Description: actionVerbs[action] + " " + spec.about + ".",
				Area:        spec.area,
				Level:       levelFor(spec, action),
			})
		}
	}
	sort.Slice(catalog, func(i, j int) bool { return catalog[i].Code < catalog[j].Code })
	catalogByCode = make(map[string]CatalogPermission, len(catalog))
	for _, p := range catalog {
		catalogByCode[p.Code] = p
	}
}

// PermissionCatalog lists every grantable permission, sorted by code. Callers must not modify it.
func PermissionCatalog() []CatalogPermission { return catalog }

// CatalogPermissionByCode looks one permission up by its `<domain>:<action>` code.
func CatalogPermissionByCode(code string) (CatalogPermission, bool) {
	p, ok := catalogByCode[code]
	return p, ok
}

// RoleAccess is a role's level in each area. A missing area is none.
type RoleAccess map[constants.PermissionArea]constants.AccessLevel

// Level is the access in area, none when unset.
func (a RoleAccess) Level(area constants.PermissionArea) constants.AccessLevel {
	if l, ok := a[area]; ok && l.IsValid() {
		return l
	}
	return constants.AccessLevelNone
}

// Complete returns the access with every area set, none where it was unset.
func (a RoleAccess) Complete() RoleAccess {
	out := make(RoleAccess, len(constants.PermissionAreas()))
	for _, area := range constants.PermissionAreas() {
		out[area] = a.Level(area)
	}
	return out
}

// LevelPermissions lists the codes the access levels alone grant, sorted.
func LevelPermissions(access RoleAccess) []string {
	var out []string
	for _, p := range catalog {
		if access.Level(p.Area).Rank() >= p.Level.Rank() {
			out = append(out, p.Code)
		}
	}
	return out
}

// EffectivePermissions is what a role holds: the codes its access levels grant, plus grants, minus revokes. Sorted.
// A revoke wins over a grant of the same code.
func EffectivePermissions(access RoleAccess, grants, revokes []string) []string {
	set := map[string]bool{}
	for _, c := range LevelPermissions(access) {
		set[c] = true
	}
	for _, c := range grants {
		if _, ok := catalogByCode[c]; ok {
			set[c] = true
		}
	}
	for _, c := range revokes {
		delete(set, c)
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// RoleDefinition is how a role is described: access per area, plus the codes added and removed on top of it.
type RoleDefinition struct {
	Access  RoleAccess
	Grants  []string
	Revokes []string
}

// DefinitionFromEffective describes a set of held codes with the fewest grants and revokes: each area takes the level
// that differs least from what the role holds there (the lower level on a tie). Codes outside the catalog are ignored.
// It describes roles saved before access levels existed, and normalizes a definition after an edit.
func DefinitionFromEffective(held []string) RoleDefinition {
	heldSet := map[string]bool{}
	for _, c := range held {
		if _, ok := catalogByCode[c]; ok {
			heldSet[c] = true
		}
	}
	access := RoleAccess{}
	for _, area := range constants.PermissionAreas() {
		best, bestCost := constants.AccessLevelNone, -1
		for _, level := range constants.AccessLevels() {
			cost := 0
			for _, p := range catalog {
				if p.Area != area {
					continue
				}
				if (level.Rank() >= p.Level.Rank()) != heldSet[p.Code] {
					cost++
				}
			}
			if bestCost < 0 || cost < bestCost {
				best, bestCost = level, cost
			}
		}
		access[area] = best
	}
	return DefinitionWithAccess(access, heldSet)
}

// DefinitionWithAccess describes held codes against fixed access levels: grants are what the levels don't give,
// revokes what they give but isn't held.
func DefinitionWithAccess(access RoleAccess, held map[string]bool) RoleDefinition {
	def := RoleDefinition{Access: access.Complete(), Grants: []string{}, Revokes: []string{}}
	for _, p := range catalog {
		byLevel := access.Level(p.Area).Rank() >= p.Level.Rank()
		switch {
		case held[p.Code] && !byLevel:
			def.Grants = append(def.Grants, p.Code)
		case !held[p.Code] && byLevel:
			def.Revokes = append(def.Revokes, p.Code)
		}
	}
	return def
}

// UnknownPermissionCodes returns the codes that aren't in the catalog, in input order.
func UnknownPermissionCodes(codes []string) []string {
	var out []string
	for _, c := range codes {
		if _, ok := catalogByCode[strings.TrimSpace(c)]; !ok {
			out = append(out, c)
		}
	}
	return out
}
