//go:build e2e

package api_test

import (
	"net/url"
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
	list, _, err := apiClient.WithAPIVersion(preview9Version).GetList(accountStatusesPath, nil)
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
	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(accountStatusesPath+"/"+SeedAccountStatusID, nil)
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
	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(accountStatusesPath+"/"+SeedAccountStatusID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assertNilField(t, got, "owner")
}

func TestCovSalesAccountStatuses_RetrieveIncludeOwner(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(accountStatusesPath+"/"+SeedAccountStatusID, url.Values{"include": {"owner"}})
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
	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(accountStatusesPath+"/"+SeedAccountStatusCode, url.Values{"include": {"owner"}})
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
	list, _, err := apiClient.WithAPIVersion(preview9Version).GetList(accountStatusesPath, url.Values{"include": {"owner"}})
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

// ──────────────────────────────────────────────
// validation — dedicated per-path limit/cursor/q checks
// ──────────────────────────────────────────────

func TestCovSalesAccountStatuses_LimitNonNumericRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.WithAPIVersion(preview9Version).GetListRaw(accountStatusesPath, url.Values{"limit": {"abc"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}
