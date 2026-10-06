package main

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"testing"

	"github.com/open-mrp/api/services/api-gateway/pkg/costguard"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
)

// costFieldAllowlist names fields costguard.IsCostName matches that are not the seller's cost data, keyed by "<Go type>.<Go field>".
var costFieldAllowlist = map[string]string{
	"analyticsep.AnalyzeCustomerPricingRequest.TargetGrossMargin": "a threshold the caller supplies, not a recorded cost",
	"analyticsep.AnalyzeRealizedMarginsRequest.TargetGrossMargin": "a threshold the caller supplies, not a recorded cost",
}

// TestEndpointCostFieldsAreMarked walks the request and response type of every registered endpoint and fails when a field named like cost or margin data is not tagged sensitive:"cost", so a new cost field cannot ship visible to callers without costs:read, or to customer and supplier portals.
//
// A tagged response field must serialize as null once cleared, so it has to be a pointer, slice, map or interface; a number or string would read as a real zero. An endpoint that requires costs:read outright needs no marks on its response, since nobody without the permission reaches it.
func TestEndpointCostFieldsAreMarked(t *testing.T) {
	t.Parallel()

	found := 0
	for _, group := range buildAllGroups() {
		for _, ep := range group.Endpoints {
			where := ep.GetMethod() + " " + ep.GetRoute()
			sites := costSites(ep.GetRequestType())
			if !requiresOnlyCostsRead(ep) {
				response := costSites(ep.GetResponseType())
				for i := range response {
					response[i].response = true
				}
				sites = append(sites, response...)
			}
			for _, site := range sites {
				found++
				switch {
				case site.unreachable != "":
					t.Errorf("%s: %s is cost data but %s, so costguard cannot clear it", where, site.field, site.unreachable)
				case !site.tagged:
					t.Errorf("%s: %s (json %q) reads as cost or margin data but is not tagged sensitive:\"cost\"; tag it, or add it to costFieldAllowlist with the reason it is not", where, site.field, site.path)
				case site.response && !nullableKind(site.kind):
					t.Errorf("%s: %s is tagged sensitive:\"cost\" but is a %s, which serializes as a zero rather than null once cleared; make it a pointer", where, site.field, site.kind)
				}
			}
			if typ := ep.GetResponseType(); typ != nil && hasTaggedResponseSite(typ) && !costguard.HasCostFields(typ) {
				t.Errorf("%s: its response carries a sensitive:\"cost\" field that costguard's plan does not reach", where)
			}
		}
	}

	if found == 0 {
		t.Fatal("no cost fields found across the endpoint surface; the walk is no longer reaching request and response types")
	}
}

func requiresOnlyCostsRead(ep any) bool {
	spec := reflect.ValueOf(ep)
	if spec.Kind() == reflect.Pointer {
		spec = spec.Elem()
	}
	perms, ok := spec.FieldByName("RequiredPermissions").Interface().(types.AnyOfPermissions)
	if !ok || len(perms) == 0 {
		return false
	}
	return !slices.ContainsFunc(perms, func(p types.Permission) bool {
		return p != types.Permission{Domain: types.PermissionDomainCosts, Action: types.ActionRead}
	})
}

type costSite struct {
	path        string
	field       string
	kind        reflect.Kind
	tagged      bool
	response    bool
	unreachable string
}

func costSites(typ reflect.Type) []costSite {
	if typ == nil {
		return nil
	}
	var out []costSite
	walkCostSites(typ, "", "", make(map[reflect.Type]bool), &out, 0)
	return out
}

func walkCostSites(typ reflect.Type, prefix, unreachable string, visited map[reflect.Type]bool, out *[]costSite, depth int) {
	typ = derefType(typ)
	if typ == nil || typ.Kind() != reflect.Struct || depth > maxWalkDepth || visited[typ] {
		return
	}
	visited[typ] = true
	defer delete(visited, typ)

	for i := range typ.NumField() {
		sf := typ.Field(i)
		name, skip := jsonFieldName(sf)
		if skip {
			continue
		}
		if sf.Anonymous {
			reason := unreachable
			if !sf.IsExported() && reason == "" {
				reason = fmt.Sprintf("it is promoted from the unexported embedded %s", sf.Type)
			}
			walkCostSites(sf.Type, joinPath(prefix, name), reason, visited, out, depth+1)
			continue
		}
		if !sf.IsExported() || name == "" {
			continue
		}

		path := joinPath(prefix, name)
		field := fmt.Sprintf("%s.%s", typ, sf.Name)
		tagged := costguard.IsCostField(sf)
		if tagged || costguard.IsCostName(name) {
			if _, allowed := costFieldAllowlist[field]; !allowed || tagged {
				*out = append(*out, costSite{path: path, field: field, kind: sf.Type.Kind(), tagged: tagged, unreachable: unreachable})
			}
			if tagged {
				continue
			}
		}

		ft := derefType(sf.Type)
		switch ft.Kind() {
		case reflect.Struct:
			if inner, wrapped := unexportedWrapperInner(ft); wrapped {
				walkCostSites(inner, path, unreachable, visited, out, depth+1)
				continue
			}
			walkCostSites(ft, path, unreachable, visited, out, depth+1)
		case reflect.Slice, reflect.Array, reflect.Map:
			walkCostSites(ft.Elem(), path, unreachable, visited, out, depth+1)
		}
	}
}

func nullableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return true
	default:
		return false
	}
}

func hasTaggedResponseSite(typ reflect.Type) bool {
	return slices.ContainsFunc(costSites(typ), func(s costSite) bool { return s.tagged })
}

// TestCostSites_flagsUnmarkedAndUnclearable guards the guard: the walk must still notice an unmarked cost field, a marked one that cannot read as null, and one costguard cannot reach.
func TestCostSites_flagsUnmarkedAndUnclearable(t *testing.T) {
	t.Parallel()

	type rate struct {
		Value string `json:"value"`
	}
	type promoted struct {
		Margin *string `json:"margin" sensitive:"cost"`
	}
	type resource struct {
		ID          string            `json:"id"`
		UnitCost    *rate             `json:"unit_cost" sensitive:"cost"`
		TotalCost   float64           `json:"total_cost" sensitive:"cost"`
		GrossMargin *string           `json:"gross_margin"`
		LaborRate   *rate             `json:"labor_rate"`
		Lines       []map[string]rate `json:"lines"`
		Costing     struct {
			Cogs *string `json:"cogs"`
		} `json:"costing"`
		promoted
	}

	got := map[string]costSite{}
	for _, s := range costSites(reflect.TypeFor[*resource]()) {
		got[s.path] = s
	}
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if want := []string{"costing.cogs", "gross_margin", "labor_rate", "margin", "total_cost", "unit_cost"}; !slices.Equal(keys, want) {
		t.Fatalf("found %v, want %v", keys, want)
	}
	if !got["unit_cost"].tagged || got["gross_margin"].tagged || got["labor_rate"].tagged || got["costing.cogs"].tagged {
		t.Errorf("tagged flags wrong: %+v", got)
	}
	if nullableKind(got["total_cost"].kind) {
		t.Errorf("a float cost field must be reported as not nullable")
	}
	if got["margin"].unreachable == "" {
		t.Errorf("a cost field promoted from an unexported embedded struct must be reported as unreachable")
	}
	if !costguard.IsCostName("weighted_average_unit_cost") || costguard.IsCostName("unit_value") || costguard.IsCostName("costume") {
		t.Errorf("IsCostName must match whole tokens only")
	}
}
