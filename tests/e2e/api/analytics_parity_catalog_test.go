//go:build e2e

package api_test

import (
	"fmt"
	"math/rand/v2"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Weeks of sales, open batches and the demand forecast, each read through a product line, items and a
// scanning station made for the test, so the account's other activity cannot move the figures.

// --- Weeks of sales ---

func TestAnalyticsParityWeeksOfSales_StockAndSalesInTheLinesBaseUnit(t *testing.T) {
	t.Parallel()
	line := parityProductLine(t)
	sold1, _ := parityProduct(t, line)
	sold2, _ := parityProduct(t, line)
	stocked, stockedItem := parityProduct(t, line)
	customer := parityCustomer(t, "", line)

	// Sold this week: 4 pr and 1 dz (6 pr) = 10 pr.
	issueParityOrder(t, customer, "",
		parityLine(sold1, "4", SeedUnitID, "1.00"),
		parityLine(sold2, "1", seedDozenUnitID, "1.00"))
	// On hand: 3 pr and 2 dz (12 pr) = 15 pr, on an item since deleted, whose stock is still on the shelf.
	mustUpdateInventory(t, stockedItem, "3", SeedUnitID, "adjust")
	mustUpdateInventory(t, stockedItem, "2", seedDozenUnitID, "adjust")
	deleteStatus, deleteBody, err := apiClient.Delete(productsPath + "/" + stocked)
	require.NoError(t, err)
	requireStatus(t, 200, deleteStatus, deleteBody)

	// A period of its own keeps this run off a report another cached; 10 pr spread over it divides exactly.
	periods := []int{5, 8, 10, 16, 20, 25, 40, 50, 80, 100, 125, 160, 200, 250, 400, 500}
	weeks := periods[rand.IntN(len(periods))]
	var row map[string]any
	eventually(t, 75*time.Second, time.Second, func() error {
		status, body, err := apiClient.GetListRaw(weeksOfSalesPath, url.Values{"period_in_weeks": {strconv.Itoa(weeks)}})
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("weeks of sales answered %d: %s", status, string(body))
		}
		got := parseJSON(body)
		for _, r := range jsonArray(got, "data") {
			m := r.(map[string]any)
			if jsonField(jsonObject(m, "product_line"), "id") == line {
				row = m
				return nil
			}
		}
		return fmt.Errorf("product line %s not reported yet", line)
	})

	assert.Contains(t, jsonField(jsonObject(row, "product_line"), "name"), "e2e-parity-pdln")
	measure := func(field string) float64 {
		q := jsonObject(row, field)
		require.NotNil(t, q, field)
		unit := jsonObject(q, "unit")
		require.NotNil(t, unit, field)
		assert.Equal(t, "pr", jsonField(unit, "abbreviation"), field)
		assert.Equal(t, "quantity", jsonField(unit, "type"), field)
		v, err := strconv.ParseFloat(jsonField(q, "value"), 64)
		require.NoError(t, err, field)
		return v
	}
	assert.InDelta(t, 15, measure("quantity_on_hand"), 1e-9)
	assert.InDelta(t, 10/float64(weeks), measure("average_sales_quantity"), 1e-9)
	wos, err := strconv.ParseFloat(jsonField(row, "weeks_of_sales"), 64)
	require.NoError(t, err)
	assert.InDelta(t, 1.5*float64(weeks), wos, 1e-9)
}

// --- Open batches ---

// seedBatch writes one batch of itemID at stationID; batches are made by scanning, which has no API that
// takes a quantity and a station directly. A zero scannedAt leaves the batch unscanned.
func seedBatch(t *testing.T, itemID, stationID, value, unitID string, scannedAt time.Time, closed bool) string {
	t.Helper()
	db := authDB(t)
	suffix := uuid.New().String()
	batchID, quantityID := "btch_"+suffix, "qu_"+suffix
	_, err := db.Exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))", quantityID, value, unitID)
	require.NoError(t, err)
	var scanned, closedAt any
	if !scannedAt.IsZero() {
		scanned = scannedAt
	}
	if closed {
		closedAt = time.Now().UTC()
	}
	_, err = db.Exec(`INSERT INTO batch (id, account_id, item_id, quantity_id, scanning_station_id, scanned_at, closed_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))`, batchID, SeedAccountID, itemID, quantityID, stationID, scanned, closedAt)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM _batch_flow WHERE A = ? OR B = ?", batchID, batchID)
		_, _ = db.Exec("DELETE FROM batch WHERE id = ?", batchID)
		_, _ = db.Exec("DELETE FROM quantity WHERE id = ?", quantityID)
	})
	return batchID
}

// feeds records that output came from input: A is the downstream batch, B the upstream one.
func feeds(t *testing.T, input, output string) {
	t.Helper()
	_, err := authDB(t).Exec("INSERT INTO _batch_flow (A, B) VALUES (?, ?)", output, input)
	require.NoError(t, err)
}

func openBatchRows(t *testing.T, stationID string, body map[string]any) []map[string]any {
	t.Helper()
	got := mustPutAnalytics(t, apiClient, openBatchesPath, nil, body)
	var out []map[string]any
	for _, r := range listRows(t, got) {
		if jsonField(jsonObject(r, "scanning_station"), "id") == stationID {
			out = append(out, r)
		}
	}
	return out
}

type openBatchWant struct{ itemID, count, unit string }

func assertOpenBatchRows(t *testing.T, rows []map[string]any, department string, want ...openBatchWant) {
	t.Helper()
	require.Len(t, rows, len(want), "%v", rows)
	for i, w := range want {
		r := rows[i]
		assert.Equal(t, "open_batch_summary", jsonField(r, "object"))
		assert.Equal(t, department, jsonField(r, "department_name"))
		assert.Equal(t, w.itemID, jsonField(jsonObject(r, "item"), "id"), "row %d", i)
		assert.Equal(t, w.count, jsonField(r, "count"), "row %d", i)
		assert.Equal(t, w.unit, jsonField(r, "unit"), "row %d", i)
	}
}

func TestAnalyticsParityOpenBatches_CountsWhatIsLeftPerStationAndItem(t *testing.T) {
	t.Parallel()
	station := jsonField(createAndCleanup(t, scanningStationsPath, map[string]any{
		"name":                 uniqueName("e2e-parity-station"),
		"type":                 "init_batch",
		"operator_requirement": "none",
		"department_id":        SeedDepartmentID,
	}), "id")
	department := jsonField(parseJSON(mustGet(t, "/v1/operations/departments/"+SeedDepartmentID)), "name")
	line := parityProductLine(t)
	_, sock := parityProduct(t, line)
	emptyLine := parityProductLine(t)

	t0 := time.Now().UTC().Add(-time.Hour)
	// 10 pr, 3 of which have gone on into an output batch (itself closed), and 2 dz recorded in dozens: the
	// dashboard summed what was recorded, without conversion, under the first-scanned batch's unit.
	first := seedBatch(t, sock, station, "10", SeedUnitID, t0, false)
	feeds(t, first, seedBatch(t, sock, station, "3", SeedUnitID, t0.Add(time.Minute), true))
	seedBatch(t, sock, station, "2", seedDozenUnitID, t0.Add(2*time.Minute), false)
	seedBatch(t, sock, station, "100", SeedUnitID, t0, true)           // closed
	seedBatch(t, sock, station, "100", SeedUnitID, time.Time{}, false) // never scanned
	seedBatch(t, SeedPartItemID, station, "4", SeedUnitID, t0, false)  // LKN, knit
	seedBatch(t, SeedLsnItemID, station, "5", SeedUnitID, t0, false)   // LSN, sewn from LKN

	all := []openBatchWant{{sock, "9", "pr"}, {SeedLsnItemID, "5", "pr"}, {SeedPartItemID, "4", "pr"}}
	assertOpenBatchRows(t, openBatchRows(t, station, nil), department, all...)

	// Selecting an item selects the parts its production consumes upstream, with the item itself.
	assertOpenBatchRows(t, openBatchRows(t, station, map[string]any{"item_ids": []string{SeedLsnItemID}}), department,
		openBatchWant{SeedLsnItemID, "5", "pr"}, openBatchWant{SeedPartItemID, "4", "pr"})
	assertOpenBatchRows(t, openBatchRows(t, station, map[string]any{"item_ids": []string{SeedPartItemID}}), department,
		openBatchWant{SeedPartItemID, "4", "pr"})
	// A product line selects its products' items; the two filters add up.
	assertOpenBatchRows(t, openBatchRows(t, station, map[string]any{"product_line_ids": []string{line}}), department,
		openBatchWant{sock, "9", "pr"})
	assertOpenBatchRows(t, openBatchRows(t, station, map[string]any{"product_line_ids": []string{line}, "item_ids": []string{SeedPartItemID}}), department,
		openBatchWant{sock, "9", "pr"}, openBatchWant{SeedPartItemID, "4", "pr"})
	// A selection that leads to nothing does not filter.
	assertOpenBatchRows(t, openBatchRows(t, station, map[string]any{"product_line_ids": []string{emptyLine}}), department, all...)
}

// --- Demand forecast ---

func forecastRow(t *testing.T, itemID string, body map[string]any) map[string]any {
	t.Helper()
	got := mustPutAnalytics(t, apiClient, demandForecastPath, nil, body)
	assert.Equal(t, "analyze_demand_forecast_response", jsonField(got, "object"))
	for _, r := range listRows(t, jsonObject(got, "data")) {
		if jsonField(jsonObject(r, "item"), "id") == itemID {
			return r
		}
	}
	return nil
}

func TestAnalyticsParityDemandForecast_ForecastsAnItemFirstOrderedThisMonth(t *testing.T) {
	t.Parallel()
	line := parityProductLine(t)
	product, item := parityProduct(t, line)
	other, otherItem := parityProduct(t, line)
	customer := parityCustomer(t, "", line)

	// Demand counts every sales order created this month, estimates included, in pairs:
	//   2 dz (12 pr) at $3/pr = $36, an estimate of 1 pr at $2/pr = $2, and 3 pr at $1/pr = $3 shipped.
	issueParityOrder(t, customer, "", parityLine(product, "2", seedDozenUnitID, "3.00"))
	createParityEstimate(t, customer, parityLine(product, "1", SeedUnitID, "2.00"))
	shipped := issueParityOrder(t, customer, "", parityLine(product, "3", SeedUnitID, "1.00"))
	shipWholeOrder(t, jsonField(shipped, "id"))
	issueParityOrder(t, customer, "", parityLine(other, "5", SeedUnitID, "1.00"))
	// A purchase order of the same product is not demand.
	po := createPurchaseOrder(t, func(body map[string]any) {
		body["lines"] = []map[string]any{{
			"product_id": product, "product_sku": "E2E-PO-SKU",
			"quantity":   map[string]any{"value": "4", "unit_id": SeedUnitID},
			"unit_price": map[string]any{"value": "9.50", "numerator_unit_id": e2eCurrencyUnitID, "denominator_unit_id": SeedUnitID},
		}}
	})
	status, raw := changePurchaseOrderStatus(t, jsonField(po, "id"), "issue")
	requireStatus(t, 200, status, raw)
	t.Cleanup(func() { changePurchaseOrderStatus(t, jsonField(po, "id"), "unissue") })

	var row map[string]any
	eventually(t, 75*time.Second, time.Second, func() error {
		row = forecastRow(t, item, map[string]any{"item_ids": []string{item}, "forecast_months": 3})
		if row == nil {
			return fmt.Errorf("item %s not forecast yet", item)
		}
		if jsonField(row, "current_month_sales") != "3" {
			return fmt.Errorf("invoiced sales not in yet: %v", row["current_month_sales"])
		}
		return nil
	})
	assert.Equal(t, line, jsonField(jsonObject(row, "product_line"), "id"))
	assert.Equal(t, "pr", jsonField(row, "unit"))
	assert.Equal(t, "$", jsonField(row, "currency"))
	assert.Equal(t, "16", jsonField(row, "current_month_demand"), "12 + 1 + 3 pr; the purchase order's 4 are not demand")
	assert.Equal(t, "41", jsonField(row, "current_month_revenue"))
	assert.Equal(t, "3", jsonField(row, "current_month_sales"))

	// No complete month to learn from: no history, and a forecast of zeros rather than none.
	assert.Empty(t, jsonArray(row, "history"))
	assert.Empty(t, jsonArray(row, "revenue_history"))
	assert.Empty(t, jsonArray(row, "sales_history"))
	assert.Empty(t, jsonArray(row, "sales_forecast"))
	now := time.Now().UTC()
	for _, field := range []string{"forecast", "revenue_forecast"} {
		points := jsonArray(row, field)
		require.Len(t, points, 3, field)
		for i, p := range points {
			m := p.(map[string]any)
			assert.Equal(t, "0", jsonField(m, "forecast"), field)
			assert.Equal(t, "0", jsonField(m, "lower_bound"), field)
			assert.Equal(t, "0", jsonField(m, "upper_bound"), field)
			// Point k covers the month k+1 past the last complete one, stamped a month on.
			want := time.Date(now.Year(), now.Month()+time.Month(i+1), 1, 0, 0, 0, 0, time.UTC)
			assert.Equal(t, want.Format(time.RFC3339), jsonField(m, "at"), field)
		}
	}

	// The filters reach the query: the other item answers only when asked for.
	assert.Nil(t, forecastRow(t, otherItem, map[string]any{"item_ids": []string{item}}))
	otherRow := forecastRow(t, otherItem, map[string]any{"product_line_ids": []string{line}})
	require.NotNil(t, otherRow, "the product line filter covers both of its items")
	assert.Equal(t, "5", jsonField(otherRow, "current_month_demand"))
}
