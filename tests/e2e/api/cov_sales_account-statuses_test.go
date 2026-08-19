//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// covSalesAccountStatusesValidCodes mirrors shared/constants.AccountStatusCode's enum set.
var covSalesAccountStatusesValidCodes = []string{"normal", "preferred", "hold_shipment", "hold_all"}

// ──────────────────────────────────────────────
// responseShape / allFields — stronger assertions (id prefix, code enum)
// ──────────────────────────────────────────────

func TestCovSalesAccountStatuses_IDPrefixAndCodeEnum(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 4)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		id := jsonField(m, "id")
		assertIDFormat(t, id, "acss")
		code := jsonField(m, "code")
		assert.Contains(t, covSalesAccountStatusesValidCodes, code, "code %q should be one of the known enum values", code)
		assertObjectField(t, m, "account_status")
	}
}

func TestCovSalesAccountStatuses_RetrieveIDPrefixAndCodeEnum(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath+"/"+SeedAccountStatusID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertIDFormat(t, jsonField(got, "id"), "acss")
	assert.Contains(t, covSalesAccountStatusesValidCodes, jsonField(got, "code"))
	assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")
}

// ──────────────────────────────────────────────
// expandable — owner include/no-include on the Retrieve endpoint
// (List-level coverage already exists in list_account_statuses_test.go)
// ──────────────────────────────────────────────

func TestCovSalesAccountStatuses_RetrieveOwnerNullWithoutInclude(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath+"/"+SeedAccountStatusID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertNilField(t, got, "owner")
}

func TestCovSalesAccountStatuses_RetrieveIncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath+"/"+SeedAccountStatusID, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assertObjectField(t, owner, "owner")
	assert.Equal(t, "system", jsonField(owner, "type"), "account_status owner should always be type=system")
}

func TestCovSalesAccountStatuses_RetrieveByCodeIncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath+"/"+SeedAccountStatusCode, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Equal(t, SeedAccountStatusID, jsonField(got, "id"))
	owner := jsonObject(got, "owner")
	require.NotNil(t, owner, "owner should be present with ?include=owner")
	assert.Equal(t, "system", jsonField(owner, "type"))
}

func TestCovSalesAccountStatuses_ListIncludeOwnerExactType(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(accountStatusesPath, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		owner := jsonObject(m, "owner")
		require.NotNil(t, owner, "owner should be present with ?include=owner")
		assert.Equal(t, "system", jsonField(owner, "type"), "account_status owner is always system-owned")
	}
}

// ──────────────────────────────────────────────
// include — unknown include value rejected (list + retrieve)
// ──────────────────────────────────────────────

func TestCovSalesAccountStatuses_ListUnknownIncludeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assert.Equal(t, "include[]", errObj["param"])
}

func TestCovSalesAccountStatuses_RetrieveUnknownIncludeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath+"/"+SeedAccountStatusID, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assert.Equal(t, "include[]", errObj["param"])
}

// ──────────────────────────────────────────────
// validation — dedicated per-path limit/cursor/q checks
// ──────────────────────────────────────────────

func TestCovSalesAccountStatuses_LimitZeroRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath, url.Values{"limit": {"0"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovSalesAccountStatuses_LimitNegativeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath, url.Values{"limit": {"-1"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovSalesAccountStatuses_LimitTooLargeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath, url.Values{"limit": {"1001"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovSalesAccountStatuses_LimitNonNumericRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath, url.Values{"limit": {"abc"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestCovSalesAccountStatuses_InvalidCursorRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath, url.Values{"cursor": {"not-a-cursor"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestCovSalesAccountStatuses_QueryTooLongRejected(t *testing.T) {
	t.Parallel()
	longQ := strings.Repeat("a", 501)
	status, body, err := apiClient.GetListRaw(accountStatusesPath, url.Values{"q": {longQ}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovSalesAccountStatuses_UnknownQueryParamRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(accountStatusesPath, url.Values{"foo": {"bar"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assert.Equal(t, "foo", errObj["param"])
}
