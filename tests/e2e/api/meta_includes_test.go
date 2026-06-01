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

			path, query, ok := resolveGetScenario(ep)
			require.True(t, ok, "resolveGetScenario(%s) operationId=%s", ep.Path, ep.OperationID)

			for _, include := range ep.IncludeEnum {
				include := include
				t.Run(include, func(t *testing.T) {
					t.Parallel()

					if reason, skip := includesOptOut[optOutKey(ep.OperationID, include)]; skip {
						t.Skipf("opt-out: %s", reason)
						return
					}

					status, body, err := apiClient.GetListRaw(path, withIncludeQuery(query, include))
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
var includesOptOut = map[string]string{
	"retrieve-settlement::responsible_user": "bug: settlement stores user ID not account_user ID; loader cannot resolve",
}

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

func assertIncludeCollapsedWithoutRequest(t *testing.T, resp map[string]any, include string) {
	t.Helper()

	if obj, _ := resp["object"].(string); obj == "list" {
		assertIncludeCollapsedOnList(t, resp, include)
		return
	}
	if err := checkIncludeCollapsed(resp, include); err != nil {
		t.Errorf("%s", err)
	}
}

func assertIncludeCollapsedOnList(t *testing.T, resp map[string]any, include string) {
	t.Helper()

	rawItems, ok := resp["data"].([]any)
	require.True(t, ok, "list response must have a data array")
	require.NotEmpty(t, rawItems, "list response data should contain at least one item (seeded)")

	for i, raw := range rawItems {
		item, ok := raw.(map[string]any)
		require.Truef(t, ok, "data[%d] should be an object", i)
		if err := checkIncludeCollapsed(item, include); err != nil {
			t.Errorf("data[%d]: %s", i, err)
		}
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

func checkIncludeCollapsed(obj map[string]any, include string) error {
	parts := strings.Split(include, ".")
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return nil
	}
	return collapsedIncludePath(obj, parts, 0, include)
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

func collapsedIncludePath(cur any, parts []string, i int, fullInclude string) error {
	curMap, ok := cur.(map[string]any)
	if !ok {
		return fmt.Errorf("navigating %q: parent at segment %d is not an object", fullInclude, i)
	}
	if objType, _ := curMap["object"].(string); objType == "list" {
		data, ok := curMap["data"].([]any)
		if !ok {
			return fmt.Errorf("include %q: list.data is not an array", fullInclude)
		}
		for idx, raw := range data {
			item, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("include %q: data[%d] is not an object", fullInclude, idx)
			}
			if err := collapsedIncludePath(item, parts, i, fullInclude); err != nil {
				return fmt.Errorf("include %q: data[%d]: %v", fullInclude, idx, err)
			}
		}
		return nil
	}

	part := parts[i]
	v, present := curMap[part]
	if !present {
		return fmt.Errorf("include %q: key %q missing from response", fullInclude, part)
	}

	if i == len(parts)-1 {
		if v != nil {
			return fmt.Errorf("include %q: %q should be null when not requested", fullInclude, part)
		}
		return nil
	}

	if v == nil {
		return nil
	}

	return collapsedIncludePath(v, parts, i+1, fullInclude)
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

type includeGetScenario struct {
	pathValues map[string]string
	query      url.Values
}

// HasParam reports whether the endpoint accepts a given query parameter.
// Implemented to make IncludeGetEndpoint compatible with ResolvePath via a
// local adapter.
func (e *IncludeGetEndpoint) ResolvePath() (string, bool) {
	adapter := ListEndpointSpec{Path: e.Path, OperationID: e.OperationID, PathParams: e.PathParams}
	return adapter.ResolvePath()
}

var includeGetScenarioByOperationID = map[string]includeGetScenario{}

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

func substituteIncludePath(path string, pathParams []string, vals map[string]string) (string, bool) {
	if len(pathParams) == 0 {
		return path, true
	}
	if len(vals) == 0 {
		return "", false
	}
	resolved := path
	for _, name := range pathParams {
		v, ok := vals[name]
		if !ok || v == "" {
			return "", false
		}
		resolved = strings.ReplaceAll(resolved, "{"+name+"}", v)
	}
	return resolved, true
}

func cloneQuery(v url.Values) url.Values {
	if len(v) == 0 {
		return nil
	}
	out := make(url.Values, len(v))
	for key, vals := range v {
		out[key] = append([]string(nil), vals...)
	}
	return out
}

func withIncludeQuery(base url.Values, include string) url.Values {
	out := cloneQuery(base)
	if out == nil {
		out = url.Values{}
	}
	out["include"] = []string{include}
	return out
}

func resolveGetScenario(ep IncludeGetEndpoint) (string, url.Values, bool) {
	scenario, ok := includeGetScenarioByOperationID[ep.OperationID]
	if !ok {
		path, resolved := ep.ResolvePath()
		return path, nil, resolved
	}
	if scenario.pathValues == nil {
		path, resolved := ep.ResolvePath()
		return path, cloneQuery(scenario.query), resolved
	}
	path, resolved := substituteIncludePath(ep.Path, ep.PathParams, scenario.pathValues)
	return path, cloneQuery(scenario.query), resolved
}

// IncludePutEndpoint is a PUT endpoint whose OpenAPI parameters declare include[]
// with a non-empty enum (subset of expandable fields on the JSON response root
// or a caller-provided extractor).
type IncludePutEndpoint struct {
	Path         string
	OperationID  string
	PathParams   []string
	IncludeEnum  []string
	ExampleRoute string
}

// ResolvePath mirrors IncludeGetEndpoint: substitute seeded path IDs.
func (e *IncludePutEndpoint) ResolvePath() (string, bool) {
	adapter := ListEndpointSpec{Path: e.Path, OperationID: e.OperationID, PathParams: e.PathParams}
	return adapter.ResolvePath()
}

// loadIncludePutEndpoints parses the OpenAPI spec and returns every PUT
// endpoint whose include[] query parameter declares an enum.
func loadIncludePutEndpoints() ([]IncludePutEndpoint, error) {
	spec, err := LoadFullSpec()
	if err != nil {
		return nil, err
	}

	var out []IncludePutEndpoint
	for path, methods := range spec.Paths {
		if isExcludedPath(path) || isExcludedFromIncludes(path) {
			continue
		}
		op, ok := methods["put"]
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
		out = append(out, IncludePutEndpoint{
			Path:        path,
			OperationID: op.OperationID,
			PathParams:  pathParams,
			IncludeEnum: includeEnum,
		})
	}
	return out, nil
}

func substituteIncludePutPath(ep IncludePutEndpoint, vals map[string]string) (string, bool) {
	return substituteIncludePath(ep.Path, ep.PathParams, vals)
}

func resolvePutScenarioPath(ep IncludePutEndpoint, pathValues map[string]string) (string, bool) {
	if pathValues == nil {
		return ep.ResolvePath()
	}
	return substituteIncludePutPath(ep, pathValues)
}

func extractRootObjectTyped(root map[string]any, wantObject string) []map[string]any {
	if jsonField(root, "object") == wantObject {
		return []map[string]any{root}
	}
	return nil
}

func splitOptOutKey(key string) (operationID, include string, ok bool) {
	parts := strings.SplitN(key, "::", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func isTaggedOptOutReason(reason string) bool {
	for _, prefix := range []string{"seed-gap:", "bug:", "schema:"} {
		if strings.HasPrefix(reason, prefix) {
			return true
		}
	}
	return false
}

func lookupIncludeGetEndpoint(endpoints []IncludeGetEndpoint, operationID string) (IncludeGetEndpoint, bool) {
	for _, ep := range endpoints {
		if ep.OperationID == operationID {
			return ep, true
		}
	}
	return IncludeGetEndpoint{}, false
}

func lookupIncludePutEndpoint(endpoints []IncludePutEndpoint, operationID string) (IncludePutEndpoint, bool) {
	for _, ep := range endpoints {
		if ep.OperationID == operationID {
			return ep, true
		}
	}
	return IncludePutEndpoint{}, false
}

func endpointDeclaresInclude(includes []string, include string) bool {
	for _, candidate := range includes {
		if candidate == include {
			return true
		}
	}
	return false
}

func executeIncludeOptOutProbe(t *testing.T, operationID, include string) bool {
	t.Helper()

	getEndpoints, err := loadIncludeGetEndpoints()
	require.NoError(t, err, "load include-supporting GET endpoints")

	if ep, ok := lookupIncludeGetEndpoint(getEndpoints, operationID); ok {
		path, query, resolved := resolveGetScenario(ep)
		require.True(t, resolved, "resolveGetScenario(%s) operationId=%s", ep.Path, ep.OperationID)
		status, body, err := apiClient.GetListRaw(path, withIncludeQuery(query, include))
		require.NoError(t, err, "GET %s?include=%s failed", path, include)
		if status != 200 {
			return false
		}
		got := parseJSON(body)
		require.NotNil(t, got, "response should be valid JSON")
		return checkIncludePopulated(got, include) == nil
	}

	putEndpoints, err := loadIncludePutEndpoints()
	require.NoError(t, err, "load include-supporting PUT endpoints")

	if ep, ok := lookupIncludePutEndpoint(putEndpoints, operationID); ok {
		scenario, hasScenario := includesPutScenarioByOperationID[ep.OperationID]
		require.True(t, hasScenario, "register includesPutScenarioByOperationID[%q]", ep.OperationID)
		path, resolved := resolvePutScenarioPath(ep, scenario.pathValues)
		require.True(t, resolved, "resolvePutScenarioPath(%s) operationId=%s", ep.Path, ep.OperationID)
		status, body, err := apiClient.PutRaw(path, url.Values{"include": {include}}, scenario.buildBody(path))
		require.NoError(t, err, "PUT %s?include=%s failed", path, include)
		if status != 200 {
			return false
		}
		got := parseJSON(body)
		require.NotNil(t, got, "response should be valid JSON")

		targets := scenario.extractTargets(got)
		require.NotEmpty(t, targets, "%s (%s): no extraction targets in response", ep.OperationID, include)
		for _, tgt := range targets {
			if checkIncludePopulated(tgt, include) == nil {
				return true
			}
		}
		return false
	}

	t.Fatalf("opt-out %q references unknown include operation", optOutKey(operationID, include))
	return false
}

func TestIncludes_GetFixtureCoverage(t *testing.T) {
	t.Parallel()

	endpoints, err := loadIncludeGetEndpoints()
	require.NoError(t, err, "load include-supporting GET endpoints")

	var unresolved []string
	for _, ep := range endpoints {
		path, _, ok := resolveGetScenario(ep)
		if !ok || path == "" {
			unresolved = append(unresolved, fmt.Sprintf("%s %s", ep.OperationID, ep.Path))
		}
	}

	require.Empty(t, unresolved,
		"every GET include endpoint must resolve to a seeded fixture path; add seed mappings or includeGetScenarioByOperationID entries:\n%s",
		strings.Join(unresolved, "\n"))
}

func TestIncludes_PutFixtureCoverage(t *testing.T) {
	t.Parallel()

	endpoints, err := loadIncludePutEndpoints()
	require.NoError(t, err, "load include-supporting PUT endpoints")

	var missing []string
	for _, ep := range endpoints {
		if _, ok := includesPutScenarioByOperationID[ep.OperationID]; !ok {
			missing = append(missing, fmt.Sprintf("%s %s", ep.OperationID, ep.Path))
		}
	}

	require.Empty(t, missing,
		"every PUT include endpoint must have a seeded scenario in includesPutScenarioByOperationID:\n%s",
		strings.Join(missing, "\n"))
}

func TestIncludes_ExpandableFieldsCollapseWithoutInclude(t *testing.T) {
	t.Parallel()

	endpoints, err := loadIncludeGetEndpoints()
	require.NoError(t, err, "load include-supporting GET endpoints")
	require.NotEmpty(t, endpoints, "spec contained no GET endpoints with include[] enums")

	for _, ep := range endpoints {
		ep := ep

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()

			path, query, ok := resolveGetScenario(ep)
			require.True(t, ok, "resolveGetScenario(%s) operationId=%s", ep.Path, ep.OperationID)

			status, body, err := apiClient.GetListRaw(path, query)
			require.NoError(t, err, "GET %s failed", path)
			requireStatus(t, 200, status, body)

			got := parseJSON(body)
			require.NotNil(t, got, "response should be valid JSON")

			for _, include := range ep.IncludeEnum {
				assertIncludeCollapsedWithoutRequest(t, got, include)
			}
		})
	}
}

func TestIncludes_OptOutMetadata(t *testing.T) {
	t.Parallel()

	getEndpoints, err := loadIncludeGetEndpoints()
	require.NoError(t, err, "load include-supporting GET endpoints")
	putEndpoints, err := loadIncludePutEndpoints()
	require.NoError(t, err, "load include-supporting PUT endpoints")

	for key, reason := range includesOptOut {
		operationID, include, ok := splitOptOutKey(key)
		require.Truef(t, ok, "invalid opt-out key format: %q", key)
		require.Truef(t, isTaggedOptOutReason(reason),
			"opt-out %q must start with one of seed-gap:, bug:, schema:, got %q", key, reason)

		if ep, ok := lookupIncludeGetEndpoint(getEndpoints, operationID); ok {
			require.Truef(t, endpointDeclaresInclude(ep.IncludeEnum, include),
				"GET opt-out %q references include %q that is not declared by operation %q", key, include, operationID)
			continue
		}
		if ep, ok := lookupIncludePutEndpoint(putEndpoints, operationID); ok {
			require.Truef(t, endpointDeclaresInclude(ep.IncludeEnum, include),
				"PUT opt-out %q references include %q that is not declared by operation %q", key, include, operationID)
			continue
		}

		t.Fatalf("opt-out %q references unknown include operation %q", key, operationID)
	}
}

func TestIncludes_OptOutsRemainNecessary(t *testing.T) {
	t.Parallel()

	for key := range includesOptOut {
		key := key
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			operationID, include, ok := splitOptOutKey(key)
			require.Truef(t, ok, "invalid opt-out key format: %q", key)
			if executeIncludeOptOutProbe(t, operationID, include) {
				t.Fatalf("opt-out %q is stale: the include now populates successfully and the opt-out should be removed", key)
			}
		})
	}
}

// putIncludeScenario wires a PUT+include[] walker to seeded fixtures that can
// tolerate the mutation safely (often idempotent repeats to the same path).
type putIncludeScenario struct {
	pathValues              map[string]string
	buildBody               func(resolvedPath string) map[string]any
	extractTargets          func(root map[string]any) []map[string]any
	disableParallelIncludes bool
	resetBetweenIncludes    func(t *testing.T, index int, path string)
}

func validateProductsPutBody(_ string) map[string]any {
	return map[string]any{
		"products_map": map[string]string{"0": SeedItemSKU},
	}
}

func extractValidateProductsRoots(root map[string]any) []map[string]any {
	pm, ok := root["products"].(map[string]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, raw := range pm {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if jsonField(m, "object") == "product" {
			out = append(out, m)
		}
	}
	return out
}

// includesPutScenarioByOperationID maps OpenAPI operationId values to seeded PUT flows.
var includesPutScenarioByOperationID = map[string]putIncludeScenario{
	"validate-products": {
		buildBody:      validateProductsPutBody,
		extractTargets: extractValidateProductsRoots,
	},
	"add-item-attribute": {
		pathValues: map[string]string{
			"id":           SeedIncludePutAddAttributeItemID,
			"attribute_id": SeedAttributeID,
		},
		buildBody: func(_ string) map[string]any { return map[string]any{} },
		extractTargets: func(root map[string]any) []map[string]any {
			return extractRootObjectTyped(root, "item")
		},
	},
	"change-item-category": {
		pathValues: map[string]string{
			"id":          SeedIncludePutChangeCategoryItemID,
			"category_id": SeedIncludePutAlternateItemCategoryID,
		},
		buildBody: func(_ string) map[string]any { return map[string]any{} },
		extractTargets: func(root map[string]any) []map[string]any {
			return extractRootObjectTyped(root, "item")
		},
	},
	"change-product-product-line": {
		pathValues: map[string]string{
			"id":              SeedIncludePutChangeProductLineSaleID,
			"product_line_id": SeedIncludePutProductLineChangeTargetID,
		},
		buildBody: func(_ string) map[string]any { return map[string]any{} },
		extractTargets: func(root map[string]any) []map[string]any {
			return extractRootObjectTyped(root, "product")
		},
	},
	"change-sales-order-status": {
		pathValues: map[string]string{
			"id": SeedIncludePutEstimateSalesOrderID,
		},
		buildBody: func(_ string) map[string]any {
			return map[string]any{"status_change": "issue", "send_email": false}
		},
		extractTargets: func(root map[string]any) []map[string]any {
			return extractRootObjectTyped(root, "sales_order")
		},
		disableParallelIncludes: true,
		resetBetweenIncludes: func(t *testing.T, index int, path string) {
			if index == 0 {
				return
			}
			st, body, err := apiClient.Put(path, map[string]any{"status_change": "unissue", "send_email": false})
			require.NoError(t, err)
			requireStatus(t, 200, st, body)
		},
	},
	"change-purchase-order-status": {
		pathValues: map[string]string{
			"id": SeedIncludePutEstimatePurchaseOrderID,
		},
		buildBody: func(_ string) map[string]any {
			return map[string]any{"status_change": "issue", "send_email": false}
		},
		extractTargets: func(root map[string]any) []map[string]any {
			return extractRootObjectTyped(root, "purchase_order")
		},
		disableParallelIncludes: true,
		resetBetweenIncludes: func(t *testing.T, index int, path string) {
			if index == 0 {
				return
			}
			st, body, err := apiClient.Put(path, map[string]any{"status_change": "unissue", "send_email": false})
			require.NoError(t, err)
			requireStatus(t, 200, st, body)
		},
	},
	"update-agent-status": {
		pathValues: map[string]string{
			"id": SeedCustomAgentDefinitionID,
		},
		buildBody: func(_ string) map[string]any {
			return map[string]any{"status_code": "active"}
		},
		extractTargets: func(root map[string]any) []map[string]any {
			return extractRootObjectTyped(root, "agent_definition")
		},
	},
}

// TestIncludes_PutPopulateNestedResources is the PUT counterpart of
// TestIncludes_PopulateNestedResources: it discovers PUT endpoints with include[]
// enums from OpenAPI and asserts each declared include materializes populated
// nested resources on extraction targets.
func TestIncludes_PutPopulateNestedResources(t *testing.T) {
	t.Parallel()

	endpoints, err := loadIncludePutEndpoints()
	require.NoError(t, err, "load include-supporting PUT endpoints")
	require.NotEmpty(t, endpoints, "spec contained no PUT endpoints with include[] enums")

	for _, ep := range endpoints {
		ep := ep

		scenario, ok := includesPutScenarioByOperationID[ep.OperationID]
		require.Truef(t, ok, "register includesPutScenarioByOperationID[\"%s\"]", ep.OperationID)

		path, resolved := resolvePutScenarioPath(ep, scenario.pathValues)
		require.True(t, resolved, "resolvePutScenarioPath(%s) operationId=%s", ep.Path, ep.OperationID)

		body := scenario.buildBody(path)

		t.Run(ep.OperationID, func(t *testing.T) {
			t.Parallel()

			includeEnum := ep.IncludeEnum

			runOne := func(t *testing.T, include string) {
				t.Helper()

				if reason, skip := includesOptOut[optOutKey(ep.OperationID, include)]; skip {
					t.Skipf("opt-out: %s", reason)
					return
				}

				status, respBody, err := apiClient.PutRaw(path, url.Values{"include": {include}}, body)
				require.NoError(t, err, "PUT %s?include=%s failed", path, include)
				requireStatus(t, 200, status, respBody)

				got := parseJSON(respBody)
				require.NotNil(t, got, "response should be valid JSON")

				targets := scenario.extractTargets(got)
				require.NotEmpty(t, targets, "%s (%s): no extraction targets in response", ep.OperationID, include)

				var populatedCount int
				var lastPopErr string
				for i, tgt := range targets {
					if incErr := checkIncludePopulated(tgt, include); incErr != nil {
						lastPopErr = fmt.Sprintf("candidate[%d]: %v", i, incErr)
						continue
					}
					populatedCount++
				}
				require.Positive(t, populatedCount,
					"%s?include=%s: none of %d response object(s) had include populated (%s)",
					path, include, len(targets), lastPopErr)
			}

			if scenario.disableParallelIncludes {
				for i := range includeEnum {
					if scenario.resetBetweenIncludes != nil {
						scenario.resetBetweenIncludes(t, i, path)
					}
					inc := includeEnum[i]
					t.Run(inc, func(t *testing.T) {
						runOne(t, inc)
					})
				}
				return
			}

			for _, include := range includeEnum {
				include := include
				t.Run(include, func(t *testing.T) {
					t.Parallel()
					runOne(t, include)
				})
			}
		})
	}
}
