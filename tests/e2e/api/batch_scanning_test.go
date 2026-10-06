//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Batch scanning: initializing, moving, merging and splitting batches at scanning stations, what each
// scan writes, and the inventory it moves.
//
// Every test builds its own items, stations and steps, because these tests move stock and assert on
// totals. The inventory a scan moves is applied by the batch_scanned consumer, so the totals are read
// with eventually.

const (
	scanShippingCategory  = "itcg_01seedshipping000" // a product category in the each group
	scanPackagingCategory = "itcg_01seedpackaging00" // a material category in the each group
	scanInventoryTimeout  = 30 * time.Second
)

// ──────────────────────────────────────────────
// Fixture builders
// ──────────────────────────────────────────────

type scanItem struct {
	id, sku string
}

func newScanPart(t *testing.T, categoryID string) scanItem {
	t.Helper()
	sku := uniqueName("e2e-scan-part")
	body := validPartBody(sku)
	body["category_id"] = categoryID
	status, raw, err := apiClient.Post(partsPath+"?include=item", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	part := parseJSON(raw)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + jsonField(part, "id")) })
	item := jsonObject(part, "item")
	require.NotNil(t, item, "part must carry its item: %s", raw)
	return scanItem{id: jsonField(item, "id"), sku: sku}
}

func newScanMaterial(t *testing.T) scanItem {
	t.Helper()
	sku := uniqueName("e2e-scan-mat")
	status, raw, err := apiClient.Post(materialsPath+"?include=item", map[string]any{
		"sku":         sku,
		"category_id": scanPackagingCategory,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	material := parseJSON(raw)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + jsonField(material, "id")) })
	item := jsonObject(material, "item")
	require.NotNil(t, item, "material must carry its item: %s", raw)
	return scanItem{id: jsonField(item, "id"), sku: sku}
}

func newScanStation(t *testing.T, stationType string) string {
	t.Helper()
	status, raw, err := apiClient.Post(scanningStationsPath, map[string]any{
		"name":                 uniqueName("e2e-scan-" + stationType),
		"type":                 stationType,
		"operator_requirement": "material_check",
		"department_id":        SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	id := jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { apiClient.Delete(scanningStationsPath + "/" + id) })
	return id
}

type scanQty struct {
	itemID, value, unitID string
}

type scanConsumption struct {
	itemID, value, unitID, waste string
}

// newScanStep creates a step at a station that makes produce from consumes. Steps link to the steps
// that make what they consume as they are created, so a chain is built upstream first.
func newScanStep(t *testing.T, stationID string, produce scanQty, consumes ...scanConsumption) string {
	t.Helper()
	consumptions := make([]map[string]any, len(consumes))
	for i, c := range consumes {
		waste := c.waste
		if waste == "" {
			waste = "0"
		}
		consumptions[i] = map[string]any{
			"item_id": c.itemID, "quantity_value": c.value, "quantity_unit_id": c.unitID,
			"waste_quantity_value": waste, "waste_quantity_unit_id": c.unitID,
		}
	}
	status, raw, err := apiClient.Post(productionStepsPath, map[string]any{
		"name":                uniqueName("e2e-scan-step"),
		"leveling_factor":     "0",
		"allowances":          "0",
		"scanning_station_id": stationID,
		"labor_time":          map[string]any{"value": "1", "numerator_unit_id": unitSecond, "denominator_unit_id": unitEach},
		"labor_rate":          map[string]any{"value": "1", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
		"overhead_rate":       map[string]any{"value": "1", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
		"production":          map[string]any{"item_id": produce.itemID, "quantity_value": produce.value, "quantity_unit_id": produce.unitID},
		"consumptions":        consumptions,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	id := jsonField(parseJSON(raw), "id")
	t.Cleanup(func() { cleanupStepIDs(id) })
	return id
}

// planScanBatch puts one planned batch on a run of its own and returns the batch.
func planScanBatch(t *testing.T, itemID, value, unitID string) string {
	t.Helper()
	run := createRunWithBatches(t, map[string]any{"item_id": itemID, "quantity_value": value, "quantity_unit_id": unitID})
	batches := runBatches(t, jsonField(run, "id"), "run")
	require.Len(t, batches, 1)
	return jsonField(batches[0], "id")
}

func stockScanItem(t *testing.T, itemID, value, unitID string) {
	t.Helper()
	mustUpdateInventory(t, itemID, value, unitID, "adjust")
	waitForOnHand(t, itemID, value)
}

// ──────────────────────────────────────────────
// Scan calls
// ──────────────────────────────────────────────

func postScan(t *testing.T, action string, body map[string]any) (int, map[string]any, []byte) {
	t.Helper()
	status, raw, err := apiClient.Post(batchesPath+"/actions/"+action, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "%s must not 5xx: %s", action, string(raw))
	return status, parseJSON(raw), raw
}

// mustScan performs a scan that must succeed. Its cleanup waits for the scan's inventory to be applied,
// and undoes a batch the scan created, before the fixture it reads is deleted: a consumer that finds
// its step or items gone retries the message, and every scan queued behind it waits.
func mustScan(t *testing.T, action string, body map[string]any) map[string]any {
	t.Helper()
	status, got, raw := postScan(t, action, body)
	requireStatus(t, 201, status, raw)
	settleScanAtCleanup(t, action, jsonField(got, "id"))
	return got
}

// settleScanAtCleanup registers the cleanup mustScan describes for a scan that returned batch id.
func settleScanAtCleanup(t *testing.T, action, id string) {
	t.Helper()
	if action != "initialize" {
		t.Cleanup(func() {
			apiClient.Delete(batchesPath + "/" + id)
			waitForMessagesSettled(t, "core.cmd.undo_batch_scan", "core.undo_batch_scan", id)
		})
	}
	t.Cleanup(func() { waitForMessagesSettled(t, "core.event.batch_scanned", "core.batch_scanned_inventory", id) })
}

// waitForMessagesSettled waits until every message under routingKey that names batchID has been
// handled by handler.
func waitForMessagesSettled(t *testing.T, routingKey, handler, batchID string) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		var pending int
		err := authDB(t).QueryRow(`
			SELECT COUNT(*)
			FROM message_outbox o
			LEFT JOIN message_inbox i ON i.message_id = o.message_id AND i.handler = ?
			WHERE o.routing_key = ?
			AND CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(o.payload, '$.data'))) AS CHAR) LIKE ?
			AND (i.status IS NULL OR i.status = 'received')`,
			handler, routingKey, "%"+batchID+"%").Scan(&pending)
		require.NoError(t, err)
		if pending == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("%d %s message(s) for %s still unhandled by %s", pending, routingKey, batchID, handler)
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func initializeScan(t *testing.T, batchID, stationID string, extra map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{"batch_id": batchID, "scanning_station_id": stationID}
	for k, v := range extra {
		body[k] = v
	}
	return mustScan(t, "initialize", body)
}

func moveScan(t *testing.T, stationID, stepID string, batchIDs ...string) map[string]any {
	t.Helper()
	return mustScan(t, "move", map[string]any{"batch_ids": batchIDs, "production_step_id": stepID, "scanning_station_id": stationID})
}

func splitBody(stationID, stepID string, batchIDs []string, firsts, seconds, waste string, closeBatch bool) map[string]any {
	body := map[string]any{
		"batch_ids":           batchIDs,
		"production_step_id":  stepID,
		"scanning_station_id": stationID,
		"firsts":              map[string]any{"measure": firsts, "unit_id": unitEach},
		"close_batch":         closeBatch,
	}
	if seconds != "" {
		body["seconds"] = map[string]any{"measure": seconds, "unit_id": unitEach}
	}
	if waste != "" {
		body["waste"] = map[string]any{"measure": waste, "unit_id": unitEach}
	}
	return body
}

// requireScanError asserts a rejected scan and its message, which is what the station shows the operator.
func requireScanError(t *testing.T, action string, body map[string]any, wantStatus int, wantMessage string) {
	t.Helper()
	status, got, raw := postScan(t, action, body)
	requireStatus(t, wantStatus, status, raw)
	assert.Equal(t, wantMessage, jsonField(jsonObject(got, "error"), "message"), "%s error message", action)
}

func remainingToSplit(t *testing.T, stepID string, batchIDs ...string) (int, map[string]any, []byte) {
	t.Helper()
	status, raw, err := apiClient.Post(batchesPath+"/remaining-quantities?include=unit", map[string]any{
		"batch_ids": batchIDs, "production_step_id": stepID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "remaining-to-split must not 5xx: %s", string(raw))
	return status, parseJSON(raw), raw
}

func requireRemaining(t *testing.T, stepID, want, wantUnitID string, batchIDs ...string) {
	t.Helper()
	status, got, raw := remainingToSplit(t, stepID, batchIDs...)
	requireStatus(t, 200, status, raw)
	assertDecimalEqual(t, want, jsonField(got, "value"), "remaining to split")
	assert.Equal(t, wantUnitID, jsonField(jsonObject(got, "unit"), "id"), "remaining unit: %s", raw)
	assert.NotEmpty(t, jsonField(got, "id"), "the remaining quantity is addressable")
}

type scanConsumptionRow struct {
	SKU              string `json:"sku"`
	DemandMeasure    string `json:"demand_measure"`
	DemandUnit       string `json:"demand_unit"`
	InventoryMeasure string `json:"inventory_measure"`
	InventoryUnit    string `json:"inventory_unit"`
}

func consumptionPreview(t *testing.T, stationID string, body map[string]any) map[string]scanConsumptionRow {
	t.Helper()
	status, raw, err := apiClient.Post(scanningStationsPath+"/"+stationID+"/consumptions", body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	var list struct {
		Data []scanConsumptionRow `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &list))
	rows := make(map[string]scanConsumptionRow, len(list.Data))
	for _, row := range list.Data {
		rows[row.SKU] = row
	}
	return rows
}

// ──────────────────────────────────────────────
// Reads
// ──────────────────────────────────────────────

func waitForOnHand(t *testing.T, itemID, want string) {
	t.Helper()
	wantDec := decimal.RequireFromString(want)
	eventually(t, scanInventoryTimeout, 250*time.Millisecond, func() error {
		got := readInventory(t, itemID).onHand
		if !got.Equal(wantDec) {
			return fmt.Errorf("item %s on hand is %s, want %s", itemID, got, want)
		}
		return nil
	})
}

// flowBatch reads one batch out of its flow, which is where a batch's closed state is visible.
func flowBatch(t *testing.T, flowOf, batchID string) map[string]any {
	t.Helper()
	status, raw, err := apiClient.GetListRaw(batchesPath+"/"+flowOf+"/flow", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	for _, node := range jsonArray(parseJSON(raw), "data") {
		batch := jsonObject(node.(map[string]any), "batch")
		if jsonField(batch, "id") == batchID {
			return batch
		}
	}
	t.Fatalf("batch %s is not in the flow of %s: %s", batchID, flowOf, raw)
	return nil
}

func requireOpen(t *testing.T, flowOf, batchID string) {
	t.Helper()
	assertNilField(t, flowBatch(t, flowOf, batchID), "closed_at")
}

func requireClosed(t *testing.T, flowOf, batchID string) {
	t.Helper()
	assert.NotEmpty(t, jsonField(flowBatch(t, flowOf, batchID), "closed_at"), "batch %s should be closed", batchID)
}

func stationBatches(t *testing.T, stationID string) map[string]map[string]any {
	t.Helper()
	status, raw, err := apiClient.GetListRaw(scanningStationsPath+"/"+stationID+"/batches", url.Values{"limit": {"100"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	out := map[string]map[string]any{}
	for _, row := range jsonArray(parseJSON(raw), "data") {
		b := row.(map[string]any)
		out[jsonField(b, "id")] = b
	}
	return out
}

// scanEvents decodes the outbox messages a batch's scans wrote under routingKey.
func scanEvents(t *testing.T, routingKey, batchID string) []map[string]any {
	t.Helper()
	rows, err := authDB(t).Query(`
		SELECT CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR)
		FROM message_outbox
		WHERE routing_key = ?
		AND CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR) LIKE ?`,
		routingKey, "%"+batchID+"%")
	require.NoError(t, err)
	defer rows.Close()

	var events []map[string]any
	for rows.Next() {
		var data string
		require.NoError(t, rows.Scan(&data))
		events = append(events, parseJSON([]byte(data)))
	}
	require.NoError(t, rows.Err())
	return events
}

func requireOneBatchScanned(t *testing.T, batchID string) map[string]any {
	t.Helper()
	events := scanEvents(t, "core.event.batch_scanned", batchID)
	require.Len(t, events, 1, "a scan publishes exactly one batch_scanned event for %s", batchID)
	assert.Empty(t, scanEvents(t, "core.cmd.execute_production_step", batchID), "the retired command is never published")
	return events[0]
}

// ──────────────────────────────────────────────
// A single-part chain: initialize, move, split
// ──────────────────────────────────────────────

type scanChain struct {
	material     scanItem
	a, b, c      scanItem
	initStation  string
	moveStation  string
	splitStation string
	initStep     string
	moveStep     string
	splitStep    string
}

// newScanChain builds part A (initialized, 2 material each), moved into B (1 A and 1 material each),
// split into C (1 B each), with 1000 of the material on the shelf.
func newScanChain(t *testing.T) scanChain {
	t.Helper()
	c := scanChain{
		material:     newScanMaterial(t),
		a:            newScanPart(t, scanShippingCategory),
		b:            newScanPart(t, scanShippingCategory),
		c:            newScanPart(t, scanShippingCategory),
		initStation:  newScanStation(t, "init_batch"),
		moveStation:  newScanStation(t, "move_batch"),
		splitStation: newScanStation(t, "split_batch"),
	}
	c.initStep = newScanStep(t, c.initStation, scanQty{c.a.id, "1", unitEach},
		scanConsumption{itemID: c.material.id, value: "2", unitID: unitEach})
	c.moveStep = newScanStep(t, c.moveStation, scanQty{c.b.id, "1", unitEach},
		scanConsumption{itemID: c.a.id, value: "1", unitID: unitEach},
		scanConsumption{itemID: c.material.id, value: "1", unitID: unitEach})
	c.splitStep = newScanStep(t, c.splitStation, scanQty{c.c.id, "1", unitEach},
		scanConsumption{itemID: c.b.id, value: "1", unitID: unitEach})
	stockScanItem(t, c.material.id, "1000", unitEach)
	return c
}

func TestBatchScanning_ChainMovesInventoryAtEveryScan(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)

	// Initialize: the planned batch is stamped, attached to the station's step, and its output and
	// materials are booked.
	initialized := initializeScan(t, planned, c.initStation, nil)
	assert.Equal(t, planned, jsonField(initialized, "id"))
	assert.NotEmpty(t, jsonField(initialized, "scanned_at"))
	assert.Equal(t, c.initStep, jsonField(jsonObject(initialized, "production_step"), "id"))
	assertNilField(t, initialized, "closed_at")
	waitForOnHand(t, c.a.id, "10")
	waitForOnHand(t, c.material.id, "980")

	evt := requireOneBatchScanned(t, planned)
	assert.Equal(t, c.initStep, jsonField(evt, "production_step_id"))
	assert.Equal(t, c.initStation, jsonField(evt, "scanning_station_id"))
	assert.Equal(t, c.a.id, jsonField(evt, "item_id"))
	assertDecimalEqual(t, "10", jsonField(evt, "measure"), "event measure")
	assert.Equal(t, unitEach, jsonField(evt, "unit_id"))
	assert.Empty(t, scanEvents(t, "billing.cmd.report_batch_created", planned), "initializing creates no batch to meter")

	// Initializing twice is refused.
	requireScanError(t, "initialize", map[string]any{"batch_id": planned, "scanning_station_id": c.initStation},
		400, "This batch has been scanned already.")

	// Move: a new batch of B, already scanned, made from the whole of A, which closes.
	moved := moveScan(t, c.moveStation, c.moveStep, planned)
	movedID := jsonField(moved, "id")
	assert.NotEqual(t, planned, movedID)
	assert.NotEmpty(t, jsonField(moved, "scanned_at"), "a scan's output is scanned when it is created")
	assert.Equal(t, c.b.id, jsonField(jsonObject(moved, "item"), "id"))
	assertDecimalEqual(t, "10", jsonField(jsonObject(moved, "quantity"), "value"), "moved quantity")
	assertNilField(t, moved, "closed_at")
	requireClosed(t, planned, planned)
	waitForOnHand(t, c.b.id, "10")
	waitForOnHand(t, c.a.id, "0")
	waitForOnHand(t, c.material.id, "970")
	requireOneBatchScanned(t, movedID)
	require.Len(t, scanEvents(t, "billing.cmd.report_batch_created", movedID), 1, "a created batch is metered once")

	// The source is used up, so moving it again finds nothing waiting at the step.
	requireScanError(t, "move", map[string]any{"batch_ids": []string{planned}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation},
		400, "Batch not compatible with production step.")

	// Split, scanned by the batch first initialized: the flow is followed to B.
	requireRemaining(t, c.splitStep, "10", unitEach, planned)

	first := mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{planned}, "4", "1", "1", false))
	firstID := jsonField(first, "id")
	assertDecimalEqual(t, "4", jsonField(jsonObject(first, "quantity"), "value"), "firsts")
	assertDecimalEqual(t, "1", jsonField(jsonObject(first, "seconds"), "value"), "seconds")
	assertDecimalEqual(t, "1", jsonField(jsonObject(first, "waste"), "value"), "waste")
	requireOpen(t, planned, movedID)
	requireRemaining(t, c.splitStep, "4", unitEach, planned)
	waitForOnHand(t, c.c.id, "4")
	waitForOnHand(t, c.b.id, "4") // firsts, seconds and waste all ran: 10 - 6
	splitEvt := requireOneBatchScanned(t, firstID)
	assertDecimalEqual(t, "4", jsonField(splitEvt, "measure"), "split event firsts")
	assertDecimalEqual(t, "1", jsonField(splitEvt, "seconds_measure"), "split event seconds")
	assertDecimalEqual(t, "1", jsonField(splitEvt, "waste_measure"), "split event waste")

	// The rest of it closes the source.
	second := mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{planned}, "4", "", "", false))
	secondID := jsonField(second, "id")
	requireClosed(t, planned, movedID)
	status, _, raw := remainingToSplit(t, c.splitStep, planned)
	requireStatus(t, 400, status, raw)
	waitForOnHand(t, c.c.id, "8")
	waitForOnHand(t, c.b.id, "0")

	// Undoing the last split reopens the source and puts its inventory back.
	deleteBatch(t, secondID)
	requireOpen(t, planned, movedID)
	requireRemaining(t, c.splitStep, "4", unitEach, planned)
	waitForOnHand(t, c.c.id, "4")
	waitForOnHand(t, c.b.id, "4")

	// The station lists what was scanned there, with what a label prints.
	splitRows := stationBatches(t, c.splitStation)
	require.Contains(t, splitRows, firstID)
	assert.NotContains(t, splitRows, secondID)
	assert.NotNil(t, jsonObject(splitRows[firstID], "machines"))
	initRow := stationBatches(t, c.initStation)[planned]
	require.NotNil(t, initRow)
	lots := jsonArray(jsonObject(initRow, "lots"), "data")
	require.NotEmpty(t, lots, "an initialized batch carries its run's number as a lot: %v", initRow)
	runLot := lots[len(lots)-1].(map[string]any)
	assert.Equal(t, "productionRun", jsonField(runLot, "type"))
	assert.Equal(t, jsonField(jsonObject(initRow, "production_run"), "number"), jsonField(runLot, "lot_number"))
}

func TestBatchScanning_SplitCanCloseTheSourceEarly(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	moved := jsonField(moveScan(t, c.moveStation, c.moveStep, planned), "id")

	mustScan(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "2", "", "", true))
	requireClosed(t, moved, moved)
	requireScanError(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "1", "", "", false),
		400, "Batch not compatible with production step.")
}

func TestBatchScanning_RejectsWhatTheStationCannotDo(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	first := planScanBatch(t, c.a.id, "5", unitEach)
	second := planScanBatch(t, c.a.id, "5", unitEach)
	initializeScan(t, first, c.initStation, nil)
	initializeScan(t, second, c.initStation, nil)

	requireScanError(t, "move", map[string]any{"batch_ids": []string{}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation},
		400, "No batches to move.")
	requireScanError(t, "move", map[string]any{"batch_ids": []string{first, second}, "production_step_id": c.moveStep, "scanning_station_id": c.moveStation},
		400, "Single-part production step can only accept one batch at a time.")
	requireScanError(t, "move", map[string]any{"batch_ids": []string{first}, "production_step_id": c.moveStep, "scanning_station_id": "sgsn_doesnotexist"},
		404, "Scanning Station not found.")
	requireScanError(t, "move", map[string]any{"batch_ids": []string{first}, "production_step_id": "prst_doesnotexist", "scanning_station_id": c.moveStation},
		404, "Production step not found.")

	moved := jsonField(moveScan(t, c.moveStation, c.moveStep, first), "id")
	requireScanError(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved}, "0", "0", "", false),
		400, "No quantity to split.")
	requireScanError(t, "split", splitBody(c.splitStation, c.splitStep, []string{moved, moved}, "1", "", "", false),
		400, "Duplicate batches provided.")
}

// ──────────────────────────────────────────────
// Multi-part steps
// ──────────────────────────────────────────────

type multiPartFixture struct {
	material    scanItem
	p1, p2, out scanItem
	initStation string
	station     string
	step        string
}

// newMultiPartFixture builds a step that assembles one of out from one of p1 and two of p2, plus a
// material that is consumed but never scanned in.
func newMultiPartFixture(t *testing.T, stationType string) multiPartFixture {
	t.Helper()
	f := multiPartFixture{
		material:    newScanMaterial(t),
		p1:          newScanPart(t, scanShippingCategory),
		p2:          newScanPart(t, scanShippingCategory),
		out:         newScanPart(t, scanShippingCategory),
		initStation: newScanStation(t, "init_batch"),
		station:     newScanStation(t, stationType),
	}
	newScanStep(t, f.initStation, scanQty{f.p1.id, "1", unitEach})
	newScanStep(t, f.initStation, scanQty{f.p2.id, "1", unitEach})
	f.step = newScanStep(t, f.station, scanQty{f.out.id, "1", unitEach},
		scanConsumption{itemID: f.p1.id, value: "1", unitID: unitEach},
		scanConsumption{itemID: f.p2.id, value: "2", unitID: unitEach},
		scanConsumption{itemID: f.material.id, value: "1", unitID: unitEach})
	stockScanItem(t, f.material.id, "100", unitEach)
	return f
}

func (f multiPartFixture) initialized(t *testing.T, item scanItem, value string) string {
	t.Helper()
	id := planScanBatch(t, item.id, value, unitEach)
	initializeScan(t, id, f.initStation, nil)
	return id
}

func TestBatchScanning_MultiPartMoveNeedsEveryPartInStep(t *testing.T) {
	t.Parallel()
	f := newMultiPartFixture(t, "move_batch")
	p1 := f.initialized(t, f.p1, "5")
	p2 := f.initialized(t, f.p2, "10")
	p2Short := f.initialized(t, f.p2, "8")
	waitForOnHand(t, f.p2.id, "18")

	move := func(ids ...string) map[string]any {
		return map[string]any{"batch_ids": ids, "production_step_id": f.step, "scanning_station_id": f.station}
	}
	// The material is consumed, not scanned in, so the part is all that is missing.
	requireScanError(t, "move", move(p1), 400, "Missing required part: "+f.p2.sku)
	requireScanError(t, "move", move(p1, p2Short), 400, "All batches must have the same measure.")
	requireScanError(t, "move", move(p1, p1), 400, "Duplicate batches provided.")

	assembled := mustScan(t, "move", move(p1, p2))
	assertDecimalEqual(t, "5", jsonField(jsonObject(assembled, "quantity"), "value"), "assembled quantity")
	requireClosed(t, p1, p1)
	requireClosed(t, p2, p2)
	requireOpen(t, p2Short, p2Short)

	waitForOnHand(t, f.out.id, "5")
	waitForOnHand(t, f.p1.id, "0")
	waitForOnHand(t, f.p2.id, "8")
	waitForOnHand(t, f.material.id, "95")
	requireOneBatchScanned(t, jsonField(assembled, "id"))
}

func TestBatchScanning_MultiPartSplitAndRemaining(t *testing.T) {
	t.Parallel()
	f := newMultiPartFixture(t, "split_batch")
	p1 := f.initialized(t, f.p1, "6")
	p2 := f.initialized(t, f.p2, "12")

	requireRemaining(t, f.step, "6", unitEach, p1, p2)
	status, got, raw := postScan(t, "split", splitBody(f.station, f.step, []string{p1}, "1", "", "", false))
	requireStatus(t, 400, status, raw)
	assert.Equal(t, "Cannot split a single batch.", jsonField(jsonObject(got, "error"), "message"))

	mustScan(t, "split", splitBody(f.station, f.step, []string{p1, p2}, "2", "1", "", false))
	// Only firsts count against the assembly's remainder.
	requireRemaining(t, f.step, "4", unitEach, p1, p2)
	requireOpen(t, p1, p1)

	mustScan(t, "split", splitBody(f.station, f.step, []string{p1, p2}, "3", "", "", false))
	// The first part alone decides whether the parts are used up: 2+1 and 3 of its 6. With it closed
	// the set no longer resolves at the step, as on the dashboard.
	requireClosed(t, p1, p1)
	requireOpen(t, p2, p2)
	status, got, raw = remainingToSplit(t, f.step, p1, p2)
	requireStatus(t, 400, status, raw)
	assert.Equal(t, "Batch not compatible with production step.", jsonField(jsonObject(got, "error"), "message"))
	waitForOnHand(t, f.out.id, "5")
}

// ──────────────────────────────────────────────
// Merge
// ──────────────────────────────────────────────

func TestBatchScanning_MergeSinglePartTotalsTheBatches(t *testing.T) {
	t.Parallel()
	material := newScanMaterial(t)
	part := newScanPart(t, scanShippingCategory)
	out := newScanPart(t, scanShippingCategory)
	initStation := newScanStation(t, "init_batch")
	mergeStation := newScanStation(t, "merge_batch")
	newScanStep(t, initStation, scanQty{part.id, "1", unitEach})
	mergeStep := newScanStep(t, mergeStation, scanQty{out.id, "2", unitEach},
		scanConsumption{itemID: part.id, value: "1", unitID: unitEach},
		scanConsumption{itemID: material.id, value: "1", unitID: unitEach})
	stockScanItem(t, material.id, "50", unitEach)

	five := planScanBatch(t, part.id, "5", unitEach)
	seven := planScanBatch(t, part.id, "7", unitEach)
	initializeScan(t, five, initStation, nil)
	initializeScan(t, seven, initStation, nil)

	requireScanError(t, "merge", map[string]any{"batch_ids": []string{}, "production_step_id": mergeStep, "scanning_station_id": mergeStation},
		400, "No batches to merge.")

	merged := mustScan(t, "merge", map[string]any{"batch_ids": []string{five, seven}, "production_step_id": mergeStep, "scanning_station_id": mergeStation})
	// 12 of the part make 24 at two per part.
	assertDecimalEqual(t, "24", jsonField(jsonObject(merged, "quantity"), "value"), "merged quantity")
	requireClosed(t, five, five)
	requireClosed(t, seven, seven)
	waitForOnHand(t, out.id, "24")
	waitForOnHand(t, part.id, "0")
	waitForOnHand(t, material.id, "38")
	requireOneBatchScanned(t, jsonField(merged, "id"))
}

func TestBatchScanning_MergeMultiPartTotalsEachPart(t *testing.T) {
	t.Parallel()
	f := newMultiPartFixture(t, "merge_batch")
	p1a := f.initialized(t, f.p1, "3")
	p1b := f.initialized(t, f.p1, "2")
	p2 := f.initialized(t, f.p2, "10")
	p2Short := f.initialized(t, f.p2, "6")

	merge := func(ids ...string) map[string]any {
		return map[string]any{"batch_ids": ids, "production_step_id": f.step, "scanning_station_id": f.station}
	}
	requireScanError(t, "merge", merge(p1a, p1b), 400, "Missing required part: "+f.p2.sku)
	requireScanError(t, "merge", merge(p1a, p1b, p2Short), 400, "All batches must have the same measure.")

	merged := mustScan(t, "merge", merge(p1a, p1b, p2))
	assertDecimalEqual(t, "5", jsonField(jsonObject(merged, "quantity"), "value"), "merged quantity")
	waitForOnHand(t, f.out.id, "5")
	waitForOnHand(t, f.p1.id, "0")
	waitForOnHand(t, f.p2.id, "6")
	waitForOnHand(t, f.material.id, "95")
}

// ──────────────────────────────────────────────
// Units
// ──────────────────────────────────────────────

// A sock part is counted in pairs. A batch planned in eaches is stepped through in the step's pairs.
func TestBatchScanning_QuantitiesConvertIntoTheStepsUnit(t *testing.T) {
	t.Parallel()
	material := newScanMaterial(t)
	sock := newScanPart(t, SeedItemCategoryID)
	finished := newScanPart(t, SeedItemCategoryID)
	trimmed := newScanPart(t, SeedItemCategoryID)
	initStation := newScanStation(t, "init_batch")
	moveStation := newScanStation(t, "move_batch")
	splitStation := newScanStation(t, "split_batch")
	newScanStep(t, initStation, scanQty{sock.id, "1", SeedPairUnitID},
		scanConsumption{itemID: material.id, value: "1", unitID: unitEach})
	moveStep := newScanStep(t, moveStation, scanQty{finished.id, "1", SeedPairUnitID},
		scanConsumption{itemID: sock.id, value: "1", unitID: SeedPairUnitID})
	splitStep := newScanStep(t, splitStation, scanQty{trimmed.id, "1", SeedPairUnitID},
		scanConsumption{itemID: finished.id, value: "1", unitID: SeedPairUnitID})
	stockScanItem(t, material.id, "100", unitEach)

	planned := planScanBatch(t, sock.id, "12", unitEach)
	initializeScan(t, planned, initStation, nil)
	waitForOnHand(t, sock.id, "6") // pairs
	waitForOnHand(t, material.id, "94")

	moved := moveScan(t, moveStation, moveStep, planned)
	assertDecimalEqual(t, "6", jsonField(jsonObject(moved, "quantity"), "value"), "12 eaches are 6 pairs")
	waitForOnHand(t, finished.id, "6")
	waitForOnHand(t, sock.id, "0")

	// What is left to split is in the base unit of what the step makes.
	requireRemaining(t, splitStep, "6", SeedPairUnitID, planned)

	// A split is entered in the unit the remainder came back in. Any other is refused, as on the
	// dashboard, because whether the source is used up is decided in that unit.
	splitIn := func(measure, unitID string) map[string]any {
		return map[string]any{
			"batch_ids": []string{planned}, "production_step_id": splitStep, "scanning_station_id": splitStation,
			"firsts": map[string]any{"measure": measure, "unit_id": unitID}, "close_batch": false,
		}
	}
	requireScanError(t, "split", splitIn("4", unitEach), 400, "Produced unit mismatch.")

	split := mustScan(t, "split", splitIn("2", SeedPairUnitID))
	assertDecimalEqual(t, "2", jsonField(jsonObject(split, "quantity"), "value"), "split firsts as entered")
	requireRemaining(t, splitStep, "4", SeedPairUnitID, planned)
	waitForOnHand(t, trimmed.id, "2")
	waitForOnHand(t, finished.id, "4")
}

// A step written in a unit other than the base unit of what it makes cannot be split by the remaining
// quantity, as on the dashboard: the remainder is refused rather than computed in mismatched units.
func TestBatchScanning_RemainingRefusesAStepNotWrittenInItsBaseUnit(t *testing.T) {
	t.Parallel()
	sock := newScanPart(t, SeedItemCategoryID)
	trimmed := newScanPart(t, SeedItemCategoryID)
	initStation := newScanStation(t, "init_batch")
	splitStation := newScanStation(t, "split_batch")
	newScanStep(t, initStation, scanQty{sock.id, "1", SeedPairUnitID})
	splitStep := newScanStep(t, splitStation, scanQty{trimmed.id, "2", unitEach},
		scanConsumption{itemID: sock.id, value: "1", unitID: SeedPairUnitID})

	planned := planScanBatch(t, sock.id, "3", SeedPairUnitID)
	initializeScan(t, planned, initStation, nil)

	status, got, raw := remainingToSplit(t, splitStep, planned)
	requireStatus(t, 400, status, raw)
	assert.Equal(t, "Produced unit mismatch.", jsonField(jsonObject(got, "error"), "message"))
}

// A consumption's waste is part of what one of the step's outputs takes.
func TestBatchScanning_ConsumptionWasteScalesTheOutput(t *testing.T) {
	t.Parallel()
	part := newScanPart(t, scanShippingCategory)
	out := newScanPart(t, scanShippingCategory)
	initStation := newScanStation(t, "init_batch")
	moveStation := newScanStation(t, "move_batch")
	newScanStep(t, initStation, scanQty{part.id, "1", unitEach})
	moveStep := newScanStep(t, moveStation, scanQty{out.id, "1", unitEach},
		scanConsumption{itemID: part.id, value: "3", unitID: unitEach, waste: "1"})

	planned := planScanBatch(t, part.id, "10", unitEach)
	initializeScan(t, planned, initStation, nil)
	moved := moveScan(t, moveStation, moveStep, planned)
	assertDecimalEqual(t, "2.5", jsonField(jsonObject(moved, "quantity"), "value"), "10 parts at 3 + 1 waste each")
}

// ──────────────────────────────────────────────
// Initialize options
// ──────────────────────────────────────────────

func TestBatchScanning_InitializeWithoutConsumingMaterials(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	quiet := planScanBatch(t, c.a.id, "4", unitEach)
	counted := planScanBatch(t, c.a.id, "6", unitEach)

	scanned := initializeScan(t, quiet, c.initStation, map[string]any{"consume_materials": false})
	assert.NotEmpty(t, jsonField(scanned, "scanned_at"), "the batch is still stamped as scanned")
	initializeScan(t, counted, c.initStation, map[string]any{"consume_materials": true})

	// Only the counted batch moves inventory.
	waitForOnHand(t, c.a.id, "6")
	waitForOnHand(t, c.material.id, "988")
	assert.Empty(t, scanEvents(t, "core.event.batch_scanned", quiet), "a scan that consumes nothing hands nothing off")
	requireOneBatchScanned(t, counted)
}

func TestBatchScanning_InitializePicksTheEntryStep(t *testing.T) {
	t.Parallel()
	upstreamPart := newScanPart(t, scanShippingCategory)
	part := newScanPart(t, scanShippingCategory)
	station := newScanStation(t, "init_batch")
	otherStation := newScanStation(t, "init_batch")
	newScanStep(t, otherStation, scanQty{upstreamPart.id, "1", unitEach})
	// Two steps at the station make the part; only one has nothing upstream.
	newScanStep(t, station, scanQty{part.id, "1", unitEach},
		scanConsumption{itemID: upstreamPart.id, value: "1", unitID: unitEach})
	entry := newScanStep(t, station, scanQty{part.id, "1", unitEach})

	planned := planScanBatch(t, part.id, "1", unitEach)
	scanned := initializeScan(t, planned, station, nil)
	assert.Equal(t, entry, jsonField(jsonObject(scanned, "production_step"), "id"))
}

func TestBatchScanning_InitializeAsAnotherStationType(t *testing.T) {
	t.Parallel()
	part := newScanPart(t, scanShippingCategory)
	other := newScanPart(t, scanShippingCategory)
	moveStation := newScanStation(t, "move_batch")
	only := newScanStep(t, moveStation, scanQty{part.id, "1", unitEach})
	ambiguousStation := newScanStation(t, "move_batch")
	firstChoice := newScanStep(t, ambiguousStation, scanQty{other.id, "1", unitEach})
	newScanStep(t, ambiguousStation, scanQty{other.id, "2", unitEach})

	// One step at the station makes the part: initializing as an init station uses it.
	planned := planScanBatch(t, part.id, "3", unitEach)
	scanned := initializeScan(t, planned, moveStation, map[string]any{"type_override": "init_batch"})
	assert.Equal(t, only, jsonField(jsonObject(scanned, "production_step"), "id"))
	waitForOnHand(t, part.id, "3")

	// Two do: the operator has to pick, and the pick has to be one of them.
	ambiguous := planScanBatch(t, other.id, "2", unitEach)
	requireScanError(t, "initialize", map[string]any{"batch_id": ambiguous, "scanning_station_id": ambiguousStation, "type_override": "init_batch"},
		400, "Multiple production steps match. Please select one.")
	requireScanError(t, "initialize", map[string]any{"batch_id": ambiguous, "scanning_station_id": ambiguousStation, "type_override": "init_batch", "production_step_id": only},
		400, "Selected production step is not valid for this scanning station and part.")

	status, raw, err := apiClient.Post(batchesPath+"/"+ambiguous+"/init-steps", map[string]any{"scanning_station_id": ambiguousStation}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Len(t, jsonArray(parseJSON(raw), "data"), 2)

	picked := initializeScan(t, ambiguous, ambiguousStation, map[string]any{"type_override": "init_batch", "production_step_id": firstChoice})
	assert.Equal(t, firstChoice, jsonField(jsonObject(picked, "production_step"), "id"))

	status, _, raw = postScan(t, "initialize", map[string]any{"batch_id": planScanBatch(t, part.id, "1", unitEach), "scanning_station_id": moveStation, "type_override": "not_a_type"})
	requireStatus(t, 400, status, raw)
}

// ──────────────────────────────────────────────
// Material check
// ──────────────────────────────────────────────

func TestBatchScanning_ConsumptionPreviewForEachStationType(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)

	// Initialize: the entry step's materials for the batch's own quantity.
	rows := consumptionPreview(t, c.initStation, map[string]any{"batch_ids": []string{planned}})
	require.Len(t, rows, 1)
	assert.Equal(t, scanConsumptionRow{SKU: c.material.sku, DemandMeasure: "20", DemandUnit: "ea", InventoryMeasure: "1000", InventoryUnit: "ea"}, rows[c.material.sku])

	initializeScan(t, planned, c.initStation, nil)
	waitForOnHand(t, c.material.id, "980")
	waitForOnHand(t, c.a.id, "10")

	// Move: the step's consumptions, parts and materials, for what the batch makes there.
	rows = consumptionPreview(t, c.moveStation, map[string]any{"batch_ids": []string{planned}, "production_step_id": c.moveStep})
	require.Len(t, rows, 2)
	assert.Equal(t, scanConsumptionRow{SKU: c.a.sku, DemandMeasure: "10", DemandUnit: "ea", InventoryMeasure: "10", InventoryUnit: "ea"}, rows[c.a.sku])
	assert.Equal(t, scanConsumptionRow{SKU: c.material.sku, DemandMeasure: "10", DemandUnit: "ea", InventoryMeasure: "980", InventoryUnit: "ea"}, rows[c.material.sku])

	// Merge, by override: the same demand through the many-batch path.
	rows = consumptionPreview(t, c.moveStation, map[string]any{"batch_ids": []string{planned}, "production_step_id": c.moveStep, "type_override": "merge_batch"})
	assert.Equal(t, "10", rows[c.a.sku].DemandMeasure)

	moveScan(t, c.moveStation, c.moveStep, planned)

	// Split: the step's consumptions for the quantity being split off.
	rows = consumptionPreview(t, c.splitStation, map[string]any{
		"batch_ids": []string{planned}, "production_step_id": c.splitStep,
		"split_quantity": map[string]any{"measure": "3", "unit_id": unitEach},
	})
	require.Len(t, rows, 1)
	assert.Equal(t, "3", rows[c.b.sku].DemandMeasure)

	status, raw, err := apiClient.Post(scanningStationsPath+"/"+c.splitStation+"/consumptions", map[string]any{
		"batch_ids": []string{planned}, "production_step_id": c.splitStep,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, raw)
	assert.Equal(t, "Split quantity not found.", jsonField(jsonObject(parseJSON(raw), "error"), "message"))
}

// Demand reads out to four places with trailing zeros dropped, and whole numbers plainly.
func TestBatchScanning_ConsumptionPreviewFormatsDemand(t *testing.T) {
	t.Parallel()
	material := newScanMaterial(t)
	part := newScanPart(t, scanShippingCategory)
	station := newScanStation(t, "init_batch")
	newScanStep(t, station, scanQty{part.id, "3", unitEach},
		scanConsumption{itemID: material.id, value: "1", unitID: unitEach})

	planned := planScanBatch(t, part.id, "10", unitEach)
	rows := consumptionPreview(t, station, map[string]any{"batch_ids": []string{planned}})
	assert.Equal(t, "3.3333", rows[material.sku].DemandMeasure)
	assert.Equal(t, "0", rows[material.sku].InventoryMeasure)
}

// ──────────────────────────────────────────────
// Deleting
// ──────────────────────────────────────────────

func TestBatchScanning_BulkDeleteUndoesScansAndDropsPlannedBatches(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	first := planScanBatch(t, c.a.id, "4", unitEach)
	second := planScanBatch(t, c.a.id, "6", unitEach)
	unscanned := planScanBatch(t, c.a.id, "1", unitEach)
	initializeScan(t, first, c.initStation, nil)
	initializeScan(t, second, c.initStation, nil)
	waitForOnHand(t, c.a.id, "10")
	waitForOnHand(t, c.material.id, "980")

	status, raw, err := apiClient.Post(batchesPath+"/actions/bulk-delete", map[string]any{
		"batch_ids": []string{first, second, unscanned},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 300, "bulk delete: %s", raw)

	waitForOnHand(t, c.a.id, "0")
	waitForOnHand(t, c.material.id, "1000")

	// Undone init scans go back to their runs, unscanned, and can be scanned again.
	assert.NotContains(t, stationBatches(t, c.initStation), first)
	again := initializeScan(t, first, c.initStation, nil)
	assert.NotEmpty(t, jsonField(again, "scanned_at"))
	waitForOnHand(t, c.a.id, "4")
}

// A batch whose output a later scan used cannot be undone until that scan is.
func TestBatchScanning_UndoIsRefusedWhileALaterScanHoldsTheBatch(t *testing.T) {
	t.Parallel()
	c := newScanChain(t)
	planned := planScanBatch(t, c.a.id, "10", unitEach)
	initializeScan(t, planned, c.initStation, nil)
	moved := jsonField(moveScan(t, c.moveStation, c.moveStep, planned), "id")
	waitForOnHand(t, c.b.id, "10")

	status, raw, err := apiClient.Delete(batchesPath + "/" + planned)
	require.NoError(t, err)
	requireStatus(t, 400, status, raw)
	assert.Equal(t, "This batch has already been used by a later scan. Delete that batch first.", jsonField(jsonObject(parseJSON(raw), "error"), "message"))

	deleteBatch(t, moved)
	waitForOnHand(t, c.b.id, "0")
	waitForOnHand(t, c.a.id, "10")
	requireOpen(t, planned, planned)

	deleteBatch(t, planned)
	waitForOnHand(t, c.a.id, "0")
	waitForOnHand(t, c.material.id, "1000")
}
