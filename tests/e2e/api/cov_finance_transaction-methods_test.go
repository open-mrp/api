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
// /v1/finance/transaction-methods — read-only, hardcoded static reference
// list (5 rows: cash, check, credit_card, gift_card, ach). There is no
// create/update/delete/get-by-id/action endpoint, and the TransactionMethod
// resource has no expandable sub-fields and no timestamps, so the
// crudLifecycle / omittedFields / expandable / idempotency / actions
// categories are structurally n/a for this group (see
// TASK-finance_transaction-methods.md). Coverage here focuses on: response
// shape (all 4 json fields), list behavior (basic/search/limit), and the
// endpoint's hand-rolled validation (cursor rejection, limit bounds, q max
// length).
//
// Known coverage gap: this endpoint requires authentication
// (identity.CheckIsAuthenticated()), but the harness has no ready-made
// unauthenticated-client fixture, so a 401/403 case is not exercised here
// (see TASK-finance_transaction-methods.md "Failure-mode checklist").
// ──────────────────────────────────────────────

const covFinanceTransactionMethodsPath = "/v1/finance/transaction-methods"

// covFinanceTransactionMethodsCodes is the closed, fully-covered set of
// constants.TransactionMethod enum values — the static list returns exactly
// one row per code, so this doubles as the expected row count.
var covFinanceTransactionMethodsCodes = map[string]bool{
	"cash":        true,
	"check":       true,
	"credit_card": true,
	"gift_card":   true,
	"ach":         true,
}

func TestCovFinanceTransactionMethods_List(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionMethodsPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)
	require.Len(t, list.Data, 5, "static transaction-methods list should always have exactly 5 rows")

	seenCodes := map[string]bool{}
	for _, raw := range list.Data {
		row := parseJSON(raw)
		id := jsonField(row, "id")
		assertIDFormat(t, id, "txmd")
		assertObjectField(t, row, "transaction_method")
		assert.NotEmpty(t, jsonField(row, "name"))

		code := jsonField(row, "code")
		assert.True(t, covFinanceTransactionMethodsCodes[code], "unexpected code %q", code)
		seenCodes[code] = true
	}
	assert.Len(t, seenCodes, 5, "every known code should appear exactly once across the static list")
}

func TestCovFinanceTransactionMethods_ResponseShape(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionMethodsPath, url.Values{"q": {"Cash"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "txmd")
	assertObjectField(t, row, "transaction_method")
	assert.Equal(t, "Cash", jsonField(row, "name"))
	assert.Equal(t, "cash", jsonField(row, "code"))
}

func TestCovFinanceTransactionMethods_ListSearch(t *testing.T) {
	t.Parallel()

	// Lowercase query proves the Name match is case-insensitive.
	list, status, err := apiClient.GetList(covFinanceTransactionMethodsPath, url.Values{"q": {"cash"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)
	row := parseJSON(list.Data[0])
	assert.Equal(t, "Cash", jsonField(row, "name"))
	assert.Equal(t, "cash", jsonField(row, "code"))

	list, status, err = apiClient.GetList(covFinanceTransactionMethodsPath, url.Values{"q": {"Credit"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)
	row = parseJSON(list.Data[0])
	assert.Equal(t, "Credit Card", jsonField(row, "name"))
	assert.Equal(t, "credit_card", jsonField(row, "code"))
}

// q matches the machine-readable code as well as the display name, matching the transaction-types list.
func TestCovFinanceTransactionMethods_SearchMatchesCode(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionMethodsPath, url.Values{"q": {"credit_card"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1, "q=credit_card should match the credit_card method")
	assert.Equal(t, "credit_card", jsonField(parseJSON(list.Data[0]), "code"))
}

func TestCovFinanceTransactionMethods_ListSearchNoResults(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionMethodsPath, url.Values{"q": {"zzzznotamethod99999"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

func TestCovFinanceTransactionMethods_ListLimit(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionMethodsPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)

	// PageInfo is always the zero value on this endpoint — there is no real
	// pagination, so a client cannot distinguish "more rows exist" from
	// "that's everything" purely from limit truncation.
	assert.False(t, list.PageInfo.HasNextPage)
	assert.Nil(t, list.PageInfo.NextPageURL)
}

// TestCovFinanceTransactionMethods_LimitAppliedAfterSearch pins down that
// limit truncation is applied to the already-`q`-filtered slice, not the
// full static list: q=c matches Cash/Check/Credit Card
// (in that static list order), and limit=1 should truncate to just the
// first match (Cash), not an arbitrary/unfiltered row.
func TestCovFinanceTransactionMethods_LimitAppliedAfterSearch(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionMethodsPath, url.Values{"q": {"c"}, "limit": {"1"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assert.Equal(t, "Cash", jsonField(row, "name"))
	assert.True(t, strings.Contains(strings.ToLower(jsonField(row, "name")), "c"))
}

// TestCovFinanceTransactionMethods_InvalidCursor is the single most
// important net-new assertion for this group: the generic invalid-cursor
// sweep (pagination_errors_test.go's TestListEndpoints_InvalidCursor)
// structurally excludes this path via excludedPaginationPaths in
// spec_test.go, so without this test the hand-rolled "Invalid pagination
// cursor." check (this list is NOT actually cursor-paginated) has zero
// coverage.
func TestCovFinanceTransactionMethods_InvalidCursor(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covFinanceTransactionMethodsPath, url.Values{"cursor": {"anything"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Equal(t, "Invalid pagination cursor.", errObj["message"])
}

func TestCovFinanceTransactionMethods_InvalidLimit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		limit string
	}{
		{"zero", "0"},
		{"negative", "-1"},
		{"tooLarge", "1001"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(covFinanceTransactionMethodsPath, url.Values{"limit": {tc.limit}})
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
			assertErrorParam(t, errObj, "limit")
		})
	}
}

func TestCovFinanceTransactionMethods_QueryTooLong(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covFinanceTransactionMethodsPath, url.Values{"q": {strings.Repeat("a", 501)}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "q")
}
