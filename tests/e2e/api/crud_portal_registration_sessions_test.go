//go:build e2e

package api_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Portal registration session tests exercise the session-based customer-portal
// registration flow: a logged-in buyer starts (or resumes) a session for a seller
// by slug, advances it step by step saving form data, then completes it (which
// registers them as a customer) or abandons it.
//
// The endpoints are scoped to the authenticated buyer (identity.Actor.ID) and the
// seller slug, so each test registers its own fresh buyer via newPortalBuyerClient
// — that keeps every (buyer, seller) session space isolated and lets these run in
// parallel. Seed seller: acme-inc (SeedAccountSlug), with customer group DME and
// terms owned by it, which the new-customer completion path requires.

const portalRegSessionsPath = "/v1/sales/portal-registration-sessions"

// newPortalBuyerClient registers a fresh, email-verified user with no account and
// returns a client authenticated as that user — the "buyer" persona a portal
// registration is scoped to. Reuses the self-serve registration flow helpers.
func newPortalBuyerClient(t *testing.T) *Client {
	t.Helper()

	status, body, err := apiClient.Post(registrationSessionsPath, map[string]any{
		"email":     uniqueRegistrationEmail(),
		"plan_code": "free",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	sessionID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, sessionID)

	// Verify the email (stands in for the emailed link) so the user can be created.
	token := registrationVerificationToken(t, sessionID)
	status, body, err = apiClient.Put(verifyTokenPath(token), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Create the user — the response logs them in via cookie. We stop here: the
	// buyer has a user identity but no account, exactly the portal buyer persona.
	resp, err := apiClient.PostFull(registrationSessionsPath+"/"+sessionID+"/users", map[string]any{
		"name":     "E2E Portal Buyer",
		"password": "P@ssw0rd123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)

	return apiClient.WithBearerToken(accessTokenFromSetCookie(t, resp.Header), "")
}

// startPortalRegistration creates (or resumes) the buyer's session for the seed seller.
func startPortalRegistration(t *testing.T, buyer *Client) map[string]any {
	t.Helper()
	status, body, err := buyer.Post(portalRegSessionsPath, map[string]any{
		"seller_slug": SeedAccountSlug,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

// fullNewCustomerData returns a complete session-data payload satisfying the
// new-customer registration requirements (name, address, group, both terms).
func fullNewCustomerData(name string) map[string]any {
	return map[string]any{
		"customer_name":       name,
		"customer_group_id":   SeedCustomerGroupID,
		"shipping_term_id":    SeedShippingTermID,
		"payment_term_id":     SeedPaymentTermID,
		"phone":               "555-0100",
		"address_name":        name,
		"address_street_1":    "123 Buyer St",
		"address_locality":    "Columbus",
		"address_state":       "OH",
		"address_postal_code": "43004",
		"address_country":     "US",
	}
}

// TestPortalRegistration_CreateShape verifies the start response shape.
func TestPortalRegistration_CreateShape(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)

	session := startPortalRegistration(t, buyer)
	assert.Equal(t, "portal_registration_session", jsonField(session, "object"))
	assert.True(t, strings.HasPrefix(jsonField(session, "id"), "porgse_"),
		"id should carry the porgse_ prefix, got %q", jsonField(session, "id"))
	assert.Equal(t, "customer_details", jsonField(session, "step"), "a fresh session starts at customer_details")
	assert.Equal(t, SeedAccountSlug, jsonField(session, "seller_slug"))
	assert.NotEmpty(t, jsonField(session, "seller_account_id"))
	assert.Nil(t, session["is_existing_customer"], "is_existing_customer is unset until chosen")
	assert.Nil(t, session["completed_at"])
	assert.Nil(t, session["abandoned_at"])
	assert.NotEmpty(t, jsonField(session, "created_at"))
}

// TestPortalRegistration_ResumeReturnsSameSession confirms a second start for the
// same (buyer, seller) resumes the in-progress session rather than creating a new one.
func TestPortalRegistration_ResumeReturnsSameSession(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)

	first := startPortalRegistration(t, buyer)
	firstID := jsonField(first, "id")
	require.NotEmpty(t, firstID)

	second := startPortalRegistration(t, buyer)
	assert.Equal(t, firstID, jsonField(second, "id"),
		"starting again should resume the existing incomplete session")
}

// TestPortalRegistration_UpdatePersistsData advances the session and confirms the
// saved data and step are read back on a subsequent GET.
func TestPortalRegistration_UpdatePersistsData(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)
	sessionID := jsonField(startPortalRegistration(t, buyer), "id")
	sessionPath := portalRegSessionsPath + "/" + sessionID

	status, body, err := buyer.Patch(sessionPath, map[string]any{
		"step": "billing_address",
		"session_data": map[string]any{
			"customer_name":    "Wizard Buyer Co",
			"address_street_1": "500 Persisted Ave",
			"address_locality": "Dublin",
			"address_state":    "OH",
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	updated := parseJSON(body)
	assert.Equal(t, "billing_address", jsonField(updated, "step"))

	// GET reflects the persisted step + data.
	status, body, err = buyer.Do("GET", sessionPath, nil, "")
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	got := parseJSON(body)
	assert.Equal(t, "billing_address", jsonField(got, "step"))
	data := jsonObject(got, "session_data")
	require.NotNil(t, data, "session_data should be present after an update")
	assert.Equal(t, "portal_registration_session_data", jsonField(data, "object"))
	assert.Equal(t, "Wizard Buyer Co", jsonField(data, "customer_name"))
	assert.Equal(t, "500 Persisted Ave", jsonField(data, "address_street_1"))
	assert.Equal(t, "Dublin", jsonField(data, "address_locality"))
}

// TestPortalRegistration_ForwardOnlySteps confirms steps cannot move backwards.
func TestPortalRegistration_ForwardOnlySteps(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)
	sessionID := jsonField(startPortalRegistration(t, buyer), "id")
	sessionPath := portalRegSessionsPath + "/" + sessionID

	// Advance to the last step.
	status, body, err := buyer.Patch(sessionPath, map[string]any{"step": "contact"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Moving back to an earlier step is rejected.
	status, body, err = buyer.Patch(sessionPath, map[string]any{"step": "customer_details"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestPortalRegistration_GetUnknownNotFound confirms an unknown id is a clean 404.
func TestPortalRegistration_GetUnknownNotFound(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)

	status, body, err := buyer.Do("GET", portalRegSessionsPath+"/porgse_"+uuid.New().String(), nil, "")
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// TestPortalRegistration_OwnershipEnforced confirms a buyer cannot read or mutate
// another buyer's session. An existing-but-unowned session is a 403 (an unknown id
// is a 404 — see TestPortalRegistration_GetUnknownNotFound); the service deliberately
// separates "not yours" from "doesn't exist".
func TestPortalRegistration_OwnershipEnforced(t *testing.T) {
	t.Parallel()
	owner := newPortalBuyerClient(t)
	intruder := newPortalBuyerClient(t)

	sessionID := jsonField(startPortalRegistration(t, owner), "id")
	sessionPath := portalRegSessionsPath + "/" + sessionID

	status, body, err := intruder.Do("GET", sessionPath, nil, "")
	require.NoError(t, err)
	requireStatus(t, 403, status, body)

	status, body, err = intruder.Patch(sessionPath, map[string]any{"step": "billing_address"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)

	status, body, err = intruder.Post(sessionPath+"/actions/complete", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)

	status, body, err = intruder.Post(sessionPath+"/actions/abandon", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 403, status, body)
}

// TestPortalRegistration_Abandon confirms abandoning stops the session from being
// resumed — a subsequent start yields a brand-new session id.
func TestPortalRegistration_Abandon(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)
	firstID := jsonField(startPortalRegistration(t, buyer), "id")

	status, body, err := buyer.Post(portalRegSessionsPath+"/"+firstID+"/actions/abandon", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	abandoned := parseJSON(body)
	assert.Equal(t, firstID, jsonField(abandoned, "id"))
	assert.NotEmpty(t, jsonField(abandoned, "abandoned_at"), "abandoned_at should be set")

	// Start again — the abandoned session is not resumed.
	secondID := jsonField(startPortalRegistration(t, buyer), "id")
	assert.NotEqual(t, firstID, secondID, "an abandoned session must not be resumed")
}

// TestPortalRegistration_CompleteAfterAbandonRejected confirms an abandoned session
// cannot then be completed.
func TestPortalRegistration_CompleteAfterAbandonRejected(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)
	sessionID := jsonField(startPortalRegistration(t, buyer), "id")

	status, body, err := buyer.Post(portalRegSessionsPath+"/"+sessionID+"/actions/abandon", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = buyer.Post(portalRegSessionsPath+"/"+sessionID+"/actions/complete", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestPortalRegistration_CreateValidation covers request validation on start.
func TestPortalRegistration_CreateValidation(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)

	t.Run("missing seller_slug", func(t *testing.T) {
		t.Parallel()
		status, body, err := buyer.Post(portalRegSessionsPath, map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("unknown seller_slug", func(t *testing.T) {
		t.Parallel()
		status, body, err := buyer.Post(portalRegSessionsPath, map[string]any{
			"seller_slug": "no-such-seller-" + uuid.New().String()[:8],
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})
}

// TestPortalRegistration_CompleteNewCustomerJourney walks the full new-customer
// flow — start, advance through both data steps saving the full form, complete —
// and confirms completion registers the buyer (completed_at set, idempotent replay).
func TestPortalRegistration_CompleteNewCustomerJourney(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)
	sessionID := jsonField(startPortalRegistration(t, buyer), "id")
	sessionPath := portalRegSessionsPath + "/" + sessionID

	data := fullNewCustomerData("E2E New Customer " + uuid.New().String()[:8])

	// Advance through billing then contact, persisting the full snapshot each step
	// (mirrors the wizard, which sends cumulative data and replaces on each save).
	status, body, err := buyer.Patch(sessionPath, map[string]any{
		"step":                 "billing_address",
		"session_data":         data,
		"is_existing_customer": false,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = buyer.Patch(sessionPath, map[string]any{
		"step":                 "contact",
		"session_data":         data,
		"is_existing_customer": false,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Complete — registers the buyer as a new customer of the seller.
	status, body, err = buyer.Post(sessionPath+"/actions/complete", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	completed := parseJSON(body)
	assert.Equal(t, sessionID, jsonField(completed, "id"))
	assert.NotEmpty(t, jsonField(completed, "completed_at"), "completed_at should be set after completion")

	// Completion is idempotent — replaying returns the completed session, not an error.
	status, body, err = buyer.Post(sessionPath+"/actions/complete", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "completed_at"))

	// A completed session can no longer be updated.
	status, body, err = buyer.Patch(sessionPath, map[string]any{"step": "contact"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestPortalRegistration_CompleteExistingCustomerJourney confirms the existing-customer
// path: the buyer links to a seed customer by number and completion succeeds.
func TestPortalRegistration_CompleteExistingCustomerJourney(t *testing.T) {
	t.Parallel()
	buyer := newPortalBuyerClient(t)
	sessionID := jsonField(startPortalRegistration(t, buyer), "id")
	sessionPath := portalRegSessionsPath + "/" + sessionID

	// Existing-customer registration only needs the customer number; SeedCustomerNumber
	// belongs to the seed customer of the seed seller.
	status, body, err := buyer.Patch(sessionPath, map[string]any{
		"step":                 "contact",
		"is_existing_customer": true,
		"session_data": map[string]any{
			"customer_number": SeedCustomerNumber,
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, true, parseJSON(body)["is_existing_customer"])

	status, body, err = buyer.Post(sessionPath+"/actions/complete", nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.NotEmpty(t, jsonField(parseJSON(body), "completed_at"),
		"linking an existing customer should complete the session")
}
