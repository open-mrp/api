//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// /v1/messaging/email-inboxes — plain CRUD resource (create/list/get/update/
// delete, no /actions/* sub-routes; `actions` category is legitimately na —
// see TASK-messaging_email-inboxes.md). No prior test file existed for this
// group; this file is the sole coverage.
//
// Response resource: apiresource.EmailInbox (email_inbox_resource.go).
// Expandable fields: email_domain (*EmailDomain), agent_config
// (*AgentDefinition) — both null unless requested via ?include=.
//
// Known prod bug suspects (see task file §6), verified live against this
// stack before writing assertions:
//   1. [HIGH] PATCH does an unconditional SET on every column — a minimal
//      PATCH {"status":"disabled"} wipes from_name/agent_config_id/
//      agent_trigger_policy/agent_trigger_keywords to null/empty even though
//      they were never mentioned in the request. See
//      TestCovMessagingEmailInboxes_OmittedFields/UpdatePreservesOmittedFields,
//      which is intentionally left asserting the *correct* (preserving)
//      behavior and is expected to fail red until fixed.
//   2. [MED] agent_trigger_policy is never enum-validated on create/update —
//      an arbitrary string is silently accepted and persisted verbatim.
//   3. [MED] agent_config_id has no existence/ownership check — a garbage id
//      is silently accepted (201/200) with agent_config resolving null since
//      hydration for a nonexistent id yields nothing.
//   4. [LOW] from_name / agent_config_id / agent_trigger_policy are
//      field.Optional[T] (not *field.Clearable[T]) on the update request —
//      explicit null is rejected 400 "cannot be null", so none of the three
//      can ever be cleared back to null once set via PATCH.
// Each is documented at its assertion site below rather than fixed.
// ──────────────────────────────────────────────

const covMessagingEmailInboxesPath = "/v1/messaging/email-inboxes"

// covMessagingEmailDomainsPath and covMessagingEmailDomainsCreate are
// declared by the sibling cov_messaging_email-domains_test.go (same
// package) — reused here rather than redeclared, per the one-new-file rule.

// covMessagingEmailInboxCreateBody returns a minimal-but-valid create payload for a
// unique address under the seeded verified domain.
func covMessagingEmailInboxCreateBody(localPart string) map[string]any {
	return map[string]any{
		"email_domain_id": SeedEmailDomainID,
		"address":         localPart + "@mail.e2e.openmrp.ai",
	}
}

// --- 1. CRUD Lifecycle ---

func TestCovMessagingEmailInboxes_CRUD(t *testing.T) {
	t.Parallel()
	addr := uniqueName("e2e-eminb-crud")

	// CREATE
	createStatus, createBody, err := apiClient.Post(covMessagingEmailInboxesPath, covMessagingEmailInboxCreateBody(addr), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	created := parseJSON(createBody)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assertObjectField(t, created, "email_inbox")
	assert.Equal(t, addr+"@mail.e2e.openmrp.ai", jsonField(created, "address"))
	assert.Equal(t, "active", jsonField(created, "status"))

	// GET
	getStatus, getBody, err := apiClient.GetListRaw(covMessagingEmailInboxesPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, addr+"@mail.e2e.openmrp.ai", jsonField(got, "address"))

	// UPDATE
	patchStatus, patchBody, err := apiClient.Patch(covMessagingEmailInboxesPath+"/"+id, map[string]any{
		"status": "disabled",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)
	updated := parseJSON(patchBody)
	assert.Equal(t, "disabled", jsonField(updated, "status"))
	assert.Equal(t, id, jsonField(updated, "id"))

	// DELETE
	delStatus, delBody, err := apiClient.Delete(covMessagingEmailInboxesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, delStatus, delBody)

	// Verify deletion
	getStatus2, _, err := apiClient.GetListRaw(covMessagingEmailInboxesPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus2)
}

// --- 2. Create and Update All Fields ---

func TestCovMessagingEmailInboxes_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()
	addr := uniqueName("e2e-eminb-allf")

	createStatus, createBody, err := apiClient.Post(covMessagingEmailInboxesPath+"?include=email_domain,agent_config", map[string]any{
		"email_domain_id":        SeedEmailDomainID,
		"address":                addr + "@mail.e2e.openmrp.ai",
		"from_name":              "E2E Support Bot",
		"agent_config_id":        SeedAgentDefinitionID,
		"agent_trigger_policy":   "mention",
		"agent_trigger_keywords": []string{"forecast", "reorder"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, createStatus, createBody)

	got := parseJSON(createBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(covMessagingEmailInboxesPath + "/" + id)

	// Assert every response field.
	assertIDFormat(t, id, "emix")
	assertObjectField(t, got, "email_inbox")
	assert.Equal(t, "active", jsonField(got, "status"))
	assert.Equal(t, addr+"@mail.e2e.openmrp.ai", jsonField(got, "address"))
	assert.Equal(t, "E2E Support Bot", jsonField(got, "from_name"))
	assert.Equal(t, "mention", jsonField(got, "agent_trigger_policy"))
	assert.ElementsMatch(t, []string{"forecast", "reorder"}, jsonStringSlice(got, "agent_trigger_keywords"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	domain := jsonObject(got, "email_domain")
	require.NotNil(t, domain, "email_domain must be populated with ?include=email_domain")
	assert.Equal(t, SeedEmailDomainID, jsonField(domain, "id"))
	assert.Equal(t, "email_domain", jsonField(domain, "object"))
	assert.Equal(t, "mail.e2e.openmrp.ai", jsonField(domain, "domain"))
	assert.Equal(t, "verified", jsonField(domain, "status"))

	agentConfig := jsonObject(got, "agent_config")
	require.NotNil(t, agentConfig, "agent_config must be populated with ?include=agent_config")
	assert.Equal(t, SeedAgentDefinitionID, jsonField(agentConfig, "id"))
	assert.Equal(t, "agent_definition", jsonField(agentConfig, "object"))

	// ── UPDATE with different values ──
	patchStatus, patchBody, err := apiClient.Patch(covMessagingEmailInboxesPath+"/"+id+"?include=email_domain,agent_config", map[string]any{
		"status":                 "disabled",
		"from_name":              "Updated Support Bot",
		"agent_config_id":        SeedAgentDefinitionID,
		"agent_trigger_policy":   "keyword",
		"agent_trigger_keywords": []string{"escalate"},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, patchStatus, patchBody)

	updated := parseJSON(patchBody)
	assert.Equal(t, id, jsonField(updated, "id"), "ID must not change")
	assert.Equal(t, "disabled", jsonField(updated, "status"))
	assert.Equal(t, "Updated Support Bot", jsonField(updated, "from_name"))
	assert.Equal(t, "keyword", jsonField(updated, "agent_trigger_policy"))
	assert.Equal(t, []string{"escalate"}, jsonStringSlice(updated, "agent_trigger_keywords"))
	assert.Equal(t, addr+"@mail.e2e.openmrp.ai", jsonField(updated, "address"), "address is immutable via update")
	assertValidTimestamp(t, jsonField(updated, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(updated, "updated_at"), "updated_at")

	updDomain := jsonObject(updated, "email_domain")
	require.NotNil(t, updDomain, "email_domain should still be present with ?include")
	assert.Equal(t, SeedEmailDomainID, jsonField(updDomain, "id"))

	updAgentConfig := jsonObject(updated, "agent_config")
	require.NotNil(t, updAgentConfig, "agent_config should still be present with ?include")
	assert.Equal(t, SeedAgentDefinitionID, jsonField(updAgentConfig, "id"))
}

// --- 3. Omitted Fields ---

func TestCovMessagingEmailInboxes_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		addr := uniqueName("e2e-eminb-omit")
		status, body, err := apiClient.Post(covMessagingEmailInboxesPath, covMessagingEmailInboxCreateBody(addr), newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(covMessagingEmailInboxesPath + "/" + id)

		assertObjectField(t, got, "email_inbox")
		assert.Equal(t, "active", jsonField(got, "status"), "status defaults to active on create")
		assertNilField(t, got, "from_name")
		assertNilField(t, got, "email_domain") // expandable, not included
		assertNilField(t, got, "agent_config") // no agent bound
		assertNilField(t, got, "agent_trigger_policy")
		// agent_trigger_keywords is a required array field normalized to [] not null.
		keywords, ok := got["agent_trigger_keywords"]
		require.True(t, ok, "agent_trigger_keywords should be present")
		arr, ok := keywords.([]any)
		require.True(t, ok, "agent_trigger_keywords should be an array")
		assert.Empty(t, arr)
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("CreateMissingRequiredFields", func(t *testing.T) {
		status, body, err := apiClient.Post(covMessagingEmailInboxesPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"missing email_domain_id and address should return 400 or 422, got %d: %s", status, string(body))
		errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
		assert.Contains(t, errObj["message"], "email_domain_id")
		assert.Contains(t, errObj["message"], "address")
	})

	t.Run("CreateMissingAddressOnly", func(t *testing.T) {
		status, body, err := apiClient.Post(covMessagingEmailInboxesPath, map[string]any{
			"email_domain_id": SeedEmailDomainID,
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"missing address should return 400 or 422, got %d: %s", status, string(body))
	})

	// prodBugSuspect #1 (HIGH): UpdateInbox issues an unconditional SET on
	// every column (email_inbox.sql.go's UpdateEmailInbox), not a
	// COALESCE/merge. A PATCH that sends only `status` therefore wipes
	// from_name/agent_config_id/agent_trigger_policy/agent_trigger_keywords
	// to null/empty even though the request never mentioned them. This
	// sub-test asserts the CORRECT (preserving) behavior per
	// e2e-test-patterns.md §3 and is expected to fail red against the
	// current backend — do not weaken it to match the buggy overwrite.
	t.Run("UpdatePreservesOmittedFields", func(t *testing.T) {
		addr := uniqueName("e2e-eminb-pres")
		createStatus, createBody, err := apiClient.Post(covMessagingEmailInboxesPath, map[string]any{
			"email_domain_id":        SeedEmailDomainID,
			"address":                addr + "@mail.e2e.openmrp.ai",
			"from_name":              "Original Name",
			"agent_config_id":        SeedAgentDefinitionID,
			"agent_trigger_policy":   "mention",
			"agent_trigger_keywords": []string{"forecast", "reorder"},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, createStatus, createBody)

		created := parseJSON(createBody)
		id := jsonField(created, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(covMessagingEmailInboxesPath + "/" + id)
		origCreatedAt := jsonField(created, "created_at")

		// PATCH only `status`.
		patchStatus, patchBody, err := apiClient.Patch(covMessagingEmailInboxesPath+"/"+id, map[string]any{
			"status": "disabled",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)

		got := parseJSON(patchBody)

		assert.Equal(t, "disabled", jsonField(got, "status"))
		assert.Equal(t, origCreatedAt, jsonField(got, "created_at"), "created_at should not change")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		// prodBugSuspect #1: these are expected to be PRESERVED but the live
		// backend currently overwrites them to null/empty on any PATCH.
		assert.Equal(t, "Original Name", jsonField(got, "from_name"),
			"from_name should be preserved when omitted from PATCH body (prodBugSuspect #1: UpdateEmailInbox does an unconditional SET on every column)")
		assert.Equal(t, "mention", jsonField(got, "agent_trigger_policy"),
			"agent_trigger_policy should be preserved when omitted from PATCH body (prodBugSuspect #1)")
		assert.ElementsMatch(t, []string{"forecast", "reorder"}, jsonStringSlice(got, "agent_trigger_keywords"),
			"agent_trigger_keywords should be preserved when omitted from PATCH body (prodBugSuspect #1)")
	})
}

// --- 4. Response Shape ---

func TestCovMessagingEmailInboxes_CreateResponseShape(t *testing.T) {
	t.Parallel()
	addr := uniqueName("e2e-eminb-shape")
	status, body, err := apiClient.Post(covMessagingEmailInboxesPath, covMessagingEmailInboxCreateBody(addr), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(covMessagingEmailInboxesPath + "/" + id)

	// The real ID generator uses id.EmailInboxIDPrefix ("emix"), which differs
	// from the made-up "eminb_" literal used in static seed fixtures — only
	// assert the format on freshly-created ids, per task file §2.
	assertIDFormat(t, id, "emix")
	assertObjectField(t, got, "email_inbox")
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// --- 5. List ---
// No pagination test: ListEmailInboxesRequest is `struct{}` (zero declared
// query params) and the service always returns the full account result set
// wrapped in an always-empty PageInfo — this is intentional (small resource
// cardinality), so pagination coverage is legitimately na, not partial.

func TestCovMessagingEmailInboxes_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covMessagingEmailInboxesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "should have at least the seeded inbox")

	var foundSeed bool
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedEmailInboxID {
			foundSeed = true
			break
		}
	}
	assert.True(t, foundSeed, "seeded inbox should appear in the list")
}

func TestCovMessagingEmailInboxes_List_IncludeEmailDomain(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covMessagingEmailInboxesPath, url.Values{"include": {"email_domain"}})
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	require.GreaterOrEqual(t, len(list.Data), 1)

	var found map[string]any
	for _, raw := range list.Data {
		row := parseJSON(raw)
		if jsonField(row, "id") == SeedEmailInboxID {
			found = row
			break
		}
	}
	require.NotNil(t, found, "seeded inbox should appear in the list")
	domain := jsonObject(found, "email_domain")
	require.NotNil(t, domain, "email_domain should be populated with ?include=email_domain on list")
	assert.Equal(t, SeedEmailDomainID, jsonField(domain, "id"))
}

func TestCovMessagingEmailInboxes_List_UnknownQueryParamRejected(t *testing.T) {
	t.Parallel()
	// ListEmailInboxesRequest declares zero fields at all (not even limit),
	// so this exercises the generic unknown-param framework rejection on a
	// struct with no fields whatsoever.
	status, body, err := apiClient.GetListRaw(covMessagingEmailInboxesPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, covMessagingEmailInboxesPath, status, body)
}

func TestCovMessagingEmailInboxes_List_UnknownIncludeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covMessagingEmailInboxesPath, url.Values{"include": {"bogus"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// --- 6. Expandable Fields ---

func TestCovMessagingEmailInboxes_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covMessagingEmailInboxesPath+"/"+SeedEmailInboxID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertNilField(t, got, "email_domain")
	assertNilField(t, got, "agent_config")
}

func TestCovMessagingEmailInboxes_IncludeEmailDomain(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covMessagingEmailInboxesPath+"/"+SeedEmailInboxID, url.Values{"include": {"email_domain"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	domain := jsonObject(got, "email_domain")
	require.NotNil(t, domain, "email_domain should be present with ?include=email_domain")
	assert.Equal(t, SeedEmailDomainID, jsonField(domain, "id"))
	assert.Equal(t, "email_domain", jsonField(domain, "object"))
	assert.Equal(t, "mail.e2e.openmrp.ai", jsonField(domain, "domain"))
	assert.Equal(t, "verified", jsonField(domain, "status"))
	assert.NotEmpty(t, jsonStringSlice(domain, "dkim_tokens"))
	assertValidTimestamp(t, jsonField(domain, "verified_at"), "verified_at")
	assertValidTimestamp(t, jsonField(domain, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(domain, "updated_at"), "updated_at")
	// agent_config was not requested — still null.
	assertNilField(t, got, "agent_config")
}

func TestCovMessagingEmailInboxes_IncludeAgentConfig(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covMessagingEmailInboxesPath+"/"+SeedEmailInboxID, url.Values{"include": {"agent_config"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	agentConfig := jsonObject(got, "agent_config")
	require.NotNil(t, agentConfig, "agent_config should be present with ?include=agent_config")
	assert.Equal(t, SeedAgentDefinitionID, jsonField(agentConfig, "id"))
	assert.Equal(t, "agent_definition", jsonField(agentConfig, "object"))
	// email_domain was not requested — still null.
	assertNilField(t, got, "email_domain")
}

func TestCovMessagingEmailInboxes_IncludeBoth(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covMessagingEmailInboxesPath+"/"+SeedEmailInboxID, url.Values{"include": {"email_domain,agent_config"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.NotNil(t, jsonObject(got, "email_domain"))
	assert.NotNil(t, jsonObject(got, "agent_config"))
}

// --- 7. Idempotency ---

func TestCovMessagingEmailInboxes_CreateIdempotent(t *testing.T) {
	t.Parallel()
	addr := uniqueName("e2e-eminb-idem")
	body := covMessagingEmailInboxCreateBody(addr)
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(covMessagingEmailInboxesPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id1)
	defer apiClient.Delete(covMessagingEmailInboxesPath + "/" + id1)

	status2, body2, err := apiClient.Post(covMessagingEmailInboxesPath, body, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"), "replaying the same idempotency key with the same body should return the same resource id")
}

func TestCovMessagingEmailInboxes_UpdateIdempotent(t *testing.T) {
	t.Parallel()
	addr := uniqueName("e2e-eminb-updidem")
	created := createAndCleanup(t, covMessagingEmailInboxesPath, covMessagingEmailInboxCreateBody(addr))
	id := jsonField(created, "id")

	idemKey := newIdempotencyKey()
	patchBody := map[string]any{"status": "disabled"}

	status1, resp1, err := apiClient.Patch(covMessagingEmailInboxesPath+"/"+id, patchBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status1, resp1)

	status2, resp2, err := apiClient.Patch(covMessagingEmailInboxesPath+"/"+id, patchBody, idemKey)
	require.NoError(t, err)
	requireStatus(t, 200, status2, resp2)
	assert.Equal(t, jsonField(parseJSON(resp1), "updated_at"), jsonField(parseJSON(resp2), "updated_at"),
		"replaying the same idempotency key should not apply the update a second time")
}

// --- 8. Validation ---

func TestCovMessagingEmailInboxes_CreateValidation_MalformedAddressNoAt(t *testing.T) {
	t.Parallel()
	body := covMessagingEmailInboxCreateBody(uniqueName("e2e"))
	body["address"] = "not-an-email"
	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "address")
}

func TestCovMessagingEmailInboxes_CreateValidation_EmptyDomainPart(t *testing.T) {
	t.Parallel()
	body := covMessagingEmailInboxCreateBody(uniqueName("e2e"))
	body["address"] = "user@"
	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "address")
}

func TestCovMessagingEmailInboxes_CreateValidation_EmptyAddress(t *testing.T) {
	t.Parallel()
	body := covMessagingEmailInboxCreateBody(uniqueName("e2e"))
	body["address"] = ""
	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422, "empty address should return 400 or 422, got %d: %s", status, string(respBody))
}

func TestCovMessagingEmailInboxes_CreateValidation_DomainMismatch(t *testing.T) {
	t.Parallel()
	body := covMessagingEmailInboxCreateBody(uniqueName("e2e"))
	body["address"] = "user@wrongdomain.com"
	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "address")
}

func TestCovMessagingEmailInboxes_CreateValidation_DomainNotFound(t *testing.T) {
	t.Parallel()
	body := covMessagingEmailInboxCreateBody(uniqueName("e2e"))
	body["email_domain_id"] = "emdom_doesnotexist000"
	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, respBody)
	requireErrorResponse(t, respBody, "resource_not_found", "invalid_request_error")
}

// TestCovMessagingEmailInboxes_CreateValidation_DomainNotVerified creates a
// fresh domain via the sibling (out-of-scope) email-domains group without
// verifying it — the stub EmailIdentityProvider leaves it `pending` — then
// asserts the inbox create rejects it. Per task file §8, there is no
// delete-domain endpoint, so the stray domain row is left as harmless test
// debris (consistent with other e2e domain rows).
func TestCovMessagingEmailInboxes_CreateValidation_DomainNotVerified(t *testing.T) {
	t.Parallel()
	domainName := covMessagingEmailDomainsUniqueDomain("e2e-eminb-unverified")
	domain := covMessagingEmailDomainsCreate(t, domainName)
	require.Equal(t, "pending", jsonField(domain, "status"), "freshly registered domain should start pending (stub provider does not auto-verify)")
	domainID := jsonField(domain, "id")
	require.NotEmpty(t, domainID)

	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, map[string]any{
		"email_domain_id": domainID,
		"address":         "support@" + domainName,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "email_domain_id")
}

func TestCovMessagingEmailInboxes_CreateValidation_FromNameExplicitNull(t *testing.T) {
	t.Parallel()
	// prodBugSuspect #4: from_name is field.Optional[string] not
	// *field.Clearable[string], so explicit null is rejected rather than
	// clearing the field. Document the current behavior.
	body := covMessagingEmailInboxCreateBody(uniqueName("e2e"))
	body["from_name"] = nil
	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "from_name")
}

func TestCovMessagingEmailInboxes_CreateValidation_FromNameBlank(t *testing.T) {
	t.Parallel()
	body := covMessagingEmailInboxCreateBody(uniqueName("e2e"))
	body["from_name"] = ""
	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "from_name")
}

// TestCovMessagingEmailInboxes_CreateValidation_AgentConfigIDNotValidated
// documents prodBugSuspect #3: neither the gateway nor the notification
// service verifies the referenced agent config exists or belongs to the
// caller's account before binding it — an unknown id is silently accepted
// (201), unlike email_domain_id which does a real ownership-checked lookup.
// This is a data-quality / potential-authz gap, not the "nice" 404 one might
// expect — asserted as observed, not desired.
func TestCovMessagingEmailInboxes_CreateValidation_AgentConfigIDNotValidated(t *testing.T) {
	t.Parallel()
	body := covMessagingEmailInboxCreateBody(uniqueName("e2e"))
	body["agent_config_id"] = "agdf_doesnotexist00000"
	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	t.Log("prodBugSuspect #3: create accepted an agent_config_id that does not exist (no FK/ownership check) — see TASK-messaging_email-inboxes.md §6.3")

	got := parseJSON(respBody)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(covMessagingEmailInboxesPath + "/" + id)
	// The unknown id was stored, but hydration of a nonexistent agent config
	// resolves to nothing, so ?include=agent_config still comes back null.
	getStatus, getBody, err := apiClient.GetListRaw(covMessagingEmailInboxesPath+"/"+id, url.Values{"include": {"agent_config"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assertNilField(t, parseJSON(getBody), "agent_config")
}

// TestCovMessagingEmailInboxes_CreateValidation_AgentTriggerPolicyNotValidated
// documents prodBugSuspect #2: agent_trigger_policy has no IsValid() enum
// check on create (contrast with the analogous chat-group field in
// conversation_service.go). An arbitrary string is silently accepted and
// persisted verbatim.
func TestCovMessagingEmailInboxes_CreateValidation_AgentTriggerPolicyNotValidated(t *testing.T) {
	t.Parallel()
	body := covMessagingEmailInboxCreateBody(uniqueName("e2e"))
	body["agent_trigger_policy"] = "bogus"
	status, respBody, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	t.Log("prodBugSuspect #2: create accepted agent_trigger_policy='bogus' with no enum validation — see TASK-messaging_email-inboxes.md §6.2")

	got := parseJSON(respBody)
	assert.Equal(t, "bogus", jsonField(got, "agent_trigger_policy"))
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(covMessagingEmailInboxesPath + "/" + id)
}

func TestCovMessagingEmailInboxes_CreateValidation_DuplicateAddress(t *testing.T) {
	t.Parallel()
	addr := uniqueName("e2e-eminb-dup")
	body := covMessagingEmailInboxCreateBody(addr)

	status1, body1, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id1)
	defer apiClient.Delete(covMessagingEmailInboxesPath + "/" + id1)

	// Same address, different idempotency key — a genuine duplicate-create
	// attempt rather than an idempotent replay.
	status2, body2, err := apiClient.Post(covMessagingEmailInboxesPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status2, body2)
	requireErrorResponse(t, body2, "resource_exists", "invalid_request_error")
}

func TestCovMessagingEmailInboxes_UpdateValidation_MissingStatus(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, covMessagingEmailInboxesPath, covMessagingEmailInboxCreateBody(uniqueName("e2e-eminb-updval1")))
	id := jsonField(created, "id")

	status, respBody, err := apiClient.Patch(covMessagingEmailInboxesPath+"/"+id, map[string]any{
		"from_name": "X",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422, "missing status should return 400 or 422, got %d: %s", status, string(respBody))
	errObj := requireErrorResponse(t, respBody, "missing_field", "invalid_request_error")
	assertErrorParam(t, errObj, "status")
}

func TestCovMessagingEmailInboxes_UpdateValidation_InvalidStatusEnum(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, covMessagingEmailInboxesPath, covMessagingEmailInboxCreateBody(uniqueName("e2e-eminb-updval2")))
	id := jsonField(created, "id")

	status, respBody, err := apiClient.Patch(covMessagingEmailInboxesPath+"/"+id, map[string]any{
		"status": "bogus",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "status")
}

// TestCovMessagingEmailInboxes_UpdateValidation_CannotClearFromName
// documents prodBugSuspect #4 on the update side: from_name cannot be
// cleared back to null via PATCH — explicit null is rejected, and there is
// no dedicated "clear from_name" pathway at all in this group.
func TestCovMessagingEmailInboxes_UpdateValidation_CannotClearFromName(t *testing.T) {
	t.Parallel()
	created := createAndCleanup(t, covMessagingEmailInboxesPath, map[string]any{
		"email_domain_id": SeedEmailDomainID,
		"address":         uniqueName("e2e-eminb-clearfn") + "@mail.e2e.openmrp.ai",
		"from_name":       "Has A Name",
	})
	id := jsonField(created, "id")

	status, respBody, err := apiClient.Patch(covMessagingEmailInboxesPath+"/"+id, map[string]any{
		"status":    "active",
		"from_name": nil,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, respBody)
	errObj := requireErrorResponse(t, respBody, "invalid_format", "invalid_request_error")
	assertErrorParam(t, errObj, "from_name")
	t.Log("prodBugSuspect #4: from_name cannot be cleared back to null via PATCH (field.Optional, not *field.Clearable) — see TASK-messaging_email-inboxes.md §6.4")
}

func TestCovMessagingEmailInboxes_UnknownJSONFieldRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covMessagingEmailInboxesPath, map[string]any{bogusE2EJSONField: "x"}, newIdempotencyKey())
	require.NoError(t, err)
	assertJSONUnknownFieldRejected(t, "POST", covMessagingEmailInboxesPath, status, body)
}

// --- 9. Failure Modes: Not Found / Tenant Isolation ---

func TestCovMessagingEmailInboxes_NotFound(t *testing.T) {
	t.Parallel()
	const bogusID = "emix_doesnotexist0000"

	getStatus, _, err := apiClient.GetListRaw(covMessagingEmailInboxesPath+"/"+bogusID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus)

	patchStatus, _, err := apiClient.Patch(covMessagingEmailInboxesPath+"/"+bogusID, map[string]any{"status": "active"}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, patchStatus)

	delStatus, _, err := apiClient.Delete(covMessagingEmailInboxesPath + "/" + bogusID)
	require.NoError(t, err)
	assert.Equal(t, 404, delStatus)
}

// TestCovMessagingEmailInboxes_TenantIsolation asserts that a caller from a
// different account cannot read, update, or delete another account's inbox
// by id — all three return 404, never leaking a 200 or a 500.
func TestCovMessagingEmailInboxes_TenantIsolation(t *testing.T) {
	t.Parallel()
	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	getStatus, getBody, err := tenantB.GetListRaw(covMessagingEmailInboxesPath+"/"+SeedEmailInboxID, nil)
	require.NoError(t, err)
	requireStatus(t, 404, getStatus, getBody)

	patchStatus, patchBody, err := tenantB.Patch(covMessagingEmailInboxesPath+"/"+SeedEmailInboxID, map[string]any{"status": "active"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, patchStatus, patchBody)

	delStatus, delBody, err := tenantB.Delete(covMessagingEmailInboxesPath + "/" + SeedEmailInboxID)
	require.NoError(t, err)
	requireStatus(t, 404, delStatus, delBody)
}

// --- 10. Auth ---

// TestCovMessagingEmailInboxes_Unauthenticated asserts an empty bearer token
// (but valid OpenMRP-Version/OpenMRP-Account headers) is rejected 401
// invalid_credentials on all five operations, matching the pattern used
// elsewhere (e.g. TestCovMessagingBlocks_Unauthenticated).
func TestCovMessagingEmailInboxes_Unauthenticated(t *testing.T) {
	t.Parallel()
	unauth := apiClient.WithBearerToken("", SeedAccountID)

	listStatus, listBody, err := unauth.GetListRaw(covMessagingEmailInboxesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, listStatus, listBody)
	requireErrorResponse(t, listBody, "invalid_credentials", "invalid_request_error")

	getStatus, getBody, err := unauth.GetListRaw(covMessagingEmailInboxesPath+"/"+SeedEmailInboxID, nil)
	require.NoError(t, err)
	requireStatus(t, 401, getStatus, getBody)
	requireErrorResponse(t, getBody, "invalid_credentials", "invalid_request_error")

	postStatus, postBody, err := unauth.Post(covMessagingEmailInboxesPath, covMessagingEmailInboxCreateBody(uniqueName("e2e-eminb-noauth")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, postStatus, postBody)
	requireErrorResponse(t, postBody, "invalid_credentials", "invalid_request_error")

	patchStatus, patchBody, err := unauth.Patch(covMessagingEmailInboxesPath+"/"+SeedEmailInboxID, map[string]any{"status": "active"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 401, patchStatus, patchBody)
	requireErrorResponse(t, patchBody, "invalid_credentials", "invalid_request_error")

	delStatus, delBody, err := unauth.Delete(covMessagingEmailInboxesPath + "/" + SeedEmailInboxID)
	require.NoError(t, err)
	requireStatus(t, 401, delStatus, delBody)
	requireErrorResponse(t, delBody, "invalid_credentials", "invalid_request_error")
}
