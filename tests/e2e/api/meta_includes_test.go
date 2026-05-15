//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIncludes_PopulateNestedResources enumerates every GET endpoint that
// declares an `include[]` enum in the OpenAPI spec and verifies each declared
// include value actually materializes a populated nested resource when
// requested. This is the safety net that catches bugs where an include is
// advertised but the backend never fetches / attaches the nested data.
//
// The test is data-driven off the OpenAPI spec, so new includes are
// automatically covered as soon as they're added to an endpoint's
// IncludeConfig.
//
// For an (operationID, include) pair that cannot yet satisfy the populated-
// data assertion (seed data truly has no nested resource, or nested data is
// not expected on the seeded parent), add an entry to includesOptOut with the
// reason. Every opt-out is a TODO to backfill seed data.
func TestIncludes_PopulateNestedResources(t *testing.T) {
	t.Parallel()

	endpoints, err := loadIncludeGetEndpoints()
	require.NoError(t, err, "load include-supporting GET endpoints")
	require.NotEmpty(t, endpoints, "spec contained no GET endpoints with include[] enums")

	for _, ep := range endpoints {
		ep := ep
		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()

			path, ok := ep.ResolvePath()
			if !ok {
				t.Skipf("Cannot resolve path params for %s", ep.Path)
				return
			}

			for _, include := range ep.IncludeEnum {
				include := include
				t.Run(include, func(t *testing.T) {
					t.Parallel()

					if reason, skip := includesOptOut[optOutKey(ep.OperationID, include)]; skip {
						t.Skipf("opt-out: %s", reason)
						return
					}

					status, body, err := apiClient.GetListRaw(path, url.Values{"include": {include}})
					require.NoError(t, err, "GET %s?include=%s failed", path, include)
					requireStatus(t, 200, status, body)

					got := parseJSON(body)
					require.NotNil(t, got, "response should be valid JSON")

					assertIncludePopulated(t, got, include)
				})
			}
		})
	}
}

// optOutKey formats an opt-out map key.
func optOutKey(operationID, include string) string {
	return operationID + "::" + include
}

// includesOptOut lists (operationID, include) pairs whose populated-data
// assertion is knowingly skipped. Each entry is a TODO to either backfill
// seed data or fix a backend bug. The goal is an empty map — every opt-out
// is tech debt. Reasons are tagged:
//
//	seed-gap: nested resource isn't in the seed data; add it.
//	bug:      backend doesn't attach the nested resource when requested.
//	schema:   the relationship isn't representable in the current schema.
var includesOptOut = map[string]string{}

// assertIncludePopulated navigates the response to the JSON path described by
// the include key and asserts that the value is present and non-empty.
// List responses (object == "list") are handled by walking each item in data[]
// and requiring that at least one item has the include populated with a valid
// stub (id + object non-empty). Single-object responses use the same "at least
// one row" rule for each list envelope along the path (e.g. production flow
// steps) so heterogeneous nested rows are covered.
func assertIncludePopulated(t *testing.T, resp map[string]any, include string) {
	t.Helper()

	if obj, _ := resp["object"].(string); obj == "list" {
		assertIncludePopulatedOnList(t, resp, include)
		return
	}
	assertIncludePopulatedOnObject(t, resp, include, "")
}

func assertIncludePopulatedOnList(t *testing.T, resp map[string]any, include string) {
	t.Helper()

	rawItems, ok := resp["data"].([]any)
	require.True(t, ok, "list response must have a data array")
	require.NotEmpty(t, rawItems, "list response data should contain at least one item (seeded)")

	// At least one item in the list must have the include populated. We don't
	// require every item to be populated because heterogeneous resources can
	// legitimately have the nested relationship only on some rows.
	var populatedCount int
	var lastErr string
	for i, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if err := checkIncludePopulated(item, include); err != nil {
			lastErr = fmt.Sprintf("data[%d]: %s", i, err)
			continue
		}
		populatedCount++
	}
	require.Positive(t, populatedCount,
		"expected at least one list item to have %q populated; last error: %s", include, lastErr)
}

func assertIncludePopulatedOnObject(t *testing.T, obj map[string]any, include, pathLabel string) {
	t.Helper()
	if err := checkIncludePopulated(obj, include); err != nil {
		t.Errorf("%s: %s", pathLabel, err)
	}
}

// checkIncludePopulated navigates the dot-separated include path through the
// given JSON object and returns an error describing the first place it fails
// the populated-stub contract.
func checkIncludePopulated(obj map[string]any, include string) error {
	parts := strings.Split(include, ".")
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return nil
	}
	return includePath(obj, parts, 0, include)
}

// includePath walks `parts` starting at index `i`. When the current value is a
// list envelope, each element of data[] is tried so at least one row can
// complete the remaining path (mirrors assertIncludePopulatedOnList semantics).
func includePath(cur any, parts []string, i int, fullInclude string) error {
	if i >= len(parts) {
		return validateIncludeValue(fullInclude, cur)
	}
	part := parts[i]
	curMap, ok := cur.(map[string]any)
	if !ok {
		return fmt.Errorf("navigating %q: parent at segment %d is not an object", fullInclude, i)
	}
	if objType, _ := curMap["object"].(string); objType == "list" {
		data, _ := curMap["data"].([]any)
		if len(data) == 0 {
			return fmt.Errorf("include %q: list at segment %d has no items to navigate through", fullInclude, i)
		}
		var errs []string
		for idx, raw := range data {
			item, ok := raw.(map[string]any)
			if !ok {
				errs = append(errs, fmt.Sprintf("data[%d]: not an object", idx))
				continue
			}
			v, present := item[part]
			if !present {
				errs = append(errs, fmt.Sprintf("data[%d]: missing key %q", idx, part))
				continue
			}
			if v == nil {
				errs = append(errs, fmt.Sprintf("data[%d]: key %q is null", idx, part))
				continue
			}
			if err := includePath(v, parts, i+1, fullInclude); err != nil {
				errs = append(errs, fmt.Sprintf("data[%d]: %v", idx, err))
				continue
			}
			return nil
		}
		if len(errs) > 0 {
			return fmt.Errorf("include %q: no list item completed path at segment %d (%s)", fullInclude, i, strings.Join(errs, "; "))
		}
		return fmt.Errorf("include %q: list at segment %d had no traversable items", fullInclude, i)
	}
	v, present := curMap[part]
	if !present {
		return fmt.Errorf("include %q: key %q missing from response", fullInclude, part)
	}
	if v == nil {
		return fmt.Errorf("include %q: %q is null (backend did not attach data)", fullInclude, part)
	}
	return includePath(v, parts, i+1, fullInclude)
}

// validateIncludeValue enforces that the backend populated the include with
// real data. The assertion is forgiving of legitimate shape variations:
//
//   - Primitive arrays (e.g. role.permissions as []string): non-empty required.
//   - List envelopes ({object:"list", data:[...]}): data must be non-empty.
//     Individual items are not stub-checked because audit-event changes,
//     permissions, and other "value object" includes have no id/object fields.
//   - Regular resource objects: the "object" field must be non-empty. "id" is
//     only enforced when the response actually declares an id (value objects
//     like Owner, FreightPreferences, Defaults carry no id field).
func validateIncludeValue(include string, v any) error {
	switch vv := v.(type) {
	case []any:
		if len(vv) == 0 {
			return fmt.Errorf("include %q: array is empty (seed should populate at least one entry)", include)
		}
		return nil
	case map[string]any:
		if objType, _ := vv["object"].(string); objType == "list" {
			rawItems, ok := vv["data"].([]any)
			if !ok {
				return fmt.Errorf("include %q: list.data is not an array", include)
			}
			if len(rawItems) == 0 {
				return fmt.Errorf("include %q: list.data is empty (seed should populate at least one entry)", include)
			}
			return nil
		}
		// Raw JSON blobs (e.g. audit_event.metadata) have no "object" field —
		// they're not expandable resources, just attached JSON. Require only
		// that the object isn't empty.
		if _, hasObject := vv["object"]; !hasObject {
			if len(vv) == 0 {
				return fmt.Errorf("include %q: object is empty", include)
			}
			return nil
		}
		if obj, _ := vv["object"].(string); obj == "" {
			return fmt.Errorf("include %q: object field missing or empty", include)
		}
		if idRaw, hasID := vv["id"]; hasID {
			if id, _ := idRaw.(string); id == "" {
				return fmt.Errorf("include %q: id is empty", include)
			}
		}
		return nil
	default:
		return fmt.Errorf("include %q: terminal value is %T, expected array or object", include, v)
	}
}

// IncludeGetEndpoint is a GET endpoint whose OpenAPI parameters declare a
// non-empty include[] enum.
type IncludeGetEndpoint struct {
	Path         string
	OperationID  string
	PathParams   []string
	IncludeEnum  []string
	ExampleRoute string
}

// HasParam reports whether the endpoint accepts a given query parameter.
// Implemented to make IncludeGetEndpoint compatible with ResolvePath via a
// local adapter.
func (e *IncludeGetEndpoint) ResolvePath() (string, bool) {
	adapter := ListEndpointSpec{Path: e.Path, OperationID: e.OperationID, PathParams: e.PathParams}
	return adapter.ResolvePath()
}

// excludedIncludePaths lists path prefixes skipped by the includes coverage
// test because the standard e2e API key can't exercise them (internal-admin-
// only endpoints, etc.). Mirrors the pattern of excludedPaginationPaths.
var excludedIncludePaths = []string{
	"/v1/core/request-logs", // requires internal admin role
}

func isExcludedFromIncludes(path string) bool {
	for _, prefix := range excludedIncludePaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// loadIncludeGetEndpoints parses the full OpenAPI spec and returns every GET
// endpoint whose include[] query parameter declares an enum. Endpoints matched
// by excludedPaths are dropped because they can't be exercised by e2e tests.
func loadIncludeGetEndpoints() ([]IncludeGetEndpoint, error) {
	spec, err := LoadFullSpec()
	if err != nil {
		return nil, err
	}

	var out []IncludeGetEndpoint
	for path, methods := range spec.Paths {
		if isExcludedPath(path) || isExcludedFromIncludes(path) {
			continue
		}
		op, ok := methods["get"]
		if !ok {
			continue
		}

		var includeEnum []string
		var pathParams []string
		for _, p := range op.Parameters {
			if p.In == "path" {
				pathParams = append(pathParams, p.Name)
				continue
			}
			if p.In != "query" || p.Name != "include[]" || p.Schema == nil {
				continue
			}
			if p.Schema.Items != nil && len(p.Schema.Items.Enum) > 0 {
				for _, v := range p.Schema.Items.Enum {
					if s, ok := v.(string); ok {
						includeEnum = append(includeEnum, s)
					}
				}
			}
		}
		if len(includeEnum) == 0 {
			continue
		}
		out = append(out, IncludeGetEndpoint{
			Path:        path,
			OperationID: op.OperationID,
			PathParams:  pathParams,
			IncludeEnum: includeEnum,
		})
	}
	return out, nil
}
