//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Coverage top-up for sales_priorities (/v1/sales/priorities).
//
// Priority is a read-only, platform-seeded lookup resource (low/normal/high) —
// no create/update/delete/action routes exist. Existing coverage in
// list_priorities_test.go and crud_partial_includes_test.go is strong; this
// file closes the specific named gaps:
//   1. retrieve-by-code (low/normal/high) dual-resolution behavior
//   2. all 7 response fields (incl. owner via ?include=owner) in one response
//   3. dedicated limit/cursor/q boundary validation for this path
//   4. unknown include value behavior (confirmed via curl: 400 parameter_invalid)
// ──────────────────────────────────────────────

// --- Retrieve by code (dual ID-or-code resolution) ---

func TestCovSalesPriorities_RetrieveByCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		code string
		name string
		id   string
	}{
		{"low", "Low", "pi_01seedlow000000000000"},
		{"normal", "Normal", SeedPriorityID},
		{"high", "High", "pi_01seedhigh00000000000"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.code, func(t *testing.T) {
			t.Parallel()

			status, body, err := apiClient.GetListRaw(prioritiesPath+"/"+tc.code, nil)
			require.NoError(t, err)
			requireStatus(t, 200, status, body)

			got := parseJSON(body)
			require.NotNil(t, got)
			assertObjectField(t, got, "priority")
			assert.Equal(t, tc.code, jsonField(got, "code"), "code should match requested code segment")
			assert.Equal(t, tc.name, jsonField(got, "name"))
			assert.Equal(t, tc.id, jsonField(got, "id"), "retrieve-by-code should resolve to the same id as the known seeded priority")

			// Cross-check: retrieving the same priority by its ID returns an identical id.
			statusByID, bodyByID, err := apiClient.GetListRaw(prioritiesPath+"/"+tc.id, nil)
			require.NoError(t, err)
			requireStatus(t, 200, statusByID, bodyByID)
			gotByID := parseJSON(bodyByID)
			assert.Equal(t, jsonField(got, "id"), jsonField(gotByID, "id"), "retrieve-by-code and retrieve-by-id should resolve to the same underlying priority")
			assert.Equal(t, jsonField(got, "code"), jsonField(gotByID, "code"))
		})
	}
}

// --- All fields (including expandable owner) in a single response, across every seeded priority ---

func TestCovSalesPriorities_AllFieldsWithOwnerInclude(t *testing.T) {
	t.Parallel()

	list, _, err := apiClient.GetList(prioritiesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 3, "expected at least 3 seeded priorities (low, normal, high)")

	for _, item := range list.Data {
		id := DataItemField(item, "id")
		require.NotEmpty(t, id)

		status, body, err := apiClient.GetListRaw(prioritiesPath+"/"+id, url.Values{"include": {"owner"}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		got := parseJSON(body)
		require.NotNil(t, got)

		assertIDFormat(t, jsonField(got, "id"), "pi")
		assertObjectField(t, got, "priority")
		assert.Contains(t, []string{"low", "normal", "high"}, jsonField(got, "code"), "code must be one of the seeded enum values")
		assert.NotEmpty(t, jsonField(got, "name"), "name must be populated")
		assertValidTimestamp(t, jsonField(got, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(got, "updated_at"), "updated_at")

		owner := jsonObject(got, "owner")
		require.NotNil(t, owner, "owner should be populated with ?include=owner for priority %s", id)
		assertObjectField(t, owner, "owner")
		assert.Equal(t, "system", jsonField(owner, "type"), "priorities are platform-owned; owner.type should always be 'system'")
		assertNilField(t, owner, "account")
	}
}

// --- List: dedicated limit boundary validation for this path ---

func TestCovSalesPriorities_List_InvalidLimitZero(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath, url.Values{"limit": {"0"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	got := parseJSON(body)
	errObj := jsonObject(got, "error")
	require.NotNil(t, errObj, "400 response should contain an error object")
	assert.Equal(t, "invalid_format", jsonField(errObj, "code"))
}

func TestCovSalesPriorities_List_InvalidLimitNegative(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath, url.Values{"limit": {"-1"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	got := parseJSON(body)
	errObj := jsonObject(got, "error")
	require.NotNil(t, errObj, "400 response should contain an error object")
	assert.Equal(t, "invalid_format", jsonField(errObj, "code"))
}

func TestCovSalesPriorities_List_InvalidLimitTooLarge(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath, url.Values{"limit": {"1001"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	got := parseJSON(body)
	errObj := jsonObject(got, "error")
	require.NotNil(t, errObj, "400 response should contain an error object")
	assert.Equal(t, "invalid_format", jsonField(errObj, "code"))
}

func TestCovSalesPriorities_List_LimitMaxAllowed(t *testing.T) {
	t.Parallel()
	// limit=1000 is the documented max and must be accepted (not rejected).
	list, _, err := apiClient.GetList(prioritiesPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 3, "should still return all seeded priorities under the max limit")
}

// --- List: dedicated invalid cursor validation for this path ---

func TestCovSalesPriorities_List_InvalidCursor(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath, url.Values{"cursor": {"not_a_real_cursor_value"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	got := parseJSON(body)
	errObj := jsonObject(got, "error")
	require.NotNil(t, errObj, "400 response should contain an error object")
	assert.Equal(t, "validation_failed", jsonField(errObj, "code"))
}

// --- List: dedicated oversized q validation for this path ---

func TestCovSalesPriorities_List_QueryTooLong(t *testing.T) {
	t.Parallel()
	longQuery := strings.Repeat("a", 501)
	status, body, err := apiClient.GetListRaw(prioritiesPath, url.Values{"q": {longQuery}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	got := parseJSON(body)
	errObj := jsonObject(got, "error")
	require.NotNil(t, errObj, "400 response should contain an error object")
	assert.Equal(t, "invalid_format", jsonField(errObj, "code"))
}

func TestCovSalesPriorities_List_QueryAtMaxLengthAllowed(t *testing.T) {
	t.Parallel()
	// q at exactly the 500-char boundary must be accepted, not rejected.
	maxQuery := strings.Repeat("a", 500)
	status, body, err := apiClient.GetListRaw(prioritiesPath, url.Values{"q": {maxQuery}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// --- Unknown include value behavior (list + retrieve) ---

func TestCovSalesPriorities_List_UnknownIncludeValueRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	got := parseJSON(body)
	errObj := jsonObject(got, "error")
	require.NotNil(t, errObj, "400 response should contain an error object")
	assert.Equal(t, "parameter_invalid", jsonField(errObj, "code"))
}

func TestCovSalesPriorities_Retrieve_UnknownIncludeValueRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(prioritiesPath+"/"+SeedPriorityID, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)

	got := parseJSON(body)
	errObj := jsonObject(got, "error")
	require.NotNil(t, errObj, "400 response should contain an error object")
	assert.Equal(t, "parameter_invalid", jsonField(errObj, "code"))
}
