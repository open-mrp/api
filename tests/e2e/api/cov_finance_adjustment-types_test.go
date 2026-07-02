//go:build e2e

package api_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// covFinanceAdjustmentTypesPath is the only route under this prefix: a
// read-only, non-account-scoped, global lookup list. There is no get-by-id,
// create, update, delete, or /actions/* route for adjustment types.
const covFinanceAdjustmentTypesPath = "/v1/finance/adjustment-types"

// covFinanceAdjustmentTypesAllCodes are the 6 fixed rows seeded via
// INSERT IGNORE in shared/db/seed/0001_static_types.sql:23-30. Deterministic
// across environments.
var covFinanceAdjustmentTypesAllCodes = []string{
	"discount", "shipping_discrepancy", "short_payment", "write_off", "fee", "refund",
}

// --- List: seeded data + response shape + all fields ---

func TestCovFinanceAdjustmentTypes_ReturnsSeededData(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 6, "should have at least the 6 seeded adjustment types")

	codes := map[string]bool{}
	for _, item := range list.Data {
		codes[DataItemField(item, "code")] = true
	}
	for _, code := range covFinanceAdjustmentTypesAllCodes {
		assert.True(t, codes[code], "seeded adjustment type code %q not found", code)
	}
}

func TestCovFinanceAdjustmentTypes_ResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
	assert.NotNil(t, list.Data)

	for _, item := range list.Data {
		assert.Equal(t, "adjustment_type", DataItemField(item, "object"))
	}
}

func TestCovFinanceAdjustmentTypes_ByID_AllFields(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	found := false
	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		if jsonField(m, "id") != SeedAdjustmentTypeID {
			continue
		}
		found = true
		assertIDFormat(t, jsonField(m, "id"), "ajtp")
		assertObjectField(t, m, "adjustment_type")
		assert.Equal(t, SeedAdjustmentTypeName, jsonField(m, "name"))
		assert.Equal(t, SeedAdjustmentTypeCode, jsonField(m, "code"))
		assertValidTimestamp(t, jsonField(m, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(m, "updated_at"), "updated_at")
	}
	assert.True(t, found, "seeded adjustment type %s should be present in the list", SeedAdjustmentTypeID)
}

func TestCovFinanceAdjustmentTypes_ItemFields(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assert.NotEmpty(t, jsonField(m, "id"), "id should not be empty")
		assert.Equal(t, "adjustment_type", jsonField(m, "object"), "object should be 'adjustment_type'")
		assert.NotEmpty(t, jsonField(m, "name"), "name should not be empty")
		assert.NotEmpty(t, jsonField(m, "code"), "code should not be empty")
		assertValidTimestamp(t, jsonField(m, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(m, "updated_at"), "updated_at")
	}
}

// --- List: pagination ---

func TestCovFinanceAdjustmentTypes_LimitParam(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	assert.Len(t, list.Data, 1, "limit=1 should return exactly 1 item")
	assert.True(t, list.PageInfo.HasNextPage, "should have a next page with limit=1 and 6 seeded adjustment types")
}

func TestCovFinanceAdjustmentTypes_LimitMax(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covFinanceAdjustmentTypesPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"limit": {"1000"}})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(list.Data), 1000)
}

func TestCovFinanceAdjustmentTypes_PaginationCursor(t *testing.T) {
	t.Parallel()

	// Page 1
	page1, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requirePageLen(t, page1.Data, 1)
	require.NotNil(t, page1.PageInfo.NextPageURL, "page 1 should have next_page_url")

	page1ID := DataItemField(page1.Data[0], "id")
	assert.NotEmpty(t, page1ID)

	// Page 2
	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	requirePageLen(t, page2.Data, 1)

	page2ID := DataItemField(page2.Data[0], "id")
	assert.NotEmpty(t, page2ID)
	assert.NotEqual(t, page1ID, page2ID, "page 2 should have a different item than page 1")
}

func TestCovFinanceAdjustmentTypes_PrevPageURL(t *testing.T) {
	t.Parallel()

	page1, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"limit": {"1"}})
	require.NoError(t, err)
	requirePageLen(t, page1.Data, 1)
	require.NotNil(t, page1.PageInfo.NextPageURL)

	page1ID := DataItemField(page1.Data[0], "id")

	page2, _, err := apiClient.GetListFromPageURL(page1.PageInfo.NextPageURL)
	require.NoError(t, err)
	requirePageLen(t, page2.Data, 1)
	assert.True(t, page2.PageInfo.HasPrevPage, "page 2 should have has_prev_page=true")
	require.NotNil(t, page2.PageInfo.PreviousPageURL, "page 2 should have previous_page_url")

	backToPage1, _, err := apiClient.GetListFromPageURL(page2.PageInfo.PreviousPageURL)
	require.NoError(t, err)
	require.Len(t, backToPage1.Data, 1)

	backID := DataItemField(backToPage1.Data[0], "id")
	assert.Equal(t, page1ID, backID, "navigating back via previous_page_url should return the first page item")
}

// --- List: search (q) ---

func TestCovFinanceAdjustmentTypes_SearchByName(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"q": {SeedAdjustmentTypeName}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "search for %q should return at least 1 result", SeedAdjustmentTypeName)

	for _, item := range list.Data {
		name := DataItemField(item, "name")
		assert.True(t,
			strings.Contains(strings.ToLower(name), strings.ToLower(SeedAdjustmentTypeName)),
			"search result %q should contain %q", name, SeedAdjustmentTypeName,
		)
	}
}

func TestCovFinanceAdjustmentTypes_SearchCaseInsensitive(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"q": {strings.ToLower(SeedAdjustmentTypeName)}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1, "case-insensitive search should return at least 1 result")

	found := false
	for _, item := range list.Data {
		if DataItemField(item, "id") == SeedAdjustmentTypeID {
			found = true
		}
	}
	assert.True(t, found, "lowercase search should still find seeded adjustment type %s", SeedAdjustmentTypeID)
}

func TestCovFinanceAdjustmentTypes_SearchNoResults(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"q": {"zzzznotanadjustmenttype99999"}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data, "nonsense search should return empty data")
}

// TestCovFinanceAdjustmentTypes_SearchWildcardCharsEscaped confirms the raw SQL
// LIKE search escapes user-supplied '%'/'_' wildcard metacharacters rather than
// interpolating them unescaped into the pattern. If unescaped, q=% would be
// wrapped into pattern "%%%" and match every row (since a bare '%' is itself a
// wildcard), and q=Disc_unt would match "Discount" (since '_' matches any
// single char). Neither should happen: both must be treated as literal
// characters that don't appear in any seeded name, yielding zero matches.
func TestCovFinanceAdjustmentTypes_SearchWildcardCharsEscaped(t *testing.T) {
	t.Parallel()

	percentList, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"q": {"%"}})
	require.NoError(t, err)
	assertEmptyListData(t, percentList.Data, "q=%% should be treated as a literal percent sign, not an unescaped SQL wildcard matching every row")

	underscoreList, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"q": {"Disc_unt"}})
	require.NoError(t, err)
	assertEmptyListData(t, underscoreList.Data, "q=Disc_unt should be treated as a literal underscore, not an unescaped SQL single-char wildcard matching 'Discount'")
}

// --- List: expandable (owner) ---

func TestCovFinanceAdjustmentTypes_OwnerNullWithoutInclude(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		assertNilField(t, m, "owner")
	}
}

func TestCovFinanceAdjustmentTypes_OwnerPopulatedWithInclude(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(covFinanceAdjustmentTypesPath, url.Values{"include": {"owner"}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(list.Data), 1)

	for _, item := range list.Data {
		m := parseJSON(item)
		require.NotNil(t, m)
		owner := jsonObject(m, "owner")
		require.NotNil(t, owner, "owner should be populated with ?include=owner")
		assert.Equal(t, "owner", jsonField(owner, "object"), "owner.object should be 'owner'")
		// NOTE (prodBugSuspect, do not fix here): populateOwnerOnAdjustmentType in
		// registered_adjustment_type.go hardcodes OwnerTypeSystem for every row
		// regardless of actual provenance; the adjustment_type table has no
		// account_id column, so this passing today doesn't validate any
		// per-account ownership logic that might be added later.
		assert.Equal(t, "system", jsonField(owner, "type"), "owner.type should be 'system' for global lookup rows")
	}
}

// --- Validation ---

func TestCovFinanceAdjustmentTypes_UnknownQueryParamRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covFinanceAdjustmentTypesPath, url.Values{bogusE2EQueryParam: {"1"}})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, covFinanceAdjustmentTypesPath, status, body)
}

func TestCovFinanceAdjustmentTypes_LimitZeroRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covFinanceAdjustmentTypesPath, url.Values{"limit": {"0"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "limit=0 should be rejected with 400, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
}

func TestCovFinanceAdjustmentTypes_LimitNegativeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covFinanceAdjustmentTypesPath, url.Values{"limit": {"-1"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "limit=-1 should be rejected with 400, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
}

func TestCovFinanceAdjustmentTypes_LimitOverMaxRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covFinanceAdjustmentTypesPath, url.Values{"limit": {"1001"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "limit=1001 should be rejected with 400, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
}

func TestCovFinanceAdjustmentTypes_LimitNonNumericRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covFinanceAdjustmentTypesPath, url.Values{"limit": {"abc"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "limit=abc should be rejected with 400, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

func TestCovFinanceAdjustmentTypes_InvalidCursorRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covFinanceAdjustmentTypesPath, url.Values{"cursor": {"not-a-cursor"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "garbage cursor should be rejected with 400, not 500; got %d: %s", status, string(body))
	requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
}

func TestCovFinanceAdjustmentTypes_InvalidIncludeValueRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covFinanceAdjustmentTypesPath, url.Values{"include": {"bogus_field"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "include=bogus_field should be rejected with 400, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// TestCovFinanceAdjustmentTypes_CrossResourceIncludeRejected confirms an include
// value that is valid for a *different* resource (e.g. "attributes") is still
// rejected for adjustment types, since only "owner" is registered in this
// resource's include registry.
func TestCovFinanceAdjustmentTypes_CrossResourceIncludeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covFinanceAdjustmentTypesPath, url.Values{"include": {"attributes"}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "include=attributes (valid on other resources) should be rejected here, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// --- Auth: reachable by internal and non-internal (customer) actors ---

// TestCovFinanceAdjustmentTypes_CustomerActorCanRead confirms non-internal
// actors (customer portal) can read adjustment types unconditionally, per
// checkAdjustmentTypeReadPermission treating this as global lookup data with
// no per-account access control.
func TestCovFinanceAdjustmentTypes_CustomerActorCanRead(t *testing.T) {
	t.Parallel()
	customer := apiClient.WithBearerToken(SeedCustomerAPIKey, SeedAccountID)
	status, body, err := customer.GetListRaw(covFinanceAdjustmentTypesPath, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	list, _, err := customer.GetList(covFinanceAdjustmentTypesPath, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list.Data), 6, "customer actor should see all 6 seeded adjustment types")
}
