//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIncludes_HydratedToOneMatchesCanonical is the safety net for the class of bug where a to-one
// include is populated but with the WRONG or INCOMPLETE data: a stub synthesized from the parent's
// join that drops the related resource's own base fields, or the wrong record stitched onto a row.
//
// TestIncludes_PopulateNestedResources only proves an include is present and non-empty — a stub that
// carries id+object but nulls out every other base field sails through it. This test closes that gap:
// for every declared include that resolves to a single id-bearing resource which also has its own
// canonical GET-by-id endpoint, it fetches that resource directly and asserts the hydrated include
// carries the same value for every non-null base (scalar) field. `?include=x` is a promise to return
// the full x, not a reference to it.
//
// It is data-driven off the OpenAPI spec, so every current and future include is covered
// automatically. Includes with no canonical single-id retrieve (polymorphic Entity/Actor references,
// value objects like Quantity or Owner, list-valued includes) have nothing to cross-check and get no
// subtest; their presence is already guarded by TestIncludes_PopulateNestedResources.
func TestIncludes_HydratedToOneMatchesCanonical(t *testing.T) {
	t.Parallel()

	spec, err := LoadFullSpec()
	require.NoError(t, err, "load OpenAPI spec")
	retrieveByType := buildRetrieveEndpointsByObjectType(spec)
	require.NotEmpty(t, retrieveByType, "spec should expose at least one single-id retrieve endpoint")

	endpoints, err := loadIncludeGetEndpoints()
	require.NoError(t, err, "load include-supporting GET endpoints")
	require.NotEmpty(t, endpoints, "spec contained no GET endpoints with include[] enums")

	var crossChecked atomic.Int64
	t.Cleanup(func() {
		assert.Positive(t, crossChecked.Load(), "no include resolved to a resource with a canonical retrieve")
	})

	for _, ep := range endpoints {
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()

			for _, include := range ep.IncludeEnum {
				path, query, ok := resolveGetScenario(ep, include)
				require.True(t, ok, "resolveGetScenario(%s) operationId=%s include=%s", ep.Path, ep.OperationID, include)

				status, body, err := apiClient.GetListRaw(path, withIncludeQuery(query, include))
				require.NoError(t, err, "GET %s?include=%s failed", path, include)
				require.Equalf(t, 200, status, "GET %s?include=%s: %s", path, include, string(body))

				got := parseJSON(body)
				require.NotNil(t, got, "GET %s?include=%s should be valid JSON", path, include)

				// A relation only some rows carry can be null on every row of the newest page once earlier
				// runs' rows pile up ahead of the seeded ones, so read on while it is null everywhere.
				// A value object is present rather than null, so it stops at the first page.
				leaves := collectIncludeLeafResources(got, include)
				for page := 0; len(leaves) == 0 && !includeHasAnyValue(got, include) && page < maxListScanPages; page++ {
					next := jsonField(jsonObject(got, "page_info"), "next_page_url")
					if next == "" {
						break
					}
					nextPath, nextQuery, ok := ListURLPathQuery(&next)
					require.True(t, ok, "next_page_url %q", next)
					status, body, err = apiClient.GetListRaw(nextPath, nextQuery)
					require.NoError(t, err, "GET %s failed", next)
					require.Equalf(t, 200, status, "GET %s: %s", next, string(body))
					path, query, got = nextPath, nextQuery, parseJSON(body)
					leaves = collectIncludeLeafResources(got, include)
				}
				if len(leaves) == 0 {
					continue // a value object, list, or primitive: no canonical resource to compare with
				}
				if _, ok := retrieveByType[jsonField(leaves[0], "object")]; !ok {
					continue // polymorphic or nested-only type: no canonical retrieve to compare with
				}

				t.Run(include, func(t *testing.T) {
					t.Parallel()
					crossChecked.Add(1)

					owners := func() []string {
						// The owning account is often itself an expandable (a sales order's customer), so
						// look again with every top-level include expanded.
						status, body, err := apiClient.GetListRaw(path, withIncludesQuery(query, topLevelIncludes(ep.IncludeEnum)))
						require.NoError(t, err)
						require.Less(t, status, 500, "GET %s with every include must not 5xx: %s", path, string(body))
						return append(collectAccountIDs(got), collectAccountIDs(parseJSON(body))...)
					}
					// A parallel test may delete a listed row between the two reads. A leaf the including
					// endpoint no longer hands out is that race; one it still hands out must be retrievable.
					stillIncluded := func(id string) bool {
						status, body, err := apiClient.GetListRaw(path, withIncludeQuery(query, include))
						require.NoError(t, err)
						requireStatus(t, 200, status, body)
						for _, leaf := range collectIncludeLeafResources(parseJSON(body), include) {
							if jsonField(leaf, "id") == id {
								return true
							}
						}
						return false
					}

					for _, leaf := range leaves {
						rt, ok := retrieveByType[jsonField(leaf, "object")]
						require.True(t, ok, "include %q mixes object types: %v", include, leaf)
						id := jsonField(leaf, "id")
						canonPath, status, canon := retrieveCanonical(t, rt, id, owners)
						if status == 200 {
							assertHydratedMatchesCanonical(t, include, leaf, canon)
							return
						}
						require.Falsef(t, stillIncluded(id),
							"the include hands out %s, which no account in the response can retrieve (status %d)", canonPath, status)
					}
					t.Fatalf("every %q the include handed out was deleted before it could be cross-checked", include)
				})
			}
		})
	}
}

// retrieveCanonical fetches the resource through its own retrieve endpoint. A counterparty's
// resource, such as a customer's address, is only retrievable while targeting that account, so the
// accounts named in the including response are tried when the caller's own scope does not have it.
func retrieveCanonical(t *testing.T, rt retrieveEndpoint, id string, owners func() []string) (string, int, map[string]any) {
	t.Helper()

	canonPath := strings.ReplaceAll(rt.path, "{"+rt.param+"}", id)
	status, body, err := apiClient.GetListRaw(canonPath, nil)
	require.NoError(t, err, "canonical GET %s failed", canonPath)
	require.Less(t, status, 500, "canonical GET %s must not 5xx: %s", canonPath, string(body))

	if status != 200 {
		for _, accountID := range owners() {
			status, body, err = apiClient.WithAccountID(accountID).GetListRaw(canonPath, nil)
			require.NoError(t, err, "canonical GET %s as %s failed", canonPath, accountID)
			require.Less(t, status, 500, "canonical GET %s as %s must not 5xx: %s", canonPath, accountID, string(body))
			if status == 200 {
				break
			}
		}
	}
	if status != 200 {
		return canonPath, status, nil
	}
	canon := parseJSON(body)
	require.NotNil(t, canon, "canonical response should be valid JSON")
	return canonPath, status, canon
}

func topLevelIncludes(includes []string) []string {
	var out []string
	for _, include := range includes {
		if !strings.Contains(include, ".") {
			out = append(out, include)
		}
	}
	return out
}

func withIncludesQuery(base url.Values, includes []string) url.Values {
	out := cloneQuery(base)
	if out == nil {
		out = url.Values{}
	}
	out["include"] = includes
	return out
}

// collectAccountIDs returns every distinct account id anywhere in a response, in the order found.
func collectAccountIDs(v any) []string {
	var out []string
	seen := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		case string:
			if strings.HasPrefix(v, "ac_") && !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	walk(v)
	return out
}

// includeHydrationSkipFields are base scalar fields excluded from the cross-check. Timestamps are
// skipped because a parallel test mutating the same row between the include read and the canonical
// read would drift updated_at, and per-loader timestamp formatting is not the field-drop bug this
// test hunts.
var includeHydrationSkipFields = map[string]bool{
	"created_at": true,
	"updated_at": true,
}

// includeHydrationKnownGaps records, per object type, fields a hydrated include is allowed to differ
// from the canonical resource on. This is a ledger of gaps this guard surfaced, kept green so it can
// guard against NEW regressions. The reference-include stubs it originally captured (carrier.code,
// shipment.priority, sales_order lifecycle timestamps, department.name, customer.relationship_type,
// scanning_station label fields) have been burned down by migrating each include to its resource's full
// loader (see registered_inventory_change_log.go). What remains is deliberate:
//
//   - computed aggregates: counts/rollups the full retrieve computes but a lightweight reference
//     include legitimately leaves at its zero value — a real design choice, not a bug.
//   - value-object stubs: the freight unit is embedded inline in a Money/Rate value object with
//     normalized ratios rather than loaded through the unit loader, so it diverges from the canonical
//     unit. Not a field-drop; a distinct serialization of the same value.
//
// Keyed by object type: these are properties of a resource's reference shape, shared across every
// endpoint that includes it. An empty map (all entries removed) is the goal.
var includeHydrationKnownGaps = map[string]map[string]string{
	"order_discount": {"order_count": "computed aggregate; reference include leaves it 0"},
	"shipment":       {"case_count": "computed aggregate; reference include leaves it 0"},
	"sales_order":    {"line_count": "computed aggregate; reference include leaves it 0"},
	"unit": {
		"ratio_numerator":    "value-object stub: freight unit is embedded with normalized ratios",
		"ratio_denominator":  "value-object stub: freight unit is embedded with normalized ratios",
		"offset_numerator":   "value-object stub: freight unit is embedded with normalized ratios",
		"offset_denominator": "value-object stub: freight unit is embedded with normalized ratios",
		"is_base_unit":       "value-object stub: freight unit is embedded with normalized ratios",
	},
}

func isKnownHydrationGap(objType, field string) bool {
	_, ok := includeHydrationKnownGaps[objType][field]
	return ok
}

// assertHydratedMatchesCanonical requires the hydrated include to carry the same value as the
// canonical resource for every non-null scalar field. Object- and array-valued fields (the related
// resource's own expandables and nested resources) are skipped: those are null on both sides unless
// separately requested, so comparing them would prove nothing.
func assertHydratedMatchesCanonical(t *testing.T, include string, leaf, canon map[string]any) {
	t.Helper()

	objType := jsonField(canon, "object")
	for key, cval := range canon {
		if cval == nil || !isScalarJSON(cval) {
			continue
		}
		if includeHydrationSkipFields[key] || isKnownHydrationGap(objType, key) {
			continue
		}
		lval, present := leaf[key]
		require.Truef(t, present,
			"include %q: hydrated %s dropped field %q that the canonical resource populates (%v) — the include returned a partial stub",
			include, objType, key, cval)
		assert.Equalf(t, cval, lval,
			"include %q: hydrated %s.%s is %v but the canonical resource reports %v — the include returned incomplete or mismatched data",
			include, objType, key, lval, cval)
	}
}

func isScalarJSON(v any) bool {
	switch v.(type) {
	case nil, bool, float64, string:
		return true
	default:
		return false
	}
}

// retrieveEndpoint is a GET-by-id route: a path with a single {param} slot at its tail that returns
// one resource of a known object type.
type retrieveEndpoint struct {
	path  string
	param string
}

// buildRetrieveEndpointsByObjectType indexes the spec's single-id retrieve endpoints by the object
// type they return, so a hydrated include can be matched back to the canonical resource. When a type
// is retrievable by more than one such path, the shortest wins (the top-level resource route rather
// than a nested one).
func buildRetrieveEndpointsByObjectType(spec *openAPISpec) map[string]retrieveEndpoint {
	out := map[string]retrieveEndpoint{}
	for path, methods := range spec.Paths {
		if isExcludedPath(path) {
			continue
		}
		op, ok := methods["get"]
		if !ok {
			continue
		}

		var pathParams []string
		for _, p := range op.Parameters {
			if p.In == "path" {
				pathParams = append(pathParams, p.Name)
			}
		}
		if len(pathParams) != 1 {
			continue
		}
		param := pathParams[0]
		if !strings.HasSuffix(path, "/{"+param+"}") {
			continue
		}

		schema, ok := spec.GetResponseSchema(path, "get", "200")
		if !ok {
			continue
		}
		objType := singleObjectEnum(spec, schema)
		if objType == "" || objType == "list" {
			continue
		}
		if existing, ok := out[objType]; !ok || len(path) < len(existing.path) {
			out[objType] = retrieveEndpoint{path: path, param: param}
		}
	}
	return out
}

// singleObjectEnum returns the sole enum value of a resource schema's "object" discriminator, or ""
// when the schema is not a single concrete resource (a list, or a polymorphic union whose object
// enum has more than one value).
func singleObjectEnum(spec *openAPISpec, schema *openAPISchema) string {
	prop, ok := findSchemaProperty(spec, schema, "object", 0)
	if !ok {
		return ""
	}
	if prop.Ref != "" {
		if resolved, ok := spec.ResolveSchemaRef(prop.Ref); ok {
			prop = resolved
		}
	}
	if len(prop.Enum) != 1 {
		return ""
	}
	s, _ := prop.Enum[0].(string)
	return s
}

// findSchemaProperty resolves a named property through refs and allOf/oneOf/anyOf composition.
func findSchemaProperty(spec *openAPISpec, schema *openAPISchema, name string, depth int) (*openAPISchema, bool) {
	if schema == nil || depth > 10 {
		return nil, false
	}
	if schema.Ref != "" {
		if resolved, ok := spec.ResolveSchemaRef(schema.Ref); ok {
			return findSchemaProperty(spec, resolved, name, depth+1)
		}
		return nil, false
	}
	if p, ok := schema.Properties[name]; ok {
		return &p, true
	}
	for _, group := range [][]openAPISchema{schema.AllOf, schema.OneOf, schema.AnyOf} {
		for i := range group {
			if p, ok := findSchemaProperty(spec, &group[i], name, depth+1); ok {
				return p, true
			}
		}
	}
	return nil, false
}

// includeHasAnyValue reports whether any row carries a non-null value at the include path.
func includeHasAnyValue(resp map[string]any, include string) bool {
	parts := strings.Split(include, ".")
	var walk func(cur any, i int) bool
	walk = func(cur any, i int) bool {
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		if obj, _ := m["object"].(string); obj == "list" {
			data, _ := m["data"].([]any)
			for _, item := range data {
				if walk(item, i) {
					return true
				}
			}
			return false
		}
		v, present := m[parts[i]]
		if !present || v == nil {
			return false
		}
		if i == len(parts)-1 {
			return true
		}
		return walk(v, i+1)
	}
	return walk(resp, 0)
}

// collectIncludeLeafResources navigates the dot-separated include path and returns every terminal
// value that is an id-bearing resource object (has non-empty "id" and "object"). List envelopes along
// the way are fanned out so rows with heterogeneous relationships are all reached.
func collectIncludeLeafResources(resp map[string]any, include string) []map[string]any {
	parts := strings.Split(include, ".")
	var out []map[string]any
	if obj, _ := resp["object"].(string); obj == "list" {
		for _, raw := range jsonArray(resp, "data") {
			if row, ok := raw.(map[string]any); ok {
				walkIncludeLeaves(row, parts, 0, &out)
			}
		}
		return out
	}
	walkIncludeLeaves(resp, parts, 0, &out)
	return out
}

func walkIncludeLeaves(cur any, parts []string, i int, out *[]map[string]any) {
	if i >= len(parts) {
		if m, ok := cur.(map[string]any); ok {
			if jsonField(m, "object") != "" && jsonField(m, "id") != "" {
				*out = append(*out, m)
			}
		}
		return
	}
	curMap, ok := cur.(map[string]any)
	if !ok {
		return
	}
	if objType, _ := curMap["object"].(string); objType == "list" {
		if data, ok := curMap["data"].([]any); ok {
			for _, raw := range data {
				if item, ok := raw.(map[string]any); ok {
					walkIncludeLeaves(item, parts, i, out)
				}
			}
		}
		return
	}
	v, present := curMap[parts[i]]
	if !present || v == nil {
		return
	}
	walkIncludeLeaves(v, parts, i+1, out)
}
