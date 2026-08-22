//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// SalesOrder — Additional Include Tests
// ──────────────────────────────────────────────
//
// This file covers include fields not already tested in included_fields_test.go
// (which checks carrier, service_level, payment_term, shipping_term).

func TestSalesOrders_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["customer"], "customer should be null without ?include=customer")
	assert.Nil(t, got["bill_to_address"], "bill_to_address should be null without ?include=bill_to_address")
	assert.Nil(t, got["ship_to_address"], "ship_to_address should be null without ?include=ship_to_address")
	assert.Nil(t, got["carrier"], "carrier should be null without ?include=carrier")
	assert.Nil(t, got["service_level"], "service_level should be null without ?include=service_level")
	assert.Nil(t, got["payment_term"], "payment_term should be null without ?include=payment_term")
	assert.Nil(t, got["shipping_term"], "shipping_term should be null without ?include=shipping_term")
	assert.Nil(t, got["order_discount"], "order_discount should be null without ?include=order_discount")
	assert.Nil(t, got["lines"], "lines should be null without ?include=lines")
	assert.Nil(t, got["contacts"], "contacts should be null without ?include=contacts")
}

func TestSalesOrders_IncludeContacts(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"contacts"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	contacts := jsonObject(parseJSON(body), "contacts")
	require.NotNil(t, contacts, "contacts should be present with ?include=contacts")
	assert.Equal(t, "order_contact", jsonField(contacts, "object"))
	// ORD-001 is seeded with one invoice and one acknowledgement recipient.
	assert.Contains(t, jsonStringSlice(contacts, "invoice"), "dane@openmrp.ai")
	assert.Contains(t, jsonStringSlice(contacts, "acknowledgement"), "user2@openmrp.ai")
}

func TestSalesOrders_List_IncludeContacts(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"contacts"}})
	contacts := jsonObject(row, "contacts")
	require.NotNil(t, contacts, "contacts should be populated on the list row with ?include=contacts")
	assert.Equal(t, "order_contact", jsonField(contacts, "object"))
	assert.Contains(t, jsonStringSlice(contacts, "invoice"), "dane@openmrp.ai")
	assert.Contains(t, jsonStringSlice(contacts, "acknowledgement"), "user2@openmrp.ai")
}

func TestSalesOrders_IncludeCustomer(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"customer"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	cust := jsonObject(got, "customer")
	require.NotNil(t, cust, "customer should be present with ?include=customer")
	assert.Equal(t, "customer", jsonField(cust, "object"))
	assert.NotEmpty(t, jsonField(cust, "id"))
}

func TestSalesOrders_IncludeBillToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"bill_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["bill_to_address"]
	assert.True(t, ok, "bill_to_address key should be present with ?include=bill_to_address")
	if addr := jsonObject(got, "bill_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
	}
}

func TestSalesOrders_IncludeShipToAddress(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"ship_to_address"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	_, ok := got["ship_to_address"]
	assert.True(t, ok, "ship_to_address key should be present with ?include=ship_to_address")
	if addr := jsonObject(got, "ship_to_address"); addr != nil {
		assert.Equal(t, "address", jsonField(addr, "object"))
	}
}

func TestSalesOrders_IncludeOrderDiscount(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"order_discount"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	discount := jsonObject(got, "order_discount")
	require.NotNil(t, discount, "order_discount should be populated with ?include=order_discount (seed sets order_discount_id on ORD-001)")
	assert.Equal(t, "order_discount", jsonField(discount, "object"))
	assert.NotEmpty(t, jsonField(discount, "id"))
}

func TestSalesOrders_IncludeLines(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	lines := jsonObject(got, "lines")
	require.NotNil(t, lines, "lines should be present with ?include=lines")
	assert.Equal(t, "list", jsonField(lines, "object"))
}

// firstSalesOrderLineProduct returns the product on the first order line that has one.
func firstSalesOrderLineProduct(t *testing.T, order map[string]any) map[string]any {
	t.Helper()
	for _, raw := range jsonListData(order, "lines") {
		line, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if product := jsonObject(line, "product"); product != nil {
			return product
		}
	}
	t.Fatalf("no order line with an expanded product")
	return nil
}

func TestSalesOrders_IncludeLineProductItemCategory(t *testing.T) {
	t.Parallel()

	// Without the nested include, the item is expanded but its category is not.
	bare := getSalesOrder(t, SeedSalesOrderID, url.Values{"include": {"lines", "lines.product", "lines.product.item"}})
	item := jsonObject(firstSalesOrderLineProduct(t, bare), "item")
	require.NotNil(t, item, "item should be present with ?include=lines.product.item")
	assertNilField(t, item, "category")

	full := getSalesOrder(t, SeedSalesOrderID, url.Values{"include": {"lines", "lines.product", "lines.product.item", "lines.product.item.category"}})
	item = jsonObject(firstSalesOrderLineProduct(t, full), "item")
	require.NotNil(t, item)
	category := jsonObject(item, "category")
	require.NotNil(t, category, "category should be present with ?include=lines.product.item.category")
	assert.Equal(t, "item_category", jsonField(category, "object"))
	assert.NotEmpty(t, jsonField(category, "id"))
}

func TestSalesOrders_IncludeLineProductItemCategoryUnitGroup(t *testing.T) {
	t.Parallel()

	// The category resolves without its own sub-objects when they are not requested.
	bare := getSalesOrder(t, SeedSalesOrderID, url.Values{"include": {"lines", "lines.product", "lines.product.item", "lines.product.item.category"}})
	category := jsonObject(jsonObject(firstSalesOrderLineProduct(t, bare), "item"), "category")
	require.NotNil(t, category)
	assertNilField(t, category, "unit_group")
	assertNilField(t, category, "properties")

	full := getSalesOrder(t, SeedSalesOrderID, url.Values{"include": {
		"lines", "lines.product", "lines.product.item", "lines.product.item.category",
		"lines.product.item.category.properties",
		"lines.product.item.category.unit_group",
		"lines.product.item.category.unit_group.base_unit",
		"lines.product.item.category.unit_group.associated_units",
		"lines.product.item.category.unit_group.associated_units.unit",
	}})
	category = jsonObject(jsonObject(firstSalesOrderLineProduct(t, full), "item"), "category")
	require.NotNil(t, category)

	unitGroup := jsonObject(category, "unit_group")
	require.NotNil(t, unitGroup, "unit_group should be present with ?include=lines.product.item.category.unit_group")
	assert.Equal(t, "unit_group", jsonField(unitGroup, "object"))

	baseUnit := jsonObject(unitGroup, "base_unit")
	require.NotNil(t, baseUnit, "base_unit should be present with the nested include")
	assert.Equal(t, "unit", jsonField(baseUnit, "object"))

	associated := jsonListData(unitGroup, "associated_units")
	require.NotEmpty(t, associated, "associated_units should be populated with the nested include")
	first, ok := associated[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "unit_group_unit", jsonField(first, "object"))
	assert.NotNil(t, jsonObject(first, "unit"), "associated_units[].unit should be populated with the nested include")

	require.NotNil(t, jsonObject(category, "properties"), "properties should be present with the nested include")
}

func TestSalesOrders_List_IncludeLineProductItemCategory(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"lines", "lines.product", "lines.product.item", "lines.product.item.category"}})
	item := jsonObject(firstSalesOrderLineProduct(t, row), "item")
	require.NotNil(t, item)
	category := jsonObject(item, "category")
	require.NotNil(t, category, "category should be populated on the list row with ?include=lines.product.item.category")
	assert.Equal(t, "item_category", jsonField(category, "object"))
}

// ──────────────────────────────────────────────
// SalesOrder — List parity (no summary object)
// ──────────────────────────────────────────────
//
// The list endpoint returns the full SalesOrder resource (there is no
// SalesOrderSummary). These tests pin that a list row can expand the same
// includes as detail, while inline scalars like line_count are always present.

// salesOrderListRow fetches the sales-order list with the given query params and
// returns the row for SeedSalesOrderID, failing if it is not on the page.
func salesOrderListRow(t *testing.T, params url.Values) map[string]any {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("limit", "100")

	status, body, err := apiClient.GetListRaw(salesOrdersPath, params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	for _, item := range jsonArray(got, "data") {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if jsonField(row, "id") == SeedSalesOrderID {
			return row
		}
	}
	require.FailNowf(t, "seed sales order not found in list", "id %s not in list response", SeedSalesOrderID)
	return nil
}

func TestSalesOrders_List_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, nil)

	// Inline scalars are always present on every row.
	assert.Equal(t, "sales_order", jsonField(row, "object"))
	assert.NotEmpty(t, jsonField(row, "number"))
	_, hasLineCount := row["line_count"]
	assert.True(t, hasLineCount, "line_count should always be present on a list row")

	// Expandable sub-resources are null until requested — same as detail.
	assert.Nil(t, row["customer"], "customer should be null without ?include=customer")
	assert.Nil(t, row["ship_to_address"], "ship_to_address should be null without ?include=ship_to_address")
	assert.Nil(t, row["payment_term"], "payment_term should be null without ?include=payment_term")
	assert.Nil(t, row["lines"], "lines should be null without ?include=lines")
}

func TestSalesOrders_List_IncludeShipToAddress(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"ship_to_address"}})
	addr := jsonObject(row, "ship_to_address")
	require.NotNil(t, addr, "ship_to_address should be populated on the list row with ?include=ship_to_address")
	assert.Equal(t, "address", jsonField(addr, "object"))
}

func TestSalesOrders_List_IncludePaymentTerm(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"payment_term"}})
	term := jsonObject(row, "payment_term")
	require.NotNil(t, term, "payment_term should be populated on the list row with ?include=payment_term")
	assert.Equal(t, "payment_term", jsonField(term, "object"))
}

func TestSalesOrders_List_IncludeLines(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"lines"}})
	lines := jsonObject(row, "lines")
	require.NotNil(t, lines, "lines should be populated on the list row with ?include=lines")
	assert.Equal(t, "list", jsonField(lines, "object"))
}

// ──────────────────────────────────────────────
// SalesOrder — created_by include
// ──────────────────────────────────────────────
//
// created_by is resolved from the order's create audit event (via
// platform-service) only when included. The seed order has a seeded internal
// create event; SeedInternalSalesOrderID has none, exercising the system
// fallback.

func TestSalesOrders_CreatedBy_OmittedByDefault(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["created_by"], "created_by should be null without ?include=created_by")

	row := salesOrderListRow(t, nil)
	assert.Nil(t, row["created_by"], "created_by should be null on a list row without ?include=created_by")
}

func TestSalesOrders_IncludeCreatedBy_Retrieve(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"created_by"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	createdBy := jsonObject(parseJSON(body), "created_by")
	require.NotNil(t, createdBy, "created_by should be present with ?include=created_by")
	assert.Equal(t, "created_by", jsonField(createdBy, "object"))
	assert.Equal(t, "internal", jsonField(createdBy, "relation"))

	actor := jsonObject(createdBy, "actor")
	require.NotNil(t, actor, "actor should be present for an internal creator")
	assert.Equal(t, "actor", jsonField(actor, "object"))
	assert.NotEmpty(t, jsonField(actor, "id"))
	assert.NotEmpty(t, jsonField(actor, "type"))
}

func TestSalesOrders_IncludeCreatedBy_List(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"created_by"}})
	createdBy := jsonObject(row, "created_by")
	require.NotNil(t, createdBy, "created_by should be present on a list row with ?include=created_by")
	assert.Equal(t, "created_by", jsonField(createdBy, "object"))
	assert.Equal(t, "internal", jsonField(createdBy, "relation"))
	require.NotNil(t, jsonObject(createdBy, "actor"), "actor should be present for an internal creator")
}

func TestSalesOrders_IncludeCreatedBy_SystemFallback(t *testing.T) {
	t.Parallel()

	// An order with no create audit event resolves to a system creator, no actor.
	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedInternalSalesOrderID, url.Values{"include": {"created_by"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	createdBy := jsonObject(parseJSON(body), "created_by")
	require.NotNil(t, createdBy, "created_by should be present with ?include=created_by")
	assert.Equal(t, "created_by", jsonField(createdBy, "object"))
	assert.Equal(t, "system", jsonField(createdBy, "relation"))
	assert.Nil(t, createdBy["actor"], "actor should be null for a system creator")
}

// ──────────────────────────────────────────────
// SalesOrder — sales_rep include
// ──────────────────────────────────────────────
//
// sales_rep is an expandable Actor. The backend only ships the rep's id and name
// on the order; the gateway hydrates the rep's display name, handle (email), and
// avatar URL from core-service when ?include=sales_rep is requested. ORD-001 is
// seeded with sales_rep_id = SeedAccountUserID (John Doe / dane@openmrp.ai, who has
// a seeded avatar), so the hydrated fields resolve with real data.

func TestSalesOrders_SalesRep_OmittedByDefault(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["sales_rep"], "sales_rep should be null without ?include=sales_rep")

	row := salesOrderListRow(t, nil)
	assert.Nil(t, row["sales_rep"], "sales_rep should be null on a list row without ?include=sales_rep")
}

// assertSeedSalesRepActor pins the fully hydrated shape of ORD-001's sales rep:
// the id/object/type plus the default-populated name, handle, and avatar_url that
// the gateway resolves from core-service.
func assertSeedSalesRepActor(t *testing.T, salesRep map[string]any) {
	t.Helper()
	require.NotNil(t, salesRep, "sales_rep should be present with ?include=sales_rep")
	assert.Equal(t, "actor", jsonField(salesRep, "object"))
	assert.Equal(t, "user", jsonField(salesRep, "type"))
	assert.Equal(t, SeedAccountUserID, jsonField(salesRep, "id"))
	// Name, handle (email), and avatar_url are hydrated by default — the fix under test.
	assert.Equal(t, "John Doe", jsonField(salesRep, "name"))
	assert.Equal(t, "dane@openmrp.ai", jsonField(salesRep, "handle"))
	assert.NotEmpty(t, jsonField(salesRep, "avatar_url"), "avatar_url should be hydrated for a rep with a seeded avatar")
}

func TestSalesOrders_IncludeSalesRep_Retrieve(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+SeedSalesOrderID, url.Values{"include": {"sales_rep"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	assertSeedSalesRepActor(t, jsonObject(parseJSON(body), "sales_rep"))
}

func TestSalesOrders_IncludeSalesRep_List(t *testing.T) {
	t.Parallel()

	row := salesOrderListRow(t, url.Values{"include": {"sales_rep"}})
	assertSeedSalesRepActor(t, jsonObject(row, "sales_rep"))
}

// sales_rep_id references account_user.id (acus_), not the user id. The update path
// must reject a user id (which no join resolves and which would silently blank the
// rep) and accept a real account_user id, hydrating it to name/handle/avatar.
func TestSalesOrders_UpdateSalesRep_RejectsUserIDAcceptsAccountUser(t *testing.T) {
	t.Parallel()
	customerID := setupOrderCustomer(t)

	status, body, err := apiClient.Post(salesOrdersPath, minimalSalesOrderCreateBody(t, customerID), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	orderID := jsonField(parseJSON(body), "id")
	deleteOrder(t, orderID)

	// A user id (us_) is the wrong id space — it must be rejected, not stored.
	badStatus, badBody, err := apiClient.Patch(salesOrdersPath+"/"+orderID,
		map[string]any{"sales_rep_id": SeedUserID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, badStatus, badBody)

	// The matching account_user id is accepted and hydrates on read.
	okStatus, okBody, err := apiClient.Patch(salesOrdersPath+"/"+orderID,
		map[string]any{"sales_rep_id": SeedAccountUserID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, okStatus, okBody)

	getStatus, getBody, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, url.Values{"include": {"sales_rep"}})
	require.NoError(t, err)
	requireStatus(t, 200, getStatus, getBody)
	assertSeedSalesRepActor(t, jsonObject(parseJSON(getBody), "sales_rep"))
}
