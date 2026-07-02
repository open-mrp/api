//go:build e2e

package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tenancy endpoints gained a CheckHasUserActor() guard so an API-key actor (which
// has no user actor) is rejected with a clean 4xx instead of faulting deeper with
// a 5xx. These assert the guard fires for the API key and that a real logged-in
// user is allowed through.

const tenancyPath = "/v1/identity/me/tenancy"

func TestTenancy_SwitchAccountRejectsAPIKeyActor(t *testing.T) {
	t.Parallel()
	// apiClient authenticates with an API key — no user actor.
	status, body, err := apiClient.Put(tenancyPath, map[string]any{"account_id": SeedCustomerAccountID})
	require.NoError(t, err)
	assert.Less(t, status, 500, "switch-account with an API-key actor must be a clean 4xx, not 5xx: %s", string(body))
	assert.Equal(t, http.StatusForbidden, status, "API-key actor lacks a user actor → 403: %s", string(body))
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

func TestTenancy_ListCustomerAccountsRejectsAPIKeyActor(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Do(http.MethodGet, tenancyPath+"/customer-accounts/"+SeedAccountID, nil, "")
	require.NoError(t, err)
	assert.Less(t, status, 500, "list-customer-accounts with an API-key actor must be a clean 4xx, not 5xx: %s", string(body))
	assert.Equal(t, http.StatusForbidden, status, "API-key actor lacks a user actor → 403: %s", string(body))
	requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
}

func TestTenancy_UserActorCanReadTenancy(t *testing.T) {
	t.Parallel()
	// A logged-in user has a user actor and is allowed through the guard.
	user := loginAsUser(t, seedUserEmail, seedUserPassword, SeedAccountID)
	resp, err := user.GetFull(tenancyPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, resp.StatusCode, resp.Body)
}
