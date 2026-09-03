//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sub-object hydration.
//
// A sub-object is either absent or complete. The failure this file exists to catch is the third
// state: a resource that arrives with its `id` and `object` set and every other required field
// blank, because a presenter built a shell out of a foreign key instead of resolving the record.
// That shape passes a null check, renders as an empty string, and is invisible until a screen
// tries to print it.
//
// Two rules follow from that, and every test here checks one of them:
//
//   - A field behind an include is null until asked for, and fully populated once it is.
//   - A computed figure — one summed or netted at read time, with no row of its own — carries the
//     unit it is counted in, because there is no id a caller could follow to fetch it.

// --- Shared assertions ---

// assertUnitHydrated asserts a unit arrived as a whole record rather than an id with blanks after it.
func assertUnitHydrated(t *testing.T, unit map[string]any, where string) {
	t.Helper()

	require.NotNil(t, unit, "%s must be present", where)
	assertObjectField(t, unit, "unit")
	assert.NotEmpty(t, jsonField(unit, "id"), "%s.id", where)
	assert.NotEmpty(t, jsonField(unit, "name"), "%s.name — a unit resolved from an id has a name", where)
	assert.NotEmpty(t, jsonField(unit, "abbreviation"), "%s.abbreviation — this is what a screen prints", where)
	assert.NotEmpty(t, jsonField(unit, "type"), "%s.type", where)
	assert.NotEmpty(t, jsonField(unit, "ratio_numerator"), "%s.ratio_numerator", where)
	assert.NotEmpty(t, jsonField(unit, "ratio_denominator"), "%s.ratio_denominator", where)
	assertValidTimestamp(t, jsonField(unit, "created_at"), where+".created_at")
	assertValidTimestamp(t, jsonField(unit, "updated_at"), where+".updated_at")
}

// assertRateHydrated asserts a rate arrived complete. Its two units stay behind their own includes,
// so they are not checked here.
func assertRateHydrated(t *testing.T, rate map[string]any, where string) {
	t.Helper()

	require.NotNil(t, rate, "%s must be present", where)
	assertObjectField(t, rate, "rate")
	assert.NotEmpty(t, jsonField(rate, "id"), "%s.id", where)
	assert.NotEmpty(t, jsonField(rate, "value"), "%s.value", where)
	assert.NotEmpty(t, jsonField(rate, "display_value"), "%s.display_value — required, and the only readable form of the rate", where)
	assertValidTimestamp(t, jsonField(rate, "created_at"), where+".created_at")
	assertValidTimestamp(t, jsonField(rate, "updated_at"), where+".updated_at")
}

// assertQuantityHydrated asserts a stored quantity arrived complete. Its unit is expandable, so it
// is checked separately by the tests that ask for it.
func assertQuantityHydrated(t *testing.T, quantity map[string]any, where string) {
	t.Helper()

	require.NotNil(t, quantity, "%s must be present", where)
	assertObjectField(t, quantity, "quantity")
	assert.NotEmpty(t, jsonField(quantity, "id"), "%s.id — a stored quantity is a row, so it has one", where)
	assert.NotEmpty(t, jsonField(quantity, "value"), "%s.value", where)
	assert.NotEmpty(t, jsonField(quantity, "display_value"), "%s.display_value", where)
}

// assertComputedQuantityHydrated asserts a figure derived at read time arrived with its unit. It has
// no id — that is the difference between it and a stored quantity — and the unit is not expandable,
// so it must already be here.
func assertComputedQuantityHydrated(t *testing.T, quantity map[string]any, where string) {
	t.Helper()

	require.NotNil(t, quantity, "%s must be present", where)
	assertObjectField(t, quantity, "computed_quantity")
	assert.NotContains(t, quantity, "id", "%s carries no id — it is computed, not stored", where)
	assert.NotEmpty(t, jsonField(quantity, "value"), "%s.value", where)
	assert.NotEmpty(t, jsonField(quantity, "display_value"), "%s.display_value", where)
	assertUnitHydrated(t, jsonObject(quantity, "unit"), where+".unit")
}

// firstLineWith asks for a resource with the given includes and returns its first line.
func firstLineWith(t *testing.T, path string, includes ...string) (map[string]any, []byte) {
	t.Helper()

	status, body, err := apiClient.GetListRaw(path, url.Values{"include": includes})
	require.NoError(t, err)
	require.Less(t, status, 500, "%s must not 5xx: %s", path, string(body))
	requireStatus(t, 200, status, body)

	lines := jsonListData(parseJSON(body), "lines")
	require.NotEmpty(t, lines, "the seeded record has lines: %s", string(body))
	line, ok := lines[0].(map[string]any)
	require.True(t, ok)
	return line, body
}

// --- Item costs ---

// A cost is a currency over an item unit. Both units come back resolved without being asked for,
// because "5.00" with no currency and no basis is not a cost.
func TestItemCosts_UnitsAreResolvedWithoutAnInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID+"/costs", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "item costs must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	costs := parseJSON(body)
	assertObjectField(t, costs, "item_costs")

	for _, field := range []string{"direct_material_cost", "direct_labor_cost", "overhead_cost", "total_cost"} {
		assert.NotEmpty(t, jsonField(costs, field), "%s is required", field)
	}

	assertUnitHydrated(t, jsonObject(costs, "numerator_unit"), "numerator_unit")
	assertUnitHydrated(t, jsonObject(costs, "denominator_unit"), "denominator_unit")
	assert.Equal(t, "currency", jsonField(jsonObject(costs, "numerator_unit"), "type"),
		"the numerator is the currency the costs are priced in")
}

// --- Item inventory ---

// The four stock figures are netted out of the ledger, so each is a computed quantity that arrives
// with its unit already on it.
func TestItemInventory_ExpandedFiguresCarryTheirUnit(t *testing.T) {
	t.Parallel()

	includes := []string{"on_hand", "reserved", "available_to_promise", "short"}
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID+"/inventory", url.Values{"include": includes})
	require.NoError(t, err)
	require.Less(t, status, 500, "item inventory must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	inventory := parseJSON(body)
	assertObjectField(t, inventory, "item_inventory")

	for _, field := range includes {
		assertComputedQuantityHydrated(t, jsonObject(inventory, field), field)
	}
}

func TestItemInventory_FiguresAreNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+SeedItemID+"/inventory", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	inventory := parseJSON(body)
	for _, field := range []string{"on_hand", "reserved", "available_to_promise", "short"} {
		assertNilField(t, inventory, field)
	}
}

// --- Inventories list ---

// On-hand is the only figure this list reports, so it is not expandable: every row carries the
// quantity and the unit it is counted in. The endpoint accepts no includes at all.
func TestInventories_EveryRowCarriesAResolvedQuantityAndUnit(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(inventoriesPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "inventories list must not 5xx")
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data, "the seeded account has inventory")

	for _, raw := range list.Data {
		row := parseJSON(raw)
		assertObjectField(t, row, "inventory_item")

		item := jsonObject(row, "item")
		require.NotNil(t, item, "an inventory row always names its item: %s", string(raw))
		assertObjectField(t, item, "item")
		assert.NotEmpty(t, jsonField(item, "sku"))

		assertComputedQuantityHydrated(t, jsonObject(row, "quantity"), "quantity")
	}
}

// --- Deliveries ---

// The cost a delivery line was stocked at is a rate: a value, a readable form, and two units. All
// of it comes from the delivery query, so asking for the include returns the whole record — not an
// id with a blank display value.
func TestDeliveries_LineUnitCostExpandsFullyHydrated(t *testing.T) {
	t.Parallel()

	line, body := firstLineWith(t, deliveriesPath+"/"+SeedDeliveryID, "lines", "lines.unit_cost")
	assertRateHydrated(t, jsonObject(line, "unit_cost"), "lines.unit_cost")

	// The two units stay behind their own includes.
	unitCost := jsonObject(line, "unit_cost")
	assertNilField(t, unitCost, "numerator_unit")
	assertNilField(t, unitCost, "denominator_unit")
	_ = body
}

func TestDeliveries_LineUnitCostUnitsExpandWithInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, deliveriesPath+"/"+SeedDeliveryID,
		"lines", "lines.unit_cost", "lines.unit_cost.numerator_unit", "lines.unit_cost.denominator_unit")

	unitCost := jsonObject(line, "unit_cost")
	require.NotNil(t, unitCost)
	assertUnitHydrated(t, jsonObject(unitCost, "numerator_unit"), "lines.unit_cost.numerator_unit")
	assertUnitHydrated(t, jsonObject(unitCost, "denominator_unit"), "lines.unit_cost.denominator_unit")
	assert.Equal(t, "currency", jsonField(jsonObject(unitCost, "numerator_unit"), "type"),
		"a unit cost is priced in a currency")
}

// The quantity is always on the line; its unit is a record of its own and is asked for by name.
func TestDeliveries_LineQuantityUnitExpandsWithInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, deliveriesPath+"/"+SeedDeliveryID, "lines", "lines.quantity", "lines.quantity.unit")

	quantity := jsonObject(line, "quantity")
	assertQuantityHydrated(t, quantity, "lines.quantity")
	assertUnitHydrated(t, jsonObject(quantity, "unit"), "lines.quantity.unit")
}

func TestDeliveries_LineQuantityUnitIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, deliveriesPath+"/"+SeedDeliveryID, "lines")

	quantity := jsonObject(line, "quantity")
	assertQuantityHydrated(t, quantity, "lines.quantity")
	assertNilField(t, quantity, "unit")
}

// --- Receiving orders ---

// What the purchase order asked for is the order line's own quantity, reported under that
// quantity's id. A synthesized id would look real and resolve to nothing.
func TestReceivingOrders_LineQuantityOrderedIsThePurchaseOrderQuantity(t *testing.T) {
	t.Parallel()

	line, body := firstLineWith(t, receivingOrdersPath+"/"+SeedReceivingOrderID,
		"lines", "lines.quantity_ordered", "lines.quantity_ordered.unit", "related", "related.purchase_order")

	ordered := jsonObject(line, "quantity_ordered")
	assertQuantityHydrated(t, ordered, "lines.quantity_ordered")
	assertUnitHydrated(t, jsonObject(ordered, "unit"), "lines.quantity_ordered.unit")

	orderedID := jsonField(ordered, "id")
	assert.NotContains(t, orderedID, "_ordered", "the ordered quantity is the order line's own row, not a made-up id")

	// The same quantity, reached the other way: through the purchase order the receiving order was
	// created from. Both routes must name the same row.
	related := jsonObject(parseJSON(body), "related")
	if related == nil {
		t.Skip("this receiving order carries no related purchase order")
	}
	purchaseOrder := jsonObject(related, "purchase_order")
	if purchaseOrder == nil {
		t.Skip("this receiving order carries no related purchase order")
	}

	poLine, _ := firstLineWith(t, purchaseOrdersPath+"/"+jsonField(purchaseOrder, "id"), "lines", "lines.quantity_ordered")
	assert.Equal(t, jsonField(jsonObject(poLine, "quantity_ordered"), "id"), orderedID,
		"the receiving line reports the purchase order line's quantity, not a copy of its value")
}

func TestReceivingOrders_LineQuantityUnitExpandsWithInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, receivingOrdersPath+"/"+SeedReceivingOrderID, "lines", "lines.quantity", "lines.quantity.unit")

	quantity := jsonObject(line, "quantity")
	assertQuantityHydrated(t, quantity, "lines.quantity")
	assertUnitHydrated(t, jsonObject(quantity, "unit"), "lines.quantity.unit")
}

// The rejected figure is summed across the line's deliveries, so when it is present it is a
// computed quantity carrying its unit rather than a quantity with an invented id.
func TestReceivingOrders_LineRejectedQuantityIsComputed(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(receivingOrdersPath+"/"+SeedReceivingOrderID,
		url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonListData(parseJSON(body), "lines")
	require.NotEmpty(t, lines)

	for _, raw := range lines {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		rejected := jsonObject(line, "rejected_quantity")
		if rejected == nil {
			continue
		}
		assertComputedQuantityHydrated(t, rejected, "lines.rejected_quantity")
	}
}

// --- Purchase orders ---

func TestPurchaseOrders_LineQuantityAndRatesAreHydrated(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, purchaseOrdersPath+"/"+SeedPurchaseOrderID, "lines", "lines.quantity_ordered", "lines.unit_price")

	assertQuantityHydrated(t, jsonObject(line, "quantity_ordered"), "lines.quantity_ordered")
	assertRateHydrated(t, jsonObject(line, "unit_price"), "lines.unit_price")
}

func TestPurchaseOrders_LineUnitPriceUnitsExpandWithInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, purchaseOrdersPath+"/"+SeedPurchaseOrderID,
		"lines", "lines.unit_price", "lines.unit_price.numerator_unit", "lines.unit_price.denominator_unit")

	unitPrice := jsonObject(line, "unit_price")
	require.NotNil(t, unitPrice)
	assertUnitHydrated(t, jsonObject(unitPrice, "numerator_unit"), "lines.unit_price.numerator_unit")
	assertUnitHydrated(t, jsonObject(unitPrice, "denominator_unit"), "lines.unit_price.denominator_unit")
}

func TestPurchaseOrders_LineQuantityOrderedUnitExpandsWithInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, purchaseOrdersPath+"/"+SeedPurchaseOrderID,
		"lines", "lines.quantity_ordered", "lines.quantity_ordered.unit")

	assertUnitHydrated(t, jsonObject(jsonObject(line, "quantity_ordered"), "unit"), "lines.quantity_ordered.unit")
}

// Nothing on a line is half-resolved when its include was not asked for.
func TestPurchaseOrders_LineSubUnitsAreNullWithoutInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, purchaseOrdersPath+"/"+SeedPurchaseOrderID, "lines")

	assertNilField(t, line, "item")
	assertNilField(t, jsonObject(line, "quantity_ordered"), "unit")
	assertNilField(t, jsonObject(line, "unit_price"), "numerator_unit")
	assertNilField(t, jsonObject(line, "unit_price"), "denominator_unit")
}

// What has been received against a line is rolled up from the receiving order, so when it is
// present it is computed and carries its own unit.
func TestPurchaseOrders_LineQuantityReceivedIsComputed(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(purchaseOrdersPath+"/"+SeedPurchaseOrderID, url.Values{"include": {"lines"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	lines := jsonListData(parseJSON(body), "lines")
	require.NotEmpty(t, lines)

	for _, raw := range lines {
		line, ok := raw.(map[string]any)
		require.True(t, ok)
		received := jsonObject(line, "quantity_received")
		if received == nil {
			continue
		}
		assertComputedQuantityHydrated(t, received, "lines.quantity_received")
	}
}

func TestPurchaseOrders_LineItemExpandsFullyHydrated(t *testing.T) {
	t.Parallel()

	line, body := firstLineWith(t, purchaseOrdersPath+"/"+SeedPurchaseOrderID, "lines", "lines.item")

	item := jsonObject(line, "item")
	if item == nil {
		t.Skip("this purchase order line is not linked to a catalog item")
	}
	assertObjectField(t, item, "item")
	assert.NotEmpty(t, jsonField(item, "id"))
	assert.NotEmpty(t, jsonField(item, "sku"))
	assert.NotEmpty(t, jsonField(item, "type"), "a resolved item names its type: %s", string(body))
	assertValidTimestamp(t, jsonField(item, "created_at"), "lines.item.created_at")
}

// --- Suppliers ---

// A supplier's default addresses are the same records whether the caller listed suppliers or
// retrieved one. The list used to offer no includes at all, so they were unreachable from it.
func TestSuppliers_ListAddressesExpandWithInclude(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(suppliersPath, url.Values{
		"include": {"bill_to_address", "ship_to_address"},
		"limit":   {"5"},
	})
	require.NoError(t, err)
	require.Less(t, status, 500, "suppliers list must not 5xx")
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data, "the seeded account has suppliers")

	var sawAddress bool
	for _, raw := range list.Data {
		row := parseJSON(raw)
		assertObjectField(t, row, "supplier")

		address := jsonObject(row, "bill_to_address")
		if address == nil {
			continue
		}
		sawAddress = true
		assertObjectField(t, address, "address")
		assert.NotEmpty(t, jsonField(address, "id"), "an expanded address is a whole record")
		assertValidTimestamp(t, jsonField(address, "created_at"), "bill_to_address.created_at")
	}
	assert.True(t, sawAddress, "at least one seeded supplier has a default billing address: %s",
		formatListDataForLog(list.Data))
}

func TestSuppliers_ListAddressesAreNullWithoutInclude(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(suppliersPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)

	for _, raw := range list.Data {
		row := parseJSON(raw)
		assertNilField(t, row, "bill_to_address")
		assertNilField(t, row, "ship_to_address")
	}
}

// A supplier named from a purchase order carries its identity but not its record, so the
// timestamps are nullable. Listed on its own it is a record, and they are set.
func TestSuppliers_ListRowsCarryTheirTimestamps(t *testing.T) {
	t.Parallel()

	list, status, err := apiClient.GetList(suppliersPath, url.Values{"limit": {"5"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, nil)
	require.NotEmpty(t, list.Data)

	for _, raw := range list.Data {
		row := parseJSON(raw)
		assertValidTimestamp(t, jsonField(row, "created_at"), "created_at")
		assertValidTimestamp(t, jsonField(row, "updated_at"), "updated_at")
	}
}

func TestSuppliers_ListRejectsAnUnknownInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(suppliersPath, url.Values{"include": {"not_a_real_field"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "an unknown include is a client error, not a crash: %s", string(body))
	assert.Equal(t, 400, status, "include only accepts the documented keys")
	assert.Contains(t, string(body), "bill_to_address", "the error names what is allowed")
}

// assertItemHydrated asserts an item arrived as a whole catalog record rather than the id and SKU
// the referring document happened to know. `type` and the timestamps are the tell: a document join
// carries neither, so a presenter that builds the item itself leaves them blank.
func assertItemHydrated(t *testing.T, item map[string]any, where string) {
	t.Helper()

	require.NotNil(t, item, "%s must be present", where)
	assertObjectField(t, item, "item")
	assert.NotEmpty(t, jsonField(item, "id"), "%s.id", where)
	assert.NotEmpty(t, jsonField(item, "sku"), "%s.sku", where)
	assert.NotEmpty(t, jsonField(item, "type"), "%s.type — required, and absent from every document join", where)
	assertValidTimestamp(t, jsonField(item, "created_at"), where+".created_at")
	assertValidTimestamp(t, jsonField(item, "updated_at"), where+".updated_at")

	// The item's own expandables were not asked for, so they stay null.
	for _, sub := range []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes"} {
		assertNilField(t, item, sub)
	}
}

// --- Invoices ---

// An invoice line names what it billed. The invoice query knows only the item's id and SKU, so the
// line used to ship an item with a blank `type` and zero timestamps — and shipped it whether or not
// anyone asked. It is an include now, resolved through the item loader.
func TestInvoices_LineItemExpandsFullyHydrated(t *testing.T) {
	t.Parallel()

	line, body := firstLineWith(t, invoicesPath+"/"+SeedInvoiceID, "lines", "lines.item")

	item := jsonObject(line, "item")
	require.NotNil(t, item, "the seeded invoice line bills a catalog item: %s", string(body))
	assertItemHydrated(t, item, "lines.item")
}

func TestInvoices_LineItemIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, invoicesPath+"/"+SeedInvoiceID, "lines")
	assertNilField(t, line, "item")
}

// The quantity billed is a stored row, so it carries an id; the unit it is counted in is a record
// of its own and is reachable through it.
func TestInvoices_LineQuantityUnitExpandsWithInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, invoicesPath+"/"+SeedInvoiceID, "lines", "lines.quantity", "lines.quantity.unit")

	quantity := jsonObject(line, "quantity")
	assertQuantityHydrated(t, quantity, "lines.quantity")
	assertUnitHydrated(t, jsonObject(quantity, "unit"), "lines.quantity.unit")
}

func TestInvoices_LineQuantityUnitIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, invoicesPath+"/"+SeedInvoiceID, "lines")
	assertNilField(t, jsonObject(line, "quantity"), "unit")
}

// A price is currency over an item unit. Both sides are reachable, so the line can print "$/kg"
// without a second request.
func TestInvoices_LineUnitPriceUnitsExpandWithInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, invoicesPath+"/"+SeedInvoiceID,
		"lines", "lines.unit_price", "lines.unit_price.numerator_unit", "lines.unit_price.denominator_unit")

	unitPrice := jsonObject(line, "unit_price")
	assertRateHydrated(t, unitPrice, "lines.unit_price")
	assertUnitHydrated(t, jsonObject(unitPrice, "numerator_unit"), "lines.unit_price.numerator_unit")
	assertUnitHydrated(t, jsonObject(unitPrice, "denominator_unit"), "lines.unit_price.denominator_unit")
}

func TestInvoices_LineUnitPriceUnitsAreNullWithoutInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, invoicesPath+"/"+SeedInvoiceID, "lines")
	unitPrice := jsonObject(line, "unit_price")
	require.NotNil(t, unitPrice, "unit_price is always on the line")
	assertNilField(t, unitPrice, "numerator_unit")
	assertNilField(t, unitPrice, "denominator_unit")
}

// An allocation is an amount of money. The currency is a record, and the allocation only knew its
// id — so the amount arrived with a null unit and no include that would fill it.
func TestInvoices_AllocationAmountUnitExpandsWithInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+SeedInvoiceID,
		url.Values{"include": {"allocations", "allocations.amount", "allocations.amount.unit"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "invoice retrieve must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	allocations := jsonListData(parseJSON(body), "allocations")
	if len(allocations) == 0 {
		t.Skip("the seeded invoice has no allocations")
	}
	for _, raw := range allocations {
		allocation, ok := raw.(map[string]any)
		require.True(t, ok)
		assertObjectField(t, allocation, "invoice_allocation")

		amount := jsonObject(allocation, "amount")
		assertQuantityHydrated(t, amount, "allocations.amount")
		assertUnitHydrated(t, jsonObject(amount, "unit"), "allocations.amount.unit")
	}
}

// --- Settlements and transactions ---

// The same allocation amount, reached from the two other documents that carry it.
func TestSettlements_AllocationAmountUnitExpandsWithInclude(t *testing.T) {
	t.Parallel()

	assertAllocationAmountsCarryTheirUnit(t, settlementsPath+"/"+SeedSettlementID, "transaction_allocation")
}

func TestTransactions_AllocationAmountUnitExpandsWithInclude(t *testing.T) {
	t.Parallel()

	assertAllocationAmountsCarryTheirUnit(t, transactionsPath+"/"+SeedTransactionID, "transaction_allocation")
}

func assertAllocationAmountsCarryTheirUnit(t *testing.T, path, objectType string) {
	t.Helper()

	status, body, err := apiClient.GetListRaw(path,
		url.Values{"include": {"allocations", "allocations.amount", "allocations.amount.unit"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "%s must not 5xx: %s", path, string(body))
	requireStatus(t, 200, status, body)

	allocations := jsonListData(parseJSON(body), "allocations")
	if len(allocations) == 0 {
		t.Skipf("%s has no allocations", path)
	}
	for _, raw := range allocations {
		allocation, ok := raw.(map[string]any)
		require.True(t, ok)
		assertObjectField(t, allocation, objectType)

		amount := jsonObject(allocation, "amount")
		assertQuantityHydrated(t, amount, "allocations.amount")
		assertUnitHydrated(t, jsonObject(amount, "unit"), "allocations.amount.unit")
	}
}

// --- Production steps ---

// A step's output and its inputs both name catalog items. The production query carries each item's
// SKU but not its type or its own timestamps, so the step used to report its own timestamps as the
// item's — a plausible-looking date that belongs to a different record.
func TestProductionSteps_ProducedItemExpandsFullyHydrated(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+SeedProductionStepID,
		url.Values{"include": {"production", "production.produced_item"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "production step retrieve must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	production := jsonObject(parseJSON(body), "production")
	require.NotNil(t, production, "the step was asked for its production: %s", string(body))
	assertItemHydrated(t, jsonObject(production, "produced_item"), "production.produced_item")
}

func TestProductionSteps_ProducedItemIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+SeedProductionStepID,
		url.Values{"include": {"production"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	production := jsonObject(parseJSON(body), "production")
	require.NotNil(t, production)
	assertNilField(t, production, "produced_item")
}

func TestProductionSteps_ConsumedItemExpandsFullyHydrated(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+SeedProductionStepID,
		url.Values{"include": {"consumptions", "consumptions.consumed_item"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "production step retrieve must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	consumptions := jsonListData(parseJSON(body), "consumptions")
	if len(consumptions) == 0 {
		t.Skip("the seeded production step consumes nothing")
	}
	for _, raw := range consumptions {
		consumption, ok := raw.(map[string]any)
		require.True(t, ok)
		assertObjectField(t, consumption, "consumption")
		assertItemHydrated(t, jsonObject(consumption, "consumed_item"), "consumptions.consumed_item")
	}
}

func TestProductionSteps_ConsumedItemIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+SeedProductionStepID,
		url.Values{"include": {"consumptions"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, raw := range jsonListData(parseJSON(body), "consumptions") {
		consumption, ok := raw.(map[string]any)
		require.True(t, ok)
		assertNilField(t, consumption, "consumed_item")
	}
}

// --- List/retrieve include parity ---

// A line is the same object whether it arrived from a list or a retrieve, so the two endpoints
// offer the same reach into it. The list used to stop at `lines`, which meant a caller that wanted
// the item behind each line had to retrieve every order one at a time.
func TestPurchaseOrders_ListOffersTheSameLineIncludesAsRetrieve(t *testing.T) {
	t.Parallel()

	includes := []string{
		"lines", "lines.item",
		"lines.quantity_ordered", "lines.quantity_ordered.unit",
		"lines.unit_price", "lines.unit_price.numerator_unit", "lines.unit_price.denominator_unit",
	}

	status, body, err := apiClient.GetListRaw(purchaseOrdersPath, url.Values{"include": includes, "limit": {"5"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "purchase order list must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	var sawLine bool
	// The response is itself the list, so its rows are read straight off `data`.
	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		require.True(t, ok)

		for _, rawLine := range jsonListData(row, "lines") {
			line, ok := rawLine.(map[string]any)
			require.True(t, ok)
			sawLine = true

			if item := jsonObject(line, "item"); item != nil {
				assertItemHydrated(t, item, "lines.item")
			}

			quantity := jsonObject(line, "quantity_ordered")
			assertQuantityHydrated(t, quantity, "lines.quantity_ordered")
			assertUnitHydrated(t, jsonObject(quantity, "unit"), "lines.quantity_ordered.unit")

			unitPrice := jsonObject(line, "unit_price")
			assertRateHydrated(t, unitPrice, "lines.unit_price")
			assertUnitHydrated(t, jsonObject(unitPrice, "numerator_unit"), "lines.unit_price.numerator_unit")
			assertUnitHydrated(t, jsonObject(unitPrice, "denominator_unit"), "lines.unit_price.denominator_unit")
		}
	}
	require.True(t, sawLine, "the seeded account has purchase orders with lines: %s", string(body))
}

// --- Allocation transactions ---

// assertTransactionHydrated asserts an allocation's transaction arrived as the real record. The
// allocation knows only the transaction's id, so `number`, `amount` and the type are the tell: a
// presenter that builds the transaction itself leaves every one of them blank.
func assertTransactionHydrated(t *testing.T, tx map[string]any, where string) {
	t.Helper()

	require.NotNil(t, tx, "%s must be present", where)
	assertObjectField(t, tx, "transaction")
	assert.NotEmpty(t, jsonField(tx, "id"), "%s.id", where)
	assert.NotEmpty(t, jsonField(tx, "number"), "%s.number — required, and absent from the allocation row", where)
	assert.NotEmpty(t, jsonField(tx, "is_fully_allocated"), "%s.is_fully_allocated", where)
	assertValidTimestamp(t, jsonField(tx, "created_at"), where+".created_at")
	assertValidTimestamp(t, jsonField(tx, "updated_at"), where+".updated_at")

	assertQuantityHydrated(t, jsonObject(tx, "amount"), where+".amount")

	txType := jsonObject(tx, "transaction_type")
	require.NotNil(t, txType, "%s.transaction_type is required", where)
	assertObjectField(t, txType, "transaction_type")
	assert.NotEmpty(t, jsonField(txType, "code"), "%s.transaction_type.code", where)

	// The transaction's own expandables were not asked for, so they stay null.
	assertNilField(t, tx, "customer")
	assertNilField(t, tx, "responsible_user")
	assertNilField(t, tx, "allocations")
}

// A settlement's allocations each point at the transaction the money came from. The allocation row
// carries only that id, so the transaction is an include — it used to arrive as an id with a blank
// number and no amount, and no include could fill it because the loader was a stub that errored.
func TestSettlements_AllocationTransactionExpandsFullyHydrated(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(settlementsPath+"/"+SeedSettlementID,
		url.Values{"include": {"allocations", "allocations.transaction"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "settlement retrieve must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	allocations := jsonListData(parseJSON(body), "allocations")
	require.NotEmpty(t, allocations, "the seeded settlement has allocations: %s", string(body))
	for _, raw := range allocations {
		allocation, ok := raw.(map[string]any)
		require.True(t, ok)
		assertTransactionHydrated(t, jsonObject(allocation, "transaction"), "allocations.transaction")
	}
}

func TestSettlements_AllocationTransactionIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(settlementsPath+"/"+SeedSettlementID,
		url.Values{"include": {"allocations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	allocations := jsonListData(parseJSON(body), "allocations")
	require.NotEmpty(t, allocations)
	for _, raw := range allocations {
		allocation, ok := raw.(map[string]any)
		require.True(t, ok)
		assertNilField(t, allocation, "transaction")
	}
}

// The same field on the other allocation shape, reached from an invoice.
func TestInvoices_AllocationTransactionExpandsFullyHydrated(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+SeedInvoiceID,
		url.Values{"include": {"allocations", "allocations.transaction"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "invoice retrieve must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	allocations := jsonListData(parseJSON(body), "allocations")
	if len(allocations) == 0 {
		t.Skip("the seeded invoice has no allocations")
	}
	for _, raw := range allocations {
		allocation, ok := raw.(map[string]any)
		require.True(t, ok)
		assertTransactionHydrated(t, jsonObject(allocation, "transaction"), "allocations.transaction")
	}
}

func TestInvoices_AllocationTransactionIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(invoicesPath+"/"+SeedInvoiceID, url.Values{"include": {"allocations"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, raw := range jsonListData(parseJSON(body), "allocations") {
		allocation, ok := raw.(map[string]any)
		require.True(t, ok)
		assertNilField(t, allocation, "transaction")
	}
}

// Reaching through the transaction to the currency its amount is counted in — the transaction is
// loaded here rather than presented, so this proves the loader stashes what its own sub-objects need.
func TestSettlements_AllocationTransactionAmountUnitExpands(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(settlementsPath+"/"+SeedSettlementID,
		url.Values{"include": {"allocations", "allocations.transaction", "allocations.transaction.amount", "allocations.transaction.amount.unit"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "settlement retrieve must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	allocations := jsonListData(parseJSON(body), "allocations")
	require.NotEmpty(t, allocations)
	for _, raw := range allocations {
		allocation, ok := raw.(map[string]any)
		require.True(t, ok)
		amount := jsonObject(jsonObject(allocation, "transaction"), "amount")
		assertQuantityHydrated(t, amount, "allocations.transaction.amount")
		assertUnitHydrated(t, jsonObject(amount, "unit"), "allocations.transaction.amount.unit")
	}
}

// --- Attribute properties ---

// An attribute names the property it belongs to, and the discount query knows only that property's
// id. The property used to arrive as an id with a blank name and zero timestamps — unusable for the
// "Color: Beige" label it exists to render.
func TestVolumeDiscounts_AttributePropertiesAreResolved(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(volumeDiscountsPath, url.Values{"include": {"attributes"}, "limit": {"5"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "volume discounts must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	var sawProperty bool
	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		require.True(t, ok)

		for _, rawAttr := range jsonListData(row, "attributes") {
			attr, ok := rawAttr.(map[string]any)
			require.True(t, ok)
			assertObjectField(t, attr, "attribute")

			property := jsonObject(attr, "property")
			if property == nil {
				continue
			}
			sawProperty = true
			assertObjectField(t, property, "property")
			assert.NotEmpty(t, jsonField(property, "id"), "property.id")
			assert.NotEmpty(t, jsonField(property, "name"), "property.name — this is the label the attribute renders under")
			assertValidTimestamp(t, jsonField(property, "created_at"), "property.created_at")
			assertValidTimestamp(t, jsonField(property, "updated_at"), "property.updated_at")
		}
	}
	require.True(t, sawProperty, "the seeded account has a volume discount whose attribute names a property: %s", string(body))
}

// --- Agent tools ---

// A tool is granted to an agent by slug, and its display metadata comes from the code catalog. A
// grant whose slug the catalog no longer carries used to be reported anyway, with a blank required
// `name` and an empty schema — a row the terminal renders as an unnamed tool.
func TestAgents_ToolsAreNamed(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw("/v1/ai/agents", url.Values{"include": {"tools"}, "limit": {"10"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "agents must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	var sawTool bool
	for _, raw := range jsonArray(parseJSON(body), "data") {
		row, ok := raw.(map[string]any)
		require.True(t, ok)

		for _, rawTool := range jsonListData(row, "tools") {
			entry, ok := rawTool.(map[string]any)
			require.True(t, ok)
			assertObjectField(t, entry, "agent_definition_tool")

			tool := jsonObject(entry, "tool")
			require.NotNil(t, tool, "an agent tool always names the tool it grants")
			sawTool = true
			assertObjectField(t, tool, "available_tool")
			assert.NotEmpty(t, jsonField(tool, "slug"), "tool.slug")
			assert.NotEmpty(t, jsonField(tool, "name"), "tool.name — required, and blank for a slug the catalog does not carry")
			assert.NotEmpty(t, jsonField(tool, "category"), "tool.category")
		}
	}
	require.True(t, sawTool, "the seeded account has an agent with a granted tool: %s", string(body))
}

// --- Batches ---

// A batch names the records it points at — the item it produces, the station it sits at, the step
// and department it belongs to, the machines that made it. The batch query carries an id and a
// label for each, so they are references; they used to be full records with every other required
// field blank.
func TestBatches_ReferencesAreNamedNotHalfBuilt(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(scanningStationsPath+"/"+SeedScanningStationID+"/batches",
		url.Values{"limit": {"5"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "batches must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	var sawBatch bool
	for _, raw := range jsonArray(parseJSON(body), "data") {
		batch, ok := raw.(map[string]any)
		require.True(t, ok)
		assertObjectField(t, batch, "batch")
		sawBatch = true

		for _, key := range []string{"item", "scanning_station", "department", "production_step"} {
			ref := jsonObject(batch, key)
			if ref == nil {
				continue
			}
			assertObjectField(t, ref, "entity")
			assert.NotEmpty(t, jsonField(ref, "id"), "%s.id", key)
			assert.NotEmpty(t, jsonField(ref, "type"), "%s.type names what the reference points at", key)
		}
	}
	require.True(t, sawBatch, "the seeded scanning station has batches: %s", string(body))
}

// The three measures are stored rows, so the unit each is counted in is reachable through them.
func TestBatches_MeasureUnitsExpandWithInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(scanningStationsPath+"/"+SeedScanningStationID+"/batches",
		url.Values{"limit": {"5"}, "include": {"quantity.unit"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "batches must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	for _, raw := range jsonArray(parseJSON(body), "data") {
		batch, ok := raw.(map[string]any)
		require.True(t, ok)
		quantity := jsonObject(batch, "quantity")
		if quantity == nil {
			continue
		}
		assertUnitHydrated(t, jsonObject(quantity, "unit"), "quantity.unit")
	}
}

func TestBatches_MeasureUnitIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(scanningStationsPath+"/"+SeedScanningStationID+"/batches",
		url.Values{"limit": {"5"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, raw := range jsonArray(parseJSON(body), "data") {
		batch, ok := raw.(map[string]any)
		require.True(t, ok)
		if quantity := jsonObject(batch, "quantity"); quantity != nil {
			assertNilField(t, quantity, "unit")
		}
	}
}

// --- The purchase order line a receipt was raised from ---

// assertPurchaseOrderLineHydrated asserts an order line arrived as a whole record. The receiving and
// delivery lines that name it carry only its id, so `product_sku`, the ordered quantity and the
// agreed unit price are the tell: a presenter that builds the line itself leaves them blank.
func assertPurchaseOrderLineHydrated(t *testing.T, line map[string]any, where string) {
	t.Helper()

	require.NotNil(t, line, "%s must be present", where)
	assertObjectField(t, line, "purchase_order_line")
	assert.NotEmpty(t, jsonField(line, "id"), "%s.id", where)
	assert.NotEmpty(t, jsonField(line, "product_sku"), "%s.product_sku — required, and absent from the line that names it", where)
	assertValidTimestamp(t, jsonField(line, "created_at"), where+".created_at")
	assertValidTimestamp(t, jsonField(line, "updated_at"), where+".updated_at")

	assertQuantityHydrated(t, jsonObject(line, "quantity_ordered"), where+".quantity_ordered")
	assertRateHydrated(t, jsonObject(line, "unit_price"), where+".unit_price")

	// A purchase line records one agreed price, and that price is the cost — there is no separate
	// unit_cost on it.
	assert.NotContains(t, line, "unit_cost", "%s reports one price, not a price and a cost", where)

	// The line's own expandables were not asked for, so they stay null.
	assertNilField(t, line, "item")
}

// A receiving order line is raised from a purchase order line. It carries that line's item and
// ordered quantity directly; the line itself is an include, so the agreed price is one request away
// rather than a second round trip to the order.
func TestReceivingOrders_LineOrderLineExpandsFullyHydrated(t *testing.T) {
	t.Parallel()

	line, body := firstLineWith(t, receivingOrdersPath+"/"+SeedReceivingOrderID, "lines", "lines.order_line")
	orderLine := jsonObject(line, "order_line")
	require.NotNil(t, orderLine, "a receiving line is always raised from an order line: %s", string(body))
	assertPurchaseOrderLineHydrated(t, orderLine, "lines.order_line")
}

func TestReceivingOrders_LineOrderLineIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, receivingOrdersPath+"/"+SeedReceivingOrderID, "lines")
	assertNilField(t, line, "order_line")
}

// Reaching through the order line to the price's currency proves the loader stashes what its own
// sub-objects need, the same way the order endpoint does.
func TestReceivingOrders_LineOrderLineUnitPriceUnitsExpand(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, receivingOrdersPath+"/"+SeedReceivingOrderID,
		"lines", "lines.order_line", "lines.order_line.unit_price", "lines.order_line.unit_price.numerator_unit")

	unitPrice := jsonObject(jsonObject(line, "order_line"), "unit_price")
	assertRateHydrated(t, unitPrice, "lines.order_line.unit_price")
	assertUnitHydrated(t, jsonObject(unitPrice, "numerator_unit"), "lines.order_line.unit_price.numerator_unit")
}

// A delivery line reaches the same order line, one step further along: it is stocked against a
// receiving line, which was raised from the order line.
func TestDeliveries_LineOrderLineExpandsFullyHydrated(t *testing.T) {
	t.Parallel()

	line, body := firstLineWith(t, deliveriesPath+"/"+SeedDeliveryID, "lines", "lines.order_line")
	orderLine := jsonObject(line, "order_line")
	require.NotNil(t, orderLine, "a delivery line is always stocked against an order line: %s", string(body))
	assertPurchaseOrderLineHydrated(t, orderLine, "lines.order_line")
}

func TestDeliveries_LineOrderLineIsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, deliveriesPath+"/"+SeedDeliveryID, "lines")
	assertNilField(t, line, "order_line")
}

// The unit cost a delivery line records is the price its order line was agreed at — that is where
// the figure comes from at stocking time, so the two must line up.
func TestDeliveries_LineUnitCostMatchesTheOrderLinePrice(t *testing.T) {
	t.Parallel()

	line, _ := firstLineWith(t, deliveriesPath+"/"+SeedDeliveryID, "lines", "lines.unit_cost", "lines.order_line")

	unitCost := jsonObject(line, "unit_cost")
	orderLine := jsonObject(line, "order_line")
	require.NotNil(t, unitCost)
	require.NotNil(t, orderLine)

	assert.Equal(t, jsonField(jsonObject(orderLine, "unit_price"), "value"), jsonField(unitCost, "value"),
		"stocking costs the goods at the price the order line agreed")
}

// A purchase order line reports one agreed price. The cost it becomes is recorded on the delivery
// line when the goods are stocked, not carried as a second rate on the order.
func TestPurchaseOrders_LineReportsOnePriceNotAPriceAndACost(t *testing.T) {
	t.Parallel()

	line, body := firstLineWith(t, purchaseOrdersPath+"/"+SeedPurchaseOrderID, "lines")
	assert.NotContains(t, line, "unit_cost", "a purchase line's price is its cost: %s", string(body))
	assert.Contains(t, line, "unit_price", "the agreed price is always on the line")
}
