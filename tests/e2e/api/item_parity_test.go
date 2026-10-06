//go:build e2e

package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// Item endpoints the dashboard still reads from its own API: bulk reconcile, trends, the as-of
// inventory list, costing across steps written in different units, the items export, and the JSON
// body caps. Every figure is chosen so a conversion that never happens, or one that rounds, shows up
// as a wrong answer rather than a near one.
//
// Yarn is stocked by the pound (453.59237 g), with grains (0.06479891 g) in its unit group: 7,000
// grains are exactly a pound. Socks are stocked by the pair, with eaches and dozens in the group.

const (
	sockDozenUnitID = "un_01seeddozen00000000"
	unitEachID      = "each"
)

// --- Fixtures ---

// newSockPart creates a part stocked by the pair and returns its SKU and item id.
func newSockPart(t *testing.T, prefix string) (sku, itemID string) {
	t.Helper()
	sku = uniqueName(prefix)
	status, body, err := apiClient.Post(partsPath+"?include=item", validPartBody(sku), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	part := parseJSON(body)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + jsonField(part, "id")) })
	return sku, jsonField(jsonObject(part, "item"), "id")
}

// newSockProduct creates a sale product stocked by the pair, on the seeded Socks line when lined.
func newSockProduct(t *testing.T, prefix string, lined bool) (sku, itemID string) {
	t.Helper()
	sku = uniqueName(prefix)
	body := validProductBody(sku)
	if lined {
		body["product_line_id"] = SeedProductLineID
	}
	status, respBody, err := apiClient.Post(productsPath+"?include=item", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	product := parseJSON(respBody)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + jsonField(product, "id")) })
	return sku, jsonField(jsonObject(product, "item"), "id")
}

// inventoryLogIDs lists the item's inventory logs oldest first.
func inventoryLogIDs(t *testing.T, itemID string) []string {
	t.Helper()
	rows, err := authDB(t).Query("SELECT id FROM inventory_log WHERE item_id = ? ORDER BY created_at, id", itemID)
	require.NoError(t, err)
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, rows.Err())
	return ids
}

// logInventory adjusts the item's stock and returns the inventory log the movement wrote.
func logInventory(t *testing.T, itemID, value, unitID string) string {
	t.Helper()
	before := inventoryLogIDs(t, itemID)
	mustUpdateInventory(t, itemID, value, unitID, "adjust")
	after := inventoryLogIDs(t, itemID)
	require.Len(t, after, len(before)+1, "an adjustment writes one inventory log")
	return after[len(after)-1]
}

// backdateLogs moves every listed log to its instant, and every other log of the item to long
// before any window under test, so the item's history is exactly what the test describes.
func backdateLogs(t *testing.T, itemID string, at map[string]time.Time) {
	t.Helper()
	db := authDB(t)
	_, err := db.Exec("UPDATE inventory_log SET created_at = ? WHERE item_id = ?", time.Now().UTC().AddDate(-1, 0, 0), itemID)
	require.NoError(t, err)
	for id, when := range at {
		_, err := db.Exec("UPDATE inventory_log SET created_at = ? WHERE id = ?", when.UTC(), id)
		require.NoError(t, err)
	}
}

// postRawJSON posts body exactly as given. The client's Post marshals its body, which compacts any
// whitespace a test pads a body out with.
func postRawJSON(t *testing.T, path string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, apiClient.baseURL+path, bytes.NewReader(body))
	require.NoError(t, err)
	apiClient.applyAuth(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", newIdempotencyKey())
	resp, err := apiClient.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, respBody
}

// physicalLevel is what a reconcile measures against: on hand net of demand nothing has covered. A
// reconcile downward writes an issue, which on-hand only reflects once allocation draws it.
func physicalLevel(t *testing.T, itemID string) string {
	t.Helper()
	return readInventory(t, itemID).level().String()
}

func decimalOf(t *testing.T, raw string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(raw)
	require.NoError(t, err, "%q is not a decimal", raw)
	return d
}

func assertExactDecimal(t *testing.T, want, got string, msgAndArgs ...any) {
	t.Helper()
	assert.True(t, decimalOf(t, want).Equal(decimalOf(t, got)), append([]any{"want %s, got %s: ", want, got}, msgAndArgs...)...)
}

func reconciledRows(resp map[string]any) map[string][]map[string]any {
	out := map[string][]map[string]any{}
	for _, raw := range jsonListData(resp, "reconciled_items") {
		row := raw.(map[string]any)
		sku := jsonField(jsonObject(row, "item"), "name")
		out[sku] = append(out[sku], row)
	}
	return out
}

func errorRows(resp map[string]any) map[string]string {
	out := map[string]string{}
	for _, raw := range jsonListData(resp, "errors") {
		row := raw.(map[string]any)
		out[jsonField(jsonObject(row, "item"), "name")] = jsonField(row, "error")
	}
	return out
}

// --- Bulk reconcile: units ---

// A row is converted from its own unit into the item's base unit before it is applied, and the
// ledger records exactly that base-unit figure.
func TestItemParity_BulkReconcileBooksTheRowConvertedIntoTheBaseUnit(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)

	resp := bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "gr", "quantity": "3500"}}, "force")
	rows := reconciledRows(resp)[sku]
	require.Len(t, rows, 1, "%v", resp)
	assertExactDecimal(t, "0", jsonField(jsonObject(rows[0], "previous_quantity"), "value"))
	assertExactDecimal(t, "0.5", jsonField(jsonObject(rows[0], "new_quantity"), "value"), "3,500 grains is half a pound")
	assert.Equal(t, poundUnitID, jsonField(jsonObject(jsonObject(rows[0], "new_quantity"), "unit"), "id"),
		"reconciled figures are in the item's base unit")

	bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "GR", "quantity": "7000"}}, "addition")

	receipts := ledgerRows(t, itemID, "inventory_receipt")
	require.Len(t, receipts, 2, "one receipt per row: %s", ledgerDump(t, itemID))
	for i, want := range []string{"0.5", "1"} {
		assertExactDecimal(t, want, receipts[i].Value, "receipt %d is the row in pounds", i)
		assert.Equal(t, poundUnitID, receipts[i].UnitID, "receipt %d is booked in the base unit", i)
	}
	assertExactDecimal(t, "1.5", readInventory(t, itemID).onHand.String())

	// Forcing down to 2,100 grains (0.3 lb) issues exactly the difference.
	bulkReconcile(t, []map[string]any{{"sku": sku, "unit": "gr", "quantity": "2100"}}, "force")
	issues := ledgerRows(t, itemID, "inventory_issue")
	require.Len(t, issues, 1, "%s", ledgerDump(t, itemID))
	assertExactDecimal(t, "1.2", issues[0].Value)
	assert.Equal(t, poundUnitID, issues[0].UnitID)
	settles(t, "the issue draws on the receipts", func() bool { return readInventory(t, itemID).level().Equal(decimal.RequireFromString("0.3")) })
}

// Only the item's unit group says how its units relate. A count unit against a weighed item, or a
// weight the group does not carry, is an error row and writes nothing.
func TestItemParity_BulkReconcileUnitOutsideTheItemsGroupIsAnErrorRow(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)
	logsBefore := inventoryLogIDs(t, itemID)

	for _, unit := range []string{"ea", "pr", "g"} {
		resp := bulkReconcile(t, []map[string]any{{"sku": sku, "unit": unit, "quantity": "4"}}, "force")
		errs := errorRows(resp)
		require.Contains(t, errs, sku, "%s is outside the yarn group: %v", unit, resp)
		assert.Contains(t, errs[sku], "not in this item's unit group")
		assert.Empty(t, jsonListData(resp, "reconciled_items"))
	}
	assert.Empty(t, ledgerRows(t, itemID, "inventory_receipt"), "an errored row writes nothing")
	assert.Empty(t, ledgerRows(t, itemID, "inventory_issue"))
	assert.Equal(t, logsBefore, inventoryLogIDs(t, itemID), "nor any inventory log")
}

// --- Bulk reconcile: the report ---

// Every row lands in exactly one list, in its own terms.
func TestItemParity_BulkReconcileReportsEveryRowOnce(t *testing.T) {
	t.Parallel()

	inPounds, poundsItem := newReconcilableItem(t)
	inGrains, grainsItem := newReconcilableItem(t)
	badUnit, _ := newReconcilableItem(t)
	wrongGroup, _ := newReconcilableItem(t)
	socks, socksItem := newSockPart(t, "e2e-parity-rec-sock")
	missing := uniqueName("e2e-parity-rec-absent")

	resp := bulkReconcile(t, []map[string]any{
		{"sku": inPounds, "unit": "lbs", "quantity": "2.25"},
		{"sku": inGrains, "unit": "gr", "quantity": "14000"},
		{"sku": missing, "unit": "lbs", "quantity": "1"},
		{"sku": badUnit, "unit": "not-a-unit", "quantity": "1"},
		{"sku": wrongGroup, "unit": "ea", "quantity": "1"},
		{"sku": socks, "unit": "dz", "quantity": "2"},
	}, "force")

	reconciled := reconciledRows(resp)
	assert.Len(t, jsonListData(resp, "reconciled_items"), 3, "%v", resp)
	assertExactDecimal(t, "2.25", jsonField(jsonObject(reconciled[inPounds][0], "new_quantity"), "value"))
	assertExactDecimal(t, "2", jsonField(jsonObject(reconciled[inGrains][0], "new_quantity"), "value"))
	assertExactDecimal(t, "12", jsonField(jsonObject(reconciled[socks][0], "new_quantity"), "value"), "2 dz is 12 pairs")
	assert.Equal(t, SeedPairUnitID, jsonField(jsonObject(jsonObject(reconciled[socks][0], "new_quantity"), "unit"), "id"))

	skipped := jsonListData(resp, "skipped_items")
	require.Len(t, skipped, 1, "%v", resp)
	assert.Equal(t, missing, jsonField(skipped[0].(map[string]any), "sku"))

	errs := errorRows(resp)
	require.Len(t, errs, 2, "%v", resp)
	assert.Contains(t, errs[badUnit], "not found")
	assert.Contains(t, errs[wrongGroup], "not in this item's unit group")

	assertExactDecimal(t, "2.25", readInventory(t, poundsItem).onHand.String())
	assertExactDecimal(t, "2", readInventory(t, grainsItem).onHand.String())
	assertExactDecimal(t, "12", readInventory(t, socksItem).onHand.String())
}

// `addition` adds the converted row to the level; `force` replaces the level with it. Each reports
// the level it found and the level it left.
func TestItemParity_BulkReconcileAdditionAddsAndForceReplaces(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)

	step := func(reconcileType, unit, quantity, wantPrevious, wantNew string) {
		t.Helper()
		resp := bulkReconcile(t, []map[string]any{{"sku": sku, "unit": unit, "quantity": quantity}}, reconcileType)
		rows := reconciledRows(resp)[sku]
		require.Len(t, rows, 1, "%v", resp)
		assertExactDecimal(t, wantPrevious, jsonField(jsonObject(rows[0], "previous_quantity"), "value"), "%s %s %s previous", reconcileType, quantity, unit)
		assertExactDecimal(t, wantNew, jsonField(jsonObject(rows[0], "new_quantity"), "value"), "%s %s %s new", reconcileType, quantity, unit)
		assertExactDecimal(t, wantNew, physicalLevel(t, itemID))
	}

	step("force", "lbs", "10", "0", "10")
	step("addition", "gr", "7000", "10", "11")
	step("addition", "lbs", "-0.5", "11", "10.5")
	step("force", "gr", "3500", "10.5", "0.5")
	step("force", "lbs", "0.5", "0.5", "0.5")
}

// A SKU listed twice applies each row to what the row before it left, in both modes.
func TestItemParity_BulkReconcileRepeatedSKUAppliesInTurn(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)

	resp := bulkReconcile(t, []map[string]any{
		{"sku": sku, "unit": "lbs", "quantity": "5"},
		{"sku": sku, "unit": "lbs", "quantity": "2"},
	}, "force")
	rows := reconciledRows(resp)[sku]
	require.Len(t, rows, 2)
	assertExactDecimal(t, "5", jsonField(jsonObject(rows[1], "previous_quantity"), "value"))
	assertExactDecimal(t, "2", physicalLevel(t, itemID), "the second force wins, not both")

	bulkReconcile(t, []map[string]any{
		{"sku": sku, "unit": "lbs", "quantity": "1"},
		{"sku": sku, "unit": "gr", "quantity": "7000"},
	}, "addition")
	assertExactDecimal(t, "4", physicalLevel(t, itemID))
}

// --- Bulk reconcile: limits ---

func TestItemParity_BulkReconcileRefusesMoreThanTheRowCap(t *testing.T) {
	t.Parallel()

	rows := make([]map[string]any, 1001)
	for i := range rows {
		rows[i] = map[string]any{"sku": "E2E-ABSENT", "unit": "lbs", "quantity": "1"}
	}
	status, body, err := apiClient.Post(bulkReconcilePath, map[string]any{"data": rows, "reconcile_type": "force"}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "%s", string(body))
	requireStatus(t, 400, status, body)
	assertErrorParam(t, requireErrorResponse(t, body, "invalid_format", "invalid_request_error"), "data")
}

// An import past the default 1 MB is what the raised bulk cap is for: it is read whole and applied.
func TestItemParity_BulkReconcileOverOneMegabyteUnderTheBulkCapSucceeds(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)
	rows := []map[string]any{{"sku": sku, "unit": "gr", "quantity": "7000"}}
	padding := strings.Repeat("x", 1100)
	for i := range 999 {
		rows = append(rows, map[string]any{"sku": fmt.Sprintf("E2E-ABSENT-%04d-%s", i, padding), "unit": "lbs", "quantity": "1"})
	}
	payload, err := json.Marshal(map[string]any{"data": rows, "reconcile_type": "force"})
	require.NoError(t, err)
	require.Greater(t, len(payload), 1<<20, "the body must be over the default cap to prove anything")

	status, body, err := apiClient.Post(bulkReconcilePath, json.RawMessage(payload), newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "%.500s", string(body))
	requireStatus(t, 200, status, body)
	resp := parseJSON(body)
	assert.Len(t, jsonListData(resp, "reconciled_items"), 1)
	assert.Len(t, jsonListData(resp, "skipped_items"), 999)
	assertExactDecimal(t, "1", readInventory(t, itemID).onHand.String())
}

// --- JSON body caps ---

// A body over the cap is refused as too large — not truncated and reported as malformed JSON.
func TestItemParity_JSONBodyOverTheCapIs413(t *testing.T) {
	t.Parallel()

	padded := func(prefix string, size int) []byte {
		var b bytes.Buffer
		b.WriteString(prefix)
		b.Write(bytes.Repeat([]byte(" "), size))
		b.WriteString("}")
		return b.Bytes()
	}

	for _, tc := range []struct {
		name, path string
		body       []byte
		limit      string
	}{
		{"default cap", materialsPath, padded(`{"sku":"e2e-too-large","category_id":"`+SeedMaterialCategoryID+`"`, (1<<20)+16), "1 MB"},
		{"bulk cap", bulkReconcilePath, padded(`{"reconcile_type":"force","data":[]`, (8<<20)+16), "8 MB"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body := postRawJSON(t, tc.path, tc.body)
			require.Less(t, status, 500, "%s", string(body))
			requireStatus(t, http.StatusRequestEntityTooLarge, status, body)
			errObj := requireErrorResponse(t, body, "request_too_large", "invalid_request_error")
			assert.Contains(t, jsonField(errObj, "message"), tc.limit)
		})
	}
}

// The product, material and part imports carry descriptions and properties, and take the bulk cap.
func TestItemParity_BulkUpsertOverOneMegabyteIsAccepted(t *testing.T) {
	t.Parallel()

	sku := uniqueName("e2e-parity-big-upsert")
	var b bytes.Buffer
	b.WriteString(`{"materials":[{"sku":"` + sku + `","category":{"id":"` + SeedMaterialCategoryID + `"}}]`)
	b.Write(bytes.Repeat([]byte(" "), (1<<20)+64))
	b.WriteString("}")

	require.Greater(t, b.Len(), 1<<20)

	status, body := postRawJSON(t, materialsBulkUpsertPath, b.Bytes())
	require.Less(t, status, 500, "%.500s", string(body))
	requireStatus(t, http.StatusAccepted, status, body)

	var materialID string
	eventually(t, 30*time.Second, 500*time.Millisecond, func() error {
		list, _, err := apiClient.GetList(materialsPath, url.Values{"q": {sku}})
		if err != nil {
			return err
		}
		if len(list.Data) != 1 {
			return fmt.Errorf("material %s not written yet", sku)
		}
		materialID = DataItemField(list.Data[0], "id")
		return nil
	})
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + materialID) })
}

// --- Trends ---

// The chart pairs each date with a value, so there are exactly 30 points, one per UTC day. Each is
// the level the day closed on, quiet days carry forward, the series opens at the level logged before
// the window, and every log is converted from the unit its movement was written in.
func TestItemParity_TrendsAreThirtyClosingLevelsInTheBaseUnit(t *testing.T) {
	t.Parallel()

	_, itemID := newSockPart(t, "e2e-parity-trend")

	seed := logInventory(t, itemID, "24", unitEachID)        // 24 ea = 12 pr
	morning := logInventory(t, itemID, "1", sockDozenUnitID) // 36 ea, logged as 3 dz
	evening := logInventory(t, itemID, "4", SeedPairUnitID)  // 44 ea, logged as 22 pr
	recent := logInventory(t, itemID, "-8", unitEachID)      // 36 ea, logged as 36 ea
	today := time.Now().UTC().Truncate(24 * time.Hour)
	backdateLogs(t, itemID, map[string]time.Time{
		seed:    today.AddDate(0, 0, -40).Add(9 * time.Hour),
		morning: today.AddDate(0, 0, -10).Add(8 * time.Hour),
		evening: today.AddDate(0, 0, -10).Add(23*time.Hour + 59*time.Minute),
		recent:  today.AddDate(0, 0, -3).Add(12 * time.Hour),
	})

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID+"/trends", url.Values{"trend_type": {"inventory"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "%s", string(body))
	requireStatus(t, 200, status, body)
	trends := parseJSON(body)

	unit := jsonObject(trends, "unit")
	assertUnitHydrated(t, unit, "unit")
	assert.Equal(t, SeedPairUnitID, jsonField(unit, "id"), "values are in the category's base unit")

	points := jsonListData(trends, "points")
	require.Len(t, points, 30, "one point per day for 30 days: %s", string(body))
	start := today.AddDate(0, 0, -29)
	for i, raw := range points {
		point := raw.(map[string]any)
		day := start.AddDate(0, 0, i)
		occurred, err := time.Parse(time.RFC3339, jsonField(point, "occurred_at"))
		require.NoError(t, err)
		assert.True(t, occurred.Equal(day), "point %d is %s, want %s", i, occurred, day)

		want := "12" // the seed, carried forward
		switch {
		case !day.Before(today.AddDate(0, 0, -3)):
			want = "18"
		case !day.Before(today.AddDate(0, 0, -10)):
			want = "22" // the day closed on the evening log, not the morning one
		}
		assertExactDecimal(t, want, jsonField(point, "value"), "day %s", day.Format(time.DateOnly))
	}
}

// An item that has never moved reads zero for all 30 days, with its unit.
func TestItemParity_TrendsForAnItemNeverLoggedAreZero(t *testing.T) {
	t.Parallel()

	_, itemID := newSockPart(t, "e2e-parity-trend-flat")
	_, err := authDB(t).Exec("DELETE FROM inventory_log WHERE item_id = ?", itemID)
	require.NoError(t, err)

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID+"/trends", url.Values{"trend_type": {"inventory"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	trends := parseJSON(body)
	points := jsonListData(trends, "points")
	require.Len(t, points, 30)
	for _, raw := range points {
		assertExactDecimal(t, "0", jsonField(raw.(map[string]any), "value"))
	}
	assert.Equal(t, SeedPairUnitID, jsonField(jsonObject(trends, "unit"), "id"))
}

// --- Inventories as of a date ---

func inventoryRowAsOf(t *testing.T, sku string, asOf *time.Time, include ...string) map[string]any {
	t.Helper()
	params := url.Values{}
	if asOf != nil {
		params.Set("as_of", asOf.UTC().Format(time.RFC3339))
	}
	if len(include) > 0 {
		params["include"] = include
	}
	return inventoryRowForItem(t, sku, params)
}

// As of a past instant, an item reports the last level logged by then, converted into its base unit;
// an item with nothing logged by then reports zero.
func TestItemParity_InventoriesAsOfReportTheLevelLoggedByThen(t *testing.T) {
	t.Parallel()

	sku, itemID := newSockPart(t, "e2e-parity-asof")
	first := logInventory(t, itemID, "24", unitEachID)      // 12 pr
	second := logInventory(t, itemID, "1", sockDozenUnitID) // 36 ea, logged as 3 dz = 18 pr
	// Whole seconds, because as_of travels as RFC 3339 and a log a fraction after it is not "at" it.
	now := time.Now().UTC().Truncate(time.Second)
	backdateLogs(t, itemID, map[string]time.Time{
		first:  now.Add(-5 * 24 * time.Hour),
		second: now.Add(-2 * 24 * time.Hour),
	})

	for _, tc := range []struct {
		name string
		asOf time.Time
		want string
	}{
		{"before any log", now.Add(-10 * 24 * time.Hour), "0"},
		{"between the logs", now.Add(-3 * 24 * time.Hour), "12"},
		{"exactly at the second log", now.Add(-2 * 24 * time.Hour), "18"},
		{"after both", now.Add(-time.Hour), "18"},
	} {
		row := inventoryRowAsOf(t, sku, &tc.asOf)
		quantity := jsonObject(row, "quantity")
		assertExactDecimal(t, tc.want, jsonField(quantity, "value"), tc.name)
		assert.Equal(t, SeedPairUnitID, jsonField(jsonObject(quantity, "unit"), "id"), "%s: in the base unit", tc.name)
	}

	row := inventoryRowAsOf(t, sku, nil)
	assertExactDecimal(t, "18", jsonField(jsonObject(row, "quantity"), "value"), "without as_of the figure is current on-hand")
}

func TestItemParity_InventoriesAsOfForAnItemNeverLoggedIsZero(t *testing.T) {
	t.Parallel()

	sku, itemID := newReconcilableItem(t)
	_, err := authDB(t).Exec("DELETE FROM inventory_log WHERE item_id = ?", itemID)
	require.NoError(t, err)

	asOf := time.Now().UTC()
	row := inventoryRowAsOf(t, sku, &asOf)
	quantity := jsonObject(row, "quantity")
	assertExactDecimal(t, "0", jsonField(quantity, "value"))
	assert.Equal(t, poundUnitID, jsonField(jsonObject(quantity, "unit"), "id"))
}

func TestItemParity_InventoriesRejectAMalformedAsOf(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(inventoriesPath, url.Values{"as_of": {"last tuesday"}})
	require.NoError(t, err)
	require.Less(t, status, 500, "%s", string(body))
	requireStatus(t, 400, status, body)
}

// The export shows each item's product line; it is an include, null until asked for and null for an
// item that sells under none.
func TestItemParity_InventoriesProductLineIsAnInclude(t *testing.T) {
	t.Parallel()

	linedSKU, _ := newSockProduct(t, "e2e-parity-lined", true)
	unlinedSKU, _ := newReconcilableItem(t)

	assertNilField(t, inventoryRowAsOf(t, linedSKU, nil), "product_line")

	line := jsonObject(inventoryRowAsOf(t, linedSKU, nil, "product_line"), "product_line")
	require.NotNil(t, line, "the line expands when asked for")
	assertObjectField(t, line, "product_line")
	assert.Equal(t, SeedProductLineID, jsonField(line, "id"))
	assert.Equal(t, SeedProductLineName, jsonField(line, "name"))

	assertNilField(t, inventoryRowAsOf(t, unlinedSKU, nil, "product_line"), "product_line")

	asOf := time.Now().UTC()
	line = jsonObject(inventoryRowAsOf(t, linedSKU, &asOf, "product_line"), "product_line")
	require.NotNil(t, line, "as_of does not change what the row expands to")
	assert.Equal(t, SeedProductLineID, jsonField(line, "id"))
}

// --- Costing ---

// A knit step makes 12 eaches of a component from 12 eaches of a $0.10 packaging; a sew step makes a
// pair of the finished part from 2 dozen of the component. A pair takes two knit runs: $2.40 of
// material. Read as written, 2 over 12, it is a sixth of a run — $0.20. Drawn as 24 eaches instead,
// the flow costs the same.
func TestItemParity_CostsScaleUpstreamStepsThroughTheirUnits(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, drawn, drawnUnit string
	}{
		{"component drawn in dozens", "2", sockDozenUnitID},
		{"component drawn in eaches", "24", unitEachID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := createLaborCostFixture(t)
			component := createPartItem(t, "e2e-parity-cost-component")
			finished := fixture.partItemID
			createZeroLaborStep(t, "e2e-parity-knit", component, "12", unitEachID, fixture.materialItemID, "12", unitEachID)
			createZeroLaborStep(t, "e2e-parity-sew", finished, "1", SeedPairUnitID, component, tc.drawn, tc.drawnUnit)

			costs := getItemCosts(t, finished)
			assertExactDecimal(t, "2.4", jsonField(costs, "direct_material_cost"), "material per pair")
			assertExactDecimal(t, "0", jsonField(costs, "direct_labor_cost"))
			assertExactDecimal(t, "0", jsonField(costs, "overhead_cost"))
			assertExactDecimal(t, "2.4", jsonField(costs, "total_cost"))
			assert.Equal(t, SeedPairUnitID, jsonField(jsonObject(costs, "denominator_unit"), "id"))
		})
	}
}

// createZeroLaborStep makes a step producing produced from consumed, priced by its material alone.
func createZeroLaborStep(t *testing.T, name, produced, producedQty, producedUnit, consumed, consumedQty, consumedUnit string) string {
	t.Helper()
	zeroRate := func(num, den string) map[string]any {
		return map[string]any{"value": "0", "numerator_unit_id": num, "denominator_unit_id": den}
	}
	status, body, err := apiClient.Post(productionStepsPath, map[string]any{
		"name":            uniqueName(name),
		"leveling_factor": "0",
		"allowances":      "0",
		"labor_time":      zeroRate(unitSecond, unitEach),
		"labor_rate":      zeroRate(unitDollar, unitHour),
		"overhead_rate":   zeroRate(unitDollar, unitHour),
		"production":      map[string]any{"item_id": produced, "quantity_value": producedQty, "quantity_unit_id": producedUnit},
		"consumptions": []map[string]any{{"item_id": consumed, "quantity_value": consumedQty, "quantity_unit_id": consumedUnit,
			"waste_quantity_value": "0", "waste_quantity_unit_id": consumedUnit}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { cleanupStepIDs(id) })
	return id
}

// A producing step that makes nothing cannot be costed per unit — the same not-found as no flow.
func TestItemParity_CostsForAZeroProductionQuantityAreNotFound(t *testing.T) {
	t.Parallel()

	fixture := createLaborCostFixture(t)
	step := createLinkedStep(t, "e2e-parity-zero", fixture.partItemID, fixture.materialItemID)
	_, err := authDB(t).Exec(`UPDATE quantity q JOIN production p ON p.quantity_id = q.id
		SET q.value = 0 WHERE p.production_step_id = ?`, step)
	require.NoError(t, err)

	status, body, err := apiClient.GetListRaw(itemsPath+"/"+fixture.partItemID+"/costs", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "a zero quantity is not a server error: %s", string(body))
	requireStatus(t, 404, status, body)
	requireErrorResponse(t, body, "resource_not_found", "invalid_request_error")
}

// --- Items export ---

type exportRow struct{ onHand, unit string }

func itemsExportRows(t *testing.T) map[string]exportRow {
	t.Helper()
	resp, err := apiClient.GetFull(itemsPath+"/actions/export", nil)
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 500, "%s", string(resp.Body))
	requireStatus(t, 200, resp.StatusCode, resp.Body)

	f, err := excelize.OpenReader(bytes.NewReader(resp.Body))
	require.NoError(t, err)
	defer f.Close()
	rows, err := f.GetRows("Items")
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	header := rows[0]
	col := func(name string) int {
		for i, h := range header {
			if h == name {
				return i
			}
		}
		t.Fatalf("export has no %q column: %v", name, header)
		return -1
	}
	sku, onHand, unit := col("SKU"), col("On Hand Qty"), col("Unit")
	out := map[string]exportRow{}
	for _, row := range rows[1:] {
		cell := func(i int) string {
			if i < len(row) {
				return row[i]
			}
			return ""
		}
		out[cell(sku)] = exportRow{onHand: cell(onHand), unit: cell(unit)}
	}
	return out
}

// The export's on-hand is the inventory list's figure in the category base unit, labelled by the
// unit's abbreviation; non-sale products are left out as the list leaves them out.
func TestItemParity_ItemsExportConvertsAndLabelsTheUnit(t *testing.T) {
	t.Parallel()

	sockSKU, sockItem := newSockPart(t, "e2e-parity-export-sock")
	mustUpdateInventory(t, sockItem, "24", unitEachID, "adjust")
	mustUpdateInventory(t, sockItem, "1", sockDozenUnitID, "adjust")
	yarnSKU, yarnItem := newReconcilableItem(t)
	mustUpdateInventory(t, yarnItem, "3500", grainUnitID, "adjust")

	serviceSKU := uniqueName("e2e-parity-export-service")
	body := validProductBody(serviceSKU)
	body["type"] = "service"
	status, respBody, err := apiClient.Post(productsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	t.Cleanup(func() { apiClient.Delete(productsPath + "/" + jsonField(parseJSON(respBody), "id")) })

	rows := itemsExportRows(t)
	require.Contains(t, rows, sockSKU)
	assertExactDecimal(t, "18", rows[sockSKU].onHand, "24 ea and a dozen are 18 pairs")
	assert.Equal(t, "pr", rows[sockSKU].unit, "the unit is labelled by abbreviation, not id")
	require.Contains(t, rows, yarnSKU)
	assertExactDecimal(t, "0.5", rows[yarnSKU].onHand)
	assert.Equal(t, "lbs", rows[yarnSKU].unit)
	assert.NotContains(t, rows, serviceSKU, "a non-sale product is not stock")
}
