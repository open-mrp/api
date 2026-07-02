//go:build e2e

package api_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for /v1/messaging/email-domains: create -> get -> verify lifecycle
// for SES DKIM domain registration. This resource is intentionally narrow:
// no PATCH, no DELETE (domains are immutable once registered; only
// status/verified_at/dkim_tokens change server-side via the verify action),
// and no expandable fields or list query params (list is always the full,
// unfiltered, unpaginated account result set) — those categories are marked
// `na` in the comments below rather than silently skipped.

const covMessagingEmailDomainsPath = "/v1/messaging/email-domains"

// covMessagingEmailDomainsCreate creates a domain with a guaranteed-unique
// name and returns the parsed resource.
func covMessagingEmailDomainsCreate(t *testing.T, domain string) map[string]any {
	t.Helper()
	status, body, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{
		"domain": domain,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	return parseJSON(body)
}

// covMessagingEmailDomainsUniqueDomain returns a fresh, collision-free domain
// name for this test run, mirroring the task-spec convention.
func covMessagingEmailDomainsUniqueDomain(prefix string) string {
	return strings.ToLower(uniqueName(prefix)) + ".example.com"
}

// TestCovMessagingEmailDomains_Lifecycle covers the create -> get -> verify
// -> re-verify (idempotent no-op) shape of the resource. There is no
// update/delete lifecycle for this resource (see file header).
func TestCovMessagingEmailDomains_Lifecycle(t *testing.T) {
	t.Parallel()

	domain := covMessagingEmailDomainsUniqueDomain("e2e-domain-lc")
	created := covMessagingEmailDomainsCreate(t, domain)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	assertIDFormat(t, id, "emdn")
	assertObjectField(t, created, "email_domain")
	assert.Equal(t, domain, jsonField(created, "domain"))
	assert.Equal(t, "pending", jsonField(created, "status"))
	assertNilField(t, created, "verified_at")

	// GET reflects the same freshly-created state.
	getStatus, getBody, err := apiClient.GetListRaw(covMessagingEmailDomainsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assertObjectField(t, got, "email_domain")
	assert.Equal(t, domain, jsonField(got, "domain"))
	assert.Equal(t, "pending", jsonField(got, "status"))
	assertNilField(t, got, "verified_at")

	// dkim_tokens must be a non-null array (stub identity provider in e2e).
	rawTokens, ok := got["dkim_tokens"]
	require.True(t, ok, "dkim_tokens field should be present")
	tokens, ok := rawTokens.([]any)
	require.True(t, ok, "dkim_tokens should be an array")
	assert.NotEmpty(t, tokens, "dkim_tokens should be non-empty for a freshly created domain")

	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

	// VERIFY flips pending -> verified.
	verifyStatus, verifyBody, err := apiClient.Post(covMessagingEmailDomainsPath+"/"+id+"/actions/verify", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, verifyStatus, verifyBody)
	verified := parseJSON(verifyBody)
	assert.Equal(t, id, jsonField(verified, "id"))
	assert.Equal(t, "verified", jsonField(verified, "status"))
	assertValidTimestamp(t, jsonField(verified, "verified_at"), "verified_at")
	updatedAtAfterVerify := jsonField(verified, "updated_at")
	assertValidTimestamp(t, updatedAtAfterVerify, "updated_at")
	verifiedAtAfterVerify := jsonField(verified, "verified_at")

	// Re-VERIFY is a no-op: already-verified domains are returned unchanged
	// (status/verified_at/updated_at must NOT move on a second verify call).
	reverifyStatus, reverifyBody, err := apiClient.Post(covMessagingEmailDomainsPath+"/"+id+"/actions/verify", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, reverifyStatus, reverifyBody)
	reverified := parseJSON(reverifyBody)
	assert.Equal(t, "verified", jsonField(reverified, "status"))
	assert.Equal(t, verifiedAtAfterVerify, jsonField(reverified, "verified_at"),
		"verified_at should not change on a second verify call")
	assert.Equal(t, updatedAtAfterVerify, jsonField(reverified, "updated_at"),
		"updated_at should not bump on a second verify call (idempotent no-op)")
}

// TestCovMessagingEmailDomains_CreateResponseShape asserts response-shape
// invariants (id prefix, object, timestamp formats) independent of the
// lifecycle test above.
func TestCovMessagingEmailDomains_CreateResponseShape(t *testing.T) {
	t.Parallel()

	domain := covMessagingEmailDomainsUniqueDomain("e2e-domain-shape")
	created := covMessagingEmailDomainsCreate(t, domain)

	id := jsonField(created, "id")
	assertIDFormat(t, id, "emdn")
	assertObjectField(t, created, "email_domain")
	assert.Equal(t, "pending", jsonField(created, "status"))
	assertNilField(t, created, "verified_at")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	rawTokens, ok := created["dkim_tokens"]
	require.True(t, ok, "dkim_tokens field should be present")
	tokens, ok := rawTokens.([]any)
	require.True(t, ok, "dkim_tokens should be an array (never null)")
	assert.NotEmpty(t, tokens)
}

// TestCovMessagingEmailDomains_CreateAllFields is the "assert every response
// field" test required by the pattern doc. There is no update leg for this
// resource (immutable after create — see file header), so this only covers
// the create response.
func TestCovMessagingEmailDomains_CreateAllFields(t *testing.T) {
	t.Parallel()

	domain := covMessagingEmailDomainsUniqueDomain("e2e-domain-allf")
	created := covMessagingEmailDomainsCreate(t, domain)

	// Every json field on the EmailDomain resource struct.
	assert.NotEmpty(t, jsonField(created, "id"))
	assert.Equal(t, "email_domain", jsonField(created, "object"))
	assert.Equal(t, domain, jsonField(created, "domain"))
	assert.Equal(t, "pending", jsonField(created, "status"))
	assert.NotNil(t, created["dkim_tokens"])
	assertNilField(t, created, "verified_at")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")
}

// TestCovMessagingEmailDomains_DomainNormalization asserts the server
// lower-cases and trims the domain before storing/returning it.
func TestCovMessagingEmailDomains_DomainNormalization(t *testing.T) {
	t.Parallel()

	base := covMessagingEmailDomainsUniqueDomain("e2e-domain-Norm")
	mixed := "  " + strings.ToUpper(base[:1]) + base[1:] + "  "

	status, body, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{
		"domain": mixed,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	got := parseJSON(body)
	assert.Equal(t, strings.ToLower(strings.TrimSpace(mixed)), jsonField(got, "domain"),
		"domain should be lower-cased and trimmed server-side")
}

// TestCovMessagingEmailDomains_CreateValidation_MissingDomain covers the
// omitted-required-field case: `domain` absent from the body entirely.
func TestCovMessagingEmailDomains_CreateValidation_MissingDomain(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	// The framework-level required-field check may surface as missing_field
	// or validation_failed depending on which layer catches it first; only
	// assert the status per docs/patterns/e2e-test-patterns.md §8.
	assert.Equal(t, 400, status, "missing domain should return 400: %s", string(body))
}

// TestCovMessagingEmailDomains_CreateValidation_EmptyDomain covers `domain: ""`.
func TestCovMessagingEmailDomains_CreateValidation_EmptyDomain(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{
		"domain": "",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "empty domain should return 400: %s", string(body))
}

// TestCovMessagingEmailDomains_CreateValidation_NoDot covers a domain missing
// a `.` — an explicit apierror.NewParameterInvalidError case.
func TestCovMessagingEmailDomains_CreateValidation_NoDot(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{
		"domain": "nodotdomain",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "domain")
}

// TestCovMessagingEmailDomains_CreateValidation_ContainsSpace covers a
// domain containing whitespace.
func TestCovMessagingEmailDomains_CreateValidation_ContainsSpace(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{
		"domain": "has space.com",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "domain")
}

// TestCovMessagingEmailDomains_CreateValidation_ContainsAt covers a domain
// containing `@` (i.e. an email address rather than a bare domain).
func TestCovMessagingEmailDomains_CreateValidation_ContainsAt(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{
		"domain": "has@at.com",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "domain")
}

// TestCovMessagingEmailDomains_CreateDuplicateConflict covers the
// already-registered-domain case: 409 resource_exists, distinct from the
// 400 format-validation cases above.
func TestCovMessagingEmailDomains_CreateDuplicateConflict(t *testing.T) {
	t.Parallel()

	// First registration succeeds so the conflict is isolated from other
	// parallel tests (rather than depending on the globally-seeded domain).
	domain := covMessagingEmailDomainsUniqueDomain("e2e-domain-dup")
	first := covMessagingEmailDomainsCreate(t, domain)
	require.NotEmpty(t, jsonField(first, "id"))

	status, body, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{
		"domain": domain,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	requireErrorResponse(t, body, "resource_exists", "invalid_request_error")
}

// TestCovMessagingEmailDomains_CreateIdempotent covers create idempotency
// per docs/patterns/e2e-test-patterns.md §7: two POSTs with the same
// Idempotency-Key and the same body must return the SAME id with 201 both
// times (a gateway-level idempotency-key cache/replay, not a per-service
// concern — verified live against the running stack).
func TestCovMessagingEmailDomains_CreateIdempotent(t *testing.T) {
	t.Parallel()

	domain := covMessagingEmailDomainsUniqueDomain("e2e-domain-idem")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{
		"domain": domain,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id1)

	status2, body2, err := apiClient.Post(covMessagingEmailDomainsPath, map[string]any{
		"domain": domain,
	}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"),
		"replaying the same Idempotency-Key + body should return the original resource id, not create a second row")
}

// TestCovMessagingEmailDomains_List asserts the seeded verified domain is
// present in the account-scoped, unpaginated list response. There is no
// limit/cursor/q/include/sort param on this endpoint (ListEmailDomainsRequest
// is an empty struct) — pagination/search sub-facets are `na` for this group
// and intentionally not tested here.
func TestCovMessagingEmailDomains_List(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covMessagingEmailDomainsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)
	assert.GreaterOrEqual(t, len(list.Data), 1, "should have at least the seeded email domain")

	var foundSeed bool
	for _, raw := range list.Data {
		var item map[string]any
		require.NoError(t, json.Unmarshal(raw, &item))
		assertObjectField(t, item, "email_domain")
		if jsonField(item, "id") == SeedEmailDomainID {
			foundSeed = true
			assert.Equal(t, "mail.e2e.augno.com", jsonField(item, "domain"))
			assert.Equal(t, "verified", jsonField(item, "status"))
			assertValidTimestamp(t, jsonField(item, "verified_at"), "verified_at")
		}
	}
	assert.True(t, foundSeed, "seeded email domain %s should appear in the list", SeedEmailDomainID)
}

// TestCovMessagingEmailDomains_GetNotFound covers GET on a well-formed but
// nonexistent id.
func TestCovMessagingEmailDomains_GetNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covMessagingEmailDomainsPath+"/emdn_0000000000000000000", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovMessagingEmailDomains_VerifyNotFound covers the verify action on a
// well-formed but nonexistent id (the generic 404 sweep in
// error_response_test.go only targets plain GET-by-id, not POST actions, so
// this action endpoint needs its own dedicated coverage).
func TestCovMessagingEmailDomains_VerifyNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covMessagingEmailDomainsPath+"/emdn_0000000000000000000/actions/verify", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovMessagingEmailDomains_DeleteNotAllowed documents (rather than
// silently omitting) that this resource has no DELETE route: it is
// immutable once created. Method routing returns 405, not 404.
func TestCovMessagingEmailDomains_DeleteNotAllowed(t *testing.T) {
	t.Parallel()
	domain := covMessagingEmailDomainsUniqueDomain("e2e-domain-nodel")
	created := covMessagingEmailDomainsCreate(t, domain)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	status, body, err := apiClient.Delete(covMessagingEmailDomainsPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 405, status, "DELETE should not be routed for email domains: %s", string(body))
}

// TestCovMessagingEmailDomains_NoAuth covers the unauthenticated/no-account
// case as a cheap smoke test (not a required category for this group, but
// low-cost given the standard messaging:read permission check).
func TestCovMessagingEmailDomains_NoAuth(t *testing.T) {
	t.Parallel()
	noAuthClient := apiClient.WithBearerToken("", "")
	status, body, err := noAuthClient.GetListRaw(covMessagingEmailDomainsPath, nil)
	require.NoError(t, err)
	assert.True(t, status == 401 || status == 403,
		"unauthenticated list should return 401 or 403, got %d: %s", status, string(body))
}

// TestCovMessagingEmailDomains_TenantIsolationGet mirrors
// tenant_isolation_test.go: Tenant B must not be able to GET Tenant A's
// email domain, even though email_domain.domain has a global (not
// per-account) unique index — a cross-tenant GET must still 404, not leak
// existence via any other status.
func TestCovMessagingEmailDomains_TenantIsolationGet(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	domain := covMessagingEmailDomainsUniqueDomain("e2e-domain-iso")
	created := covMessagingEmailDomainsCreate(t, domain)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)

	status, body, err := clientB.GetListRaw(covMessagingEmailDomainsPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, status,
		"Tenant B should get 404 (not 403) for Tenant A's email domain: %s", string(body))
}
