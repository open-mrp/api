//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// covAiMemoriesPath mirrors agentMemoriesPath from crud_agent_memories_test.go — a
// distinct package-level const is not declared to avoid redeclaring that symbol.
// Instead this file reuses agentMemoriesPath directly wherever a path const is needed.

// --- CRUD lifecycle ---

func TestCovAiMemories_CRUD(t *testing.T) {
	t.Parallel()

	// CREATE
	createStatus, createBody, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category": "preference",
		"content":  "CRUD lifecycle memory.",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assertIDFormat(t, id, "agmm")
	assertObjectField(t, created, "agent_memory")
	assert.Equal(t, "preference", jsonField(created, "category"))
	assert.Equal(t, "CRUD lifecycle memory.", jsonField(created, "content"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(agentMemoriesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, "CRUD lifecycle memory.", jsonField(got, "content"))

	// UPDATE
	patchStatus, patchBody, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"content": "CRUD lifecycle memory (updated).",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, "CRUD lifecycle memory (updated).", jsonField(updated, "content"))
	assert.Equal(t, "preference", jsonField(updated, "category"), "category should be preserved")

	// DELETE
	delStatus, delBody, err := apiClient.Delete(agentMemoriesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)
	assert.JSONEq(t, `{}`, string(delBody))

	// Verify deletion
	getStatus2, getBody2, err := apiClient.GetListRaw(agentMemoriesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2, "GET after delete should 404: %s", string(getBody2))
}

// --- Create and Update: all fields ---

// TestCovAiMemories_CreateAndUpdateAllFields creates a memory with every settable
// field (including entity_type=customer, whose Entity sub-object is the only case
// where name/handle hydrate — see HydrateCustomerEntities) and asserts every
// response-struct json field, then updates and re-asserts.
func TestCovAiMemories_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	createStatus, createBody, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category":    "preference",
		"content":     "Customer prefers express shipping.",
		"metadata":    map[string]any{"source": "support_ticket"},
		"entity_type": "customer",
		"entity_id":   SeedCustomerAccountID,
		"importance":  0.7,
		"expires_at":  "2027-01-02T15:04:05Z",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	got := parseJSON(createBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(agentMemoriesPath + "/" + id)

	assertIDFormat(t, id, "agmm")
	assertObjectField(t, got, "agent_memory")
	assert.Equal(t, "preference", jsonField(got, "category"))
	assert.Equal(t, "Customer prefers express shipping.", jsonField(got, "content"))

	metadata := jsonObject(got, "metadata")
	require.NotNil(t, metadata, "metadata should echo back the object sent on create")
	assert.Equal(t, "support_ticket", jsonField(metadata, "source"))

	entity := jsonObject(got, "entity")
	require.NotNil(t, entity, "entity should be materialized when entity_type+entity_id are both set")
	assert.Equal(t, SeedCustomerAccountID, jsonField(entity, "id"))
	assert.Equal(t, "entity", jsonField(entity, "object"))
	assert.Equal(t, "customer", jsonField(entity, "type"))
	// entity.name / entity.handle only hydrate for entity_type=="customer" (HydrateCustomerEntities).
	assert.NotEmpty(t, jsonField(entity, "name"), "customer entity should have a hydrated name")
	assert.NotEmpty(t, jsonField(entity, "handle"), "customer entity should have a hydrated handle")

	assert.Equal(t, "0.7", jsonField(got, "importance"))
	assertValidTimestamp(t, jsonField(got, "expires_at"), "expires_at")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// UPDATE with different values
	updateStatus, updateBody, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"category":   "fact",
		"content":    "Customer prefers ground shipping now.",
		"metadata":   map[string]any{"source": "phone_call"},
		"importance": 0.9,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, updateStatus, updateBody)

	updated := parseJSON(updateBody)
	assert.Equal(t, id, jsonField(updated, "id"), "id must not change")
	assertObjectField(t, updated, "agent_memory")
	assert.Equal(t, "fact", jsonField(updated, "category"))
	assert.Equal(t, "Customer prefers ground shipping now.", jsonField(updated, "content"))
	updatedMetadata := jsonObject(updated, "metadata")
	require.NotNil(t, updatedMetadata)
	assert.Equal(t, "phone_call", jsonField(updatedMetadata, "source"))
	assert.Equal(t, "0.9", jsonField(updated, "importance"))

	// entity/expires_at were omitted from the PATCH body, so they should be preserved.
	updatedEntity := jsonObject(updated, "entity")
	require.NotNil(t, updatedEntity, "entity should be preserved when omitted from PATCH")
	assert.Equal(t, SeedCustomerAccountID, jsonField(updatedEntity, "id"))
	assertValidTimestamp(t, jsonField(updated, "expires_at"), "expires_at")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")
}

// TestCovAiMemories_EntityNonCustomerTypeHasNilNameHandle proves the entity name/handle
// only populate for entity_type=="customer" — every other entity_type (e.g. "product")
// always renders name:null, handle:null even though entity.id/type are set.
func TestCovAiMemories_EntityNonCustomerTypeHasNilNameHandle(t *testing.T) {
	t.Parallel()

	id := covAiMemoriesCreate(t, map[string]any{
		"category":    "fact",
		"content":     "A product-scoped memory.",
		"entity_type": "product",
		"entity_id":   SeedProductID,
	})

	status, body, err := apiClient.GetListRaw(agentMemoriesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	entity := jsonObject(got, "entity")
	require.NotNil(t, entity)
	assert.Equal(t, SeedProductID, jsonField(entity, "id"))
	assert.Equal(t, "entity", jsonField(entity, "object"))
	assert.Equal(t, "product", jsonField(entity, "type"))
	assertNilField(t, entity, "name")
	assertNilField(t, entity, "handle")
}

// --- Omitted fields ---

func TestCovAiMemories_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
			"category": "instruction",
			"content":  "Always confirm the shipping address before quoting freight.",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(agentMemoriesPath + "/" + id)

		assertObjectField(t, got, "agent_memory")
		assert.Equal(t, "instruction", jsonField(got, "category"))
		assertNilField(t, got, "metadata")
		assertNilField(t, got, "entity")
		assertNilField(t, got, "expires_at")
		assert.Equal(t, "0", jsonField(got, "importance"), "importance defaults to 0 when omitted")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("CreateMissingCategory", func(t *testing.T) {
		status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
			"content": "Missing category.",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		// The category enum is checked before the required-field rule, so an omitted category surfaces as an invalid value rather than a missing one. Still a 400 naming the same param.
		errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
		assertErrorParam(t, errObj, "category")
	})

	t.Run("CreateMissingContent", func(t *testing.T) {
		status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
			"category": "fact",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "content")
	})

	t.Run("CreateEmptyContent", func(t *testing.T) {
		status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
			"category": "fact",
			"content":  "",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
		errObj := requireErrorResponse(t, body, "missing_field", "invalid_request_error")
		assertErrorParam(t, errObj, "content")
	})

	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		createStatus, createBody, err := apiClient.Post(agentMemoriesPath, map[string]any{
			"category":   "fact",
			"content":    "Original content.",
			"metadata":   map[string]any{"k": "v"},
			"importance": 0.4,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(agentMemoriesPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		patchStatus, patchBody, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
			"content": "Updated content only.",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)
		assert.Equal(t, "Updated content only.", jsonField(got, "content"))
		assert.Equal(t, "fact", jsonField(got, "category"), "category should be preserved")
		metadata := jsonObject(got, "metadata")
		require.NotNil(t, metadata, "metadata should be preserved when omitted from PATCH")
		assert.Equal(t, "v", jsonField(metadata, "k"))
		assert.Equal(t, "0.4", jsonField(got, "importance"), "importance should be preserved")
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})
}

// --- Response shape ---

func TestCovAiMemories_CreateResponseShape(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category": "fact",
		"content":  "Response shape check.",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(agentMemoriesPath + "/" + id)

	assertIDFormat(t, id, "agmm")
	assertObjectField(t, got, "agent_memory")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

func TestCovAiMemories_CreateResponseIncludesLocationHeader(t *testing.T) {
	t.Parallel()
	full, err := apiClient.PostFull(agentMemoriesPath, map[string]any{
		"category": "fact",
		"content":  "Location header check.",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, full.StatusCode, full.Body)

	id := jsonField(parseJSON(full.Body), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(agentMemoriesPath + "/" + id)

	assertCreatedLocation(t, full.Header, id)
}

// --- List ---

func TestCovAiMemories_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(agentMemoriesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "should have at least the seeded memory")
}

func TestCovAiMemories_ListPagination(t *testing.T) {
	t.Parallel()
	assertCursorPaginationAdvances(t, agentMemoriesPath, nil)
}

func TestCovAiMemories_ListSearch(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(agentMemoriesPath, url.Values{"q": {"ground shipping"}})
	require.NoError(t, err)
	assertListContainsID(t, agentMemoriesPath, url.Values{"q": {"ground shipping"}}, SeedAgentMemoryID)
	assert.GreaterOrEqual(t, len(list.Data), 1)
}

func TestCovAiMemories_ListSearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(agentMemoriesPath, url.Values{"q": {"zzzznotamemory99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data)
}

func TestCovAiMemories_ListCategoryFilter(t *testing.T) {
	t.Parallel()

	id := covAiMemoriesCreate(t, map[string]any{
		"category": "instruction",
		"content":  uniqueName("category-filter-target"),
	})

	list, _, err := apiClient.GetList(agentMemoriesPath, url.Values{"category": {"instruction"}, "limit": {"1000"}})
	require.NoError(t, err)
	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "category=instruction filter should include the created memory")

	// A category outside the enum is rejected rather than filtered to nothing: the value cannot match any row, so accepting it would only ever return an empty list that looks like real data.
	status, body, err := apiClient.GetListRaw(agentMemoriesPath, url.Values{"category": {"zzzznocategory99999"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestCovAiMemories_ListEntityTypeFilter(t *testing.T) {
	t.Parallel()

	id := covAiMemoriesCreate(t, map[string]any{
		"category":    "fact",
		"content":     uniqueName("entity-type-filter-target"),
		"entity_type": "product",
		"entity_id":   SeedProductID,
	})

	list, _, err := apiClient.GetList(agentMemoriesPath, url.Values{"entity_type": {"product"}, "limit": {"1000"}})
	require.NoError(t, err)
	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == id {
			found = true
			break
		}
	}
	assert.True(t, found, "entity_type=product filter should include the created memory")

	empty, status, err := apiClient.GetList(agentMemoriesPath, url.Values{"entity_type": {"zzzznoentitytype99999"}})
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assertEmptyListData(t, empty.Data)
}

// TestCovAiMemories_ExpiredMemoryVisibleOnGetButExcludedFromList exercises the
// documented Get/List asymmetry (prodBugSuspect #3): List filters out memories whose
// expires_at is in the past, but Get/Update/Delete apply no such filter and still
// operate on the row normally.
func TestCovAiMemories_ExpiredMemoryVisibleOnGetButExcludedFromList(t *testing.T) {
	t.Parallel()

	uniqueContent := uniqueName("expired-memory")
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category":   "fact",
		"content":    uniqueContent,
		"expires_at": "2020-01-01T00:00:00Z",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body) // create does not reject a past expires_at
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(agentMemoriesPath + "/" + id)

	// GET still returns the expired memory.
	getStatus, getBody, err := apiClient.GetListRaw(agentMemoriesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assert.Equal(t, id, jsonField(parseJSON(getBody), "id"))

	// List excludes it, even scoped by a search term unique to this row.
	list, _, err := apiClient.GetList(agentMemoriesPath, url.Values{"q": {uniqueContent}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "expired memory should be excluded from list results")

	// PATCH and DELETE still operate on it (proving no filter there either).
	patchStatus, patchBody, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"content": uniqueContent + "-updated",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody) // update does not exclude expired memories
}

func TestCovAiMemories_ListValidation_UnknownQueryParam(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(agentMemoriesPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, agentMemoriesPath, status, body)
}

func TestCovAiMemories_ListValidation_LimitOutOfRange(t *testing.T) {
	t.Parallel()
	for _, limit := range []string{"0", "-1", "1001"} {
		status, body, err := apiClient.GetListRaw(agentMemoriesPath, url.Values{"limit": {limit}})
		require.NoError(t, err)
		assert.Equal(t, 400, status, "limit=%s should be rejected: %s", limit, string(body))
	}
}

func TestCovAiMemories_ListValidation_CursorInvalid(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(agentMemoriesPath, url.Values{"cursor": {"not-a-real-cursor"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// --- Expandable fields ---

// expandable = na: AgentMemory has no field tagged `expandable:"true"` (grep of
// agent_memory_resource.go confirms this), and ListMemoriesRequest/RetrieveMemoryRequest
// define no `include` query param at all. The `entity` sub-object is always
// materialized inline regardless of query params (see resourceloaders doc comment
// "no expandable sub-resources"), so there is nothing to gate behind ?include — its
// shape is covered by TestCovAiMemories_CreateAndUpdateAllFields and
// TestCovAiMemories_EntityNonCustomerTypeHasNilNameHandle instead.

// --- actions ---

// actions = na: this group is pure CRUD (List/Create/Retrieve/Update/Delete); there
// are no /actions/<verb> or other non-CRUD routes in services/api-gateway/endpoints/agent-memories.

// --- Idempotency ---

func TestCovAiMemories_CreateIdempotent(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()
	body := map[string]any{
		"category": "fact",
		"content":  uniqueName("idempotent-create"),
	}

	status1, body1, err := apiClient.Post(agentMemoriesPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id1)
	defer apiClient.Delete(agentMemoriesPath + "/" + id1)

	status2, body2, err := apiClient.Post(agentMemoriesPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"), "repeat POST with the same idempotency key should return the original resource")
}

func TestCovAiMemories_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category": "fact",
		"content":  "Idempotent update base.",
	})

	idemKey := newIdempotencyKey()
	patchBody := map[string]any{"content": "Idempotent update applied."}

	status1, resp1, err := apiClient.Patch(agentMemoriesPath+"/"+id, patchBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, resp1)
	updatedAt1 := jsonField(parseJSON(resp1), "updated_at")

	status2, resp2, err := apiClient.Patch(agentMemoriesPath+"/"+id, patchBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, resp2)
	got2 := parseJSON(resp2)
	assert.Equal(t, "Idempotent update applied.", jsonField(got2, "content"))
	assert.Equal(t, updatedAt1, jsonField(got2, "updated_at"), "repeat PATCH with the same idempotency key should be a no-op recovery, not a second mutation")
}

// --- Validation ---

func TestCovAiMemories_CreateValidation_CategoryInvalidEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category": "insight",
		"content":  "Not a recognized category.",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestCovAiMemories_CreateValidation_EntityTypeTooLong(t *testing.T) {
	t.Parallel()
	longType := make([]byte, 256)
	for i := range longType {
		longType[i] = 'a'
	}
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category":    "fact",
		"content":     "Entity type too long.",
		"entity_type": string(longType),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "entity_type")
}

func TestCovAiMemories_CreateValidation_EntityTypeExplicitNullRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category":    "fact",
		"content":     "x",
		"entity_type": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "entity_type")
}

func TestCovAiMemories_CreateValidation_EntityIDExplicitNullRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category":  "fact",
		"content":   "x",
		"entity_id": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "entity_id")
}

func TestCovAiMemories_CreateValidation_ImportanceExplicitNullRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category":   "fact",
		"content":    "x",
		"importance": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "importance")
}

func TestCovAiMemories_CreateValidation_ExpiresAtMalformed(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category":   "fact",
		"content":    "x",
		"expires_at": "not-a-date",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "expires_at")
}

func TestCovAiMemories_CreateValidation_UnknownJSONField(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category":        "fact",
		"content":         "x",
		bogusE2EJSONField: 1,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "POST", agentMemoriesPath, status, body)
}

// TestCovAiMemories_CreateValidation_MetadataExplicitNullAccepted documents
// prodBugSuspect #2: unlike entity_type/entity_id/importance, metadata is a bare
// json.RawMessage (not field.Optional/field.Clearable), so RejectExplicitJSONNulls
// does not guard it — an explicit "metadata": null is silently accepted (201).
func TestCovAiMemories_CreateValidation_MetadataExplicitNullAccepted(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(agentMemoriesPath, map[string]any{
		"category": "fact",
		"content":  "Metadata explicit null.",
		"metadata": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	// explicit metadata:null is accepted (prodBugSuspect #2), not rejected like other optional fields
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(agentMemoriesPath + "/" + id)
	assertNilField(t, got, "metadata")
}

// TestCovAiMemories_CreateMetadataEmptyObjectRendersNull documents prodBugSuspect #1:
// AgentMemoryFromProto treats MetadataJson=="{}" the same as unset, so an explicit
// "metadata": {} on create renders as "metadata": null in the response.
func TestCovAiMemories_CreateMetadataEmptyObjectRendersNull(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category": "fact",
		"content":  "Metadata empty object.",
		"metadata": map[string]any{},
	})

	status, body, err := apiClient.GetListRaw(agentMemoriesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "metadata")
}

// TestCovAiMemories_CreateEntityTypeOnlyYieldsNilEntity documents prodBugSuspect #6:
// there is no cross-field validation pairing entity_type+entity_id on create; sending
// only entity_type is accepted (201) and the response's entity renders null because
// the loader requires both non-empty.
func TestCovAiMemories_CreateEntityTypeOnlyYieldsNilEntity(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category":    "fact",
		"content":     "Entity type without entity id.",
		"entity_type": "product",
	})

	status, body, err := apiClient.GetListRaw(agentMemoriesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "entity")
}

func TestCovAiMemories_GetValidation_NotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(agentMemoriesPath+"/agmm_doesnotexist000", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovAiMemories_UpdateValidation_NotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Patch(agentMemoriesPath+"/agmm_doesnotexist000", map[string]any{
		"content": "x",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

func TestCovAiMemories_UpdateValidation_ContentExplicitNullRejected(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category": "fact",
		"content":  "Content null rejection base.",
	})

	status, body, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"content": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "", "invalid_request_error")
	assertErrorParam(t, errObj, "content")
}

func TestCovAiMemories_UpdateValidation_CategoryInvalidEnum(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category": "fact",
		"content":  "Category invalid enum base.",
	})

	status, body, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"category": "insight",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestCovAiMemories_UpdateValidation_EntityTypeTooLong(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category": "fact",
		"content":  "Entity type too long (update) base.",
	})

	longType := make([]byte, 256)
	for i := range longType {
		longType[i] = 'b'
	}
	status, body, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"entity_type": string(longType),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestCovAiMemories_UpdateValidation_ExpiresAtMalformed(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category": "fact",
		"content":  "Expires-at malformed (update) base.",
	})

	status, body, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"expires_at": "not-a-date",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "expires_at")
}

func TestCovAiMemories_UpdateValidation_UnknownJSONField(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category": "fact",
		"content":  "Unknown field (update) base.",
	})

	status, body, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		bogusE2EJSONField: 1,
	}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "PATCH", agentMemoriesPath+"/"+id, status, body)
}

func TestCovAiMemories_DeleteValidation_NotFoundIsIdempotentNoOp(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(agentMemoriesPath + "/agmm_doesnotexist000")
	require.NoError(t, err)
	requireStatus(t, 200, status, body) // delete of a nonexistent memory is a no-op, not a 404
	assert.JSONEq(t, `{}`, string(body))
}

// TestCovAiMemories_UpdateEntityClearingClearsBothFields documents prodBugSuspect #5:
// service.go sets ClearEntity = req.EntityType.IsClear() || req.EntityID.IsClear(), so
// sending a real entity_type alongside entity_id:null discards the just-provided
// entity_type too — both columns clear, not just entity_id.
func TestCovAiMemories_UpdateEntityClearingClearsBothFields(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category":    "fact",
		"content":     "Entity clear-both base.",
		"entity_type": "customer",
		"entity_id":   SeedCustomerAccountID,
	})

	status, body, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"entity_type": "product",
		"entity_id":   nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertNilField(t, got, "entity")
}

// TestCovAiMemories_UpdateEntityTypeClearableThreeState exercises the
// omit/null/value three-state contract of the value-type field.Clearable[string]
// used for EntityType/EntityID on update (prodBugSuspect #4).
func TestCovAiMemories_UpdateEntityTypeClearableThreeState(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category":    "fact",
		"content":     "Entity clearable three-state base.",
		"entity_type": "customer",
		"entity_id":   SeedCustomerAccountID,
	})

	// omit -> leaves entity untouched
	status, body, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"content": "Entity clearable three-state base (touched).",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	entity := jsonObject(parseJSON(body), "entity")
	require.NotNil(t, entity, "omitting entity_type/entity_id should leave entity untouched")
	assert.Equal(t, SeedCustomerAccountID, jsonField(entity, "id"))

	// value -> sets a new entity (entity_type + entity_id both provided)
	status, body, err = apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"entity_type": "product",
		"entity_id":   SeedProductID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	entity = jsonObject(parseJSON(body), "entity")
	require.NotNil(t, entity)
	assert.Equal(t, SeedProductID, jsonField(entity, "id"))
	assert.Equal(t, "product", jsonField(entity, "type"))

	// null on either field -> clears both (see TestCovAiMemories_UpdateEntityClearingClearsBothFields)
	status, body, err = apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"entity_id": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "entity")
}

// TestCovAiMemories_UpdateExpiresAtClearableThreeState exercises the
// omit/null/value three-state contract of the value-type field.Clearable[string]
// used for ExpiresAt on update (prodBugSuspect #4).
func TestCovAiMemories_UpdateExpiresAtClearableThreeState(t *testing.T) {
	t.Parallel()
	id := covAiMemoriesCreate(t, map[string]any{
		"category":   "fact",
		"content":    "Expires-at clearable three-state base.",
		"expires_at": "2027-06-01T00:00:00Z",
	})

	// omit -> leaves expires_at untouched
	status, body, err := apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"content": "Expires-at clearable three-state base (touched).",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertValidTimestamp(t, jsonField(parseJSON(body), "expires_at"), "expires_at")

	// value -> sets a new expires_at
	status, body, err = apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"expires_at": "2028-01-01T00:00:00Z",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "2028-01-01T00:00:00Z", jsonField(parseJSON(body), "expires_at"))

	// null -> clears permanently
	status, body, err = apiClient.Patch(agentMemoriesPath+"/"+id, map[string]any{
		"expires_at": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assertNilField(t, parseJSON(body), "expires_at")
}

// --- Auth ---

// TestCovAiMemories_CustomerPortalActorForbidden proves every operation requires an
// internal (non-customer-portal) actor: identity.CheckIsInternalActor() runs before
// the PermissionDomainAgentMemories check, so a customer-portal key gets 403
// "insufficient_permissions" ("You must be an internal user...") rather than a
// resource-level 401/403 based on RBAC.
func TestCovAiMemories_CustomerPortalActorForbidden(t *testing.T) {
	t.Parallel()
	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)

	t.Run("List", func(t *testing.T) {
		status, body, err := customer.GetListRaw(agentMemoriesPath, nil)
		require.NoError(t, err)
		requireStatus(t, 403, status, body)
	})

	t.Run("Create", func(t *testing.T) {
		status, body, err := customer.Post(agentMemoriesPath, map[string]any{
			"category": "fact",
			"content":  "x",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 403, status, body)
	})

	t.Run("Retrieve", func(t *testing.T) {
		status, body, err := customer.GetListRaw(agentMemoriesPath+"/"+SeedAgentMemoryID, nil)
		require.NoError(t, err)
		requireStatus(t, 403, status, body)
	})

	t.Run("Update", func(t *testing.T) {
		status, body, err := customer.Patch(agentMemoriesPath+"/"+SeedAgentMemoryID, map[string]any{
			"content": "x",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 403, status, body)
	})

	t.Run("Delete", func(t *testing.T) {
		status, body, err := customer.Delete(agentMemoriesPath + "/" + SeedAgentMemoryID)
		require.NoError(t, err)
		requireStatus(t, 403, status, body)
	})
}

// TestCovAiMemories_CrossTenantIsolation proves tenant B cannot read, mutate, or
// (meaningfully) delete tenant A's seeded memory: Get/Update return 404
// resource_not_found (scoped by account), and Delete is a no-op that still 200s
// (matching the general not-found-delete-is-idempotent contract) without actually
// removing the row — verified by re-fetching it from tenant A afterward.
func TestCovAiMemories_CrossTenantIsolation(t *testing.T) {
	t.Parallel()
	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	t.Run("Get", func(t *testing.T) {
		status, body, err := tenantB.GetListRaw(agentMemoriesPath+"/"+SeedAgentMemoryID, nil)
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("Update", func(t *testing.T) {
		status, body, err := tenantB.Patch(agentMemoriesPath+"/"+SeedAgentMemoryID, map[string]any{
			"content": "cross-tenant should not apply",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("DeleteIsNoOpAndDoesNotRemoveTheRow", func(t *testing.T) {
		status, body, err := tenantB.Delete(agentMemoriesPath + "/" + SeedAgentMemoryID)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		getStatus, getBody, err := apiClient.GetListRaw(agentMemoriesPath+"/"+SeedAgentMemoryID, nil)
		require.NoError(t, err)
		requireStatus(t, 200, getStatus, getBody) // tenant B's delete must not remove tenant A's memory
		assert.Equal(t, SeedAgentMemoryID, jsonField(parseJSON(getBody), "id"))
	})
}

// --- helper ---

// covAiMemoriesCreate creates a memory and registers t.Cleanup to delete it,
// returning the new id. Distinct from createMemory (crud_agent_memories_test.go)
// only in name, to avoid any ambiguity about which file owns it.
func covAiMemoriesCreate(t *testing.T, body map[string]any) string {
	t.Helper()
	status, respBody, err := apiClient.Post(agentMemoriesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	id := jsonField(parseJSON(respBody), "id")
	require.NotEmpty(t, id, "created memory should have an id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(agentMemoriesPath + "/" + id) })
	return id
}
