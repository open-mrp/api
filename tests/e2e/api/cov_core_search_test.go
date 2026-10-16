//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/id"
)

// covCoreSearchPath is the unified fan-out search endpoint (/v1/core/search).
// It is a read-only, GET-only, query-param-only endpoint: no create/update/
// delete, no request body, and no /actions/* routes. Its only include is
// `resource`, the matched record. See services/api-gateway/endpoints/search.
const covCoreSearchPath = "/v1/core/search"

// covSearchCase is one searchable type's seeded hit: the query that finds it, and the entity reference preview.9
// callers read for it.
type covSearchCase struct {
	name     string
	q        string
	typ      string
	id       string
	idPrefix id.IDPrefix
	// entityName and entityHandle are the preview.9 entity's name and handle; a nil handle reads as null.
	entityName   string
	entityHandle *string
}

// covSearchCases cover every searchable type. Purchase orders and sales orders share the sales_order table, so the
// id prefix is "or" for both. The customer's handle is the account-relation customer number ("45678", seed
// shared/db/seed/0010_customers.sql). A product is named for its item. SeedShipmentID has no master tracking number,
// so its handle is null. A contact's id is its account_user id and its handle the directory type.
var covSearchCases = []covSearchCase{
	{"sales_order", "ORD-001", "sales_order", SeedSalesOrderID, "or", "ORD-001", new(SeedCustomerName)},
	{"purchase_order", "PO-001", "purchase_order", "or_01seedpurchord1_000", "or", "PO-001", new("Yarn Supply Co")},
	{"invoice", "INV-001", "invoice", SeedInvoiceID, "iv", "INV-001", new(SeedCustomerName)},
	{"customer", SeedCustomerName, "customer", SeedCustomerAccountID, "ac", SeedCustomerName, new("45678")},
	{"item", SeedItemSKU, "item", SeedItemID, "it", SeedItemSKU, new("Small white sock")},
	{"product", SeedItemSKU, "product", SeedProductID, "pd", SeedItemSKU, new("Small white sock")},
	{"shipment", "SHP-001", "shipment", SeedShipmentID, "sh", "SHP-001", nil},
	{"account_user", "Doe", "account_user", SeedAccountUserID, "acus", "John Doe", new("user")},
	{"agent_definition", "Order Processing Bot", "agent_definition", SeedAgentDefinitionID, "agdf", "Order Processing Bot", new("order_processing_bot")},
}

// covSearchHit searches one type and returns the seeded hit, picked out of the results: search is account-wide, and
// a test running alongside can add a second match (another "Doe" in the directory, say).
func covSearchHit(t *testing.T, client *Client, tc covSearchCase, idField string, extra url.Values) map[string]any {
	t.Helper()
	params := url.Values{"q": {tc.q}, "types": {tc.typ}}
	for k, v := range extra {
		params[k] = v
	}
	list, status, err := client.GetList(covCoreSearchPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	for _, raw := range list.Data {
		if row := parseJSON(raw); jsonField(row, idField) == tc.id {
			return row
		}
	}
	t.Fatalf("q=%s&types=%s should surface %s: %v", tc.q, tc.typ, tc.id, list.Data)
	return nil
}

// TestCovCoreSearch_ResultFields asserts every search_result field for each searchable type, with the matched
// record's resource null unless included.
func TestCovCoreSearch_ResultFields(t *testing.T) {
	t.Parallel()
	for _, tc := range covSearchCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := covSearchHit(t, apiClient, tc, "resource_id", nil)
			assert.ElementsMatch(t, []string{"object", "type", "resource_id", "resource", "entity"}, mapKeys(row))
			assertObjectField(t, row, "search_result")
			assert.Equal(t, tc.typ, jsonField(row, "type"))
			assertIDFormat(t, jsonField(row, "resource_id"), tc.idPrefix)
			assertNilField(t, row, "resource")
			assertNilField(t, row, "entity")
		})
	}
}

// TestCovCoreSearch_IncludeResource asserts include=resource expands each hit to its record, in the record's own shape.
func TestCovCoreSearch_IncludeResource(t *testing.T) {
	t.Parallel()
	for _, tc := range covSearchCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := covSearchHit(t, apiClient, tc, "resource_id", url.Values{"include": {"resource"}})
			resource := jsonObject(row, "resource")
			require.NotNil(t, resource, "resource is set with include=resource: %v", row)
			assert.Equal(t, tc.id, jsonField(resource, "id"))
			assert.Equal(t, tc.typ, jsonField(resource, "object"))
			assertNilField(t, row, "entity")
		})
	}
}

// messaging_contact was renamed: contacts are account users.
func TestCovCoreSearch_MessagingContactTypeRejected(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{"q": {"Doe"}, "types": {"messaging_contact"}})
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error"), "types")
}

// TestCovCoreSearch_ResponseShape_ListEnvelope asserts the List[Entity]
// envelope: object="list" and all four page_info sub-fields are present
// (always the zero PageInfo{} today — see TestCovCoreSearch_Pagination_*).
func TestCovCoreSearch_ResponseShape_ListEnvelope(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
		"q":     {"ORD-001"},
		"types": {"sales_order"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assertObjectField(t, parsed, "list")

	pageInfo := jsonObject(parsed, "page_info")
	require.NotNil(t, pageInfo, "page_info should be an object")
	assert.NotContains(t, pageInfo, "has_next_page", "forge's page_info has no booleans")
	assert.NotContains(t, pageInfo, "has_prev_page", "forge's page_info has no booleans")
	assertNilField(t, pageInfo, "next_page_url")
	assertNilField(t, pageInfo, "previous_page_url")

	data := jsonArray(parsed, "data")
	require.NotEmpty(t, data)
}

// ──────────────────────────────────────────────
// List: basic / multi-type / no-results
// ──────────────────────────────────────────────

// TestCovCoreSearch_List_TypesFilter_MultipleTypes asserts a multi-value
// `types` filter (browse mode, no `q`) includes rows from every requested
// type and excludes every other type.
func TestCovCoreSearch_List_TypesFilter_MultipleTypes(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"types": {"sales_order", "purchase_order"},
		"limit": {"50"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)

	var sawSalesOrder, sawPurchaseOrder bool
	for _, raw := range list.Data {
		row := parseJSON(raw)
		typ := jsonField(row, "type")
		assert.Contains(t, []string{"sales_order", "purchase_order"}, typ,
			"types=sales_order,purchase_order should never surface a row of another type")
		switch typ {
		case "sales_order":
			sawSalesOrder = true
		case "purchase_order":
			sawPurchaseOrder = true
		}
	}
	assert.True(t, sawSalesOrder, "expected at least one sales_order row")
	assert.True(t, sawPurchaseOrder, "expected at least one purchase_order row")
}

// TestCovCoreSearch_List_QOnly_NoTypes asserts a bare `q` search (no `types`)
// fans out across resource kinds and surfaces the seeded sales order.
func TestCovCoreSearch_List_QOnly_NoTypes(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q": {"ORD-001"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)

	var sawSalesOrder bool
	for _, raw := range list.Data {
		row := parseJSON(raw)
		assertObjectField(t, row, "search_result")
		require.NotEmpty(t, jsonField(row, "resource_id"))
		require.NotEmpty(t, jsonField(row, "type"))
		if jsonField(row, "resource_id") == SeedSalesOrderID {
			sawSalesOrder = true
			assert.Equal(t, "sales_order", jsonField(row, "type"))
		}
	}
	assert.True(t, sawSalesOrder, "unscoped q=ORD-001 should surface the seeded sales order")
}

// TestCovCoreSearch_List_NoResults asserts a query matching nothing returns
// 200 with an empty data array, not a 404.
func TestCovCoreSearch_List_NoResults(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q": {"zzzznotaresource99999"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assert.Equal(t, "list", list.Object)
	assertEmptyListData(t, list.Data)
}

// TestCovCoreSearch_List_LimitCapsResultCount asserts `limit` caps the number
// of interleaved rows returned, using a browse-mode (`types`-only) query
// against a type known to have more than one seeded row.
func TestCovCoreSearch_List_LimitCapsResultCount(t *testing.T) {
	t.Parallel()

	uncapped, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"types": {"sales_order"},
		"limit": {"100"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.GreaterOrEqual(t, len(uncapped.Data), 2,
		"need at least 2 seeded sales orders for a meaningful limit-capping assertion")

	capped, status2, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"types": {"sales_order"},
		"limit": {"1"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status2, nil)
	assert.Len(t, capped.Data, 1, "limit=1 should cap the result to exactly 1 row")
}

// ──────────────────────────────────────────────
// Customer scoping (?customer_ids=)
// ──────────────────────────────────────────────

// TestCovCoreSearch_CustomerScope_SafeType_RestrictsToCustomer asserts that a
// customer-safe type (sales_order) scoped with ?customer_ids= returns only that
// customer's matching records.
func TestCovCoreSearch_CustomerScope_SafeType_RestrictsToCustomer(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"customer_ids": {SeedCustomerAccountID},
		"q":            {SeedSalesOrderPONumber},
		"types":        {"sales_order"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assert.Equal(t, SeedPOSalesOrderID, jsonField(row, "resource_id"))
	assert.Equal(t, "sales_order", jsonField(row, "type"))
}

// TestCovCoreSearch_CustomerScope_UnsafeType_Excluded asserts that a type
// which is NOT customer-safe (item) is silently dropped from a
// customer-scoped search even though the identical unscoped query matches it.
func TestCovCoreSearch_CustomerScope_UnsafeType_Excluded(t *testing.T) {
	t.Parallel()

	unscoped, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {SeedItemSKU},
		"types": {"item"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, unscoped.Data, "control: unscoped item search should find the seeded item")

	scoped, status2, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"customer_ids": {SeedCustomerAccountID},
		"q":            {SeedItemSKU},
		"types":        {"item"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status2, nil)
	assertEmptyListData(t, scoped.Data,
		"item is not a customer-safe type; a ?customer_ids= scoped search must drop it entirely")
}

// TestCovCoreSearch_CustomerScope_GarbageID_ReturnsEmptyNotError asserts an
// unrecognized ?customer_ids= value is not validated/rejected — it just yields no
// matches for customer-safe types (no dedicated format check in the handler).
func TestCovCoreSearch_CustomerScope_GarbageID_ReturnsEmptyNotError(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"customer_ids": {"ac_bogus_garbage_id_not_real"},
		"q":            {"ORD-001"},
		"types":        {"sales_order"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	assertEmptyListData(t, list.Data)
}

// ──────────────────────────────────────────────
// Permission gating (customer-portal relation actor)
// ──────────────────────────────────────────────

// TestCovCoreSearch_CustomerPortal_RelationActorGetsInsufficientPermissions
// documents the real (empirically confirmed) behavior of the customer-portal
// identity against this endpoint: unlike most other endpoints (see
// customer_portal_access_test.go), a customer/supplier relation actor is NOT
// a supported caller of /v1/core/search. authorize() bypasses the coarse
// gateway gate for relation actors (their Permissions map is intentionally
// empty — see Identity.IsRelationActor), deferring to Search()'s own
// downstream check. But selectActiveSearchProviders gates every provider with
// a plain identity.CheckHasPermission(domain, action) call, which a relation
// actor can never satisfy (empty Permissions map), so `active` is always
// empty and the subsequent identity.CheckHasAnyPermission(searchReadPermissions...)
// fallback also always fails. The result: EVERY request from a relation actor
// 403s with insufficient_permissions, regardless of `types`/`customer`/`q` —
// including a self-scoped `?customer_ids=<their own account>` search, which is
// the one case the searchScope customerID design (see service.go's doc
// comment on searchScope) most plausibly intended to support. This is
// consistent with the endpoint's own coded contract (403 iff the caller holds
// none of the 8 searchReadPermissions domains, which is unconditionally true
// for every relation actor) so it is not flagged as a bug in this endpoint in
// isolation — but it does mean unified search cannot be used to power a
// customer-facing (portal) search/picker UI without adding a relation-actor
// bypass to selectActiveSearchProviders, mirroring the pattern already used
// by ListSalesOrders et al.
func TestCovCoreSearch_CustomerPortal_RelationActorGetsInsufficientPermissions(t *testing.T) {
	t.Parallel()
	client := getCustomerPortalClient()

	cases := []struct {
		name  string
		query url.Values
	}{
		{"unscoped_sales_order", url.Values{"q": {"ORD-001"}, "types": {"sales_order"}}},
		{"self_scoped_customer_param", url.Values{
			"customer_ids": {SeedCustomerAccountID}, "q": {"ORD-001"}, "types": {"sales_order"},
		}},
		{"item_type", url.Values{"q": {SeedItemSKU}, "types": {"item"}}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := client.GetListRaw(covCoreSearchPath, tc.query)
			require.NoError(t, err)
			assert.Equal(t, 403, status, "customer-portal relation actor should get 403, got %d: %s", status, string(body))
			errObj := requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
			assert.Nil(t, errObj["param"])
		})
	}
}

// ──────────────────────────────────────────────
// Validation
// ──────────────────────────────────────────────

// TestCovCoreSearch_Validation_QRequiredUnlessTypes asserts the hand-rolled
// business rule: q empty/whitespace AND types empty → 400 parameter_missing
// naming "q". This is not schema-level required, so it needs its own test.
func TestCovCoreSearch_Validation_QRequiredUnlessTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		query url.Values
	}{
		{"no_params_at_all", url.Values{}},
		{"empty_q", url.Values{"q": {""}}},
		{"whitespace_q", url.Values{"q": {"   "}}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(covCoreSearchPath, tc.query)
			require.NoError(t, err)
			assert.Equal(t, 400, status, "expected 400, got %d: %s", status, string(body))
			errObj := requireErrorResponse(t, body, "parameter_missing", "invalid_request_error")
			assertErrorParam(t, errObj, "q")
		})
	}
}

// TestCovCoreSearch_Validation_QAllowedEmptyWhenTypesSet asserts that an
// empty `q` is fine as long as `types` narrows the search (browse mode).
func TestCovCoreSearch_Validation_QAllowedEmptyWhenTypesSet(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{"types": {"sales_order"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// TestCovCoreSearch_Validation_QTooLong asserts the shared max=500 validator
// rejects an over-length `q`.
func TestCovCoreSearch_Validation_QTooLong(t *testing.T) {
	t.Parallel()

	tooLong := make([]byte, 501)
	for i := range tooLong {
		tooLong[i] = 'a'
	}
	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{"q": {string(tooLong)}})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "501-char q should be rejected, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// TestCovCoreSearch_Validation_TypesInvalid_KnownButUnsupportedObjectType
// asserts that `types=user` — a real constants.ObjectType globally, but not
// in this endpoint's search allow-list — is rejected with 400 parameter_invalid
// naming "types" (not silently ignored, not 200).
func TestCovCoreSearch_Validation_TypesInvalid_KnownButUnsupportedObjectType(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
		"q": {"x"}, "types": {"user"},
	})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "types=user should 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "types")
}

// TestCovCoreSearch_Validation_TypesInvalid_GarbageValue asserts that a
// completely bogus `types` value (not a constants.ObjectType at all) is
// rejected the same way as a known-but-unsupported one.
func TestCovCoreSearch_Validation_TypesInvalid_GarbageValue(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
		"q": {"x"}, "types": {"not_a_real_type"},
	})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "garbage types value should 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "types")
}

// TestCovCoreSearch_Validation_TypesInvalid_MixedWithValid asserts that one
// bad value in a multi-value `types` list still 400s the whole request (it is
// not a partial/best-effort filter).
func TestCovCoreSearch_Validation_TypesInvalid_MixedWithValid(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
		"q": {"x"}, "types": {"sales_order", "not_a_real_type"},
	})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "expected 400, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
	assertErrorParam(t, errObj, "types")
}

// TestCovCoreSearch_Validation_LimitOutOfRange covers limit=0, limit=-1, and
// limit=1001 (the limit is 1 to 100).
func TestCovCoreSearch_Validation_LimitOutOfRange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		limit string
	}{
		{"zero", "0"},
		{"negative", "-1"},
		{"too_large", "1001"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
				"q": {"x"}, "limit": {tc.limit},
			})
			require.NoError(t, err)
			assert.Equal(t, 400, status, "limit=%s should 400, got %d: %s", tc.limit, status, string(body))
			errObj := requireErrorResponse(t, body, "", "invalid_request_error")
			code := errObj["code"]
			assert.True(t, code == "invalid_format" || code == "validation_failed" || code == "parameter_invalid",
				"limit=%s: error.code should be invalid_format, validation_failed, or parameter_invalid, got %v", tc.limit, code)
		})
	}
}

// TestCovCoreSearch_Validation_LimitBoundaries_Valid asserts the documented
// boundary values (1, 100) are accepted.
func TestCovCoreSearch_Validation_LimitBoundaries_Valid(t *testing.T) {
	t.Parallel()

	for _, limit := range []string{"1", "100"} {
		limit := limit
		t.Run(limit, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
				"types": {"sales_order"}, "limit": {limit},
			})
			require.NoError(t, err)
			requireStatus(t, 200, status, body)
		})
	}
}

// TestCovCoreSearch_Validation_UnknownQueryParam uses the shared helper to
// assert an undeclared query parameter is rejected.
func TestCovCoreSearch_Validation_UnknownQueryParam(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
		"q":                {"x"},
		bogusE2EQueryParam: {"1"},
	})
	require.NoError(t, err)
	assertUnknownQueryParamRejected(t, covCoreSearchPath, status, body)
}

// TestCovCoreSearch_Validation_UnknownIncludeRejected asserts an include key other than `resource` is rejected,
// not silently ignored.
func TestCovCoreSearch_Validation_UnknownIncludeRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
		"q": {"x"}, "include": {"related.sales_order"},
	})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "an unknown include should be rejected, got %d: %s", status, string(body))
	requireErrorResponse(t, body, "parameter_invalid", "invalid_request_error")
}

// TestCovCoreSearch_Validation_CursorGarbage_NoOp asserts a garbage `cursor`
// value neither 400s nor 500s: it is documented as completely unused by the
// implementation today (see TestCovCoreSearch_Pagination_CursorIsNoOp_BUG),
// so passing one is a harmless no-op rather than a validation error.
func TestCovCoreSearch_Validation_CursorGarbage_NoOp(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
		"q": {"ORD-001"}, "types": {"sales_order"}, "cursor": {"totally-garbage-cursor-value"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// TestCovCoreSearch_Auth_InvalidBearerToken_401 is a smoke check that the
// gateway-wide auth middleware rejects a garbage bearer token before this
// endpoint's handler runs.
func TestCovCoreSearch_Auth_InvalidBearerToken_401(t *testing.T) {
	t.Parallel()
	bogusClient := apiClient.WithBearerToken("garbage_not_a_real_token_xyz", SeedAccountID)

	status, body, err := bogusClient.GetListRaw(covCoreSearchPath, url.Values{"q": {"x"}})
	require.NoError(t, err)
	assert.Equal(t, 401, status, "garbage bearer token should 401, got %d: %s", status, string(body))
}

// ──────────────────────────────────────────────
// Pagination — fan-out picker does not paginate
// ──────────────────────────────────────────────

// TestCovCoreSearch_Pagination_NotSupported asserts the true contract of this
// endpoint: unified search is a concurrent fan-out picker, not a keyset-
// paginated list. Search() calls up to nine per-type list RPCs in parallel,
// interleaves their rows, and caps the merged set at `limit` — there is no
// single upstream cursor to thread through, so it returns the zero-value
// apiresource.PageInfo{} by the same convention every other aggregate list in
// the gateway follows (compare endpoints/messages/service.go, which likewise
// returns PageInfo{} for its drafts/scheduled aggregate reads and only maps a
// real proto page_info for its single-source ListMessages keyset read).
//
// This test proves the truncation is real (limit=1 returns 1 of ≥2 rows) and
// then asserts the honest consequence: even with more matching rows than the
// page fits, page_info stays entirely zero — has_next_page/has_prev_page
// false, next/previous_page_url nil. Reporting has_next_page=true here would
// be a lie, because `cursor` is a no-op (see
// TestCovCoreSearch_Validation_CursorGarbage_NoOp) and no next page can be
// fetched. Clients narrow results by refining `q`/`types`, not by paging.
func TestCovCoreSearch_Pagination_NotSupported(t *testing.T) {
	t.Parallel()

	uncapped, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"types": {"sales_order"}, "limit": {"100"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.GreaterOrEqual(t, len(uncapped.Data), 2,
		"need at least 2 seeded sales orders to prove limit actually truncates")

	page1, status2, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"types": {"sales_order"}, "limit": {"1"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status2, nil)
	require.Len(t, page1.Data, 1, "limit=1 must truncate the %d-row result set to 1", len(uncapped.Data))

	// The result set was demonstrably truncated, yet this fan-out picker exposes
	// no pagination cursor: page_info is the zero value, matching every other
	// aggregate list in the gateway (endpoints/messages/service.go PageInfo{}).
	assert.False(t, page1.PageInfo.HasNextPage(),
		"fan-out search does not paginate; has_next_page stays false even when limit truncates results")
	assert.False(t, page1.PageInfo.HasPrevPage(),
		"fan-out search does not paginate; has_prev_page stays false")
	assert.Nil(t, page1.PageInfo.NextPageURL,
		"fan-out search exposes no next-page cursor; next_page_url stays nil")
	assert.Nil(t, page1.PageInfo.PreviousPageURL,
		"fan-out search exposes no previous-page cursor; previous_page_url stays nil")
}
