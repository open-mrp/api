//go:build e2e

package api_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The production costs report costs every batch scanned at a production step in a window. These build a
// plant of their own — two departments with a station each, two categories, and three steps — and scan
// batches into a window long past, so every figure is exact and no other test's data is in it:
//
//   - knit makes 6 pr of part X a run from 12 ea of a $0.50 material, 30 s a pair: $6 material, 0.05 h,
//     $1.20 labor at $24/h and $0.60 overhead at $12/h.
//   - sew makes 10 ea of part Y a run from 10 ea (and 2 ea of waste) of the material, 1 min a PAIR: half a
//     minute an each, 1/12 h a run, $2 labor and $1 overhead. The dashboard read that labor time in the
//     wrong unit and reported 1/60 of it.
//   - finish makes 2 ea of good G (on a product line) a run from part Y, 0.1 h a DOZEN: 1/60 h a run,
//     $0.40 labor and $0.20 overhead, no material (a part is not material).

const productionCostsPath = "/v1/core/analytics/production-costs"

var (
	pcWindowStart = time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	pcWindowEnd   = time.Date(2021, 3, 31, 23, 59, 59, 999_000_000, time.UTC)
)

func pcTimestamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z07:00") }

type productionCostPlant struct {
	deptA, deptB, emptyDept string
	deptAName, deptBName    string
	stationA, stationB      string
	catX, catY              string
	catXName, catYName      string
	partX, partY, good      string
	productLine             string
	knit, sew, finish       string
}

func createProductionCostPlant(t *testing.T) productionCostPlant {
	t.Helper()
	var p productionCostPlant

	p.deptAName, p.deptBName = uniqueName("e2e-pcost-a"), uniqueName("e2e-pcost-b")
	p.deptA = jsonField(createAndCleanup(t, departmentsPath, map[string]any{"name": p.deptAName}), "id")
	p.deptB = jsonField(createAndCleanup(t, departmentsPath, map[string]any{"name": p.deptBName}), "id")
	p.emptyDept = jsonField(createAndCleanup(t, departmentsPath, map[string]any{"name": uniqueName("e2e-pcost-empty")}), "id")
	station := func(dept string) string {
		return jsonField(createAndCleanup(t, scanningStationsPath, map[string]any{
			"name": uniqueName("e2e-pcost-station"), "type": "init_batch", "operator_requirement": "none", "department_id": dept,
		}), "id")
	}
	p.stationA, p.stationB = station(p.deptA), station(p.deptB)

	p.catXName, p.catYName = uniqueName("e2e-pcost-x"), uniqueName("e2e-pcost-y")
	category := func(name string) string {
		return jsonField(createAndCleanup(t, itemCategoriesPath, map[string]any{"name": name, "type": "product_category", "unit_group_id": SeedUnitGroupID}), "id")
	}
	p.catX, p.catY = category(p.catXName), category(p.catYName)

	status, body, err := apiClient.Post(materialsPath+"?include=item", map[string]any{
		"sku":         uniqueName("e2e-pcost-material"),
		"category_id": packagingID,
		"unit_cost":   map[string]any{"value": "0.50", "numerator_unit_id": unitDollar, "denominator_unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	material := parseJSON(body)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + jsonField(material, "id")) })
	materialItem := jsonField(jsonObject(material, "item"), "id")

	part := func(category string) string {
		status, body, err := apiClient.Post(partsPath+"?include=item", map[string]any{"sku": uniqueName("e2e-pcost-part"), "category_id": category}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		created := parseJSON(body)
		t.Cleanup(func() { apiClient.Delete(partsPath + "/" + jsonField(created, "id")) })
		return jsonField(jsonObject(created, "item"), "id")
	}
	p.partX, p.partY = part(p.catX), part(p.catY)

	p.productLine = parityProductLine(t)
	sku := uniqueName("e2e-pcost-good")
	status, body, err = apiClient.Post(productsPath, map[string]any{"sku": sku, "type": "sale", "category_id": p.catY, "product_line_id": p.productLine}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	productID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(productsPath + "/" + productID) })
	items, _, err := apiClient.GetList(itemsPath, url.Values{"q": {sku}})
	require.NoError(t, err)
	require.Len(t, items.Data, 1)
	p.good = DataItemField(items.Data[0], "id")

	step := func(laborTime map[string]any, item, qty, unit string, consumptions ...map[string]any) string {
		if consumptions == nil {
			consumptions = []map[string]any{}
		}
		status, body, err := apiClient.Post(productionStepsPath, map[string]any{
			"name":            uniqueName("e2e-pcost-step"),
			"leveling_factor": "0",
			"allowances":      "0",
			"labor_time":      laborTime,
			"labor_rate":      map[string]any{"value": "24", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
			"overhead_rate":   map[string]any{"value": "12", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
			"production":      map[string]any{"item_id": item, "quantity_value": qty, "quantity_unit_id": unit},
			"consumptions":    consumptions,
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 201, status, body)
		id := jsonField(parseJSON(body), "id")
		t.Cleanup(func() { cleanupStepIDs(id) })
		return id
	}
	consume := func(item, qty, waste string) map[string]any {
		return map[string]any{"item_id": item, "quantity_value": qty, "quantity_unit_id": unitEach, "waste_quantity_value": waste, "waste_quantity_unit_id": unitEach}
	}
	p.knit = step(map[string]any{"value": "30", "numerator_unit_id": unitSecond, "denominator_unit_id": SeedPairUnitID},
		p.partX, "6", SeedPairUnitID, consume(materialItem, "12", "0"))
	p.sew = step(map[string]any{"value": "1", "numerator_unit_id": unitMinute, "denominator_unit_id": SeedPairUnitID},
		p.partY, "10", unitEach, consume(materialItem, "10", "2"))
	p.finish = step(map[string]any{"value": "0.1", "numerator_unit_id": unitHour, "denominator_unit_id": unitDozen},
		p.good, "2", unitEach, consume(p.partY, "2", "0"))
	return p
}

type pcBatch struct {
	step, item, station string
	qty, unit           string
	waste, wasteUnit    string
	seconds, secUnit    string
	scannedAt           time.Time
	closed              bool
}

// scan writes one batch as the floor would have recorded it: there is no API that scans into the past.
func (p productionCostPlant) scan(t *testing.T, b pcBatch) {
	t.Helper()
	db := authDB(t)
	suffix := uuid.New().String()
	batchID := "btch_" + suffix
	quantity := func(kind, value, unit string) any {
		if value == "" {
			return nil
		}
		id := "qu_" + kind + suffix
		_, err := db.Exec("INSERT INTO quantity (id, value, unit_id, created_at, updated_at) VALUES (?, ?, ?, NOW(3), NOW(3))", id, value, unit)
		require.NoError(t, err)
		t.Cleanup(func() { _, _ = db.Exec("DELETE FROM quantity WHERE id = ?", id) })
		return id
	}
	nullable := func(s string) any {
		if s == "" {
			return nil
		}
		return s
	}
	var scanned, closed any
	if !b.scannedAt.IsZero() {
		scanned = b.scannedAt.UTC()
	}
	if b.closed {
		closed = b.scannedAt.UTC().Add(time.Hour)
	}
	_, err := db.Exec(`INSERT INTO batch (id, account_id, item_id, quantity_id, waste_quantity_id, seconds_quantity_id, scanning_station_id,
    production_step_id, scanned_at, closed_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))`,
		batchID, SeedAccountID, b.item, quantity("", b.qty, b.unit), quantity("w", b.waste, b.wasteUnit), quantity("s", b.seconds, b.secUnit),
		nullable(b.station), nullable(b.step), scanned, closed)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = db.Exec("DELETE FROM batch WHERE id = ?", batchID) })
}

// scanTheMonth records March 2021: batches of every kind at both stations and none, two on the window's
// very edges, and four that must never count — a millisecond either side of it, one never scanned, one at
// no production step.
func (p productionCostPlant) scanTheMonth(t *testing.T) {
	t.Helper()
	day := func(d int) time.Time { return time.Date(2021, 3, d, 12, 0, 0, 0, time.UTC) }
	for _, b := range []pcBatch{
		{step: p.knit, item: p.partX, station: p.stationA, qty: "12", unit: SeedPairUnitID, waste: "6", wasteUnit: SeedPairUnitID, scannedAt: day(10), closed: true},
		{step: p.knit, item: p.partX, station: p.stationB, qty: "2", unit: unitDozen, seconds: "3", secUnit: SeedPairUnitID, scannedAt: day(11)},
		{step: p.sew, item: p.partY, station: p.stationA, qty: "20", unit: unitEach, waste: "5", wasteUnit: unitEach, scannedAt: day(12)},
		{step: p.sew, item: p.partY, station: p.stationB, qty: "10", unit: unitEach, seconds: "10", secUnit: unitEach, scannedAt: day(13), closed: true},
		{step: p.finish, item: p.good, station: p.stationB, qty: "4", unit: unitEach, scannedAt: day(14)},
		{step: p.knit, item: p.partX, qty: "6", unit: SeedPairUnitID, scannedAt: day(15)},
		{step: p.knit, item: p.partX, station: p.stationA, qty: "6", unit: SeedPairUnitID, scannedAt: pcWindowStart},
		{step: p.knit, item: p.partX, station: p.stationA, qty: "6", unit: SeedPairUnitID, scannedAt: pcWindowEnd},

		{step: p.knit, item: p.partX, station: p.stationA, qty: "6", unit: SeedPairUnitID, scannedAt: pcWindowStart.Add(-time.Millisecond)},
		{step: p.knit, item: p.partX, station: p.stationA, qty: "6", unit: SeedPairUnitID, scannedAt: pcWindowEnd.Add(time.Millisecond)},
		{step: p.knit, item: p.partX, station: p.stationA, qty: "6", unit: SeedPairUnitID},
		{item: p.partX, station: p.stationA, qty: "6", unit: SeedPairUnitID, scannedAt: day(16)},
	} {
		p.scan(t, b)
	}
}

// pcCost is one kind of output's figures: money in dollars, labor time in hours, produced in eaches.
type pcCost struct{ materials, labor, overhead, total, laborTime, produced string }

type pcCosts struct{ productive, seconds, waste, total pcCost }

var pcZero = pcCost{"0", "0", "0", "0", "0", "0"}

func assertDecimal(t *testing.T, want, got, label string) {
	t.Helper()
	g, err := decimal.NewFromString(got)
	if !assert.NoError(t, err, "%s: %q is not a decimal", label, got) {
		return
	}
	assert.True(t, g.Equal(decimal.RequireFromString(want)), "%s = %s, want %s", label, got, want)
}

func assertProductionCost(t *testing.T, label string, got map[string]any, want pcCost) {
	t.Helper()
	require.NotNil(t, got, "%s is missing", label)
	assert.Equal(t, "production_cost", jsonField(got, "object"), label)
	assertDecimal(t, want.materials, jsonField(got, "materials"), label+" materials")
	assertDecimal(t, want.labor, jsonField(got, "labor"), label+" labor")
	assertDecimal(t, want.overhead, jsonField(got, "overhead"), label+" overhead")
	assertDecimal(t, want.total, jsonField(got, "total"), label+" total")
	assertDecimal(t, want.laborTime, jsonField(got, "labor_time"), label+" labor_time")
	produced := listRows(t, jsonObject(got, "produced"))
	require.Len(t, produced, 1, "%s produces in one dimension: %v", label, produced)
	assert.Equal(t, "computed_quantity", jsonField(produced[0], "object"))
	assertDecimal(t, want.produced, jsonField(produced[0], "value"), label+" produced")
	assert.Equal(t, "each", jsonField(jsonObject(produced[0], "unit"), "id"), label+" produced unit")
	assert.Equal(t, "ea", jsonField(jsonObject(produced[0], "unit"), "abbreviation"))
}

func assertProductionCosts(t *testing.T, label string, got map[string]any, want pcCosts) {
	t.Helper()
	assertProductionCost(t, label+" productive", jsonObject(got, "productive"), want.productive)
	assertProductionCost(t, label+" seconds", jsonObject(got, "seconds"), want.seconds)
	assertProductionCost(t, label+" waste", jsonObject(got, "waste"), want.waste)
	assertProductionCost(t, label+" total", jsonObject(got, "total"), want.total)
}

func (p productionCostPlant) report(t *testing.T, client *Client, body map[string]any) map[string]any {
	t.Helper()
	if body == nil {
		body = map[string]any{}
	}
	if _, ok := body["starts_at"]; !ok {
		body["starts_at"] = pcTimestamp(pcWindowStart)
	}
	if _, ok := body["ends_at"]; !ok {
		body["ends_at"] = pcTimestamp(pcWindowEnd)
	}
	got := mustPutAnalytics(t, client, productionCostsPath, nil, body)
	assert.Equal(t, "analyze_production_costs_response", jsonField(got, "object"))
	return got
}

func (p productionCostPlant) categories() []string { return []string{p.catX, p.catY} }

// Expected figures, from the runs each group's batches amount to.
var (
	pcAX = pcCosts{ // knit at A: 24 + 12 + 12 ea productive is 4 runs, 12 ea waste 1 run
		productive: pcCost{"24", "4.8", "2.4", "31.2", "0.2", "48"},
		seconds:    pcZero,
		waste:      pcCost{"6", "1.2", "0.6", "7.8", "0.05", "12"},
		total:      pcCost{"30", "6", "3", "39", "0.25", "60"},
	}
	pcBX = pcCosts{ // knit at B: 2 dz is 2 runs, 3 pr of seconds half a run
		productive: pcCost{"12", "2.4", "1.2", "15.6", "0.1", "24"},
		seconds:    pcCost{"3", "0.6", "0.3", "3.9", "0.025", "6"},
		waste:      pcZero,
		total:      pcCost{"15", "3", "1.5", "19.5", "0.125", "30"},
	}
	pcNoneX = pcCosts{ // knit at no station: 6 pr is a run
		productive: pcCost{"6", "1.2", "0.6", "7.8", "0.05", "12"},
		seconds:    pcZero,
		waste:      pcZero,
		total:      pcCost{"6", "1.2", "0.6", "7.8", "0.05", "12"},
	}
	pcAY = pcCosts{ // sew at A: 20 ea is 2 runs, 5 ea of waste half a run; the dashboard would show $0.0667 of labor, not $4
		productive: pcCost{"12", "4", "2", "18", "0.1666666667", "20"},
		seconds:    pcZero,
		waste:      pcCost{"3", "1", "0.5", "4.5", "0.0416666667", "5"},
		total:      pcCost{"15", "5", "2.5", "22.5", "0.2083333333", "25"},
	}
	pcBY = pcCosts{ // sew at B, a run and a run of seconds; finish at B, 4 ea is 2 runs
		productive: pcCost{"6", "2.8", "1.4", "10.2", "0.1166666667", "14"},
		seconds:    pcCost{"6", "2", "1", "9", "0.0833333333", "10"},
		waste:      pcZero,
		total:      pcCost{"12", "4.8", "2.4", "19.2", "0.2", "24"},
	}
	pcDeptA = pcCosts{
		productive: pcCost{"36", "8.8", "4.4", "49.2", "0.3666666667", "68"},
		seconds:    pcZero,
		waste:      pcCost{"9", "2.2", "1.1", "12.3", "0.0916666667", "17"},
		total:      pcCost{"45", "11", "5.5", "61.5", "0.4583333333", "85"},
	}
	pcDeptB = pcCosts{
		productive: pcCost{"18", "5.2", "2.6", "25.8", "0.2166666667", "38"},
		seconds:    pcCost{"9", "2.6", "1.3", "12.9", "0.1083333333", "16"},
		waste:      pcZero,
		total:      pcCost{"27", "7.8", "3.9", "38.7", "0.325", "54"},
	}
	pcCatX = pcCosts{
		productive: pcCost{"42", "8.4", "4.2", "54.6", "0.35", "84"},
		seconds:    pcCost{"3", "0.6", "0.3", "3.9", "0.025", "6"},
		waste:      pcCost{"6", "1.2", "0.6", "7.8", "0.05", "12"},
		total:      pcCost{"51", "10.2", "5.1", "66.3", "0.425", "102"},
	}
	pcCatY = pcCosts{
		productive: pcCost{"18", "6.8", "3.4", "28.2", "0.2833333333", "34"},
		seconds:    pcCost{"6", "2", "1", "9", "0.0833333333", "10"},
		waste:      pcCost{"3", "1", "0.5", "4.5", "0.0416666667", "5"},
		total:      pcCost{"27", "9.8", "4.9", "41.7", "0.4083333333", "49"},
	}
	pcAll = pcCosts{
		productive: pcCost{"60", "15.2", "7.6", "82.8", "0.6333333333", "118"},
		seconds:    pcCost{"9", "2.6", "1.3", "12.9", "0.1083333333", "16"},
		waste:      pcCost{"9", "2.2", "1.1", "12.3", "0.0916666667", "17"},
		total:      pcCost{"78", "20", "10", "108", "0.8333333333", "151"},
	}
)

func entityID(row map[string]any, key string) string { return jsonField(jsonObject(row, key), "id") }

func TestAnalyticsProductionCosts_CostsEveryBatchByDepartmentAndCategory(t *testing.T) {
	t.Parallel()
	p := createProductionCostPlant(t)
	p.scanTheMonth(t)

	got := p.report(t, apiClient, map[string]any{"category_ids": p.categories()})

	assert.Equal(t, pcTimestamp(pcWindowStart), pcTimestamp(mustParseTime(t, jsonField(got, "starts_at"))))
	assert.Equal(t, pcTimestamp(pcWindowEnd), pcTimestamp(mustParseTime(t, jsonField(got, "ends_at"))))
	currency, timeUnit := jsonObject(got, "currency_unit"), jsonObject(got, "time_unit")
	assert.Equal(t, "dollar", jsonField(currency, "id"))
	assert.Equal(t, "unit", jsonField(currency, "object"))
	assert.Equal(t, "$", jsonField(currency, "abbreviation"))
	assert.Equal(t, "hour", jsonField(timeUnit, "id"))
	assert.Equal(t, "time", jsonField(timeUnit, "type"))

	totals := jsonObject(got, "totals")
	assert.Equal(t, "production_cost_totals", jsonField(totals, "object"))
	assertProductionCosts(t, "totals", totals, pcAll)

	departments := listRows(t, jsonObject(got, "departments"))
	require.Len(t, departments, 3, "%v", departments)
	for _, d := range departments {
		assert.Equal(t, "production_cost_department", jsonField(d, "object"))
	}
	assert.Equal(t, p.deptA, entityID(departments[0], "department"))
	assert.Equal(t, p.deptAName, jsonField(jsonObject(departments[0], "department"), "name"))
	assert.Equal(t, "entity", jsonField(jsonObject(departments[0], "department"), "object"))
	assert.Equal(t, "department", jsonField(jsonObject(departments[0], "department"), "type"))
	assert.Equal(t, p.deptB, entityID(departments[1], "department"))
	assert.Nil(t, departments[2]["department"], "batches scanned at no station come last, with no department")
	assertProductionCosts(t, "department A", departments[0], pcDeptA)
	assertProductionCosts(t, "department B", departments[1], pcDeptB)
	assertProductionCosts(t, "no department", departments[2], pcNoneX)

	categories := listRows(t, jsonObject(got, "categories"))
	require.Len(t, categories, 2, "%v", categories)
	assert.Equal(t, "production_cost_category", jsonField(categories[0], "object"))
	assert.Equal(t, p.catX, entityID(categories[0], "category"))
	assert.Equal(t, p.catXName, jsonField(jsonObject(categories[0], "category"), "name"))
	assert.Equal(t, "item_category", jsonField(jsonObject(categories[0], "category"), "type"))
	assert.Equal(t, p.catY, entityID(categories[1], "category"))
	assertProductionCosts(t, "category X", categories[0], pcCatX)
	assertProductionCosts(t, "category Y", categories[1], pcCatY)

	rows := listRows(t, jsonObject(got, "department_categories"))
	require.Len(t, rows, 5, "%v", rows)
	want := []struct {
		dept, cat string
		costs     pcCosts
	}{{p.deptA, p.catX, pcAX}, {p.deptA, p.catY, pcAY}, {p.deptB, p.catX, pcBX}, {p.deptB, p.catY, pcBY}, {"", p.catX, pcNoneX}}
	for i, w := range want {
		assert.Equal(t, "production_cost_department_category", jsonField(rows[i], "object"))
		assert.Equal(t, w.dept, entityID(rows[i], "department"), "row %d", i)
		assert.Equal(t, w.cat, entityID(rows[i], "category"), "row %d", i)
		assertProductionCosts(t, "row", rows[i], w.costs)
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, s)
	require.NoError(t, err)
	return parsed
}

func TestAnalyticsProductionCosts_Filters(t *testing.T) {
	t.Parallel()
	p := createProductionCostPlant(t)
	p.scanTheMonth(t)

	t.Run("departments select the batches scanned at their stations", func(t *testing.T) {
		got := p.report(t, apiClient, map[string]any{"category_ids": p.categories(), "department_ids": []string{p.deptA}})
		assertProductionCosts(t, "totals", jsonObject(got, "totals"), pcDeptA)
		departments := listRows(t, jsonObject(got, "departments"))
		require.Len(t, departments, 1)
		assert.Equal(t, p.deptA, entityID(departments[0], "department"))
		assert.Len(t, listRows(t, jsonObject(got, "department_categories")), 2)
	})

	t.Run("a department with no stations has no batches", func(t *testing.T) {
		got := p.report(t, apiClient, map[string]any{"department_ids": []string{p.emptyDept}})
		assertEmptyProductionCostReport(t, got)
	})

	t.Run("categories select the batches of their items", func(t *testing.T) {
		got := p.report(t, apiClient, map[string]any{"category_ids": []string{p.catY}})
		assertProductionCosts(t, "totals", jsonObject(got, "totals"), pcCatY)
		categories := listRows(t, jsonObject(got, "categories"))
		require.Len(t, categories, 1)
		assert.Equal(t, p.catY, entityID(categories[0], "category"))
	})

	t.Run("an item selects itself and the parts it consumes upstream", func(t *testing.T) {
		got := p.report(t, apiClient, map[string]any{"category_ids": p.categories(), "item_ids": []string{p.good}})
		assertProductionCosts(t, "totals", jsonObject(got, "totals"), pcCatY)

		got = p.report(t, apiClient, map[string]any{"category_ids": p.categories(), "item_ids": []string{p.partX}})
		assertProductionCosts(t, "totals", jsonObject(got, "totals"), pcCatX)
	})

	// As the dashboard's report did, a product line selects the parts its goods consume, not a good a step
	// produces: the sewing of part Y counts, the finishing of the good does not.
	t.Run("a product line selects the parts its goods consume", func(t *testing.T) {
		sewn := pcCosts{
			productive: pcCost{"18", "6", "3", "27", "0.25", "30"},
			seconds:    pcCost{"6", "2", "1", "9", "0.0833333333", "10"},
			waste:      pcCost{"3", "1", "0.5", "4.5", "0.0416666667", "5"},
			total:      pcCost{"27", "9", "4.5", "40.5", "0.375", "45"},
		}
		got := p.report(t, apiClient, map[string]any{"category_ids": p.categories(), "product_line_ids": []string{p.productLine}})
		assertProductionCosts(t, "totals", jsonObject(got, "totals"), sewn)

		got = p.report(t, apiClient, map[string]any{"category_ids": p.categories(), "product_line_ids": []string{p.productLine}, "item_ids": []string{p.good}})
		assertProductionCosts(t, "totals with the good selected too", jsonObject(got, "totals"), pcCatY)
	})

	t.Run("filters combine", func(t *testing.T) {
		got := p.report(t, apiClient, map[string]any{"department_ids": []string{p.deptB}, "item_ids": []string{p.partX}, "category_ids": []string{p.catX}})
		assertProductionCosts(t, "totals", jsonObject(got, "totals"), pcBX)
		rows := listRows(t, jsonObject(got, "department_categories"))
		require.Len(t, rows, 1)
		assert.Equal(t, p.deptB, entityID(rows[0], "department"))
		assert.Equal(t, p.catX, entityID(rows[0], "category"))
	})
}

func TestAnalyticsProductionCosts_WindowIncludesBothEnds(t *testing.T) {
	t.Parallel()
	p := createProductionCostPlant(t)
	p.scanTheMonth(t)
	knitAtA := func(got map[string]any) map[string]any {
		for _, r := range listRows(t, jsonObject(got, "department_categories")) {
			if entityID(r, "department") == p.deptA && entityID(r, "category") == p.catX {
				return r
			}
		}
		t.Fatalf("no row for department A and category X: %v", got)
		return nil
	}

	// A millisecond in from each end leaves out the two batches scanned on its edges: 2 runs, not 4.
	got := p.report(t, apiClient, map[string]any{
		"category_ids": p.categories(),
		"starts_at":    pcTimestamp(pcWindowStart.Add(time.Millisecond)),
		"ends_at":      pcTimestamp(pcWindowEnd.Add(-time.Millisecond)),
	})
	assertProductionCost(t, "inside the edges", jsonObject(knitAtA(got), "productive"), pcCost{"12", "2.4", "1.2", "15.6", "0.1", "24"})

	// A millisecond out from each end takes in the two scanned just outside the month: 6 runs.
	got = p.report(t, apiClient, map[string]any{
		"category_ids": p.categories(),
		"starts_at":    pcTimestamp(pcWindowStart.Add(-time.Millisecond)),
		"ends_at":      pcTimestamp(pcWindowEnd.Add(time.Millisecond)),
	})
	assertProductionCost(t, "past the edges", jsonObject(knitAtA(got), "productive"), pcCost{"36", "7.2", "3.6", "46.8", "0.3", "72"})

	// A window holding only the batch scanned on its first millisecond.
	got = p.report(t, apiClient, map[string]any{
		"category_ids": p.categories(),
		"starts_at":    pcTimestamp(pcWindowStart),
		"ends_at":      pcTimestamp(pcWindowStart),
	})
	assertProductionCosts(t, "the first millisecond", jsonObject(got, "totals"), pcCosts{
		productive: pcCost{"6", "1.2", "0.6", "7.8", "0.05", "12"}, seconds: pcZero, waste: pcZero,
		total: pcCost{"6", "1.2", "0.6", "7.8", "0.05", "12"},
	})
}

func assertEmptyProductionCostReport(t *testing.T, got map[string]any) {
	t.Helper()
	totals := jsonObject(got, "totals")
	for _, kind := range []string{"productive", "seconds", "waste", "total"} {
		cost := jsonObject(totals, kind)
		for _, field := range []string{"materials", "labor", "overhead", "total", "labor_time"} {
			assertDecimal(t, "0", jsonField(cost, field), kind+" "+field)
		}
		assert.Empty(t, listRows(t, jsonObject(cost, "produced")))
	}
	assert.Empty(t, listRows(t, jsonObject(got, "departments")))
	assert.Empty(t, listRows(t, jsonObject(got, "categories")))
	assert.Empty(t, listRows(t, jsonObject(got, "department_categories")))
	assert.Equal(t, "dollar", jsonField(jsonObject(got, "currency_unit"), "id"), "an empty report still names its units")
}

func TestAnalyticsProductionCosts_AnotherAccountSeesNoneOfIt(t *testing.T) {
	t.Parallel()
	p := createProductionCostPlant(t)
	p.scanTheMonth(t)
	tenantB := apiClient.WithBearerToken(SeedTenantBAPIKey, SeedTenantBAccountID)

	assertEmptyProductionCostReport(t, p.report(t, tenantB, map[string]any{"category_ids": p.categories()}))
	assertEmptyProductionCostReport(t, p.report(t, tenantB, map[string]any{"department_ids": []string{p.deptA, p.deptB}}))
	assertEmptyProductionCostReport(t, p.report(t, tenantB, map[string]any{"item_ids": []string{p.partX, p.good}}))

	got := p.report(t, tenantB, nil)
	for _, r := range listRows(t, jsonObject(got, "department_categories")) {
		assert.NotContains(t, []string{p.deptA, p.deptB}, entityID(r, "department"))
		assert.NotContains(t, []string{p.catX, p.catY}, entityID(r, "category"))
	}
}

func TestAnalyticsProductionCosts_NeedsCostsRead(t *testing.T) {
	t.Parallel()
	body := map[string]any{"starts_at": pcTimestamp(pcWindowStart), "ends_at": pcTimestamp(pcWindowEnd), "category_ids": []string{SeedItemCategoryID}}

	for _, perm := range []string{"items:read", "batches:read"} {
		status, _, raw := putAnalytics(t, customRoleClient(t, perm), productionCostsPath, nil, body)
		assert.Equal(t, 403, status, "%s: %s", perm, string(raw))
	}

	got := mustPutAnalytics(t, customRoleClient(t, "costs:read"), productionCostsPath, nil, body)
	assert.Equal(t, "analyze_production_costs_response", jsonField(got, "object"))
	assert.Equal(t, "hour", jsonField(jsonObject(got, "time_unit"), "id"), "a cost reader sees the units the report is counted in without units:read")
}

func TestAnalyticsProductionCosts_Validation(t *testing.T) {
	t.Parallel()

	status, _, raw := putAnalytics(t, apiClient, productionCostsPath, nil, map[string]any{"ends_at": pcTimestamp(pcWindowEnd)})
	requireStatus(t, 400, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "", "invalid_request_error"), "starts_at")

	status, _, raw = putAnalytics(t, apiClient, productionCostsPath, nil, map[string]any{"starts_at": pcTimestamp(pcWindowStart)})
	requireStatus(t, 400, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "", "invalid_request_error"), "ends_at")

	status, _, raw = putAnalytics(t, apiClient, productionCostsPath, nil, map[string]any{"starts_at": pcTimestamp(pcWindowEnd), "ends_at": pcTimestamp(pcWindowStart)})
	requireStatus(t, 400, status, raw)
	assertErrorParam(t, requireErrorResponse(t, raw, "", "invalid_request_error"), "ends_at")
}
