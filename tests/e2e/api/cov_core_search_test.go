//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// covCoreSearchPath is the unified fan-out search endpoint (/v1/core/search).
// It is a read-only, GET-only, query-param-only endpoint: no create/update/
// delete, no request body, no `include` param (Entity has no expandable
// fields), and no /actions/* routes. See services/api-gateway/endpoints/search.
const covCoreSearchPath = "/v1/core/search"

// ──────────────────────────────────────────────
// Response shape / all fields (per searchable type)
// ──────────────────────────────────────────────

// TestCovCoreSearch_EntityFields_SalesOrder asserts every Entity json field
// for a sales_order search hit.
func TestCovCoreSearch_EntityFields_SalesOrder(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {"ORD-001"},
		"types": {"sales_order"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1, "q=ORD-001&types=sales_order should return exactly the seeded order")

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "or")
	assert.Equal(t, SeedSalesOrderID, jsonField(row, "id"))
	assertObjectField(t, row, "entity")
	assert.Equal(t, "sales_order", jsonField(row, "type"))
	assert.Equal(t, "ORD-001", jsonField(row, "name"))
	assert.Equal(t, SeedCustomerName, jsonField(row, "handle"))
}

// TestCovCoreSearch_EntityFields_PurchaseOrder asserts every Entity json field
// for a purchase_order search hit. Purchase orders and sales orders share the
// sales_order table, so the id prefix is "or" for both.
func TestCovCoreSearch_EntityFields_PurchaseOrder(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {"PO-001"},
		"types": {"purchase_order"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "or")
	assert.Equal(t, "or_01seedpurchord1_000", jsonField(row, "id")) // seed shared/db/seed/0014_e2e_extras.sql:704
	assertObjectField(t, row, "entity")
	assert.Equal(t, "purchase_order", jsonField(row, "type"))
	assert.Equal(t, "PO-001", jsonField(row, "name"))
	assert.Equal(t, "Yarn Supply Co", jsonField(row, "handle"))
}

// TestCovCoreSearch_EntityFields_Invoice asserts every Entity json field for
// an invoice search hit.
func TestCovCoreSearch_EntityFields_Invoice(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {"INV-001"},
		"types": {"invoice"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "iv")
	assert.Equal(t, SeedInvoiceID, jsonField(row, "id"))
	assertObjectField(t, row, "entity")
	assert.Equal(t, "invoice", jsonField(row, "type"))
	assert.Equal(t, "INV-001", jsonField(row, "name"))
	assert.Equal(t, SeedCustomerName, jsonField(row, "handle"))
}

// TestCovCoreSearch_EntityFields_Customer asserts every Entity json field for
// a customer search hit. The handle is the account-relation customer number
// ("45678", seed shared/db/seed/0010_customers.sql:24) rather than a free
// field on the customer account itself.
func TestCovCoreSearch_EntityFields_Customer(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {SeedCustomerName},
		"types": {"customer"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "ac")
	assert.Equal(t, SeedCustomerAccountID, jsonField(row, "id"))
	assertObjectField(t, row, "entity")
	assert.Equal(t, "customer", jsonField(row, "type"))
	assert.Equal(t, SeedCustomerName, jsonField(row, "name"))
	assert.Equal(t, "45678", jsonField(row, "handle"))
}

// TestCovCoreSearch_EntityFields_Item asserts every Entity json field for an
// item search hit. Name is the SKU, handle is the item description.
func TestCovCoreSearch_EntityFields_Item(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {SeedItemSKU},
		"types": {"item"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "it")
	assert.Equal(t, SeedItemID, jsonField(row, "id"))
	assertObjectField(t, row, "entity")
	assert.Equal(t, "item", jsonField(row, "type"))
	assert.Equal(t, SeedItemSKU, jsonField(row, "name"))
	assert.Equal(t, "Small white sock", jsonField(row, "handle"))
}

// TestCovCoreSearch_EntityFields_Product asserts every Entity json field for
// a product search hit. A product's display name/handle are derived from its
// linked item's SKU/description, so this shares the item's SKU query term.
func TestCovCoreSearch_EntityFields_Product(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {SeedItemSKU},
		"types": {"product"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "pd")
	assert.Equal(t, SeedProductID, jsonField(row, "id"))
	assertObjectField(t, row, "entity")
	assert.Equal(t, "product", jsonField(row, "type"))
	assert.Equal(t, SeedItemSKU, jsonField(row, "name"))
	assert.Equal(t, "Small white sock", jsonField(row, "handle"))
}

// TestCovCoreSearch_EntityFields_Shipment asserts every Entity json field for
// a shipment search hit. SeedShipmentID has no master tracking number, so its
// handle is null — this is the case that proves handle is genuinely nullable.
func TestCovCoreSearch_EntityFields_Shipment(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {"SHP-001"},
		"types": {"shipment"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "sh")
	assert.Equal(t, SeedShipmentID, jsonField(row, "id"))
	assertObjectField(t, row, "entity")
	assert.Equal(t, "shipment", jsonField(row, "type"))
	assert.Equal(t, "SHP-001", jsonField(row, "name"))
	assertNilField(t, row, "handle")
}

// TestCovCoreSearch_EntityFields_MessagingContact asserts every Entity json
// field for a messaging_contact search hit. The contact's addressable id is
// the account_user id, and handle carries the contact's directory type.
func TestCovCoreSearch_EntityFields_MessagingContact(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {"Doe"},
		"types": {"messaging_contact"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1, "q=Doe&types=messaging_contact should surface only John Doe")

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "acus")
	assert.Equal(t, SeedAccountUserID, jsonField(row, "id"))
	assertObjectField(t, row, "entity")
	assert.Equal(t, "messaging_contact", jsonField(row, "type"))
	assert.Equal(t, "John Doe", jsonField(row, "name"))
	assert.Equal(t, "user", jsonField(row, "handle"))
}

// TestCovCoreSearch_EntityFields_AgentDefinition asserts every Entity json
// field for an agent_definition search hit.
func TestCovCoreSearch_EntityFields_AgentDefinition(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"q":     {"Order Processing Bot"},
		"types": {"agent_definition"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assertIDFormat(t, jsonField(row, "id"), "agdf")
	assert.Equal(t, SeedAgentDefinitionID, jsonField(row, "id"))
	assertObjectField(t, row, "entity")
	assert.Equal(t, "agent_definition", jsonField(row, "type"))
	assert.Equal(t, "Order Processing Bot", jsonField(row, "name"))
	assert.Equal(t, "order_processing_bot", jsonField(row, "handle"))
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
	_, hasNext := pageInfo["has_next_page"]
	_, hasPrev := pageInfo["has_prev_page"]
	assert.True(t, hasNext, "page_info.has_next_page should be present")
	assert.True(t, hasPrev, "page_info.has_prev_page should be present")
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
		assertObjectField(t, row, "entity")
		require.NotEmpty(t, jsonField(row, "id"))
		require.NotEmpty(t, jsonField(row, "type"))
		if jsonField(row, "id") == SeedSalesOrderID {
			sawSalesOrder = true
			assert.Equal(t, "sales_order", jsonField(row, "type"))
			assert.Equal(t, "ORD-001", jsonField(row, "name"))
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
		"limit": {"1000"},
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
// Customer scoping (?customer=)
// ──────────────────────────────────────────────

// TestCovCoreSearch_CustomerScope_SafeType_RestrictsToCustomer asserts that a
// customer-safe type (sales_order) scoped with ?customer= returns only that
// customer's matching records.
func TestCovCoreSearch_CustomerScope_SafeType_RestrictsToCustomer(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"customer": {SeedCustomerAccountID},
		"q":        {SeedSalesOrderPONumber},
		"types":    {"sales_order"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.Len(t, list.Data, 1)

	row := parseJSON(list.Data[0])
	assert.Equal(t, SeedPOSalesOrderID, jsonField(row, "id"))
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
		"customer": {SeedCustomerAccountID},
		"q":        {SeedItemSKU},
		"types":    {"item"},
	})
	require.NoError(t, err)
	requireStatus(t, 200, status2, nil)
	assertEmptyListData(t, scoped.Data,
		"item is not a customer-safe type; a ?customer= scoped search must drop it entirely")
}

// TestCovCoreSearch_CustomerScope_GarbageID_ReturnsEmptyNotError asserts an
// unrecognized ?customer= value is not validated/rejected — it just yields no
// matches for customer-safe types (no dedicated format check in the handler).
func TestCovCoreSearch_CustomerScope_GarbageID_ReturnsEmptyNotError(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(covCoreSearchPath, url.Values{
		"customer": {"ac_bogus_garbage_id_not_real"},
		"q":        {"ORD-001"},
		"types":    {"sales_order"},
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
// including a self-scoped `?customer=<their own account>` search, which is
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
			"customer": {SeedCustomerAccountID}, "q": {"ORD-001"}, "types": {"sales_order"},
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
	requireErrorResponse(t, body, "invalid_format", "invalid_request_error")
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
// limit=1001 (validator is min=1,max=1000).
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
// boundary values (1, 1000) are accepted.
func TestCovCoreSearch_Validation_LimitBoundaries_Valid(t *testing.T) {
	t.Parallel()

	for _, limit := range []string{"1", "1000"} {
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

// TestCovCoreSearch_Validation_IncludeParamRejected asserts `include` is not
// a recognized query param on this endpoint (Entity has no expandable
// fields, so there is nothing to gate) — it should 400 like any other
// unrecognized param, not be silently ignored.
func TestCovCoreSearch_Validation_IncludeParamRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(covCoreSearchPath, url.Values{
		"q": {"x"}, "include": {"sales_order"},
	})
	require.NoError(t, err)
	assert.Equal(t, 400, status, "include should be rejected as unknown, got %d: %s", status, string(body))
	errObj := requireErrorResponse(t, body, "parameter_unknown", "invalid_request_error")
	assertErrorParam(t, errObj, "include")
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
		"types": {"sales_order"}, "limit": {"1000"},
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
	assert.False(t, page1.PageInfo.HasNextPage,
		"fan-out search does not paginate; has_next_page stays false even when limit truncates results")
	assert.False(t, page1.PageInfo.HasPrevPage,
		"fan-out search does not paginate; has_prev_page stays false")
	assert.Nil(t, page1.PageInfo.NextPageURL,
		"fan-out search exposes no next-page cursor; next_page_url stays nil")
	assert.Nil(t, page1.PageInfo.PreviousPageURL,
		"fan-out search exposes no previous-page cursor; previous_page_url stays nil")
}
