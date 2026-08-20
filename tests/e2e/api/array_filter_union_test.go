//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file exercises EVERY list endpoint that accepts an array-valued filter
// against the union/exclusion invariant the product requires:
//
//	results(filter=[A, B]) == results(filter=[A]) ∪ results(filter=[B])
//
// expressed over the returned row ids. That single invariant catches the three
// ways an array filter goes wrong:
//   - "only the first value is applied"  → S12 misses S2's rows (union fails)
//   - "AND instead of OR / IN"           → S12 ⊊ S1 ∪ S2        (union fails)
//   - "filter ignored / over-broad"      → S12 ⊋ S1 ∪ S2        (exclusion fails)
//
// Each case discovers two real, distinct filter values from live seed data via a
// known response field path, so the test is self-adjusting: where seed data is
// too thin to surface two values it skips with a logged reason (making the
// coverage gap visible) and automatically becomes real coverage once the seed
// data grows. The response field is used ONLY to discover values; the invariant
// itself is checked over row ids, so it works uniformly for scalar, nested and
// array-valued filters.

// arrayFilterCase declares one (endpoint, array-filter) pair to verify.
type arrayFilterCase struct {
	// name is the subtest name (endpoint + param).
	name string
	// path is the list endpoint path.
	path string
	// param is the array query parameter under test (e.g. "category_ids").
	param string
	// valuePath is a dotted field path on a list-response item that carries the
	// value the filter matches on. Supports nested objects ("category.id") and
	// arrays of objects ("attributes[].id", "lines[].product.item.id").
	valuePath string
	// include is the ?include= value required to populate valuePath ("" if none).
	include string
	// fromSelf is true when valuePath lives on the endpoint's own response, so a
	// discovered value MUST return rows when filtered (an empty result is a dead /
	// broken filter and fails the test). When false the values are sourced from a
	// sibling endpoint (see fromPath) and an empty result only means the seed data
	// has no linked rows, so the case skips instead of failing.
	fromSelf bool
	// fromPath / fromInclude source candidate values from a sibling list endpoint
	// when the filtered value is not reflected on the endpoint's own response.
	fromPath    string
	fromInclude string
}

func arrayFilterCases() []arrayFilterCase {
	return []arrayFilterCase{
		// --- catalog ---
		{name: "items/types", path: "/v1/catalog/items", param: "types", valuePath: "type", fromSelf: true},
		{name: "items/category_ids", path: "/v1/catalog/items", param: "category_ids", valuePath: "category.id", include: "category", fromSelf: true},
		{name: "items/attribute_ids", path: "/v1/catalog/items", param: "attribute_ids", valuePath: "attributes.data[].id", include: "attributes", fromSelf: true},
		{name: "items/product_line_ids", path: "/v1/catalog/items", param: "product_line_ids", valuePath: "product_line.id", include: "product_line", fromSelf: false, fromPath: "/v1/catalog/products", fromInclude: "product_line"},
		{name: "items/customer_ids", path: "/v1/catalog/items", param: "customer_ids", valuePath: "id", fromSelf: false, fromPath: "/v1/sales/customers"},

		{name: "materials/category_ids", path: "/v1/catalog/materials", param: "category_ids", valuePath: "item.category.id", include: "item,item.category", fromSelf: true},
		{name: "materials/attribute_ids", path: "/v1/catalog/materials", param: "attribute_ids", valuePath: "item.attributes.data[].id", include: "item,item.attributes", fromSelf: true},

		{name: "parts/category_ids", path: "/v1/catalog/parts", param: "category_ids", valuePath: "item.category.id", include: "item,item.category", fromSelf: true},
		{name: "parts/attribute_ids", path: "/v1/catalog/parts", param: "attribute_ids", valuePath: "item.attributes.data[].id", include: "item,item.attributes", fromSelf: true},

		{name: "products/product_line_ids", path: "/v1/catalog/products", param: "product_line_ids", valuePath: "product_line.id", include: "product_line", fromSelf: true},
		{name: "products/category_ids", path: "/v1/catalog/products", param: "category_ids", valuePath: "item.category.id", include: "item,item.category", fromSelf: true},
		{name: "products/attribute_ids", path: "/v1/catalog/products", param: "attribute_ids", valuePath: "item.attributes.data[].id", include: "item,item.attributes", fromSelf: true},
		{name: "products/customer_ids", path: "/v1/catalog/products", param: "customer_ids", valuePath: "id", fromSelf: false, fromPath: "/v1/sales/customers"},

		{name: "units/unit_group_ids", path: "/v1/catalog/units", param: "unit_group_ids", valuePath: "id", fromSelf: false, fromPath: "/v1/catalog/unit-groups"},

		// --- sales / finance ---
		{name: "customers/status_codes", path: "/v1/sales/customers", param: "status_codes", valuePath: "status", fromSelf: true},
		{name: "customers/commission_status_codes", path: "/v1/sales/customers", param: "commission_status_codes", valuePath: "commission_policy", fromSelf: true},
		{name: "customers/sales_rep_ids", path: "/v1/sales/customers", param: "sales_rep_ids", valuePath: "defaults.sales_rep.id", include: "defaults.sales_rep", fromSelf: true},
		{name: "customers/pricing_group_ids", path: "/v1/sales/customers", param: "pricing_group_ids", valuePath: "price_groups.data[].id", include: "price_groups", fromSelf: true},

		{name: "sales-orders/status_codes", path: "/v1/sales/sales-orders", param: "status_codes", valuePath: "status", fromSelf: true},
		{name: "sales-orders/customer_ids", path: "/v1/sales/sales-orders", param: "customer_ids", valuePath: "customer.id", include: "customer", fromSelf: true},
		{name: "sales-orders/sales_rep_ids", path: "/v1/sales/sales-orders", param: "sales_rep_ids", valuePath: "sales_rep.id", include: "sales_rep", fromSelf: true},
		{name: "sales-orders/item_ids", path: "/v1/sales/sales-orders", param: "item_ids", valuePath: "lines.data[].product.item.id", include: "lines.product.item", fromSelf: true},
		{name: "sales-orders/product_line_ids", path: "/v1/sales/sales-orders", param: "product_line_ids", valuePath: "lines.data[].product.product_line.id", include: "lines.product.product_line", fromSelf: true},

		{name: "invoices/customer_ids", path: "/v1/finance/invoices", param: "customer_ids", valuePath: "customer.id", include: "customer", fromSelf: true},
		{name: "invoices/item_ids", path: "/v1/finance/invoices", param: "item_ids", valuePath: "lines.data[].item.id", include: "lines", fromSelf: true},

		{name: "transactions/types", path: "/v1/finance/transactions", param: "types", valuePath: "transaction_type.code", fromSelf: true},
		{name: "transactions/adjustment_types", path: "/v1/finance/transactions", param: "adjustment_types", valuePath: "adjustment_type.code", fromSelf: true},
		{name: "transactions/methods", path: "/v1/finance/transactions", param: "methods", valuePath: "transaction_method.code", fromSelf: true},
		{name: "transactions/customer_ids", path: "/v1/finance/transactions", param: "customer_ids", valuePath: "customer.id", include: "customer", fromSelf: true},

		{name: "open-credits/customer_ids", path: "/v1/finance/open-credits", param: "customer_ids", valuePath: "customer.id", fromSelf: true},

		{name: "settlements/transaction_ids", path: "/v1/finance/settlements", param: "transaction_ids", valuePath: "id", fromSelf: false, fromPath: "/v1/finance/transactions"},
		{name: "settlements/invoice_ids", path: "/v1/finance/settlements", param: "invoice_ids", valuePath: "id", fromSelf: false, fromPath: "/v1/finance/invoices"},

		// --- operations ---
		{name: "deliveries/supplier_ids", path: "/v1/operations/deliveries", param: "supplier_ids", valuePath: "purchase_order.supplier.id", include: "purchase_order.supplier", fromSelf: true},
		{name: "deliveries/item_ids", path: "/v1/operations/deliveries", param: "item_ids", valuePath: "lines.data[].item.id", include: "lines", fromSelf: true},

		{name: "picks/customer_ids", path: "/v1/operations/picks", param: "customer_ids", valuePath: "customer.id", include: "customer", fromSelf: true},

		{name: "shipments/customer_ids", path: "/v1/operations/shipments", param: "customer_ids", valuePath: "customer.id", include: "customer", fromSelf: true},
		{name: "shipments/item_ids", path: "/v1/operations/shipments", param: "item_ids", valuePath: "lines.data[].item.id", include: "lines", fromSelf: true},

		// Suppliers link to items via supplier_material (materials), so source the
		// candidate item ids from the materials feed rather than /v1/catalog/items
		// (whose first rows are sock products that no supplier carries).
		{name: "suppliers/item_ids", path: "/v1/operations/suppliers", param: "item_ids", valuePath: "item.id", fromSelf: false, fromPath: "/v1/catalog/materials", fromInclude: "item"},

		{name: "receiving-orders/supplier_ids", path: "/v1/operations/receiving-orders", param: "supplier_ids", valuePath: "supplier.id", include: "supplier", fromSelf: true},
		{name: "receiving-orders/item_ids", path: "/v1/operations/receiving-orders", param: "item_ids", valuePath: "lines.data[].item.id", include: "lines", fromSelf: true},

		{name: "purchase-orders/status_codes", path: "/v1/operations/purchase-orders", param: "status_codes", valuePath: "status", fromSelf: true},
		{name: "purchase-orders/supplier_ids", path: "/v1/operations/purchase-orders", param: "supplier_ids", valuePath: "supplier.id", include: "supplier", fromSelf: true},
		{name: "purchase-orders/item_ids", path: "/v1/operations/purchase-orders", param: "item_ids", valuePath: "lines.data[].item.id", include: "lines", fromSelf: true},

		{name: "inventory-change-logs/action_type_codes", path: "/v1/operations/inventory-change-logs", param: "action_type_codes", valuePath: "action_type", fromSelf: true},
		{name: "inventory-change-logs/item_ids", path: "/v1/operations/inventory-change-logs", param: "item_ids", valuePath: "item.id", include: "item", fromSelf: true},
		{name: "inventory-change-logs/changed_by_user_ids", path: "/v1/operations/inventory-change-logs", param: "changed_by_user_ids", valuePath: "responsible_user.id", include: "responsible_user", fromSelf: true},

		{name: "production-runs/item_ids", path: "/v1/operations/production-runs", param: "item_ids", valuePath: "id", fromSelf: false, fromPath: "/v1/catalog/items"},

		{name: "production-steps/item_ids", path: "/v1/operations/production-steps", param: "item_ids", valuePath: "production.produced_item.id", include: "production.produced_item", fromSelf: true},
		{name: "production-steps/machine_ids", path: "/v1/operations/production-steps", param: "machine_ids", valuePath: "machines.data[].id", include: "machines", fromSelf: true},
		{name: "production-steps/scanning_station_ids", path: "/v1/operations/production-steps", param: "scanning_station_ids", valuePath: "scanning_station.id", include: "scanning_station", fromSelf: true},
		{name: "production-steps/input_step_ids", path: "/v1/operations/production-steps", param: "input_step_ids", valuePath: "in_steps.data[].id", include: "in_steps", fromSelf: true},
		{name: "production-steps/output_step_ids", path: "/v1/operations/production-steps", param: "output_step_ids", valuePath: "out_steps.data[].id", include: "out_steps", fromSelf: true},

		// --- identity / ai ---
		{name: "agents/statuses", path: "/v1/ai/agents", param: "statuses", valuePath: "status", fromSelf: true},
		{name: "agents/definition_types", path: "/v1/ai/agents", param: "definition_types", valuePath: "definition_type", fromSelf: true},
		{name: "agents/trigger_types", path: "/v1/ai/agents", param: "trigger_types", valuePath: "trigger_type", fromSelf: true},

		{name: "roles/types", path: "/v1/identity/roles", param: "types", valuePath: "type", fromSelf: true},
	}
}

func TestArrayFilters_UnionExclusion(t *testing.T) {
	for _, c := range arrayFilterCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			runArrayFilterUnion(t, c)
		})
	}
}

func runArrayFilterUnion(t *testing.T, c arrayFilterCase) {
	t.Helper()

	// 1. Discover two distinct candidate filter values.
	discoverPath, discoverInclude, discoverField := c.path, c.include, c.valuePath
	if !c.fromSelf {
		discoverPath, discoverInclude, discoverField = c.fromPath, c.fromInclude, c.valuePath
	}
	values := discoverFieldValues(t, discoverPath, discoverInclude, discoverField, 2)
	require.GreaterOrEqualf(t, len(values), 2,
		"%s %s: fewer than 2 distinct values for %q available in seed data — every array filter must have seed coverage (no skips)",
		c.path, c.param, c.valuePath)
	a, b := values[0], values[1]

	// 2. Single-value result sets.
	s1 := filteredIDSet(t, c.path, c.param, a)
	s2 := filteredIDSet(t, c.path, c.param, b)
	// Filtering by a value that was sourced from real data must return rows — for
	// fromSelf cases the value is a live row of this endpoint, and for sibling-fed
	// cases the seed links the sourced values. Empty means a dead/broken filter or
	// a seed gap; either way it must fail, never skip.
	require.NotEmptyf(t, s1, "%s: filtering by a sourced value (%q) produced no results — filter %q broken or missing seed coverage", c.path, a, c.param)
	require.NotEmptyf(t, s2, "%s: filtering by a sourced value (%q) produced no results — filter %q broken or missing seed coverage", c.path, b, c.param)

	// 3. Combined result set and the union/exclusion checks, with one retry to
	//    absorb a row created/deleted by another parallel test between calls.
	for attempt := 0; attempt < 2; attempt++ {
		s12 := filteredIDSet(t, c.path, c.param, a, b)

		missing := idsMissing(s1, s12)
		missing = append(missing, idsMissing(s2, s12)...)
		foreign := foreignIDs(s12, s1, s2)

		if len(missing) == 0 && len(foreign) == 0 {
			return // invariant holds
		}
		if attempt == 0 {
			// Re-snapshot the single-value sets too before the final assert.
			s1 = filteredIDSet(t, c.path, c.param, a)
			s2 = filteredIDSet(t, c.path, c.param, b)
			continue
		}

		assert.Empty(t, missing,
			"%s filter %q union FAILED: rows matching %q or %q individually are missing from the combined [%q, %q] result (filter does not union its values)",
			c.path, c.param, a, b, a, b)
		assert.Empty(t, foreign,
			"%s filter %q exclusion FAILED: combined [%q, %q] result contains rows matching neither value (filter is ignored or over-broad): %v",
			c.path, c.param, a, b, foreign)
	}
}

// discoverFieldValues fetches a page from path and returns up to n distinct
// string values found at the dotted valuePath across the returned items.
func discoverFieldValues(t *testing.T, path, include, valuePath string, n int) []string {
	t.Helper()
	params := url.Values{"limit": {"50"}}
	if include != "" {
		params.Set("include", include)
	}
	list, _, err := apiClient.GetList(path, params)
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, item := range list.Data {
		var row map[string]any
		if json.Unmarshal(item, &row) != nil {
			continue
		}
		for _, v := range valuesAtPath(row, strings.Split(valuePath, ".")) {
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
			if len(out) >= n {
				return out
			}
		}
	}
	return out
}

// filteredIDSet returns the set of row ids from path when filtered by param=values.
func filteredIDSet(t *testing.T, path, param string, values ...string) map[string]struct{} {
	t.Helper()
	params := url.Values{"limit": {"1000"}}
	params[param] = values
	list, _, err := apiClient.GetList(path, params)
	require.NoError(t, err, "%s filter %s=%v request failed", path, param, values)
	out := make(map[string]struct{}, len(list.Data))
	for _, item := range list.Data {
		if id := DataItemField(item, "id"); id != "" {
			out[id] = struct{}{}
		}
	}
	return out
}

// valuesAtPath walks a decoded JSON object along tokens and returns all string
// leaf values reached. A token ending in "[]" expects an array of objects and
// fans out across its elements.
func valuesAtPath(node any, tokens []string) []string {
	if len(tokens) == 0 {
		switch v := node.(type) {
		case string:
			return []string{v}
		case float64:
			// ids are strings in this API; numbers are not used as filter values.
			return nil
		default:
			_ = v
			return nil
		}
	}
	tok := tokens[0]
	isArray := strings.HasSuffix(tok, "[]")
	key := strings.TrimSuffix(tok, "[]")

	obj, ok := node.(map[string]any)
	if !ok {
		return nil
	}
	child, ok := obj[key]
	if !ok || child == nil {
		return nil
	}
	if isArray {
		arr, ok := child.([]any)
		if !ok {
			return nil
		}
		var out []string
		for _, el := range arr {
			out = append(out, valuesAtPath(el, tokens[1:])...)
		}
		return out
	}
	return valuesAtPath(child, tokens[1:])
}

// idsMissing returns ids present in want but absent from got.
func idsMissing(want, got map[string]struct{}) []string {
	var out []string
	for id := range want {
		if _, ok := got[id]; !ok {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// foreignIDs returns ids in got that are in neither a nor b.
func foreignIDs(got, a, b map[string]struct{}) []string {
	var out []string
	for id := range got {
		_, inA := a[id]
		_, inB := b[id]
		if !inA && !inB {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
