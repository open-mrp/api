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
// /v1/finance/transaction-types — read-only, hardcoded static reference list
// (4 rows: payment, credit_memo, adjustment, rebate). There is no
// create/update/delete/get-by-id/action endpoint, and the TransactionType
// resource has no expandable sub-fields and no timestamps, so the
// crudLifecycle / omittedFields / expandable / idempotency / actions
// categories are structurally n/a for this group (see
// TASK-finance_transaction-types.md). Coverage here focuses on: response
// shape (all 4 json fields, name<->code pairing), list behavior
// (basic/search/limit), and the endpoint's hand-rolled validation (cursor
// rejection, limit bounds incl. non-numeric, q max length, unknown query
// params including the undeclared `include`), plus the auth requirement.
// ──────────────────────────────────────────────

const covFinanceTransactionTypesPath = "/v1/finance/transaction-types"

const (
	covFinanceTransactionTypesPaymentID    = "txtp_01seedpayment000000"
	covFinanceTransactionTypesCreditMemoID = "txtp_01seedcreditmemo000"
	covFinanceTransactionTypesAdjustID     = "txtp_01seedadjustment000"
	covFinanceTransactionTypesRebateID     = "txtp_01seedrebate0000000"
)

// covFinanceTransactionTypesExpected is the closed, fully-covered set of
// (id -> name/code) pairs for the static list — this doubles as the
// expected row count (4) and pins the name<->code pairing per row.
var covFinanceTransactionTypesExpected = map[string]struct {
	name string
	code string
}{
	covFinanceTransactionTypesPaymentID:    {"Payment", "payment"},
	covFinanceTransactionTypesCreditMemoID: {"Credit Memo", "credit_memo"},
	covFinanceTransactionTypesAdjustID:     {"Adjustment", "adjustment"},
	covFinanceTransactionTypesRebateID:     {"Rebate", "rebate"},
}

func TestCovFinanceTransactionTypes_List(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionTypesPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assert.Equal(t, "list", list.Object)
	require.Len(t, list.Data, 4, "static transaction-types list should always have exactly 4 rows")

	seen := map[string]bool{}
	for _, raw := range list.Data {
		row := parseJSON(raw)
		id := jsonField(row, "id")
		assertIDFormat(t, id, "txtp")
		assertObjectField(t, row, "transaction_type")

		expected, ok := covFinanceTransactionTypesExpected[id]
		require.True(t, ok, "unexpected transaction type id %q", id)
		assert.Equal(t, expected.name, jsonField(row, "name"), "name mismatch for id %q", id)
		assert.Equal(t, expected.code, jsonField(row, "code"), "code mismatch for id %q", id)
		seen[id] = true
	}
	assert.Len(t, seen, 4, "every known transaction type id should appear exactly once")
}

func TestCovFinanceTransactionTypes_ResponseFields(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionTypesPath, url.Values{"q": {"Payment"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assert.Equal(t, covFinanceTransactionTypesPaymentID, jsonField(row, "id"))
	assertIDFormat(t, jsonField(row, "id"), "txtp")
	assertObjectField(t, row, "transaction_type")
	assert.Equal(t, "Payment", jsonField(row, "name"))
	assert.Equal(t, "payment", jsonField(row, "code"))
}

func TestCovFinanceTransactionTypes_SearchByName(t *testing.T) {
	t.Parallel()

	// Uppercase query.
	list, status, err := apiClient.GetList(covFinanceTransactionTypesPath, url.Values{"q": {"Payment"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)
	row := parseJSON(list.Data[0])
	assert.Equal(t, "Payment", jsonField(row, "name"))
	assert.Equal(t, "payment", jsonField(row, "code"))

	// Lowercase query proves the Name match is case-insensitive.
	list, status, err = apiClient.GetList(covFinanceTransactionTypesPath, url.Values{"q": {"payment"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)
	row = parseJSON(list.Data[0])
	assert.Equal(t, "Payment", jsonField(row, "name"))
	assert.Equal(t, "payment", jsonField(row, "code"))

	// Substring match.
	list, status, err = apiClient.GetList(covFinanceTransactionTypesPath, url.Values{"q": {"Credit"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 1)
	row = parseJSON(list.Data[0])
	assert.Equal(t, "Credit Memo", jsonField(row, "name"))
	assert.Equal(t, "credit_memo", jsonField(row, "code"))
}

func TestCovFinanceTransactionTypes_SearchNoResults(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionTypesPath, url.Values{"q": {"zzzznotarealtype99999"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data)
}

// TestCovFinanceTransactionTypes_SearchByCodeReturnsNothing locks in
// prodBugSuspect #3 from the task file: q matches Name via
// strings.Contains only — searching by the exact Code value ("credit_memo")
// returns zero rows, even though the same value's display Name substring
// ("Credit") matches. Not asking for a fix; this test exists so a future
// refactor that changes q to also match Code doesn't do so silently.
func TestCovFinanceTransactionTypes_SearchByCodeReturnsNothing(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionTypesPath, url.Values{"q": {"credit_memo"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	assertEmptyListData(t, list.Data, "q=credit_memo (a Code value, not a Name substring) should match nothing")
}

// TestCovFinanceTransactionTypes_LimitTruncates pins down prodBugSuspect #1:
// when limit < total rows, the response is truncated but PageInfo stays the
// zero value (has_next_page stays false, next_page_url stays nil) even
// though more rows exist. This locks in the current (possibly buggy)
// behavior as an explicit regression trip-wire.
func TestCovFinanceTransactionTypes_LimitTruncates(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covFinanceTransactionTypesPath, url.Values{"limit": {"2"}})
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 2)

	assert.False(t, list.PageInfo.HasNextPage, "prodBugSuspect: has_next_page stays false even though limit truncated the static 4-row list")
	assert.Nil(t, list.PageInfo.NextPageURL)
	assert.False(t, list.PageInfo.HasPrevPage)
	assert.Nil(t, list.PageInfo.PreviousPageURL)
}

// TestCovFinanceTransactionTypes_CursorRejected asserts the endpoint's
// hand-rolled "not actually cursor-paginated" validation (prodBugSuspect
// #2): ANY non-empty cursor value is rejected, regardless of content. This
// path is excluded from the generic cursor-pagination sweep
// (excludedPaginationPaths in spec_test.go), so without this test that
// behavior has zero coverage.
func TestCovFinanceTransactionTypes_CursorRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covFinanceTransactionTypesPath, url.Values{"cursor": {"anything"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assert.Equal(t, "Invalid pagination cursor.", errObj["message"])
}

func TestCovFinanceTransactionTypes_LimitValidation(t *testing.T) {
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
			status, body, err := apiClient.GetListRaw(covFinanceTransactionTypesPath, url.Values{"limit": {tc.limit}})
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
			errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
			assertErrorParam(t, errObj, "limit")
		})
	}
}

// TestCovFinanceTransactionTypes_LimitNonNumeric asserts a non-numeric limit
// value is rejected with the raw query-string-parsing error (distinct from
// the bound-check "invalid_format" errors above, since it never reaches
// struct validation).
func TestCovFinanceTransactionTypes_LimitNonNumeric(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covFinanceTransactionTypesPath, url.Values{"limit": {"abc"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "limit")
}

func TestCovFinanceTransactionTypes_QueryTooLong(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covFinanceTransactionTypesPath, url.Values{"q": {strings.Repeat("a", 501)}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "q")
}

func TestCovFinanceTransactionTypes_UnknownQueryParamRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covFinanceTransactionTypesPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, covFinanceTransactionTypesPath, status, body)
}

// TestCovFinanceTransactionTypes_IncludeRejected asserts `include` is
// rejected as an unknown query param: ListTransactionTypesRequest embeds
// only PaginationRequest (no include field), and TransactionType has zero
// expandable fields, so the framework should treat `include` the same as
// any other undeclared param.
func TestCovFinanceTransactionTypes_IncludeRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covFinanceTransactionTypesPath, url.Values{"include": {"anything"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "include")
}

// TestCovFinanceTransactionTypes_NoAccountScoping confirms the static list
// is not filtered by account: tenant B (a different account) sees the same
// 4 global rows as the primary test account.
func TestCovFinanceTransactionTypes_NoAccountScoping(t *testing.T) {
	t.Parallel()

	tenantBClient := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)
	list, status, err := tenantBClient.GetList(covFinanceTransactionTypesPath, nil)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.Len(t, list.Data, 4, "static transaction-types list should be identical across accounts")

	seen := map[string]bool{}
	for _, raw := range list.Data {
		row := parseJSON(raw)
		id := jsonField(row, "id")
		expected, ok := covFinanceTransactionTypesExpected[id]
		require.True(t, ok, "unexpected transaction type id %q for tenant B", id)
		assert.Equal(t, expected.name, jsonField(row, "name"))
		assert.Equal(t, expected.code, jsonField(row, "code"))
		seen[id] = true
	}
	assert.Len(t, seen, 4)
}

// TestCovFinanceTransactionTypes_RequiresAuth asserts the endpoint requires
// authentication: a request with an empty bearer token (but valid
// Augno-Version/Augno-Account headers, so the request reaches auth
// middleware rather than failing an earlier header check) is rejected with
// 401 invalid_credentials.
func TestCovFinanceTransactionTypes_RequiresAuth(t *testing.T) {
	t.Parallel()

	unauth := apiClient.WithBearerToken("", SeedAccountID)
	status, body, err := unauth.GetListRaw(covFinanceTransactionTypesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 401, status, body)
	requireErrorResponse(t, body, "invalid_credentials", "invalid_request_error")
}
