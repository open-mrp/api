//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	signingPinMePath   = "/v1/identity/users/me/signing-pin"
	signingPinStartPin = "2580"
	signingPinNextPin  = "8642"
	signingPinWrongPin = "1357"
)

func setTemporarySigningPinPath(accountUserID string) string {
	return accountUsersPath + "/" + accountUserID + "/actions/set-temporary-signing-pin"
}

func clearSigningPinPath(accountUserID string) string {
	return accountUsersPath + "/" + accountUserID + "/actions/clear-signing-pin"
}

func changeSigningPinPath(accountUserID string) string {
	return accountUsersPath + "/" + accountUserID + "/actions/change-signing-pin"
}

// createPinOperator creates an operator who never signs in, the person a temporary PIN is for.
func createPinOperator(t *testing.T) string {
	t.Helper()
	status, body, err := apiClient.Post(accountUsersPath, map[string]any{
		"username": uniqueName("e2e-pin-op"),
		"password": "OperatorPass123!",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, id)
	t.Cleanup(func() { removeAccountUser(id) })
	return id
}

func requireAPIError(t *testing.T, wantStatus int, wantCode string, status int, body []byte) map[string]any {
	t.Helper()
	requireStatus(t, wantStatus, status, body)
	apiErr := jsonObject(parseJSON(body), "error")
	require.NotNil(t, apiErr, "error body expected: %s", string(body))
	assert.Equal(t, wantCode, jsonField(apiErr, "code"), "body: %s", string(body))
	return apiErr
}

func changeSigningPin(t *testing.T, c *Client, accountUserID, currentPin, newPin string) (int, []byte) {
	t.Helper()
	status, body, err := c.Post(changeSigningPinPath(accountUserID), map[string]any{
		"current_pin": currentPin,
		"new_pin":     newPin,
	}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

// --- Set own PIN ---

func TestSigningPins_SetOwnPin(t *testing.T) {
	t.Parallel()
	session := loginAsSeedUser(t)

	status, body, err := session.Put(signingPinMePath, map[string]any{"pin": signingPinStartPin})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.JSONEq(t, `{}`, string(body))

	status, body, err = session.Put(signingPinMePath, map[string]any{"pin": signingPinNextPin})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

func TestSigningPins_SetOwnPinRefusesAPIKey(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Put(signingPinMePath, map[string]any{"pin": signingPinStartPin})
	require.NoError(t, err)
	requireAPIError(t, 403, "insufficient_permissions", status, body)
}

func TestSigningPins_SetOwnPinValidation(t *testing.T) {
	t.Parallel()
	session := loginAsSeedUser(t)

	for _, pin := range []string{"123", "123456789", "12a4", "0000", "1234", "98765"} {
		status, body, err := session.Put(signingPinMePath, map[string]any{"pin": pin})
		require.NoError(t, err)
		apiErr := requireAPIError(t, 400, "validation_failed", status, body)
		assert.Equal(t, "pin", jsonField(apiErr, "param"), "pin %q", pin)
	}

	status, _, err := session.Put(signingPinMePath, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "pin is required")
}

// --- Temporary PIN and change ---

func TestSigningPins_TemporaryPinChangedAtStation(t *testing.T) {
	t.Parallel()
	operatorID := createPinOperator(t)
	station := roleScopedClient(t, SeedScannerRoleID)

	status, body, err := apiClient.Post(setTemporarySigningPinPath(operatorID), map[string]any{"pin": signingPinStartPin}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.JSONEq(t, `{}`, string(body))

	status, body = changeSigningPin(t, station, operatorID, signingPinWrongPin, signingPinNextPin)
	apiErr := requireAPIError(t, 403, "signing_pin_invalid", status, body)
	assert.Equal(t, "pin", jsonField(apiErr, "param"))

	status, body = changeSigningPin(t, station, operatorID, signingPinStartPin, signingPinStartPin)
	apiErr = requireAPIError(t, 400, "validation_failed", status, body)
	assert.Equal(t, "new_pin", jsonField(apiErr, "param"))

	status, body = changeSigningPin(t, station, operatorID, signingPinStartPin, "4321")
	apiErr = requireAPIError(t, 400, "validation_failed", status, body)
	assert.Equal(t, "new_pin", jsonField(apiErr, "param"))

	status, body = changeSigningPin(t, station, operatorID, signingPinStartPin, signingPinNextPin)
	requireStatus(t, 200, status, body)

	status, body = changeSigningPin(t, station, operatorID, signingPinStartPin, "9513")
	requireAPIError(t, 403, "signing_pin_invalid", status, body)

	status, body = changeSigningPin(t, station, operatorID, signingPinNextPin, signingPinStartPin)
	requireStatus(t, 200, status, body)
}

func TestSigningPins_ChangeIsIdempotent(t *testing.T) {
	t.Parallel()
	operatorID := createPinOperator(t)

	status, body, err := apiClient.Post(setTemporarySigningPinPath(operatorID), map[string]any{"pin": signingPinStartPin}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	key := newIdempotencyKey()
	req := map[string]any{"current_pin": signingPinStartPin, "new_pin": signingPinNextPin}
	status, body, err = apiClient.Post(changeSigningPinPath(operatorID), req, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Post(changeSigningPinPath(operatorID), req, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body = changeSigningPin(t, apiClient, operatorID, signingPinNextPin, signingPinStartPin)
	requireStatus(t, 200, status, body)
}

func TestSigningPins_FiveWrongPinsLock(t *testing.T) {
	t.Parallel()
	operatorID := createPinOperator(t)

	status, body, err := apiClient.Post(setTemporarySigningPinPath(operatorID), map[string]any{"pin": signingPinStartPin}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for range 4 {
		status, body = changeSigningPin(t, apiClient, operatorID, signingPinWrongPin, signingPinNextPin)
		requireAPIError(t, 403, "signing_pin_invalid", status, body)
	}
	status, body = changeSigningPin(t, apiClient, operatorID, signingPinWrongPin, signingPinNextPin)
	requireAPIError(t, 429, "signing_pin_locked", status, body)

	status, body = changeSigningPin(t, apiClient, operatorID, signingPinStartPin, signingPinNextPin)
	requireAPIError(t, 429, "signing_pin_locked", status, body)

	status, body, err = apiClient.Post(setTemporarySigningPinPath(operatorID), map[string]any{"pin": signingPinStartPin}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body = changeSigningPin(t, apiClient, operatorID, signingPinStartPin, signingPinNextPin)
	requireStatus(t, 200, status, body)
}

func TestSigningPins_SetTemporaryRequiresTeamUpdate(t *testing.T) {
	t.Parallel()
	operatorID := createPinOperator(t)
	station := roleScopedClient(t, SeedScannerRoleID)

	status, body, err := station.Post(setTemporarySigningPinPath(operatorID), map[string]any{"pin": signingPinStartPin}, newIdempotencyKey())
	require.NoError(t, err)
	requireAPIError(t, 403, "insufficient_permissions", status, body)

	status, body, err = station.Post(clearSigningPinPath(operatorID), nil, newIdempotencyKey())
	require.NoError(t, err)
	requireAPIError(t, 403, "insufficient_permissions", status, body)
}

func TestSigningPins_UnknownAccountUser(t *testing.T) {
	t.Parallel()
	missing := "acus_e2emissing0000"

	status, body, err := apiClient.Post(setTemporarySigningPinPath(missing), map[string]any{"pin": signingPinStartPin}, newIdempotencyKey())
	require.NoError(t, err)
	requireAPIError(t, 404, "resource_not_found", status, body)

	status, body, err = apiClient.Post(clearSigningPinPath(missing), nil, newIdempotencyKey())
	require.NoError(t, err)
	requireAPIError(t, 404, "resource_not_found", status, body)

	status, body = changeSigningPin(t, apiClient, missing, signingPinStartPin, signingPinNextPin)
	requireAPIError(t, 404, "resource_not_found", status, body)
}

func TestSigningPins_OtherAccountsUserIsNotFound(t *testing.T) {
	t.Parallel()
	operatorID := createPinOperator(t)
	other := NewClient(envOr("E2E_BASE_URL", defaultBaseURL), SeedTenantBAPIKey, SeedTenantBAccountID)

	status, body, err := other.Post(setTemporarySigningPinPath(operatorID), map[string]any{"pin": signingPinStartPin}, newIdempotencyKey())
	require.NoError(t, err)
	requireAPIError(t, 404, "resource_not_found", status, body)
}

// --- Clear ---

func TestSigningPins_ClearForcesANewPin(t *testing.T) {
	t.Parallel()
	operatorID := createPinOperator(t)

	status, body, err := apiClient.Post(setTemporarySigningPinPath(operatorID), map[string]any{"pin": signingPinStartPin}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Post(clearSigningPinPath(operatorID), nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.JSONEq(t, `{}`, string(body))

	status, body = changeSigningPin(t, apiClient, operatorID, signingPinStartPin, signingPinNextPin)
	requireAPIError(t, 409, "signing_pin_not_set", status, body)

	status, body, err = apiClient.Post(clearSigningPinPath(operatorID), nil, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}
