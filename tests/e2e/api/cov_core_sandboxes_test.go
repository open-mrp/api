//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCovCoreSandboxes_CreateResponseShape closes the responseShape gap: the
// existing crud_sandboxes_test.go tests only assert NotEmpty on id/timestamps.
// This asserts the sbac_ id-prefix format and validates both timestamps as
// parseable RFC3339, per docs/patterns/e2e-test-patterns.md category 4.
func TestCovCoreSandboxes_CreateResponseShape(t *testing.T) {
	t.Parallel()
	name := uniqueName("e2e-sb-shape")
	status, body, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": name,
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	id := jsonField(got, "id")
	require.NotEmpty(t, id)
	defer apiClient.Delete(sandboxesPath + "/" + id)

	assertIDFormat(t, id, "sbac")
	assertObjectField(t, got, "sandbox")
	assert.Equal(t, name, jsonField(got, "name"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// TestCovCoreSandboxes_OmittedFields closes the omittedFields gap: no existing
// test omits the optional `mode` field entirely (every existing create call
// passes it explicitly), and none tests a missing `name` *key* (as opposed to
// an empty-string name).
func TestCovCoreSandboxes_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("CreateWithOnlyRequiredFields", func(t *testing.T) {
		t.Parallel()
		name := uniqueName("e2e-sb-omit")
		status, body, err := apiClient.Post(sandboxesPath, map[string]any{
			"name": name,
			// mode omitted entirely — should default to "blank" per
			// CreateSandboxRequest's `default:"blank"` tag.
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)

		got := parseJSON(body)
		id := jsonField(got, "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(sandboxesPath + "/" + id)

		assertObjectField(t, got, "sandbox")
		assert.Equal(t, name, jsonField(got, "name"))
		assertNilField(t, got, "owner_account")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
	})

	t.Run("CreateMissingNameKey", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.Post(sandboxesPath, map[string]any{
			"mode": "blank",
			// "name" key entirely absent, not just empty.
		}, newIdempotencyKey())
		require.NoError(t, err)
		assert.True(t, status == 400 || status == 422,
			"missing name key should return 400 or 422, got %d: %s", status, string(body))
		requireErrorResponse(t, body, "missing_field", "invalid_request_error")
	})
}

// TestCovCoreSandboxes_CreateValidation_NameTooLong closes the validation gap
// for the `max=255` constraint on `name` — only empty-name is tested elsewhere.
func TestCovCoreSandboxes_CreateValidation_NameTooLong(t *testing.T) {
	t.Parallel()
	longName := strings.Repeat("a", 256)
	status, body, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": longName,
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"256-char name should return 400 or 422, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
}

// TestCovCoreSandboxes_CreateValidation_InvalidMode targets the prodBugSuspect
// flagged in the task spec: an invalid `mode` enum value. Verified live against
// the running stack, the gateway correctly rejects this with 400
// parameter_invalid — so this test pins the CORRECT behavior (no bug found
// here; the suspected silent-coercion-to-blank bug does not reproduce against
// current source).
func TestCovCoreSandboxes_CreateValidation_InvalidMode(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": uniqueName("e2e-sb-badmode"),
		"mode": "bogus_mode",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.True(t, status == 400 || status == 422,
		"invalid mode value should return 400 or 422, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// TestCovCoreSandboxes_ListPagination closes the list gap: no existing test
// walks a cursor-paginated multi-page result. Scoped to rows this test owns
// via a unique `q` search term so parallel tests can't shift the window.
func TestCovCoreSandboxes_ListPagination(t *testing.T) {
	t.Parallel()
	prefix := uniqueName("e2e-sb-pg")
	var ids []string
	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("%s-%d", prefix, i)
		status, body, err := apiClient.Post(sandboxesPath, map[string]any{
			"name": name,
			"mode": "blank",
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		id := jsonField(parseJSON(body), "id")
		require.NotEmpty(t, id)
		defer apiClient.Delete(sandboxesPath + "/" + id)
		ids = append(ids, id)
	}

	assertScopedCursorPagination(t, sandboxesPath, url.Values{"q": {prefix}}, ids)
}

// TestCovCoreSandboxes_DeleteAlreadyDeleted closes the 410 failure-mode gap:
// deleting a sandbox that was already deleted must return 410 resource_gone,
// distinct from the 404-after-delete happy path already covered by
// TestSandboxes_Delete.
func TestCovCoreSandboxes_DeleteAlreadyDeleted(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": uniqueName("e2e-sb-del2x"),
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)

	delStatus1, delBody1, err := apiClient.Delete(sandboxesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 202, delStatus1, delBody1)

	delStatus2, delBody2, err := apiClient.Delete(sandboxesPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 410, delStatus2, delBody2)
	requireErrorResponse(t, delBody2, "resource_gone", "invalid_request_error")
}

// TestCovCoreSandboxes_DeleteNeverExisted closes the 404 failure-mode gap for
// DELETE specifically: an ID that was never created, distinguishing it from the
// already-deleted (410) case above.
func TestCovCoreSandboxes_DeleteNeverExisted(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(sandboxesPath + "/sbac_covneverexisted0000")
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovCoreSandboxes_CreateIdempotent_DifferentBodySameKey closes the
// idempotency gap: same key + DIFFERENT body. Verified live against the
// running stack, the gateway's idempotency mediator correctly rejects this
// with 400 idempotency_error (NOT a silent replay of the first response, as
// the task spec's prodBugSuspect speculated) — this test pins that correct
// behavior.
func TestCovCoreSandboxes_CreateIdempotent_DifferentBodySameKey(t *testing.T) {
	t.Parallel()
	idemKey := newIdempotencyKey()

	name1 := uniqueName("e2e-sb-idem-a")
	status1, body1, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": name1,
		"mode": "blank",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id1)
	defer apiClient.Delete(sandboxesPath + "/" + id1)

	name2 := uniqueName("e2e-sb-idem-b")
	status2, body2, err := apiClient.Post(sandboxesPath, map[string]any{
		"name": name2,
		"mode": "seeded",
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 400, status2, body2)
	requireErrorResponse(t, body2, "validation_failed", "idempotency_error")
}

// TestCovCoreSandboxes_BlockedInSandboxMode_Create closes the 403 failure-mode
// gap for the documented (but previously untested) CheckNotSandboxMode
// business rule: sandbox management is unavailable while acting inside any
// sandbox account. Matches the doc comment on CreateSandboxEndpoint.
func TestCovCoreSandboxes_BlockedInSandboxMode_Create(t *testing.T) {
	t.Parallel()
	sandboxUser := loginAsSandboxUser(t)

	status, body, err := sandboxUser.Post(sandboxesPath, map[string]any{
		"name": uniqueName("e2e-sb-blocked"),
		"mode": "blank",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// TestCovCoreSandboxes_BlockedInSandboxMode_List extends the CheckNotSandboxMode
// coverage to the list-sandboxes operation.
func TestCovCoreSandboxes_BlockedInSandboxMode_List(t *testing.T) {
	t.Parallel()
	sandboxUser := loginAsSandboxUser(t)

	status, body, err := sandboxUser.GetListRaw(sandboxesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// TestCovCoreSandboxes_BlockedInSandboxMode_Get extends the CheckNotSandboxMode
// coverage to the retrieve-sandbox operation.
func TestCovCoreSandboxes_BlockedInSandboxMode_Get(t *testing.T) {
	t.Parallel()
	sandboxUser := loginAsSandboxUser(t)

	status, body, err := sandboxUser.GetListRaw(sandboxesPath+"/"+SeedSandboxID, nil)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

// TestCovCoreSandboxes_BlockedInSandboxMode_Delete extends the
// CheckNotSandboxMode coverage to the delete-sandbox operation. Targets the
// stable seeded sandbox rather than deleting it — the 403 must fire before any
// deletion is attempted, so the seeded fixture is left intact.
func TestCovCoreSandboxes_BlockedInSandboxMode_Delete(t *testing.T) {
	t.Parallel()
	sandboxUser := loginAsSandboxUser(t)

	status, body, err := sandboxUser.Delete(sandboxesPath + "/" + SeedSandboxID)
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")

	// Confirm the seeded sandbox was NOT actually deleted by the blocked call.
	getStatus, getBody, err := apiClient.GetListRaw(sandboxesPath+"/"+SeedSandboxID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
}
