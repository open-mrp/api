//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The make-to-order endpoints an agent is allowed to reach.
//
// Flagging an endpoint as an agent tool is what puts it in the catalog agents are
// granted from; an endpoint that quietly falls out of the catalog stops being
// reachable without anything failing. The split matters as much as the presence:
// reading the plan is a tool, and changing what a factory builds is not.

// The read endpoints of this feature are exposed as agent tools, gated on the
// permission their endpoint declares, and none of them is reported as mutating.
//
// Two of them are a PUT and a POST — measuring delivery performance takes a date
// window and quoting takes a quantity, neither of which fits a query string — so
// this is the assertion that `mutating` describes the effect rather than the
// method. A quote listed alongside create_sales_order under the same warning is
// how a merchant learns to ignore the warning.
func TestMakeToOrderTools_ReadEndpointsAreExposed(t *testing.T) {
	t.Parallel()

	catalog := agentToolCatalog(t)

	for slug, wantPermission := range map[string]string{
		"analyze_delivery_performance":              "sales_orders:read",
		"retrieve_customer_lead_time":               "customers:read",
		"list_schedule_at_risk_orders":              "production_schedules:read",
		"quote_promise_date":                        "production_schedules:read",
		"list_fulfillment_recommendations":          "production_schedules:read",
		"list_production_schedule_item_settings":    "production_schedules:read",
		"retrieve_production_schedule_item_setting": "production_schedules:read",
	} {
		tool, ok := catalog[slug]
		require.True(t, ok, "%s must be in the tool catalog, or agents cannot be granted it", slug)

		assert.Equal(t, "api_endpoint", jsonField(tool, "category"))
		assert.NotEmpty(t, jsonField(tool, "description"),
			"a tool an agent has to choose between is only as usable as its description")
		assert.Equal(t, false, tool["mutating"], "%s reads and reports; it changes nothing", slug)
		assert.Contains(t, jsonStringSlice(tool, "required_permissions"), wantPermission,
			"%s must be gated on the permission its endpoint declares", slug)
	}
}

// The other side of the same claim: a tool that genuinely writes still says so.
// A flag that reported everything as safe would be no better than one that
// reported everything as dangerous.
func TestMakeToOrderTools_WritingToolsStillReportMutating(t *testing.T) {
	t.Parallel()

	catalog := agentToolCatalog(t)

	for _, slug := range []string{"create_sales_order", "update_customer", "delete_customer"} {
		tool, ok := catalog[slug]
		require.True(t, ok, "%s should be in the catalog", slug)
		assert.Equal(t, true, tool["mutating"], "%s changes state and must be reported as mutating", slug)
	}
}

// Writing a planning override changes what the floor builds next cycle, so those
// endpoints are deliberately not agent tools — an agent can read the advice and
// report it, and a person adopts it.
func TestMakeToOrderTools_WriteEndpointsAreNotExposed(t *testing.T) {
	t.Parallel()

	catalog := agentToolCatalog(t)

	for _, slug := range []string{
		"upsert_production_schedule_item_setting",
		"delete_production_schedule_item_setting",
		"apply_fulfillment_recommendations",
		"update_production_schedule_settings",
	} {
		_, ok := catalog[slug]
		assert.False(t, ok,
			"%s changes how a factory plans and must not be reachable as an agent tool", slug)
	}
}

// agentToolCatalog returns the whole tool catalog keyed by slug.
func agentToolCatalog(t *testing.T) map[string]map[string]any {
	t.Helper()

	list, status, err := apiClient.GetList(aiToolsPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)

	out := make(map[string]map[string]any, len(list.Data))
	for _, raw := range list.Data {
		tool := parseJSON(raw)
		out[jsonField(tool, "slug")] = tool
	}
	return out
}
