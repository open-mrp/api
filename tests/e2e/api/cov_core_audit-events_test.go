//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file closes gaps in the otherwise-extensive coverage of the
// audit-events group (auditEventsPath, declared in crud_audit_events_test.go):
// GetByID 404, the never-asserted source_ip/idempotency_key/AuditFieldChange.object
// fields, actor_types + start_date/end_date filters, cursor-based list
// pagination, and the list-level "account" include-isolation gap. See
// tests/e2e/api/crud_audit_events_test.go for the primary suite (32 pre-existing
// tests) and async_side_effects_test.go for cross-resource changes-field coverage
// — do not duplicate either here.

// --- GetByID: not found ---

func TestCovCoreAuditEvents_GetNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(auditEventsPath+"/adev_000000000000", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// --- Response shape: id format + created_at ---

// TestCovCoreAuditEvents_GetResponseShape closes the id-format/created_at gap.
// It deliberately does NOT reuse the "adev_..." seed fixtures for the id-prefix
// check: those are hand-authored literal ids in shared/db/seed/0014_e2e_extras.sql
// and don't reflect the real generator prefix. Verified live: the generator
// (shared/id/id_prefix_values.go AuditEventIDPrefix = composePrefix(VocAudit,
// VocEvent) = "au"+"ev") issues real ids like "auev_9uwb6800ic1tffc437p", so this
// test triggers a genuine, freshly-generated event and checks against "auev".
func TestCovCoreAuditEvents_GetResponseShape(t *testing.T) {
	t.Parallel()
	path := attributesPath(SeedPropertyID)

	status, body, err := apiClient.Post(path, map[string]any{
		"value": uniqueName("e2e-audit-shape"),
		"color": "blue",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	attrID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, attrID)
	defer apiClient.Delete(path + "/" + attrID)

	expectAuditEvent(t, attrID, "attribute", "create")

	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_ids":   {attrID},
		"resource_types": {"attribute"},
		"actions":        {"create"},
	})
	require.NoError(t, err)
	require.Len(t, list.Data, 1, "should find exactly one create audit event for the new attribute")

	m := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(m, "id"), "auev")
	assertObjectField(t, m, "audit_event")
	assert.Equal(t, "create", jsonField(m, "action"))
	assert.Equal(t, "attribute", jsonField(m, "resource_type"))
	assert.Equal(t, attrID, jsonField(m, "resource_id"))
	assertValidTimestamp(t, jsonField(m, "occurred_at"), "occurred_at")
	assertValidTimestamp(t, jsonField(m, "created_at"), "created_at")
}

// --- source_ip / idempotency_key ---

func TestCovCoreAuditEvents_SourceIPAndIdempotencyKeyPopulated(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(auditEventsPath+"/"+SeedAuditEventWithSourceIPID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assert.Equal(t, SeedAuditEventSourceIP, jsonField(m, "source_ip"))
	assert.Equal(t, SeedAuditEventIdempotencyKey, jsonField(m, "idempotency_key"))
}

func TestCovCoreAuditEvents_SourceIPAndIdempotencyKeyNullByDefault(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(auditEventsPath+"/"+SeedAuditEventID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	assertNilField(t, m, "source_ip")
	assertNilField(t, m, "idempotency_key")
}

// --- AuditFieldChange.object ---

func TestCovCoreAuditEvents_ChangesFieldChangeObjectType(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(auditEventsPath+"/"+SeedAuditEventWithSourceIPID, url.Values{"include": {"changes"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	m := parseJSON(body)
	require.NotNil(t, jsonObject(m, "changes"), "changes should be present with ?include=changes")
	data := jsonListData(m, "changes")
	require.NotEmpty(t, data, "seeded row should have at least one field change")

	change, ok := changeForField(data, "name")
	require.True(t, ok, "seeded row should include a 'name' field change")
	assertObjectField(t, change, "audit_field_change")
}

// --- List-level include isolation: "account" (missing from the shared table
// in crud_include_isolation_test.go, which this file does not modify) ---

func TestCovCoreAuditEvents_ListAccountNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"10"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data, "no audit events available")

	for i, item := range list.Data {
		m := parseJSON(item)
		assert.Nilf(t, m["account"], "item[%d]: account must be null without ?include=account", i)
	}
}

func TestCovCoreAuditEvents_ListAccountPopulatedWithInclude(t *testing.T) {
	t.Parallel()
	list, status, err := apiClient.GetList(auditEventsPath, url.Values{"include": {"account"}, "limit": {"10"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)

	for _, item := range list.Data {
		m := parseJSON(item)
		account := jsonObject(m, "account")
		if account == nil {
			continue
		}
		assertObjectField(t, account, "account")
		assert.NotEmpty(t, jsonField(account, "id"))
		return
	}
	t.Fatal("No audit events with a populated account found in the first 10 events")
}

// --- Cursor-based list pagination ---

// TestCovCoreAuditEvents_ListCursorPagination walks a 3-page cursor result
// scoped via resource_ids to the actor-or-target scope cohort's in-scope rows
// (SeedAuditScopeActorID/TargetID/BothID — the fourth, SeedAuditScopeNeitherID,
// is deliberately out of default actor-or-target scope and excluded). This is
// a purely read-only, non-mutating scope: safe to run in parallel with every
// other test that also reads these rows.
func TestCovCoreAuditEvents_ListCursorPagination(t *testing.T) {
	t.Parallel()
	assertScopedCursorPagination(t, auditEventsPath, url.Values{
		"resource_ids": {SeedAuditScopeActorRes, SeedAuditScopeTargetRes, SeedAuditScopeBothRes},
	}, []string{SeedAuditScopeActorID, SeedAuditScopeTargetID, SeedAuditScopeBothID})
}

// --- actor_types filter ---

func TestCovCoreAuditEvents_FilterByActorTypeUser(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_types": {"user"},
		"include":     {"actor"},
		"limit":       {"10"},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "should find at least 1 audit event with actor_type=user")
	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor, "actor should be present with ?include=actor")
		assert.Equal(t, "user", jsonField(actor, "type"))
	}
}

func TestCovCoreAuditEvents_FilterByActorTypeAPIKey(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_types": {"api_key"},
		"include":     {"actor"},
		"limit":       {"10"},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "should find at least 1 audit event with actor_type=api_key")
	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor, "actor should be present with ?include=actor")
		assert.Equal(t, "api_key", jsonField(actor, "type"))
	}
}

func TestCovCoreAuditEvents_FilterByMultipleActorTypesUnion(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_types": {"user", "api_key"},
		"include":     {"actor"},
		"limit":       {"25"},
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	allowed := map[string]bool{"user": true, "api_key": true}
	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor, "actor should be present with ?include=actor")
		got := jsonField(actor, "type")
		assert.True(t, allowed[got], "actor.type %q not in requested set", got)
	}
}

func TestCovCoreAuditEvents_FilterByInvalidActorTypeRejected(t *testing.T) {
	t.Parallel()
	// Unrecognized enum values in a list filter are rejected with 400 (platform convention).
	status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{"actor_types": {"zzz_no_such_type"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// --- start_date / end_date filters ---

func TestCovCoreAuditEvents_FilterByStartDateExcludesAll(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"start_date": {"2099-01-01T00:00:00Z"},
		"limit":      {"5"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "start_date far in the future should exclude all audit events")
}

func TestCovCoreAuditEvents_FilterByEndDateExcludesAll(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"end_date": {"2000-01-01T00:00:00Z"},
		"limit":    {"5"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "end_date far in the past should exclude all audit events")
}

func TestCovCoreAuditEvents_FilterByStartDateMalformed(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{"start_date": {"not-a-date"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "start_date")
}

func TestCovCoreAuditEvents_FilterByEndDateMalformed(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{"end_date": {"not-a-date"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "end_date")
}

// --- limit boundary validation ---

func TestCovCoreAuditEvents_ListLimitTooLow(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{"limit": {"0"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "Limit")
}

func TestCovCoreAuditEvents_ListLimitTooHigh(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{"limit": {"1001"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "Limit")
}
