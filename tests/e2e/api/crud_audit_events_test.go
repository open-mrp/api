//go:build e2e

package api_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	auditEventsPath             = "/v1/core/audit-events"
	auditEventResourceTypesPath = "/v1/core/audit-events/resource-types"
)

func TestAuditEvents_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.NotNil(t, list.Data)
}

func TestAuditEvents_ListSearchByResourceType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"q": {"unit"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 1, "Search for 'unit' should return at least 1 result")

	// Search matches the token across resource_type, action, resource_id and
	// request_id. Every result must contain "unit" in at least one of the fields
	// visible on the list item (request_id is only exposed via ?include=request).
	for _, item := range list.Data {
		m := parseJSON(item)
		fields := []string{
			jsonField(m, "resource_type"),
			jsonField(m, "action"),
			jsonField(m, "resource_id"),
		}
		var matched bool
		for _, f := range fields {
			if strings.Contains(strings.ToLower(f), "unit") {
				matched = true
				break
			}
		}
		assert.True(t, matched,
			"Search result should contain 'unit' in resource_type/action/resource_id; got resource_type=%q action=%q resource_id=%q",
			fields[0], fields[1], fields[2],
		)
	}
}

func TestAuditEvents_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"q": {"zzzznotanevent99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestAuditEvents_FilterByAction(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"actions": {"create"}, "limit": {"5"}})
	require.NoError(t, err)
	for _, item := range list.Data {
		assert.Equal(t, "create", DataItemField(item, "action"))
	}
}

func TestAuditEvents_FilterBySingleResourceType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_types": {"user"},
		"limit":          {"25"},
	})
	require.NoError(t, err)
	for _, item := range list.Data {
		assert.Equal(t, "user", DataItemField(item, "resource_type"))
	}
}

func TestAuditEvents_FilterByMultipleResourceTypes(t *testing.T) {
	t.Parallel()
	allowed := map[string]struct{}{"user": {}, "account": {}}
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_types": {"user", "account"},
		"limit":          {"50"},
	})
	require.NoError(t, err)
	for _, item := range list.Data {
		got := DataItemField(item, "resource_type")
		_, ok := allowed[got]
		assert.True(t, ok, "resource_type %q not in {user, account}", got)
	}
}

func TestAuditEvents_FilterByResourceTypeNoMatch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_types": {"zzz_definitely_not_a_real_resource_type"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

// --- Multi-value filters ---

// discoverDistinctAuditValues returns up to `n` distinct values of the given
// JSON field across recent audit events. Used to seed multi-value filter tests
// with real data the harness has produced.
func discoverDistinctAuditValues(t *testing.T, field string, n int) []string {
	t.Helper()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"50"}})
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, n)
	for _, item := range list.Data {
		m := parseJSON(item)
		v := jsonField(m, field)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
		if len(out) >= n {
			break
		}
	}
	return out
}

func TestAuditEvents_FilterByMultipleActions(t *testing.T) {
	t.Parallel()
	actions := discoverDistinctAuditValues(t, "action", 2)
	if len(actions) < 2 {
		t.Fatal("Need at least 2 distinct audit actions in recent events")
		return
	}

	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actions": actions,
		"limit":   {"50"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filter by actions %v should return results", actions)

	allowed := map[string]bool{actions[0]: true, actions[1]: true}
	for _, item := range list.Data {
		got := DataItemField(item, "action")
		assert.True(t, allowed[got], "action %q not in requested set %v", got, actions)
	}
}

func TestAuditEvents_FilterByMultipleActionsAllImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actions": {"zzz_no_match_a", "zzz_no_match_b"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Filter by impossible actions should return no results")
}

func TestAuditEvents_FilterBySingleResourceID(t *testing.T) {
	t.Parallel()
	ids := discoverDistinctAuditValues(t, "resource_id", 1)
	if len(ids) == 0 {
		t.Fatal("No resource_ids available in recent audit events")
		return
	}
	resourceID := ids[0]

	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_ids": {resourceID},
		"limit":        {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filtering by resource_id %q should return at least one event", resourceID)

	for _, item := range list.Data {
		assert.Equal(t, resourceID, DataItemField(item, "resource_id"), "All results should match the filtered resource_id")
	}
}

func TestAuditEvents_FilterByMultipleResourceIDs(t *testing.T) {
	t.Parallel()
	ids := discoverDistinctAuditValues(t, "resource_id", 2)
	if len(ids) < 2 {
		t.Fatal("Need at least 2 distinct resource_ids in recent audit events")
		return
	}

	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_ids": ids,
		"limit":        {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "Filter by resource_ids %v should return results", ids)

	allowed := map[string]bool{ids[0]: true, ids[1]: true}
	for _, item := range list.Data {
		got := DataItemField(item, "resource_id")
		assert.True(t, allowed[got], "resource_id %q not in requested set %v", got, ids)
	}
}

func TestAuditEvents_FilterByResourceIDsImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"resource_ids": {"zzz_no_such_resource_id"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Filtering by impossible resource_id should return no results")
}

// discoverAuditActorID fetches recent events with ?include=actor and returns
// the first actor ID found, or empty string if none.
func discoverAuditActorID(t *testing.T) string {
	t.Helper()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"include": {"actor"},
		"limit":   {"10"},
	})
	if err != nil || len(list.Data) == 0 {
		return ""
	}
	for _, item := range list.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		if actor == nil {
			continue
		}
		if id := jsonField(actor, "id"); id != "" {
			return id
		}
	}
	return ""
}

func TestAuditEvents_FilterByActorIDSingle(t *testing.T) {
	t.Parallel()
	actorID := discoverAuditActorID(t)
	if actorID == "" {
		t.Fatal("No audit events with an actor ID available")
		return
	}

	filtered, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_ids": {actorID},
		"include":   {"actor"},
		"limit":     {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, filtered.Data, "Filtering by a known actor_id should return at least one event")

	for _, item := range filtered.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor, "actor should be present with ?include=actor")
		assert.Equal(t, actorID, jsonField(actor, "id"), "All results should match the filtered actor_id")
	}
}

func TestAuditEvents_FilterByMultipleActorIDs(t *testing.T) {
	t.Parallel()
	actorID := discoverAuditActorID(t)
	if actorID == "" {
		t.Fatal("No audit events with an actor ID available")
		return
	}

	filtered, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_ids": {actorID, "actu_zzzzzzzzzzzzzzzz"},
		"include":   {"actor"},
		"limit":     {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, filtered.Data, "Filtering by a known actor_id should match at least one event")

	for _, item := range filtered.Data {
		m := parseJSON(item)
		actor := jsonObject(m, "actor")
		require.NotNil(t, actor)
		assert.Equal(t, actorID, jsonField(actor, "id"))
	}
}

// TestAuditEvents_ListSearchByResourceID verifies free-text search ('q') matches
// resource_id, so an operator can paste a resource id and find every audit event
// about it. Before this fix, search only matched resource_type and action, so an
// id query returned nothing. Seed event adev_01seedsearchtgt01 carries
// SeedAuditEventSearchResourceID.
func TestAuditEvents_ListSearchByResourceID(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"q":     {SeedAuditEventSearchResourceID},
		"limit": {"50"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "search for resource_id %q should return the seeded event", SeedAuditEventSearchResourceID)

	var found bool
	for _, item := range list.Data {
		if DataItemField(item, "resource_id") == SeedAuditEventSearchResourceID {
			found = true
			break
		}
	}
	assert.True(t, found, "search for %q should include the seeded event with that resource_id", SeedAuditEventSearchResourceID)
}

// TestAuditEvents_ListSearchByRequestID verifies search ('q') matches request_id,
// linking an audit event back to the request that produced it. Seed event
// adev_01seedsearchtgt01 carries SeedAuditEventSearchRequestID.
func TestAuditEvents_ListSearchByRequestID(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"q":     {SeedAuditEventSearchRequestID},
		"limit": {"50"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "search for request_id %q should return the seeded event", SeedAuditEventSearchRequestID)

	var found bool
	for _, item := range list.Data {
		if DataItemField(item, "resource_id") == SeedAuditEventSearchResourceID {
			found = true
			break
		}
	}
	assert.True(t, found, "search for request_id %q should include the seeded event", SeedAuditEventSearchRequestID)
}

// TestAuditEvents_FilterByMultipleActorsUnion is the strong union check for the
// actor array filter: filtering by two distinct, real actors must return events
// for BOTH. The seed data provides one user actor and one api_key actor.
func TestAuditEvents_FilterByMultipleActorsUnion(t *testing.T) {
	t.Parallel()
	probe, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"include": {"actor"},
		"limit":   {"100"},
	})
	if err != nil || len(probe.Data) == 0 {
		t.Fatal("No audit events available to discover actor IDs")
		return
	}

	idByType := map[string]string{}
	var ordered []string
	for _, item := range probe.Data {
		actor := jsonObject(parseJSON(item), "actor")
		if actor == nil {
			continue
		}
		id := jsonField(actor, "id")
		typ := jsonField(actor, "type")
		if id == "" {
			continue
		}
		if _, seen := idByType[typ]; !seen {
			idByType[typ] = id
			ordered = append(ordered, id)
		}
		if len(ordered) >= 2 {
			break
		}
	}
	if len(ordered) < 2 {
		t.Fatal("Need at least 2 distinct actors in recent audit events for a union test")
		return
	}
	actorA, actorB := ordered[0], ordered[1]

	filtered, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_ids": {actorA, actorB},
		"include":   {"actor"},
		"limit":     {"200"},
	})
	require.NoError(t, err)

	allowed := map[string]bool{actorA: true, actorB: true}
	sawA, sawB := false, false
	for _, item := range filtered.Data {
		actor := jsonObject(parseJSON(item), "actor")
		require.NotNil(t, actor, "actor should be present with ?include=actor")
		got := jsonField(actor, "id")
		assert.True(t, allowed[got], "actor.id %q not in requested set {%s, %s}", got, actorA, actorB)
		switch got {
		case actorA:
			sawA = true
		case actorB:
			sawB = true
		}
	}
	assert.True(t, sawA, "union filter must return events for actor %s", actorA)
	assert.True(t, sawB, "union filter must return events for actor %s", actorB)
}

func TestAuditEvents_FilterByActorIDsImpossible(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"actor_ids": {"actu_zzzzzzzzzzzzzzzz", "ak_zzzzzzzzzzzzzzzz"},
	})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "Filtering by impossible actor_ids should return no results")
}

// account_id was a dead filter (never reached SQL) and has been removed.
// Unknown query params are explicitly rejected with 400.
func TestAuditEvents_ListRejectsRemovedAccountIDFilter(t *testing.T) {
	t.Parallel()
	statusCode, body, err := apiClient.GetListRaw(auditEventsPath, url.Values{
		"account_id": {"acct_anything"},
		"limit":      {"5"},
	})
	require.NoError(t, err)
	requireStatus(t, 400, statusCode, body)
}

func TestAuditEventResourceTypes_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventResourceTypesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.Greater(t, len(list.Data), 50, "resource type enum should be substantial")

	values := make(map[string]struct{}, len(list.Data))
	for _, item := range list.Data {
		var v string
		require.NoError(t, json.Unmarshal(item, &v))
		values[v] = struct{}{}
	}
	for _, expected := range []string{"audit_event", "user", "account", "sales_order"} {
		_, ok := values[expected]
		assert.True(t, ok, "expected resource type %q to be present", expected)
	}
}

func TestAuditEvents_GetByID(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Fatal("No audit events available")
	}

	eventID := DataItemField(list.Data[0], "id")
	assert.NotEmpty(t, eventID)

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Equal(t, "audit_event", jsonField(got, "object"))
	assert.Equal(t, eventID, jsonField(got, "id"))
	assert.NotEmpty(t, jsonField(got, "action"))
	assert.NotEmpty(t, jsonField(got, "resource_type"))
	assert.NotEmpty(t, jsonField(got, "resource_id"))
	assert.NotEmpty(t, jsonField(got, "occurred_at"))
}

func TestAuditEvents_IncludeChanges(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Fatal("No audit events available")
	}
	eventID := DataItemField(list.Data[0], "id")

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, url.Values{"include": {"changes"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	_, ok := got["changes"]
	assert.True(t, ok, "changes should be present with ?include=changes")
}

func TestAuditEvents_IncludeMetadata(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Fatal("No audit events available")
	}
	eventID := DataItemField(list.Data[0], "id")

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, url.Values{"include": {"metadata"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Contains(t, got, "metadata", "metadata key should be present with ?include=metadata")
}

func TestAuditEvents_IncludeRequest(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"include": {"request"},
		"limit":   {"25"},
	})
	require.NoError(t, err)

	for _, item := range list.Data {
		m := parseJSON(item)
		req := jsonObject(m, "request")
		if req == nil {
			continue
		}
		assert.Equal(t, "request_log", jsonField(req, "object"),
			"request sub-resource object type should be request_log")
		assert.NotEmpty(t, jsonField(req, "id"), "request sub-resource should have an id")
		return
	}
	t.Fatal("No audit events with a request_id found in the first 25 events")
}

func TestAuditEvents_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Fatal("No audit events available")
	}
	eventID := DataItemField(list.Data[0], "id")

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	got := parseJSON(getBody)
	assert.Nil(t, got["actor"], "actor should be null without ?include=actor")
	assert.Nil(t, got["account"], "account should be null without ?include=account")
	assert.Nil(t, got["changes"], "changes should be null without ?include=changes")
	assert.Nil(t, got["metadata"], "metadata should be null without ?include=metadata")
	assert.Nil(t, got["request"], "request should be null without ?include=request")
}

func TestAuditEvents_IncludeActor(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	if len(list.Data) == 0 {
		t.Fatal("No audit events available")
	}
	eventID := DataItemField(list.Data[0], "id")

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, url.Values{"include": {"actor"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	actor := jsonObject(parseJSON(getBody), "actor")
	require.NotNil(t, actor)
	assert.NotEmpty(t, jsonField(actor, "id"))
	assert.Equal(t, "actor", jsonField(actor, "object"))
	assert.NotEmpty(t, jsonField(actor, "type"))
}

// firstAuditEventWithAccount lists audit events with ?include=account and returns
// the first event id together with its resolved target-account id. The far-future
// seed events (0014_e2e_extras.sql) carry a target_account_id, so at least one
// event on the first page always exposes an account.
func firstAuditEventWithAccount(t *testing.T) (eventID, accountID string) {
	t.Helper()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"include": {"account"},
		"limit":   {"25"},
	})
	require.NoError(t, err)
	for _, item := range list.Data {
		m := parseJSON(item)
		account := jsonObject(m, "account")
		if account == nil {
			continue
		}
		return jsonField(m, "id"), jsonField(account, "id")
	}
	t.Fatal("No audit events with a target account found in the first 25 events")
	return "", ""
}

func TestAuditEvents_IncludeAccount(t *testing.T) {
	t.Parallel()
	eventID, _ := firstAuditEventWithAccount(t)

	getStatus, getBody, err := apiClient.GetListRaw(auditEventsPath+"/"+eventID, url.Values{"include": {"account"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)

	account := jsonObject(parseJSON(getBody), "account")
	require.NotNil(t, account, "account should be present with ?include=account")
	assert.Equal(t, "account", jsonField(account, "object"))
	assert.NotEmpty(t, jsonField(account, "id"))
	assert.NotEmpty(t, jsonField(account, "name"))
}

func TestAuditEvents_FilterByTargetAccountID(t *testing.T) {
	t.Parallel()
	wantEventID, accountID := firstAuditEventWithAccount(t)

	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"target_account_ids": {accountID},
		"include":            {"account"},
		"limit":              {"25"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, list.Data, "filtering by a real target account should return events")

	foundWanted := false
	for _, item := range list.Data {
		m := parseJSON(item)
		account := jsonObject(m, "account")
		require.NotNil(t, account, "every event returned under a target_account_ids filter must have a target account")
		assert.Equal(t, accountID, jsonField(account, "id"),
			"target_account_ids filter must only return events targeting the requested account")
		if jsonField(m, "id") == wantEventID {
			foundWanted = true
		}
	}
	assert.True(t, foundWanted, "the seed event targeting the account should appear in the filtered results")
}

func TestAuditEvents_FilterByTargetAccountIDNoMatch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(auditEventsPath, url.Values{
		"target_account_ids": {"ac_doesnotexist00000000000"},
		"limit":              {"25"},
	})
	require.NoError(t, err)
	assert.Empty(t, list.Data, "filtering by an account with no audit events should return no results")
}

// auditEventIDSet collects the `id` of each audit event in a list response.
func auditEventIDSet(data []json.RawMessage) map[string]bool {
	ids := make(map[string]bool, len(data))
	for _, item := range data {
		ids[jsonField(parseJSON(item), "id")] = true
	}
	return ids
}

func assertAuditEventMembership(t *testing.T, data []json.RawMessage, wantPresent, wantAbsent []string) {
	t.Helper()
	ids := auditEventIDSet(data)
	for _, id := range wantPresent {
		assert.True(t, ids[id], "expected audit event %s in filtered results", id)
	}
	for _, id := range wantAbsent {
		assert.False(t, ids[id], "audit event %s should have been excluded by the filter", id)
	}
}

// auditScopeCohort lists the actor-or-target scope cohort by its four resource_ids
// (the caller is the seed account) with the given extra params applied.
func auditScopeCohort(t *testing.T, extra url.Values) *ListResponse {
	t.Helper()
	params := url.Values{
		"resource_ids": {
			SeedAuditScopeActorRes,
			SeedAuditScopeTargetRes,
			SeedAuditScopeBothRes,
			SeedAuditScopeNeitherRes,
		},
		"limit": {"50"},
	}
	for k, vs := range extra {
		for _, v := range vs {
			params.Add(k, v)
		}
	}
	list, _, err := apiClient.GetList(auditEventsPath, params)
	require.NoError(t, err)
	return list
}

// TestAuditEvents_ActorOrTargetScope_DefaultReturnsBothSides verifies the security
// scope: with no account filter the caller (seed account) sees every event where
// its account is the actor account or the target account, and never one where it is
// neither.
func TestAuditEvents_ActorOrTargetScope_DefaultReturnsBothSides(t *testing.T) {
	t.Parallel()
	list := auditScopeCohort(t, nil)
	assert.Len(t, list.Data, 3, "scope should return the actor-side, target-side, and both events")
	assertAuditEventMembership(t, list.Data,
		[]string{SeedAuditScopeActorID, SeedAuditScopeTargetID, SeedAuditScopeBothID},
		[]string{SeedAuditScopeNeitherID})
}

// TestAuditEvents_FilterByActorAccountIDs_NarrowsWithinScope filters the scope
// cohort to the acting-account side: only events whose account_id is the seed
// account remain (actor-side + both).
func TestAuditEvents_FilterByActorAccountIDs_NarrowsWithinScope(t *testing.T) {
	t.Parallel()
	list := auditScopeCohort(t, url.Values{"actor_account_ids": {SeedAccountID}})
	assert.Len(t, list.Data, 2, "actor_account_ids=[seed] should return the actor-side and both events")
	assertAuditEventMembership(t, list.Data,
		[]string{SeedAuditScopeActorID, SeedAuditScopeBothID},
		[]string{SeedAuditScopeTargetID, SeedAuditScopeNeitherID})
}

// TestAuditEvents_FilterByTargetAccountIDs_NarrowsWithinScope filters the scope
// cohort to the target-account side: only events whose target_account_id is the
// seed account remain (target-side + both).
func TestAuditEvents_FilterByTargetAccountIDs_NarrowsWithinScope(t *testing.T) {
	t.Parallel()
	list := auditScopeCohort(t, url.Values{"target_account_ids": {SeedAccountID}})
	assert.Len(t, list.Data, 2, "target_account_ids=[seed] should return the target-side and both events")
	assertAuditEventMembership(t, list.Data,
		[]string{SeedAuditScopeTargetID, SeedAuditScopeBothID},
		[]string{SeedAuditScopeActorID, SeedAuditScopeNeitherID})
}
