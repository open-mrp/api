//go:build e2e

package api_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const notesPath = "/v1/core/notes"

func createNote(t *testing.T, c *Client, body map[string]any) map[string]any {
	t.Helper()
	status, respBody, err := c.Post(notesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(notesPath + "/" + id) })
	return created
}

func noteIDs(t *testing.T, params url.Values) []string {
	t.Helper()
	list, _, err := apiClient.GetList(notesPath, params)
	require.NoError(t, err)
	var ids []string
	for _, raw := range list.Data {
		ids = append(ids, jsonField(parseJSON(raw), "id"))
	}
	return ids
}

// --- CRUD ---

func TestNotes_CRUD(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(notesPath, map[string]any{
		"record": map[string]any{"type": "customer", "id": SeedCustomerAccountID},
		"body":   uniqueName("e2e-note-crud"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	assertIDFormat(t, id, "nt")

	status, body, err = apiClient.GetListRaw(notesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, id, jsonField(parseJSON(body), "id"))

	status, body, err = apiClient.Patch(notesPath+"/"+id, map[string]any{"body": "Edited."}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "Edited.", jsonField(parseJSON(body), "body"))

	status, body, err = apiClient.Delete(notesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertDeletedStub(t, body, id, "note")

	status, body, err = apiClient.GetListRaw(notesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 410, status, body)

	status, body, err = apiClient.Delete(notesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 410, status, body)
}

func TestNotes_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	expires := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	text := uniqueName("e2e-note-all")

	got := createNote(t, apiClient, map[string]any{
		"record":     map[string]any{"type": "sales_order", "id": SeedSalesOrderID},
		"body":       text,
		"type":       "instruction",
		"importance": "high",
		"expires_at": expires.Format(time.RFC3339),
	})
	id := jsonField(got, "id")
	assertIDFormat(t, id, "nt")
	assertObjectField(t, got, "note")
	assert.Equal(t, text, jsonField(got, "body"))
	assert.Equal(t, "instruction", jsonField(got, "type"))
	assert.Equal(t, "high", jsonField(got, "importance"))
	assert.Equal(t, "active", jsonField(got, "status"))
	assertValidTimestamp(t, jsonField(got, "expires_at"), "expires_at")
	assertNilField(t, got, "edited_at")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	record := jsonObject(got, "record")
	require.NotNil(t, record)
	assert.Equal(t, SeedSalesOrderID, jsonField(record, "id"))
	assert.Equal(t, "record", jsonField(record, "object"))
	assert.Equal(t, "sales_order", jsonField(record, "type"))
	assert.NotEmpty(t, jsonField(record, "number"), "a sales order's stub carries its number")
	assert.NotEmpty(t, jsonField(record, "status"))
	assert.NotNil(t, jsonObject(record, "metadata"))

	actor := jsonObject(got, "actor")
	require.NotNil(t, actor)
	assert.Equal(t, "api_key", jsonField(actor, "type"))
	assert.Equal(t, SeedAPIKeyID, jsonField(actor, "id"))
	assert.NotEmpty(t, jsonField(actor, "name"))
	_, hasClient := actor["client"]
	assert.True(t, hasClient, "actor always carries client")

	status, body, err := apiClient.Patch(notesPath+"/"+id, map[string]any{
		"body":       text + " (edited)",
		"type":       "fact",
		"importance": "low",
		"expires_at": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	updated := parseJSON(body)
	assert.Equal(t, text+" (edited)", jsonField(updated, "body"))
	assert.Equal(t, "fact", jsonField(updated, "type"))
	assert.Equal(t, "low", jsonField(updated, "importance"))
	assertNilField(t, updated, "expires_at")
	assertValidTimestamp(t, jsonField(updated, "edited_at"), "edited_at")
	assert.Equal(t, SeedSalesOrderID, jsonField(jsonObject(updated, "record"), "id"), "record is preserved")
	assert.Equal(t, jsonField(got, "created_at"), jsonField(updated, "created_at"))
}

func TestNotes_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("AccountWideDefaults", func(t *testing.T) {
		got := createNote(t, apiClient, map[string]any{"body": uniqueName("e2e-note-min")})
		assertNilField(t, got, "record")
		assert.Equal(t, "comment", jsonField(got, "type"))
		assert.Equal(t, "normal", jsonField(got, "importance"))
		assert.Equal(t, "active", jsonField(got, "status"))
		assertNilField(t, got, "expires_at")
		assertNilField(t, got, "edited_at")
	})

	t.Run("MissingBody", func(t *testing.T) {
		status, body, err := apiClient.Post(notesPath, map[string]any{"type": "fact"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
	})

	t.Run("BlankBody", func(t *testing.T) {
		status, body, err := apiClient.Post(notesPath, map[string]any{"body": "   "}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
	})

	t.Run("UnknownType", func(t *testing.T) {
		status, body, err := apiClient.Post(notesPath, map[string]any{"body": "x", "type": "memo"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
	})

	t.Run("UnknownRecordType", func(t *testing.T) {
		status, body, err := apiClient.Post(notesPath, map[string]any{"body": "x", "record": map[string]any{"type": "widget", "id": "w_1"}}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
	})

	t.Run("MissingRecord", func(t *testing.T) {
		status, body, err := apiClient.Post(notesPath, map[string]any{"body": "x", "record": map[string]any{"type": "sales_order", "id": "or_doesnotexist"}}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 422, status, body)
	})

	t.Run("PatchOneFieldPreservesTheRest", func(t *testing.T) {
		got := createNote(t, apiClient, map[string]any{"body": "keep", "type": "preference", "importance": "high"})
		status, body, err := apiClient.Patch(notesPath+"/"+jsonField(got, "id"), map[string]any{"importance": "low"}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		updated := parseJSON(body)
		assert.Equal(t, "keep", jsonField(updated, "body"))
		assert.Equal(t, "preference", jsonField(updated, "type"))
		assert.Equal(t, "low", jsonField(updated, "importance"))
		assertNilField(t, updated, "edited_at")
	})
}

// --- List ---

func TestNotes_ListFilters(t *testing.T) {
	t.Parallel()
	tag := uniqueName("e2e-note-list")
	onOrder := createNote(t, apiClient, map[string]any{"body": tag + " order", "type": "fact", "record": map[string]any{"type": "sales_order", "id": SeedSalesOrderID}})
	accountWide := createNote(t, apiClient, map[string]any{"body": tag + " account", "type": "instruction"})
	expired := createNote(t, apiClient, map[string]any{"body": tag + " expired", "expires_at": time.Now().UTC().Add(time.Second).Format(time.RFC3339)})
	time.Sleep(2 * time.Second)

	all := noteIDs(t, url.Values{"q": {tag}})
	assert.ElementsMatch(t, []string{jsonField(onOrder, "id"), jsonField(accountWide, "id"), jsonField(expired, "id")}, all, "omitting statuses returns every note")

	assert.Equal(t, []string{jsonField(onOrder, "id")}, noteIDs(t, url.Values{"q": {tag}, "record_ids[]": {SeedSalesOrderID}}))
	assert.Equal(t, []string{jsonField(onOrder, "id")}, noteIDs(t, url.Values{"q": {tag}, "record_types[]": {"sales_order"}}))
	assert.Equal(t, []string{jsonField(accountWide, "id")}, noteIDs(t, url.Values{"q": {tag}, "types[]": {"instruction"}}))
	assert.Equal(t, []string{jsonField(expired, "id")}, noteIDs(t, url.Values{"q": {tag}, "statuses[]": {"expired"}}))
	assert.ElementsMatch(t, []string{jsonField(onOrder, "id"), jsonField(accountWide, "id")}, noteIDs(t, url.Values{"q": {tag}, "statuses[]": {"active"}}))
	assert.Len(t, noteIDs(t, url.Values{"q": {tag}, "actor_types[]": {"api_key"}}), 3)
	assert.Empty(t, noteIDs(t, url.Values{"q": {tag}, "actor_types[]": {"agent"}}))
	assert.Empty(t, noteIDs(t, url.Values{"q": {uniqueName("e2e-note-nothing")}}))

	expiredGot := parseJSON(getNoteBody(t, notesPath+"/"+jsonField(expired, "id")))
	assert.Equal(t, "expired", jsonField(expiredGot, "status"))
}

func TestNotes_ListPagination(t *testing.T) {
	t.Parallel()
	tag := uniqueName("e2e-note-page")
	for range 3 {
		createNote(t, apiClient, map[string]any{"body": tag})
	}
	list, _, err := apiClient.GetList(notesPath, url.Values{"q": {tag}, "limit": {"2"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 2)
	require.NotNil(t, list.PageInfo.NextPageURL, "a third note is on the next page")

	status, body, err := apiClient.GetListRaw(notesPath, url.Values{"limit": {"101"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// --- Authorship ---

func TestNotes_OnlyAuthorEditsOnlyAuthorOrAdminDeletes(t *testing.T) {
	t.Parallel()
	writer := customRoleClient(t, "notes:read", "notes:create", "notes:update", "notes:delete")
	other := customRoleClient(t, "notes:read", "notes:create", "notes:update", "notes:delete")

	note := createNote(t, writer, map[string]any{"body": uniqueName("e2e-note-author")})
	id := jsonField(note, "id")

	status, body, err := other.Patch(notesPath+"/"+id, map[string]any{"body": "hijack"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)

	status, body, err = other.Delete(notesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)

	status, body, err = apiClient.Patch(notesPath+"/"+id, map[string]any{"body": "admin edit"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)

	status, body, err = writer.Patch(notesPath+"/"+id, map[string]any{"body": "own edit"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Delete(notesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertDeletedStub(t, body, id, "note")
}

func TestNotes_RequireNotesPermission(t *testing.T) {
	t.Parallel()
	reader := customRoleClient(t, "notes:read")
	status, body, err := reader.Post(notesPath, map[string]any{"body": "x"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)

	none := customRoleClient(t, "customers:read")
	status, body, err = none.GetListRaw(notesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
}

func TestNotes_PortalUsersNeverSeeNotes(t *testing.T) {
	t.Parallel()
	createNote(t, apiClient, map[string]any{"body": uniqueName("e2e-note-portal"), "record": map[string]any{"type": "customer", "id": SeedCustomerAccountID}})
	portal := NewClient(envOr("E2E_BASE_URL", defaultBaseURL), SeedCustomerAPIKey, SeedAccountID)
	status, body, err := portal.GetListRaw(notesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
}

func TestNotes_TenantIsolation(t *testing.T) {
	t.Parallel()
	note := createNote(t, apiClient, map[string]any{"body": uniqueName("e2e-note-tenant")})
	tenantB := NewClient(envOr("E2E_BASE_URL", defaultBaseURL), SeedTenantBAPIKey, SeedTenantBAccountID)
	status, body, err := tenantB.GetListRaw(notesPath+"/"+jsonField(note, "id"), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// --- Idempotency ---

func TestNotes_CreateAndUpdateIdempotent(t *testing.T) {
	t.Parallel()
	key := newIdempotencyKey()
	body := map[string]any{"body": uniqueName("e2e-note-idem")}
	s1, b1, err := apiClient.Post(notesPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, s1, b1)
	id := jsonField(parseJSON(b1), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(notesPath + "/" + id) })
	s2, b2, err := apiClient.Post(notesPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, s2, b2)
	assert.Equal(t, id, jsonField(parseJSON(b2), "id"))

	patchKey := newIdempotencyKey()
	for range 2 {
		s, b, err := apiClient.Patch(notesPath+"/"+id, map[string]any{"type": "fact"}, patchKey)
		require.NoError(t, err)
		requireStatus(t, 200, s, b)
		assert.Equal(t, id, jsonField(parseJSON(b), "id"))
	}
}

// --- Version ---

func TestNotes_MemoriesRouteGoneOnForge(t *testing.T) {
	t.Parallel()
	req, err := http.NewRequest(http.MethodGet, envOr("E2E_BASE_URL", defaultBaseURL)+agentMemoriesPath, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+SeedAPIKey)
	req.Header.Set("OpenMRP-Account", SeedAccountID)
	req.Header.Set("OpenMRP-Version", defaultAPIVersion)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 404, resp.StatusCode, "1.0.forge callers use notes")
}

func TestNotes_MemoriesOnPreview9AreNotes(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category": "preference", "content": uniqueName("e2e-note-memory"), "importance": 0.8,
		"entity_type": "customer", "entity_id": SeedCustomerAccountID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(notesPath + "/" + id) })

	note := parseJSON(getNoteBody(t, notesPath+"/"+id))
	assert.Equal(t, "preference", jsonField(note, "type"))
	assert.Equal(t, "high", jsonField(note, "importance"))
	assert.Equal(t, SeedCustomerAccountID, jsonField(jsonObject(note, "record"), "id"))
	assert.Equal(t, "customer", jsonField(jsonObject(note, "record"), "type"))
}

func getNoteBody(t *testing.T, path string) []byte {
	t.Helper()
	status, body, err := apiClient.GetListRaw(path, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return body
}
