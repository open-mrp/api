package types

import (
	"slices"
	"testing"

	"github.com/open-mrp/api/shared/constants"
)

// A domain missing from the catalog can't be granted through a role's access or grants, so every recognized domain
// needs a place in it.
func TestPermissionCatalogCoversEveryDomain(t *testing.T) {
	t.Parallel()
	for _, d := range AllPermissionDomains() {
		if _, ok := permissionDomainSpecs[d]; !ok {
			t.Errorf("domain %q has no catalog entry", d)
		}
	}
	if len(permissionDomainSpecs) != len(AllPermissionDomains()) {
		t.Errorf("catalog has %d domains, the platform recognizes %d", len(permissionDomainSpecs), len(AllPermissionDomains()))
	}
	for _, p := range PermissionCatalog() {
		if !p.Area.IsValid() || !p.Level.IsValid() || p.Name == "" || p.Description == "" {
			t.Errorf("incomplete catalog entry %+v", p)
		}
	}
}

// suppliers:delete stays its own full-level code: supplier delete requires it.
func TestSupplierDeleteIsFullPurchasing(t *testing.T) {
	t.Parallel()
	p, ok := CatalogPermissionByCode("suppliers:delete")
	if !ok || p.Area != constants.PermissionAreaPurchasing || p.Level != constants.AccessLevelFull {
		t.Fatalf("suppliers:delete = %+v, %v", p, ok)
	}
}

func TestEffectivePermissionsLevelsGrantsRevokes(t *testing.T) {
	t.Parallel()
	got := EffectivePermissions(RoleAccess{constants.PermissionAreaSales: constants.AccessLevelRead},
		[]string{"invoices:read", "not_a:code"}, []string{"customers:read", "messaging:delete"})

	for _, want := range []string{"sales_orders:read", "invoices:read", "messaging:create", "self:read", "jobs:read"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing %s", want)
		}
	}
	for _, unwanted := range []string{"customers:read", "sales_orders:create", "messaging:delete", "not_a:code", "jobs:delete"} {
		if slices.Contains(got, unwanted) {
			t.Errorf("unexpected %s", unwanted)
		}
	}
}

// A role saved before access levels existed is described by its held codes; the description must give back exactly
// what it holds.
func TestDefinitionFromEffectiveRoundTrips(t *testing.T) {
	t.Parallel()
	held := []string{"sales_orders:read", "sales_orders:create", "sales_orders:update", "sales_orders:delete", "messaging:read", "jobs:read", "customers:read"}
	def := DefinitionFromEffective(held)
	got := EffectivePermissions(def.Access, def.Grants, def.Revokes)
	want := slices.Clone(held)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("round trip = %v, want %v", got, want)
	}
	if def.Access[constants.PermissionAreaCatalog] != constants.AccessLevelNone {
		t.Errorf("catalog access = %s, want none", def.Access[constants.PermissionAreaCatalog])
	}
}

func TestDefinitionOfEverythingIsFullEverywhere(t *testing.T) {
	t.Parallel()
	var all []string
	for _, p := range PermissionCatalog() {
		all = append(all, p.Code)
	}
	def := DefinitionFromEffective(all)
	for _, area := range constants.PermissionAreas() {
		if def.Access[area] != constants.AccessLevelFull {
			t.Errorf("%s = %s, want full", area, def.Access[area])
		}
	}
	if len(def.Grants) != 0 || len(def.Revokes) != 0 {
		t.Errorf("grants %v revokes %v, want none", def.Grants, def.Revokes)
	}
}

func TestRoleTemplatesAreValid(t *testing.T) {
	t.Parallel()
	seen := map[constants.RoleTemplate]bool{}
	for _, tpl := range RoleTemplates() {
		seen[tpl.Code] = true
		if unknown := UnknownPermissionCodes(tpl.Grants); len(unknown) > 0 {
			t.Errorf("%s grants unknown codes %v", tpl.Code, unknown)
		}
		for _, area := range constants.PermissionAreas() {
			if !tpl.Access.Level(area).IsValid() {
				t.Errorf("%s has no level for %s", tpl.Code, area)
			}
		}
	}
	for _, v := range constants.RoleTemplate("").EnumValues() {
		if !seen[constants.RoleTemplate(v)] {
			t.Errorf("template %s has no spec", v)
		}
	}
}
