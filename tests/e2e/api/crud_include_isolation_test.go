//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Include isolation: every endpoint must only return expandable sub-objects
// when the client explicitly asks for them via ?include=. The gateway forwards
// only the requested includes to the backend (see resourcekit.FilterIncludes),
// so a request with no include param must yield null/absent sub-objects.
//
// Per-resource GET coverage lives in the crud_*_test.go files. The cases below
// fill the remaining gaps:
//   - list endpoints (in addition to the GET-by-id coverage), and
//   - the request_log embedded as another resource's sub-resource, which must
//     not drag along its own account/actor expansions.
// ──────────────────────────────────────────────

// listIncludeIsolationCase declares a list endpoint and the expandable keys
// that must be null on every item when no include is requested. The keys mirror
// the proven-expandable fields asserted by each resource's GET negative test.
type listIncludeIsolationCase struct {
	name string
	path string
	keys []string
}

func TestIncludeIsolation_ListEndpointsReturnNullSubobjectsWithoutInclude(t *testing.T) {
	t.Parallel()

	cases := []listIncludeIsolationCase{
		{
			name: "sales_orders",
			path: salesOrdersPath,
			keys: []string{"customer", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "order_discount", "lines"},
		},
		{
			name: "agent_runs",
			path: agentRunsPath,
			keys: []string{"actions", "definition", "steps"},
		},
		{
			name: "agent_definitions",
			path: agentDefinitionsPath,
			keys: []string{"config", "tools", "role"},
		},
		{
			name: "audit_events",
			path: auditEventsPath,
			keys: []string{"actor", "changes", "metadata", "request"},
		},
		{
			name: "customers",
			path: customersPath,
			keys: []string{"contact_info", "freight_preferences", "defaults", "notification_preferences", "bill_to_address", "ship_to_address", "type", "price_groups", "parent_account", "child_accounts", "credit_limit"},
		},
		{
			name: "email_logs",
			path: emailLogsPath,
			keys: []string{"sent_by"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			list, status, err := apiClient.GetList(tc.path, url.Values{"limit": {"5"}})
			require.NoError(t, err)
			if status != 200 {
				t.Fatalf("%s list not accessible (status %d)", tc.name, status)
			}
			if len(list.Data) == 0 {
				t.Fatalf("no %s available to assert against", tc.name)
			}
			for i, item := range list.Data {
				m := parseJSON(item)
				for _, k := range tc.keys {
					assert.Nilf(t, m[k], "%s[%d]: %q must be null on list without ?include=%s", tc.name, i, k, k)
				}
			}
		})
	}
}

// TestRequestLogs_ListExpandableFieldsNullWithoutInclude is the list-endpoint
// counterpart to TestRequestLogs_ExpandableFieldsNullWithoutInclude. This is the
// endpoint whose over-fetch (always forwarding "account") forced the heavy
// full-mode query; with includes now derived from the request, a plain list must
// expose no expandable sub-objects.
func TestRequestLogs_ListExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(requestLogsPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	if status != 200 {
		t.Fatalf("request logs list not accessible (status %d)", status)
	}
	if len(list.Data) == 0 {
		t.Fatal("no request logs available")
	}

	for i, item := range list.Data {
		m := parseJSON(item)
		assert.Nilf(t, m["account"], "item[%d]: account must be null without ?include=account", i)
		assert.Nilf(t, m["actor"], "item[%d]: actor must be null without ?include=actor", i)
		assert.Nilf(t, m["query_params"], "item[%d]: query_params must be null without ?include=query_params", i)
		assert.Nilf(t, m["request_body"], "item[%d]: request_body must be null without ?include=request_body", i)
		assert.Nilf(t, m["response_body"], "item[%d]: response_body must be null without ?include=response_body", i)
	}
}

// TestAuditEvents_IncludeRequestEmbeddedRequestLogHasNoAccountOrActor verifies
// that a request_log embedded as an audit event's "request" sub-resource is the
// base resource only. "request.account"/"request.actor" are not part of any
// endpoint's allowed includes, so the loader must not embed them — the client
// never asked for them.
func TestAuditEvents_IncludeRequestEmbeddedRequestLogHasNoAccountOrActor(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(auditEventsPath+"/"+SeedAuditEventID, url.Values{"include": {"request"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	req := jsonObject(parseJSON(body), "request")
	if req == nil {
		t.Fatal("seeded audit event has no linked request log to assert against")
	}
	assert.Equal(t, "request_log", jsonField(req, "object"), "request sub-resource should be a request_log")
	assert.NotEmpty(t, jsonField(req, "id"), "request sub-resource should carry its base id")
	assert.Nil(t, req["account"], "embedded request_log must not carry account (request.account is not requestable)")
	assert.Nil(t, req["actor"], "embedded request_log must not carry actor (request.actor is not requestable)")
}
