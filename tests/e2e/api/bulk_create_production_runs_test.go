//go:build e2e

package api_test

import (
	"fmt"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const productionRunsBulkCreatePath = productionRunsPath + "/actions/bulk-create"

func bulkCreateProductionRuns(t *testing.T, runs ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(runs))
	for i, r := range runs {
		rows[i] = r
	}
	status, body, err := apiClient.Post(productionRunsBulkCreatePath, map[string]any{"production_runs": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

// Seed IDs from shared/db/seed (0005_measures.sql, 0007_items.sql, 0009_production.sql).
const (
	seedEachUnitID          = "each"
	seedHourUnitID          = "hour"
	seedKnitLargeSockStepID = "prs_01k0a51qxceydax5036pegvzzy"
	seedKnittingStationID   = "sgsn_01k0a8201zegarjfsjaw5n7yfv"
)

// unitID builds a fuzzy unit reference that identifies a unit by its ID.
func unitID(id string) map[string]any {
	return map[string]any{"id": id}
}

// itemSKU builds a fuzzy item reference that identifies an item by its SKU.
func itemSKU(sku string) map[string]any {
	return map[string]any{"sku": sku}
}

// bulkRunRow builds a minimal valid production run row with one batch of a seeded item.
func bulkRunRow() map[string]any {
	return map[string]any{
		"responsible_user_id": SeedUserID,
		"batches": []any{
			map[string]any{"item": itemSKU(SeedItemSKU), "quantity_value": "100", "quantity_unit": unitID(seedEachUnitID)},
		},
	}
}

// bulkCreatedRuns reads the pre-generated runs a bulk-create 202 carries. The 202 body
// is the canonical job resource; a create records its pre-generated ids on the job's
// row-indexed results at accept, so each entry names the run it will create and, as its
// sub-resources, that run's batches.
func bulkCreatedRuns(body []byte) []map[string]any {
	return jobResults(parseJSON(body))
}

// cleanupBulkCreatedRuns deletes every run in a bulk-create response.
func cleanupBulkCreatedRuns(body []byte) {
	for _, run := range bulkCreatedRuns(body) {
		{
			if runID := jobResultResourceID(run); runID != "" {
				apiClient.Delete(productionRunsPath + "/" + runID)
			}
		}
	}
}

// waitForRun polls until the asynchronously created run is visible and returns it.
func waitForRun(t *testing.T, runID string) map[string]any {
	t.Helper()
	var run map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, err := apiClient.GetListRaw(productionRunsPath+"/"+runID, nil)
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("run %s not created yet (status %d)", runID, status)
		}
		run = parseJSON(body)
		return nil
	})
	return run
}

func TestProductionRuns_BulkCreate_AllCreates(t *testing.T) {
	t.Parallel()

	status, body := bulkCreateProductionRuns(t, bulkRunRow(), bulkRunRow())
	requireStatus(t, 202, status, body)
	defer cleanupBulkCreatedRuns(body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	runs := bulkCreatedRuns(body)
	require.Len(t, runs, 2)

	// The acknowledgment carries the pre-generated IDs; the runs appear
	// asynchronously with distinct sequential numbers.
	numbers := map[string]struct{}{}
	for _, run := range runs {
		runID := jobResultResourceID(run)
		assertIDFormat(t, runID, id.ProductionRunIDPrefix)

		batchIDs := jobResultSubResourceIDs(run)
		require.Len(t, batchIDs, 1)
		assertIDFormat(t, batchIDs[0], id.BatchIDPrefix)

		created := waitForRun(t, runID)
		number := jsonField(created, "number")
		require.NotEmpty(t, number)
		numbers[number] = struct{}{}
	}
	assert.Len(t, numbers, 2, "run numbers must be distinct")
}

// TestProductionRuns_BulkCreate_CreateWithAllFields exercises the full batch shape:
// seconds, waste, and step and station references — then verifies the batches landed
// on the run once the async creation completes.
func TestProductionRuns_BulkCreate_CreateWithAllFields(t *testing.T) {
	t.Parallel()

	status, body := bulkCreateProductionRuns(t, map[string]any{
		"responsible_user_id": SeedUserID,
		"batches": []any{
			map[string]any{
				"item":               itemSKU(SeedItemSKU),
				"quantity_value":     "100",
				"quantity_unit":      unitID(seedEachUnitID),
				"seconds_value":      "3600",
				"seconds_unit":       unitID(seedHourUnitID),
				"waste_value":        "2",
				"waste_unit":         unitID(seedEachUnitID),
				"production_step_id": seedKnitLargeSockStepID,
				"scanning_station":   map[string]any{"id": seedKnittingStationID},
			},
			map[string]any{"item": itemSKU("SCK-002"), "quantity_value": "50", "quantity_unit": unitID(seedEachUnitID)},
		},
	})
	requireStatus(t, 202, status, body)
	defer cleanupBulkCreatedRuns(body)

	runs := bulkCreatedRuns(body)
	require.Len(t, runs, 1)
	runID := jobResultResourceID(runs[0])
	require.Len(t, jobResultSubResourceIDs(runs[0]), 2)

	waitForRun(t, runID)
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		listStatus, listBody, err := apiClient.GetListRaw(productionRunsPath+"/"+runID+"/batches", nil)
		if err != nil {
			return err
		}
		if listStatus != 200 {
			return fmt.Errorf("batches for run %s not listable yet (status %d)", runID, listStatus)
		}
		if got := len(jsonArray(parseJSON(listBody), "data")); got != 2 {
			return fmt.Errorf("run %s has %d batches, want 2", runID, got)
		}
		return nil
	})
}

// TestProductionRuns_BulkCreate_IdempotentReplayReturnsSameAcknowledgment: replaying
// the request with the same Idempotency-Key must return the identical acknowledgment
// — same pre-generated run and batch IDs — not enqueue a second creation.
func TestProductionRuns_BulkCreate_IdempotentReplayReturnsSameAcknowledgment(t *testing.T) {
	t.Parallel()

	key := newIdempotencyKey()
	payload := map[string]any{"production_runs": []any{bulkRunRow()}}

	status, body, err := apiClient.Post(productionRunsBulkCreatePath, payload, key)
	require.NoError(t, err)
	requireStatus(t, 202, status, body)
	defer cleanupBulkCreatedRuns(body)

	replayStatus, replayBody, err := apiClient.Post(productionRunsBulkCreatePath, payload, key)
	require.NoError(t, err)
	requireStatus(t, 202, replayStatus, replayBody)

	assert.JSONEq(t, string(body), string(replayBody), "replay must acknowledge the same pre-generated IDs")
}

func TestProductionRuns_BulkCreate_TooManyRejected(t *testing.T) {
	t.Parallel()

	rows := make([]map[string]any, 1001)
	for i := range rows {
		rows[i] = bulkRunRow()
	}
	status, body := bulkCreateProductionRuns(t, rows...)
	requireStatus(t, 400, status, body)
}

func TestProductionRuns_BulkCreate_EmptyRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(productionRunsBulkCreatePath, map[string]any{"production_runs": []any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

func TestProductionRuns_BulkCreate_RejectsRunWithoutBatches(t *testing.T) {
	t.Parallel()

	status, body := bulkCreateProductionRuns(t, map[string]any{
		"responsible_user_id": SeedUserID,
		"batches":             []any{},
	})
	requireStatus(t, 400, status, body)
}

func TestProductionRuns_BulkCreate_RejectsUnknownSKU(t *testing.T) {
	t.Parallel()

	row := bulkRunRow()
	row["batches"] = []any{
		map[string]any{"item": itemSKU(uniqueName("no-such-sku")), "quantity_value": "1", "quantity_unit": unitID(seedEachUnitID)},
	}
	status, body := bulkCreateProductionRuns(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_runs[0].batches[0].item")
}

func TestProductionRuns_BulkCreate_RejectsUnknownUnit(t *testing.T) {
	t.Parallel()

	row := bulkRunRow()
	row["batches"] = []any{
		map[string]any{"item": itemSKU(SeedItemSKU), "quantity_value": "1", "quantity_unit": unitID(mustGenID(t, id.UnitIDPrefix))},
	}
	status, body := bulkCreateProductionRuns(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_runs[0].batches[0].quantity_unit")
}

// TestProductionRuns_BulkCreate_ResolvesUnitByNameAndAbbreviation: a unit reference
// resolves fuzzily — by abbreviation or name, not only by ID.
func TestProductionRuns_BulkCreate_ResolvesUnitByNameAndAbbreviation(t *testing.T) {
	t.Parallel()

	status, body := bulkCreateProductionRuns(t, map[string]any{
		"responsible_user_id": SeedUserID,
		"batches": []any{
			map[string]any{
				"item":           itemSKU(SeedItemSKU),
				"quantity_value": "5",
				"quantity_unit":  map[string]any{"abbreviation": "ea"}, // by abbreviation
				"seconds_value":  "60",
				"seconds_unit":   map[string]any{"name": "Hour"}, // by name
			},
		},
	})
	requireStatus(t, 202, status, body)
	defer cleanupBulkCreatedRuns(body)

	runs := bulkCreatedRuns(body)
	require.Len(t, runs, 1)
	waitForRun(t, jobResultResourceID(runs[0]))
}

func TestProductionRuns_BulkCreate_RejectsUnknownStep(t *testing.T) {
	t.Parallel()

	row := bulkRunRow()
	row["batches"] = []any{
		map[string]any{
			"item": itemSKU(SeedItemSKU), "quantity_value": "1", "quantity_unit": unitID(seedEachUnitID),
			"production_step_id": mustGenID(t, id.ProductionStepIDPrefix),
		},
	}
	status, body := bulkCreateProductionRuns(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_runs[0].batches[0].production_step_id")
}

func TestProductionRuns_BulkCreate_RejectsUnknownStation(t *testing.T) {
	t.Parallel()

	row := bulkRunRow()
	row["batches"] = []any{
		map[string]any{
			"item": itemSKU(SeedItemSKU), "quantity_value": "1", "quantity_unit": unitID(seedEachUnitID),
			"scanning_station": map[string]any{"id": mustGenID(t, id.ScanningStationIDPrefix)},
		},
	}
	status, body := bulkCreateProductionRuns(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_runs[0].batches[0].scanning_station")
}

// TestProductionRuns_BulkCreate_ResolvesStationByName: the scanning station is a fuzzy
// reference — it resolves by name, not only by id.
func TestProductionRuns_BulkCreate_ResolvesStationByName(t *testing.T) {
	t.Parallel()

	row := bulkRunRow()
	row["batches"] = []any{
		map[string]any{
			"item": itemSKU(SeedItemSKU), "quantity_value": "1", "quantity_unit": unitID(seedEachUnitID),
			"scanning_station": map[string]any{"name": "Knitting Station"}, // by name
		},
	}
	status, body := bulkCreateProductionRuns(t, row)
	requireStatus(t, 202, status, body)
	defer cleanupBulkCreatedRuns(body)

	runs := bulkCreatedRuns(body)
	require.Len(t, runs, 1)
	waitForRun(t, jobResultResourceID(runs[0]))
}

func TestProductionRuns_BulkCreate_RejectsUnknownResponsibleUser(t *testing.T) {
	t.Parallel()

	row := bulkRunRow()
	row["responsible_user_id"] = mustGenID(t, id.UserIDPrefix)
	status, body := bulkCreateProductionRuns(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_runs[0].responsible_user_id")
}

// TestProductionRuns_BulkCreate_RejectsNonNumericQuantity: quantity values are
// decimal strings — a non-decimal is rejected with a row-indexed param.
func TestProductionRuns_BulkCreate_RejectsNonNumericQuantity(t *testing.T) {
	t.Parallel()

	row := bulkRunRow()
	row["batches"] = []any{
		map[string]any{"item": itemSKU(SeedItemSKU), "quantity_value": "lots", "quantity_unit": unitID(seedEachUnitID)},
	}
	status, body := bulkCreateProductionRuns(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_runs[0].batches[0].quantity_value")
}
