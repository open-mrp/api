//go:build e2e

package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Coverage for the customer-portal custom-domain endpoints:
//   - POST   /v1/settings/portal-domains                       (create)
//   - GET    /v1/settings/portal-domains                       (list)
//   - GET    /v1/settings/portal-domains/{id}                  (get)
//   - POST   /v1/settings/portal-domains/{id}/actions/verify   (verify)
//   - DELETE /v1/settings/portal-domains/{id}                  (delete)
//   - GET    /v1/settings/portal-hosts/{domain}                (resolve, unauthenticated)
//
// In the e2e stack every service runs with PLATFORM=test, so core-service uses the
// in-memory stub PortalDomainProvider (no Vercel token needed). The stub walks the
// real lifecycle across successive provider state-reads: read #1 is UNVERIFIED
// (pending), read #2 is verified+routing but NOT YET SERVING (securing, as if the
// TLS certificate were still issuing), and read #3+ is verified and serving. That is
// why the verify lifecycle below calls verify three times (pending, securing, verified).
//
// Cardinality: an account may hold only ONE portal domain, so the seeded account has
// a single slot. Slot-consuming tests are intentionally NOT parallel — they run in
// `go test`'s sequential phase, each clearing the account first (self-healing against
// a leftover row from a killed prior run) and deleting via t.Cleanup afterwards, so
// the slot is always free when the next one starts. Read-only / validation tests
// (which never create a row) keep t.Parallel().

const (
	covPortalDomainsPath = "/v1/settings/portal-domains"
	covPortalHostsPath   = "/v1/settings/portal-hosts"
)

// covPortalDomainsUniqueDomain returns a fresh, collision-free SUBDOMAIN for this
// run (≥2 dots), which the stub maps to a CNAME record — mirroring the recommended
// `shop.acme.com` shape.
func covPortalDomainsUniqueDomain(prefix string) string {
	return strings.ToLower(uniqueName(prefix)) + ".example.com"
}

// covPortalDomainsClearAccount removes any portal domain currently held by the
// client's account, so a create can claim the single per-account slot regardless
// of leftover state.
func covPortalDomainsClearAccount(t *testing.T, client *Client) {
	t.Helper()
	list, status, err := client.GetList(covPortalDomainsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status, "listing portal domains to clear the account slot should succeed")
	for _, raw := range list.Data {
		var item map[string]any
		require.NoError(t, json.Unmarshal(raw, &item))
		if id := jsonField(item, "id"); id != "" {
			_, _, _ = client.Delete(covPortalDomainsPath + "/" + id)
		}
	}
}

// covPortalDomainsCreate clears the account's single slot, creates a domain, and
// registers cleanup. Returns the parsed created resource.
func covPortalDomainsCreate(t *testing.T, client *Client, domain string) map[string]any {
	t.Helper()
	covPortalDomainsClearAccount(t, client)

	status, body, err := client.Post(covPortalDomainsPath, map[string]any{
		"domain": domain,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)

	created := parseJSON(body)
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { _, _, _ = client.Delete(covPortalDomainsPath + "/" + id) })
	return created
}

// covPortalDomainsVerify calls the verify action and returns (status, parsed body).
func covPortalDomainsVerify(t *testing.T, client *Client, id string) (int, map[string]any) {
	t.Helper()
	status, body, err := client.Post(covPortalDomainsPath+"/"+id+"/actions/verify", nil, newIdempotencyKey())
	require.NoError(t, err)
	return status, parseJSON(body)
}

// covPortalHostsResolveRaw performs an UNAUTHENTICATED GET against the public
// portal-host resolver, bypassing Client (which always injects auth headers) so we
// prove the endpoint is callable without credentials. sendVersion toggles the
// Augno-Version header to cover the "version required" branch. apiClient.baseURL/
// apiVersion are unexported but reachable from this same-package test.
func covPortalHostsResolveRaw(t *testing.T, host string, sendVersion bool) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, apiClient.baseURL+covPortalHostsPath+"/"+host, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	if sendVersion {
		req.Header.Set("Augno-Version", apiClient.apiVersion)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, body
}

// ──────────────────────────────────────────────
// Create + verify lifecycle
// ──────────────────────────────────────────────

// TestCovPortalDomains_Lifecycle covers create -> get -> verify(pending) ->
// verify(securing) -> verify(verified) -> re-verify(no-op). The staged verifies are
// specific to the stub provider (routing on the 2nd state-read, serving on the 3rd).
func TestCovPortalDomains_Lifecycle(t *testing.T) {
	domain := covPortalDomainsUniqueDomain("e2e-podn-lc")
	created := covPortalDomainsCreate(t, apiClient, domain)

	id := jsonField(created, "id")
	assertIDFormat(t, id, "podn")
	assertObjectField(t, created, "portal_domain")
	assert.Equal(t, domain, jsonField(created, "domain"))
	assert.Equal(t, "pending", jsonField(created, "status"))
	assertNilField(t, created, "verified_at")
	assertValidTimestamp(t, jsonField(created, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(created, "updated_at"), "updated_at")

	// A subdomain yields exactly one routing CNAME to Vercel.
	records := jsonListData(created, "dns_records")
	require.Len(t, records, 1, "a subdomain should get exactly one DNS record")
	rec, ok := records[0].(map[string]any)
	require.True(t, ok, "dns_records[0] should be an object")
	assertObjectField(t, rec, "dns_record")
	assert.Equal(t, "CNAME", jsonField(rec, "type"))
	assert.Equal(t, domain, jsonField(rec, "name"))
	assert.Equal(t, "cname.vercel-dns.com", jsonField(rec, "value"))
	assert.Equal(t, "routing", jsonField(rec, "reason"))

	// GET reflects the same freshly-created state.
	getStatus, getBody, err := apiClient.GetListRaw(covPortalDomainsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	got := parseJSON(getBody)
	assert.Equal(t, id, jsonField(got, "id"))
	assert.Equal(t, "pending", jsonField(got, "status"))
	assertNilField(t, got, "verified_at")

	// Verify #1: stub's 1st state-read is still unverified -> stays pending.
	v1Status, v1 := covPortalDomainsVerify(t, apiClient, id)
	requireStatus(t, 200, v1Status, nil)
	assert.Equal(t, "pending", jsonField(v1, "status"), "first verify should still be pending (stub read #1)")
	assertNilField(t, v1, "verified_at")

	// Verify #2: stub's 2nd read is routing but not serving -> securing (TLS cert issuing), verified_at NOT set.
	v2Status, v2 := covPortalDomainsVerify(t, apiClient, id)
	requireStatus(t, 200, v2Status, nil)
	assert.Equal(t, "securing", jsonField(v2, "status"), "second verify should be securing while the cert issues (stub read #2)")
	assertNilField(t, v2, "verified_at")

	// Verify #3: stub's 3rd read reports serving -> flips to verified, verified_at set.
	v3Status, v3 := covPortalDomainsVerify(t, apiClient, id)
	requireStatus(t, 200, v3Status, nil)
	assert.Equal(t, "verified", jsonField(v3, "status"), "third verify should flip to verified (stub read #3)")
	assertValidTimestamp(t, jsonField(v3, "verified_at"), "verified_at")
	verifiedAt := jsonField(v3, "verified_at")
	updatedAt := jsonField(v3, "updated_at")

	// Re-verify: already-verified domains return unchanged (no provider poll, no write).
	v4Status, v4 := covPortalDomainsVerify(t, apiClient, id)
	requireStatus(t, 200, v4Status, nil)
	assert.Equal(t, "verified", jsonField(v4, "status"))
	assert.Equal(t, verifiedAt, jsonField(v4, "verified_at"), "verified_at must not move on a re-verify")
	assert.Equal(t, updatedAt, jsonField(v4, "updated_at"), "updated_at must not bump on a re-verify (no-op)")
}

// TestCovPortalDomains_CreateIdempotent covers gateway idempotency: two POSTs with
// the same Idempotency-Key + body return the SAME id, 201 both times (no second row).
func TestCovPortalDomains_CreateIdempotent(t *testing.T) {
	covPortalDomainsClearAccount(t, apiClient)
	domain := covPortalDomainsUniqueDomain("e2e-podn-idem")
	idemKey := newIdempotencyKey()

	status1, body1, err := apiClient.Post(covPortalDomainsPath, map[string]any{"domain": domain}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status1, body1)
	id1 := jsonField(parseJSON(body1), "id")
	require.NotEmpty(t, id1)
	t.Cleanup(func() { _, _, _ = apiClient.Delete(covPortalDomainsPath + "/" + id1) })

	status2, body2, err := apiClient.Post(covPortalDomainsPath, map[string]any{"domain": domain}, idemKey)
	require.NoError(t, err)
	requireStatus(t, 201, status2, body2)
	assert.Equal(t, id1, jsonField(parseJSON(body2), "id"),
		"replaying the same Idempotency-Key + body should return the original id, not create a second domain")
}

// TestCovPortalDomains_OnePerAccountConflict covers the cardinality-1 rule: a second
// create (different domain) while one already exists returns 409 resource_conflict.
func TestCovPortalDomains_OnePerAccountConflict(t *testing.T) {
	first := covPortalDomainsCreate(t, apiClient, covPortalDomainsUniqueDomain("e2e-podn-conf1"))
	require.NotEmpty(t, jsonField(first, "id"))

	status, body, err := apiClient.Post(covPortalDomainsPath, map[string]any{
		"domain": covPortalDomainsUniqueDomain("e2e-podn-conf2"),
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 409, status, body)
	errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
	assertErrorParam(t, errObj, "domain")
}

// TestCovPortalDomains_List asserts the created domain appears in the account-scoped
// list with the correct object type.
func TestCovPortalDomains_List(t *testing.T) {
	created := covPortalDomainsCreate(t, apiClient, covPortalDomainsUniqueDomain("e2e-podn-list"))
	id := jsonField(created, "id")

	list, status, err := apiClient.GetList(covPortalDomainsPath, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)

	var found bool
	for _, raw := range list.Data {
		var item map[string]any
		require.NoError(t, json.Unmarshal(raw, &item))
		assertObjectField(t, item, "portal_domain")
		if jsonField(item, "id") == id {
			found = true
		}
	}
	assert.True(t, found, "created portal domain %s should appear in the list", id)
}

// TestCovPortalDomains_Delete covers DELETE -> 200, and the domain is then gone (404).
func TestCovPortalDomains_Delete(t *testing.T) {
	created := covPortalDomainsCreate(t, apiClient, covPortalDomainsUniqueDomain("e2e-podn-del"))
	id := jsonField(created, "id")

	status, body, err := apiClient.Delete(covPortalDomainsPath + "/" + id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	getStatus, getBody, err := apiClient.GetListRaw(covPortalDomainsPath+"/"+id, nil)
	require.NoError(t, err)
	requireStatus(t, 404, getStatus, getBody)
}

// ──────────────────────────────────────────────
// Public host resolution (unauthenticated)
// ──────────────────────────────────────────────

// TestCovPortalDomains_ResolveHost covers the unauthenticated resolver end-to-end:
// an unverified domain is NOT resolvable (404), and once verified it resolves to the
// account's public profile (slug). Exercised with no auth headers at all.
func TestCovPortalDomains_ResolveHost(t *testing.T) {
	domain := covPortalDomainsUniqueDomain("e2e-podn-resolve")
	created := covPortalDomainsCreate(t, apiClient, domain)
	id := jsonField(created, "id")

	// Pending domains must not resolve, even unauthenticated.
	pendingStatus, pendingBody := covPortalHostsResolveRaw(t, domain, true)
	requireStatus(t, 404, pendingStatus, pendingBody)
	requireErrorResponse(t, pendingBody, "resource_not_found", "invalid_request_error")

	// Verify #1 -> pending, #2 -> securing. A securing domain (cert still issuing) must not resolve either.
	_, _ = covPortalDomainsVerify(t, apiClient, id)
	securingStatus, securing := covPortalDomainsVerify(t, apiClient, id)
	requireStatus(t, 200, securingStatus, nil)
	require.Equal(t, "securing", jsonField(securing, "status"))
	securingResolveStatus, securingResolveBody := covPortalHostsResolveRaw(t, domain, true)
	requireStatus(t, 404, securingResolveStatus, securingResolveBody)
	requireErrorResponse(t, securingResolveBody, "resource_not_found", "invalid_request_error")

	// Verify #3 -> verified.
	vStatus, v := covPortalDomainsVerify(t, apiClient, id)
	requireStatus(t, 200, vStatus, nil)
	require.Equal(t, "verified", jsonField(v, "status"))

	// Now the public resolver returns the account's public profile, no auth required.
	status, body := covPortalHostsResolveRaw(t, domain, true)
	requireStatus(t, 200, status, body)
	resolved := parseJSON(body)
	assertObjectField(t, resolved, "public_account")
	assert.Equal(t, SeedAccountSlug, jsonField(resolved, "slug"),
		"resolved host should map to the seeded account's portal slug")
	assert.NotEmpty(t, jsonField(resolved, "id"))
}

// TestCovPortalDomains_ResolveHost_Unknown covers an unknown host -> 404.
func TestCovPortalDomains_ResolveHost_Unknown(t *testing.T) {
	t.Parallel()
	status, body := covPortalHostsResolveRaw(t, "does-not-exist.example.com", true)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovPortalDomains_ResolveHost_RequiresVersion guards the regression that a
// missing Augno-Version header yields 400 api_version_required (the bug that had the
// frontend proxy resolving every custom host to null).
func TestCovPortalDomains_ResolveHost_RequiresVersion(t *testing.T) {
	t.Parallel()
	status, body := covPortalHostsResolveRaw(t, "does-not-exist.example.com", false)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "api_version_required", "invalid_request_error")
}

// ──────────────────────────────────────────────
// Validation + not-found + auth (no slot consumed)
// ──────────────────────────────────────────────

// TestCovPortalDomains_CreateValidation_MissingDomain covers an omitted domain.
func TestCovPortalDomains_CreateValidation_MissingDomain(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covPortalDomainsPath, map[string]any{}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "missing domain should return 400: %s", string(body))
}

// TestCovPortalDomains_CreateValidation_NoDot covers a hostname with no dot.
func TestCovPortalDomains_CreateValidation_NoDot(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covPortalDomainsPath, map[string]any{"domain": "nodotdomain"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "domain")
}

// TestCovPortalDomains_CreateValidation_ContainsAt covers an `@` in the domain.
func TestCovPortalDomains_CreateValidation_ContainsAt(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covPortalDomainsPath, map[string]any{"domain": "user@shop.example.com"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "domain")
}

// TestCovPortalDomains_CreateValidation_AugnoDomain covers the explicit rule that
// augno.com hosts cannot be used as a custom portal domain.
func TestCovPortalDomains_CreateValidation_AugnoDomain(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covPortalDomainsPath, map[string]any{"domain": "shop.augno.com"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "domain")
}

// TestCovPortalDomains_GetNotFound covers GET on a well-formed but nonexistent id.
func TestCovPortalDomains_GetNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covPortalDomainsPath+"/podn_0000000000000000000", nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovPortalDomains_VerifyNotFound covers the verify action on a nonexistent id
// (the generic 404 sweep only targets plain GET-by-id, not POST actions).
func TestCovPortalDomains_VerifyNotFound(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(covPortalDomainsPath+"/podn_0000000000000000000/actions/verify", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// TestCovPortalDomains_NoAuth covers the unauthenticated/no-account case on the
// authenticated list endpoint (distinct from the public resolver).
func TestCovPortalDomains_NoAuth(t *testing.T) {
	t.Parallel()
	noAuthClient := apiClient.WithBearerToken("", "")
	status, body, err := noAuthClient.GetListRaw(covPortalDomainsPath, nil)
	require.NoError(t, err)
	assert.True(t, status == 401 || status == 403,
		"unauthenticated list should return 401 or 403, got %d: %s", status, string(body))
}
