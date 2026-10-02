//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const productionRunsPath = "/v1/operations/production-runs"

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func productionRunPath(runID string) string { return productionRunsPath + "/" + runID }

func productionRunBatchesPath(runID string) string { return productionRunPath(runID) + "/batches" }

// plannedBatch is a minimal valid batch of the seeded greige item, which the knitting
// station's step produces, so it can also be scanned.
func plannedBatch(quantity string) map[string]any {
	return map[string]any{
		"item_id":          SeedGreigeItemID,
		"quantity_value":   quantity,
		"quantity_unit_id": seedEachUnitID,
	}
}

// createProductionRun creates a run and deletes it when the test ends.
func createProductionRun(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	status, raw, err := apiClient.Post(productionRunsPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	run := parseJSON(raw)
	runID := jsonField(run, "id")
	require.NotEmpty(t, runID)
	t.Cleanup(func() { apiClient.Delete(productionRunPath(runID)) })
	return run
}

// createRunWithBatches creates a run owned by the seed user with the given planned batches.
func createRunWithBatches(t *testing.T, batches ...map[string]any) map[string]any {
	t.Helper()
	rows := make([]any, len(batches))
	for i, b := range batches {
		rows[i] = b
	}
	return createProductionRun(t, map[string]any{"responsible_user_id": SeedUserID, "batches": rows})
}

// renameRun gives a run a unique number so list tests can scope a search to it.
func renameRun(t *testing.T, runID, number string) {
	t.Helper()
	status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"number": number}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// assertBatchSummary asserts a run's only batch total.
func assertBatchSummary(t *testing.T, run map[string]any, itemID, quantity, batchCount string) {
	t.Helper()
	summaries := jsonArray(jsonObject(run, "batch_summaries"), "data")
	require.Len(t, summaries, 1, "one item in one unit totals to one summary: %v", run["batch_summaries"])
	s := summaries[0].(map[string]any)
	assertObjectField(t, s, "production_run_batch_summary")
	assert.Equal(t, itemID, jsonField(jsonObject(s, "item"), "id"))
	assert.Equal(t, "LKN", jsonField(jsonObject(s, "item"), "name"))
	assert.Equal(t, seedEachUnitID, jsonField(jsonObject(s, "unit"), "id"))
	assert.NotEmpty(t, jsonField(jsonObject(s, "unit"), "name"))
	assert.Equal(t, quantity, jsonField(s, "quantity_value"))
	assert.Equal(t, batchCount, jsonField(s, "batch_count"))
}

// runBatches lists a run's batches at the given scope, first page only.
func runBatches(t *testing.T, runID, scope string) []map[string]any {
	t.Helper()
	params := url.Values{"limit": {"100"}}
	if scope != "" {
		params.Set("scope", scope)
	}
	status, body, err := apiClient.GetListRaw(productionRunBatchesPath(runID), params)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var out []map[string]any
	for _, raw := range jsonArray(parseJSON(body), "data") {
		out = append(out, raw.(map[string]any))
	}
	return out
}

// listRunIDs returns the ids of the first page of runs matching params.
func listRunIDs(t *testing.T, params url.Values) []string {
	t.Helper()
	merged := url.Values{"limit": {"100"}}
	for k, vs := range params {
		merged[k] = vs
	}
	status, body, err := apiClient.GetListRaw(productionRunsPath, merged)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var ids []string
	for _, raw := range jsonArray(parseJSON(body), "data") {
		ids = append(ids, jsonField(raw.(map[string]any), "id"))
	}
	return ids
}

// scanBatch initializes a planned batch at the knitting station, which starts its run and,
// when it is the run's last unscanned batch, completes it.
func scanBatch(t *testing.T, batchID string) {
	t.Helper()
	status, body, err := apiClient.Post("/v1/operations/batches/actions/initialize", map[string]any{
		"batch_id":            batchID,
		"scanning_station_id": seedKnittingStationID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 300, "initialize batch %s: %s", batchID, string(body))
}

// deleteBatch deletes a batch, which reverses its scan.
func deleteBatch(t *testing.T, batchID string) {
	t.Helper()
	status, body, err := apiClient.Delete("/v1/operations/batches/" + batchID)
	require.NoError(t, err)
	require.Less(t, status, 300, "delete batch %s: %s", batchID, string(body))
}

// assertValidationParam asserts a 400 validation error naming param.
func assertValidationParam(t *testing.T, status int, body []byte, param string) {
	t.Helper()
	requireStatus(t, 400, status, body)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(body, &envelope), string(body))
	errObj, _ := envelope["error"].(map[string]any)
	require.NotNil(t, errObj, "missing error object: %s", string(body))
	assert.Equal(t, param, errObj["param"], "error.param: %s", string(body))
}

// ──────────────────────────────────────────────
// CRUD
// ──────────────────────────────────────────────

func TestProductionRuns_CRUD(t *testing.T) {
	t.Parallel()

	// Create
	status, raw, err := apiClient.Post(productionRunsPath, map[string]any{
		"responsible_user_id": SeedUserID,
		"batches":             []any{plannedBatch("10")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, raw)
	runID := jsonField(parseJSON(raw), "id")
	assertIDFormat(t, runID, id.ProductionRunIDPrefix)

	// Get
	got := parseJSON(mustGet(t, productionRunPath(runID)))
	assert.Equal(t, runID, jsonField(got, "id"))
	assert.Equal(t, "1", jsonField(got, "batch_count"))

	// Update
	number := uniqueName("e2e-pr-crud")
	status, raw, err = apiClient.Patch(productionRunPath(runID), map[string]any{"number": number}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	assert.Equal(t, number, jsonField(parseJSON(raw), "number"))
	assert.Equal(t, number, jsonField(parseJSON(mustGet(t, productionRunPath(runID))), "number"), "number persisted")

	// Delete
	status, raw, err = apiClient.Delete(productionRunPath(runID))
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)

	// Gone
	status, raw, err = apiClient.GetListRaw(productionRunPath(runID), nil)
	require.NoError(t, err)
	requireStatus(t, 404, status, raw)

	// Deleting again reports the run as already deleted rather than unknown.
	status, raw, err = apiClient.Delete(productionRunPath(runID))
	require.NoError(t, err)
	requireStatus(t, 410, status, raw)
	requireErrorResponse(t, raw, "resource_gone", "invalid_request_error")
}

func TestProductionRuns_CreateAndUpdateAllFields(t *testing.T) {
	t.Parallel()

	full := map[string]any{
		"item_id":             SeedGreigeItemID,
		"quantity_value":      "12.5",
		"quantity_unit_id":    seedEachUnitID,
		"seconds_value":       "2",
		"seconds_unit_id":     seedEachUnitID,
		"waste_value":         "0.5",
		"waste_unit_id":       seedEachUnitID,
		"production_step_id":  seedKnitLargeSockStepID,
		"scanning_station_id": seedKnittingStationID,
		"machine_ids":         []any{SeedMachineID},
	}
	run := createRunWithBatches(t, full, plannedBatch("3"))
	runID := jsonField(run, "id")

	assertIDFormat(t, runID, id.ProductionRunIDPrefix)
	assertObjectField(t, run, "production_run")
	assert.NotEmpty(t, jsonField(run, "number"), "number is assigned")
	assertNilField(t, run, "responsible_user")
	assert.Equal(t, "2", jsonField(run, "batch_count"))
	assertBatchSummary(t, run, SeedGreigeItemID, "15.5", "2")
	assertNilField(t, run, "started_at")
	assertNilField(t, run, "completed_at")
	assertValidTimestamp(t, jsonField(run, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(run, "updated_at"), "updated_at")

	// Every field of a planned batch, read back through the run's own batches.
	batches := runBatches(t, runID, "run")
	require.Len(t, batches, 2)
	var b map[string]any
	for _, candidate := range batches {
		if q := jsonObject(candidate, "quantity"); q != nil && jsonField(q, "value") == "12.5" {
			b = candidate
		}
	}
	require.NotNil(t, b, "the fully specified batch is listed: %v", batches)
	assertIDFormat(t, jsonField(b, "id"), id.BatchIDPrefix)
	assertObjectField(t, b, "batch")
	item := jsonObject(b, "item")
	assert.Equal(t, SeedGreigeItemID, jsonField(item, "id"))
	assert.Equal(t, "LKN", jsonField(item, "name"), "item is named by its SKU")
	assert.Equal(t, "2", jsonField(jsonObject(b, "seconds"), "value"))
	assert.Equal(t, "0.5", jsonField(jsonObject(b, "waste"), "value"))
	assert.Equal(t, seedKnitLargeSockStepID, jsonField(jsonObject(b, "production_step"), "id"))
	assert.Equal(t, seedKnittingStationID, jsonField(jsonObject(b, "scanning_station"), "id"))
	assert.Equal(t, runID, jsonField(jsonObject(b, "production_run"), "id"))
	machines := jsonArray(jsonObject(b, "machines"), "data")
	require.Len(t, machines, 1)
	assert.Equal(t, SeedMachineID, jsonField(machines[0].(map[string]any), "id"))
	assert.NotEmpty(t, jsonField(machines[0].(map[string]any), "handle"), "machine handle is its serial number")
	assertNilField(t, b, "closed_at")
	assertNilField(t, b, "scanned_at")
	assertValidTimestamp(t, jsonField(b, "created_at"), "created_at")
	lots := jsonArray(jsonObject(b, "lots"), "data")
	require.NotEmpty(t, lots, "a planned batch carries its run's lot")
	assert.Equal(t, jsonField(run, "number"), jsonField(lots[len(lots)-1].(map[string]any), "lot_number"))

	// Update every writable field; the account user id and the user id both resolve.
	number := uniqueName("e2e-pr-all")
	status, raw, err := apiClient.Patch(productionRunPath(runID)+"?include=responsible_user", map[string]any{
		"number":              number,
		"responsible_user_id": SeedAdmin2AccountUserID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, raw)
	updated := parseJSON(raw)
	assert.Equal(t, number, jsonField(updated, "number"))
	assert.Equal(t, SeedAdmin2AccountUserID, jsonField(jsonObject(updated, "responsible_user"), "id"))
	// Preserved
	assert.Equal(t, "2", jsonField(updated, "batch_count"))
	assert.Equal(t, jsonField(run, "created_at"), jsonField(updated, "created_at"))

	persisted := parseJSON(mustGet(t, productionRunPath(runID)+"?include=responsible_user"))
	assert.Equal(t, number, jsonField(persisted, "number"))
	assert.Equal(t, SeedAdmin2AccountUserID, jsonField(jsonObject(persisted, "responsible_user"), "id"))
}

// A batch's item carries its description as the handle, which batch labels print. SKN is
// used because no test rewrites its seeded description.
func TestProductionRuns_BatchItemHandleIsDescription(t *testing.T) {
	t.Parallel()

	batch := plannedBatch("1")
	batch["item_id"] = SeedSknItemID
	runID := jsonField(createRunWithBatches(t, batch), "id")

	item := jsonObject(runBatches(t, runID, "run")[0], "item")
	assert.Equal(t, SeedSknItemSKU, jsonField(item, "name"))
	assert.Equal(t, "Small Knitted Sock", jsonField(item, "handle"))
}

func TestProductionRuns_OmittedFields(t *testing.T) {
	t.Parallel()

	t.Run("create without batches makes an empty run", func(t *testing.T) {
		run := createProductionRun(t, map[string]any{"responsible_user_id": SeedUserID})
		assert.Equal(t, "0", jsonField(run, "batch_count"))
		assert.Empty(t, jsonArray(jsonObject(run, "batch_summaries"), "data"))
		assert.Empty(t, runBatches(t, jsonField(run, "id"), "run"))
	})

	t.Run("create without responsible_user_id is rejected", func(t *testing.T) {
		status, body, err := apiClient.Post(productionRunsPath, map[string]any{"batches": []any{plannedBatch("1")}}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("patching the number preserves the responsible user", func(t *testing.T) {
		run := createRunWithBatches(t, plannedBatch("1"))
		runID := jsonField(run, "id")
		renameRun(t, runID, uniqueName("e2e-pr-omit"))
		got := parseJSON(mustGet(t, productionRunPath(runID)+"?include=responsible_user"))
		assert.Equal(t, SeedAccountUserID, jsonField(jsonObject(got, "responsible_user"), "id"))
	})

	t.Run("empty patch is rejected", func(t *testing.T) {
		run := createRunWithBatches(t, plannedBatch("1"))
		status, body, err := apiClient.Patch(productionRunPath(jsonField(run, "id")), map[string]any{}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})
}

func TestProductionRuns_CreateResponseShape(t *testing.T) {
	t.Parallel()

	resp, err := apiClient.PostFull(productionRunsPath, map[string]any{"responsible_user_id": SeedUserID}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, resp.StatusCode, resp.Body)
	run := parseJSON(resp.Body)
	runID := jsonField(run, "id")
	defer apiClient.Delete(productionRunPath(runID))

	assertIDFormat(t, runID, id.ProductionRunIDPrefix)
	assertObjectField(t, run, "production_run")
	assertCreatedLocation(t, resp.Header, runID)
	assertValidTimestamp(t, jsonField(run, "created_at"), "created_at")
	assertValidTimestamp(t, jsonField(run, "updated_at"), "updated_at")
}

func TestProductionRuns_SequentialNumbers(t *testing.T) {
	t.Parallel()

	a := createProductionRun(t, map[string]any{"responsible_user_id": SeedUserID})
	b := createProductionRun(t, map[string]any{"responsible_user_id": SeedUserID})
	assert.NotEqual(t, jsonField(a, "number"), jsonField(b, "number"), "each run gets its own number")
}

// ──────────────────────────────────────────────
// Create / add-batches validation
// ──────────────────────────────────────────────

// batchValidationCases are rejected identically by create and add-batches.
func batchValidationCases() []struct {
	name  string
	batch map[string]any
	param string
} {
	with := func(overrides map[string]any) map[string]any {
		b := plannedBatch("5")
		for k, v := range overrides {
			b[k] = v
		}
		return b
	}
	return []struct {
		name  string
		batch map[string]any
		param string
	}{
		{"non-decimal quantity", with(map[string]any{"quantity_value": "abc"}), "batches[0].quantity_value"},
		{"zero quantity", with(map[string]any{"quantity_value": "0"}), "batches[0].quantity_value"},
		{"negative quantity", with(map[string]any{"quantity_value": "-3"}), "batches[0].quantity_value"},
		{"non-decimal seconds", with(map[string]any{"seconds_value": "x", "seconds_unit_id": seedEachUnitID}), "batches[0].seconds_value"},
		{"negative waste", with(map[string]any{"waste_value": "-1", "waste_unit_id": seedEachUnitID}), "batches[0].waste_value"},
		{"unknown item", with(map[string]any{"item_id": "it_e2e_doesnotexist000"}), "batches[0].item_id"},
		{"unknown quantity unit", with(map[string]any{"quantity_unit_id": "un_e2e_doesnotexist000"}), "batches[0].quantity_unit_id"},
		{"unknown seconds unit", with(map[string]any{"seconds_value": "1", "seconds_unit_id": "un_e2e_doesnotexist000"}), "batches[0].seconds_unit_id"},
		{"unknown production step", with(map[string]any{"production_step_id": "prs_e2e_doesnotexist00"}), "batches[0].production_step_id"},
		{"unknown scanning station", with(map[string]any{"scanning_station_id": "sgsn_e2e_doesnotexist0"}), "batches[0].scanning_station_id"},
		{"unknown machine", with(map[string]any{"machine_ids": []any{SeedMachineID, "mc_e2e_doesnotexist00"}}), "batches[0].machine_ids[1]"},
	}
}

func TestProductionRuns_CreateRejectsInvalidBatches(t *testing.T) {
	t.Parallel()

	for _, tc := range batchValidationCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(productionRunsPath, map[string]any{
				"responsible_user_id": SeedUserID,
				"batches":             []any{tc.batch},
			}, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				apiClient.Delete(productionRunPath(jsonField(parseJSON(body), "id")))
			}
			assertValidationParam(t, status, body, tc.param)
		})
	}
}

func TestProductionRuns_CreateRejectsInvalidResponsibleUser(t *testing.T) {
	t.Parallel()

	for name, userID := range map[string]string{
		"unknown user":              "us_e2e_doesnotexist",
		"another account's user":    SeedTenantBAccountUserID,
		"a customer account's user": SeedCustomerAccountUserID,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.Post(productionRunsPath, map[string]any{"responsible_user_id": userID}, newIdempotencyKey())
			require.NoError(t, err)
			if status == 201 {
				apiClient.Delete(productionRunPath(jsonField(parseJSON(body), "id")))
			}
			assertValidationParam(t, status, body, "responsible_user_id")
		})
	}
}

// Every batch is validated before anything is written, so a bad batch after a good one
// rejects the whole create.
func TestProductionRuns_CreateRejectsWholeRunOnLaterBadBatch(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(productionRunsPath, map[string]any{
		"responsible_user_id": SeedUserID,
		"batches":             []any{plannedBatch("1"), plannedBatch("-1")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		apiClient.Delete(productionRunPath(jsonField(parseJSON(body), "id")))
	}
	assertValidationParam(t, status, body, "batches[1].quantity_value")
}

// Tenant B cannot plan batches of tenant A's items; the foreign key alone would allow it.
func TestProductionRuns_CreateRejectsCrossTenantItem(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	status, body, err := clientB.Post(productionRunsPath, map[string]any{
		"responsible_user_id": SeedTenantBAccountUserID,
		"batches":             []any{plannedBatch("1")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		clientB.Delete(productionRunPath(jsonField(parseJSON(body), "id")))
	}
	assertValidationParam(t, status, body, "batches[0].item_id")
}

func TestProductionRuns_CreateRejectsUnknownField(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Post(productionRunsPath, map[string]any{
		"responsible_user_id": SeedUserID,
		bogusE2EJSONField:     "x",
	}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		apiClient.Delete(productionRunPath(jsonField(parseJSON(body), "id")))
	}
	assertJSONUnknownFieldRejected(t, "POST", productionRunsPath, status, body)
}

func TestProductionRuns_CreateIdempotency(t *testing.T) {
	t.Parallel()

	key := newIdempotencyKey()
	body := map[string]any{"responsible_user_id": SeedUserID, "batches": []any{plannedBatch("4")}}

	status1, raw1, err := apiClient.Post(productionRunsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status1, raw1)
	runID := jsonField(parseJSON(raw1), "id")
	defer apiClient.Delete(productionRunPath(runID))

	status2, raw2, err := apiClient.Post(productionRunsPath, body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status2, raw2)
	assert.Equal(t, runID, jsonField(parseJSON(raw2), "id"), "replay returns the same run")
	assert.Len(t, runBatches(t, runID, "run"), 1, "replay does not plan the batch twice")
}

// ──────────────────────────────────────────────
// Update
// ──────────────────────────────────────────────

func TestProductionRuns_UpdateNonHappyPaths(t *testing.T) {
	t.Parallel()

	run := createRunWithBatches(t, plannedBatch("1"))
	runID := jsonField(run, "id")
	other := createRunWithBatches(t, plannedBatch("1"))

	t.Run("duplicate number conflicts", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"number": jsonField(other, "number")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
		errObj := requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
		assertErrorParam(t, errObj, "number")
	})

	t.Run("its own number is not a conflict", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"number": jsonField(run, "number")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
	})

	t.Run("blank number is rejected", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"number": ""}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("null number is rejected", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"number": nil}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("number over 255 characters is rejected", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"number": strings.Repeat("9", 256)}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("unknown responsible user is rejected", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"responsible_user_id": "acus_e2e_doesnotexist"}, newIdempotencyKey())
		require.NoError(t, err)
		assertValidationParam(t, status, body, "responsible_user_id")
	})

	t.Run("unknown run is not found", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath("pnrn_e2e_doesnotexist0"), map[string]any{"number": uniqueName("x")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	t.Run("unknown field is rejected", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{bogusE2EJSONField: "x"}, newIdempotencyKey())
		require.NoError(t, err)
		assertJSONUnknownFieldRejected(t, "PATCH", productionRunPath(runID), status, body)
	})

	t.Run("completed_at is not writable", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"completed_at": time.Now().UTC().Format(time.RFC3339)}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})
}

func TestProductionRuns_UpdateIdempotency(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")
	key := newIdempotencyKey()
	number := uniqueName("e2e-pr-idem")

	status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"number": number}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.Patch(productionRunPath(runID), map[string]any{"number": number}, key)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, number, jsonField(parseJSON(body), "number"))
}

// ──────────────────────────────────────────────
// Add batches
// ──────────────────────────────────────────────

func TestProductionRuns_AddBatches(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")

	status, body, err := apiClient.Post(productionRunBatchesPath(runID), map[string]any{
		"batches": []any{plannedBatch("2"), plannedBatch("3")},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	added := jsonArray(parseJSON(body), "data")
	require.Len(t, added, 2)
	for _, raw := range added {
		b := raw.(map[string]any)
		assertIDFormat(t, jsonField(b, "id"), id.BatchIDPrefix)
		assertNilField(t, b, "scanned_at")
	}

	got := parseJSON(mustGet(t, productionRunPath(runID)))
	assert.Equal(t, "3", jsonField(got, "batch_count"))
	assertBatchSummary(t, got, SeedGreigeItemID, "6", "3")
	assertNilField(t, got, "started_at")
	assert.Len(t, runBatches(t, runID, "run"), 3)
}

func TestProductionRuns_AddBatchesNonHappyPaths(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")

	for _, tc := range batchValidationCases() {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := apiClient.Post(productionRunBatchesPath(runID), map[string]any{"batches": []any{tc.batch}}, newIdempotencyKey())
			require.NoError(t, err)
			assertValidationParam(t, status, body, tc.param)
		})
	}

	t.Run("no batches is rejected", func(t *testing.T) {
		status, body, err := apiClient.Post(productionRunBatchesPath(runID), map[string]any{"batches": []any{}}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("missing required batch field is rejected", func(t *testing.T) {
		status, body, err := apiClient.Post(productionRunBatchesPath(runID), map[string]any{
			"batches": []any{map[string]any{"item_id": SeedGreigeItemID, "quantity_value": "1"}},
		}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("unknown run is not found", func(t *testing.T) {
		status, body, err := apiClient.Post(productionRunBatchesPath("pnrn_e2e_doesnotexist0"), map[string]any{"batches": []any{plannedBatch("1")}}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})

	// None of the rejected requests planned anything.
	assert.Len(t, runBatches(t, runID, "run"), 1)
}

func TestProductionRuns_AddBatchesIdempotency(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")
	key := newIdempotencyKey()
	body := map[string]any{"batches": []any{plannedBatch("2")}}

	status, raw1, err := apiClient.Post(productionRunBatchesPath(runID), body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, raw1)
	status, raw2, err := apiClient.Post(productionRunBatchesPath(runID), body, key)
	require.NoError(t, err)
	requireStatus(t, 201, status, raw2)

	first := jsonField(jsonArray(parseJSON(raw1), "data")[0].(map[string]any), "id")
	second := jsonField(jsonArray(parseJSON(raw2), "data")[0].(map[string]any), "id")
	assert.Equal(t, first, second, "replay returns the same batch")
	assert.Len(t, runBatches(t, runID, "run"), 2, "replay does not add the batch twice")
}

// ──────────────────────────────────────────────
// Scanned lifecycle: start, complete, and the guards on a completed run
// ──────────────────────────────────────────────

func TestProductionRuns_ScannedLifecycle(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")
	batchID := jsonField(runBatches(t, runID, "run")[0], "id")

	scanBatch(t, batchID)

	run := parseJSON(mustGet(t, productionRunPath(runID)))
	assertValidTimestamp(t, jsonField(run, "started_at"), "started_at")
	assertValidTimestamp(t, jsonField(run, "completed_at"), "completed_at")

	t.Run("completed run cannot be updated", func(t *testing.T) {
		status, body, err := apiClient.Patch(productionRunPath(runID), map[string]any{"number": uniqueName("e2e-pr-done")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("completed run takes no more batches", func(t *testing.T) {
		status, body, err := apiClient.Post(productionRunBatchesPath(runID), map[string]any{"batches": []any{plannedBatch("1")}}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("status filters place it with closed runs", func(t *testing.T) {
		number := jsonField(run, "number")
		assert.Contains(t, listRunIDs(t, url.Values{"q": {number}, "status": {"closed"}}), runID)
		assert.Contains(t, listRunIDs(t, url.Values{"q": {number}, "status": {"all"}}), runID)
		assert.NotContains(t, listRunIDs(t, url.Values{"q": {number}}), runID, "omitted status means open")
	})

	t.Run("run with a scanned batch cannot be deleted", func(t *testing.T) {
		status, body, err := apiClient.Delete(productionRunPath(runID))
		require.NoError(t, err)
		requireStatus(t, 409, status, body)
		requireErrorResponse(t, body, "resource_conflict", "invalid_request_error")
		mustGet(t, productionRunPath(runID))
	})

	// Deleting the batch reverses its scan, which frees the run to be deleted.
	deleteBatch(t, batchID)
	status, body, err := apiClient.Delete(productionRunPath(runID))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
}

// ──────────────────────────────────────────────
// Delete
// ──────────────────────────────────────────────

func TestProductionRuns_DeleteRemovesPlannedBatches(t *testing.T) {
	t.Parallel()

	run := createRunWithBatches(t, plannedBatch("1"), plannedBatch("2"))
	runID := jsonField(run, "id")
	batches := runBatches(t, runID, "run")
	require.Len(t, batches, 2)

	status, body, err := apiClient.Delete(productionRunPath(runID))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	for _, b := range batches {
		status, body, err := apiClient.GetListRaw("/v1/operations/batches/"+jsonField(b, "id")+"/flow", nil)
		require.NoError(t, err)
		assert.Equal(t, 404, status, "planned batch outlives its run: %s", string(body))
	}
}

func TestProductionRuns_DeleteUnknown(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.Delete(productionRunPath("pnrn_e2e_doesnotexist0"))
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// ──────────────────────────────────────────────
// List
// ──────────────────────────────────────────────

func TestProductionRuns_ListShapeAndDefaults(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")
	number := uniqueName("e2e-pr-list")
	renameRun(t, runID, number)

	status, body, err := apiClient.GetListRaw(productionRunsPath, url.Values{"q": {number}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	list := parseJSON(body)
	assert.Equal(t, "list", jsonField(list, "object"))
	data := jsonArray(list, "data")
	require.Len(t, data, 1)
	row := data[0].(map[string]any)
	assert.Equal(t, runID, jsonField(row, "id"))
	assertObjectField(t, row, "production_run")
	assert.Equal(t, number, jsonField(row, "number"))
	assert.Equal(t, "1", jsonField(row, "batch_count"))
	assertBatchSummary(t, row, SeedGreigeItemID, "1", "1")
	assertNilField(t, row, "responsible_user")
	assertNilField(t, row, "started_at")
	assertNilField(t, row, "completed_at")
	assertValidTimestamp(t, jsonField(row, "created_at"), "created_at")

	status, body, err = apiClient.GetListRaw(productionRunsPath, url.Values{"q": {number}, "include": {"responsible_user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	row = jsonArray(parseJSON(body), "data")[0].(map[string]any)
	assert.Equal(t, SeedAccountUserID, jsonField(jsonObject(row, "responsible_user"), "id"))
}

func TestProductionRuns_ListSearch(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")
	number := uniqueName("e2e-pr-search")
	renameRun(t, runID, number)
	batchID := jsonField(runBatches(t, runID, "run")[0], "id")

	t.Run("by number fragment", func(t *testing.T) {
		assert.Equal(t, []string{runID}, listRunIDs(t, url.Values{"q": {number[3:]}}))
	})

	// Regression: the first page used to drop the batch-id match.
	t.Run("by batch id prefix on the first page", func(t *testing.T) {
		assert.Equal(t, []string{runID}, listRunIDs(t, url.Values{"q": {batchID}}))
	})

	t.Run("LIKE wildcards are literal", func(t *testing.T) {
		assert.NotContains(t, listRunIDs(t, url.Values{"q": {"%"}}), runID)
		assert.NotContains(t, listRunIDs(t, url.Values{"q": {"_"}}), runID)
	})

	t.Run("no results", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(productionRunsPath, url.Values{"q": {uniqueName("e2e-pr-none")}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		assert.Empty(t, jsonArray(parseJSON(body), "data"))
	})
}

func TestProductionRuns_ListFilters(t *testing.T) {
	t.Parallel()

	onMachine := plannedBatch("1")
	onMachine["machine_ids"] = []any{SeedMachineID}
	runID := jsonField(createRunWithBatches(t, onMachine), "id")
	number := uniqueName("e2e-pr-filter")
	renameRun(t, runID, number)
	scoped := func(extra url.Values) []string {
		params := url.Values{"q": {number}}
		for k, vs := range extra {
			params[k] = vs
		}
		return listRunIDs(t, params)
	}
	today := time.Now().UTC().Format(time.DateOnly)
	tomorrow := time.Now().UTC().Add(24 * time.Hour).Format(time.DateOnly)

	assert.Contains(t, scoped(url.Values{"status": {"open"}}), runID)
	assert.Contains(t, scoped(url.Values{"status": {"all"}}), runID)
	assert.NotContains(t, scoped(url.Values{"status": {"closed"}}), runID)

	assert.Contains(t, scoped(url.Values{"item_ids": {SeedGreigeItemID}}), runID)
	assert.Contains(t, scoped(url.Values{"item_ids": {SeedItemID, SeedGreigeItemID}}), runID, "any listed item matches")
	assert.NotContains(t, scoped(url.Values{"item_ids": {SeedItemID}}), runID)

	assert.Contains(t, scoped(url.Values{"machine_ids": {SeedMachineID}}), runID, "a planned batch's machine matches")
	assert.NotContains(t, scoped(url.Values{"machine_ids": {"mc_e2e_doesnotexist00"}}), runID)

	assert.Contains(t, scoped(url.Values{"starts_at": {today}}), runID)
	assert.NotContains(t, scoped(url.Values{"starts_at": {tomorrow}}), runID)
	assert.Contains(t, scoped(url.Values{"ends_at": {tomorrow}}), runID)
	assert.NotContains(t, scoped(url.Values{"ends_at": {today}}), runID, "ends_at is exclusive of its own day")
	assert.Contains(t, scoped(url.Values{"starts_at": {today}, "ends_at": {tomorrow}}), runID)

	// Timestamps cut at the exact instant, which is how a viewer's local day is expressed.
	hourAgo := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	hourAhead := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	assert.Contains(t, scoped(url.Values{"starts_at": {hourAgo}, "ends_at": {hourAhead}}), runID)
	assert.NotContains(t, scoped(url.Values{"starts_at": {hourAhead}}), runID)
	assert.NotContains(t, scoped(url.Values{"ends_at": {hourAgo}}), runID)
}

func TestProductionRuns_ListRejectsInvalidParams(t *testing.T) {
	t.Parallel()

	for name, params := range map[string]url.Values{
		"unknown status":      {"status": {"archived"}},
		"malformed starts_at": {"starts_at": {"yesterday"}},
		"malformed ends_at":   {"ends_at": {"2026-13-45"}},
		"malformed cursor":    {"cursor": {"not-a-cursor"}},
		"unknown include":     {"include": {"batches"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			status, body, err := apiClient.GetListRaw(productionRunsPath, params)
			require.NoError(t, err)
			requireStatus(t, 400, status, body)
		})
	}

	t.Run("unknown query param", func(t *testing.T) {
		t.Parallel()
		status, body, err := apiClient.GetListRaw(productionRunsPath, url.Values{bogusE2EQueryParam: {"x"}})
		require.NoError(t, err)
		assertUnknownQueryParamRejected(t, productionRunsPath, status, body)
	})
}

func TestProductionRuns_ListPagination(t *testing.T) {
	t.Parallel()

	prefix := uniqueName("e2e-pr-page")
	var ids []string
	for i := 0; i < 3; i++ {
		runID := jsonField(createProductionRun(t, map[string]any{"responsible_user_id": SeedUserID}), "id")
		renameRun(t, runID, fmt.Sprintf("%s-%d", prefix, i))
		ids = append(ids, runID)
	}
	assertScopedCursorPagination(t, productionRunsPath, url.Values{"q": {prefix}}, ids)
}

// ──────────────────────────────────────────────
// List batches
// ──────────────────────────────────────────────

func TestProductionRuns_ListBatchesScopes(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1"), plannedBatch("2")), "id")

	direct := runBatches(t, runID, "run")
	require.Len(t, direct, 2)
	for _, b := range direct {
		assert.Equal(t, runID, jsonField(jsonObject(b, "production_run"), "id"))
	}

	// Planned batches feed nothing yet, so the flow is the run's own batches.
	flow := runBatches(t, runID, "")
	assert.Len(t, flow, 2)
	assert.Len(t, runBatches(t, runID, "flow"), 2)
}

func TestProductionRuns_ListBatchesPagination(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1"), plannedBatch("2"), plannedBatch("3")), "id")
	var ids []string
	for _, b := range runBatches(t, runID, "run") {
		ids = append(ids, jsonField(b, "id"))
	}
	assertScopedCursorPagination(t, productionRunBatchesPath(runID), url.Values{"scope": {"run"}}, ids)
}

func TestProductionRuns_ListBatchesNonHappyPaths(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")

	t.Run("unknown scope is rejected", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(productionRunBatchesPath(runID), url.Values{"scope": {"everything"}})
		require.NoError(t, err)
		requireStatus(t, 400, status, body)
	})

	t.Run("unknown run is not found", func(t *testing.T) {
		status, body, err := apiClient.GetListRaw(productionRunBatchesPath("pnrn_e2e_doesnotexist0"), url.Values{"scope": {"run"}})
		require.NoError(t, err)
		requireStatus(t, 404, status, body)
	})
}

// ──────────────────────────────────────────────
// Expandable fields
// ──────────────────────────────────────────────

func TestProductionRuns_ExpandableFieldsNullWithoutInclude(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(productionRunPath(SeedProductionRunID), nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	assert.Nil(t, got["responsible_user"], "responsible_user should be null without ?include=responsible_user")
}

func TestProductionRuns_IncludeResponsibleUser(t *testing.T) {
	t.Parallel()
	status, body, err := apiClient.GetListRaw(productionRunPath(SeedProductionRunID), url.Values{"include": {"responsible_user"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	got := parseJSON(body)
	// responsible_user may legitimately be null if unassigned
	_, ok := got["responsible_user"]
	assert.True(t, ok, "responsible_user key should be present with ?include=responsible_user")
	if u := jsonObject(got, "responsible_user"); u != nil {
		assert.Equal(t, "account_user", jsonField(u, "object"))
		assert.NotEmpty(t, jsonField(u, "id"))
	}
}

func TestProductionRuns_IncludeResponsibleUserUser(t *testing.T) {
	t.Parallel()
	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")
	got := parseJSON(mustGet(t, productionRunPath(runID)+"?include=responsible_user.user"))
	au := jsonObject(got, "responsible_user")
	require.NotNil(t, au)
	assert.Equal(t, SeedUserID, jsonField(jsonObject(au, "user"), "id"))
}

// ──────────────────────────────────────────────
// Tenant isolation and permissions
// ──────────────────────────────────────────────

func TestProductionRuns_TenantIsolation(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()
	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")

	checks := map[string]func() (int, []byte, error){
		"get": func() (int, []byte, error) { return clientB.GetListRaw(productionRunPath(runID), nil) },
		"patch": func() (int, []byte, error) {
			return clientB.Patch(productionRunPath(runID), map[string]any{"number": uniqueName("x")}, newIdempotencyKey())
		},
		"delete": func() (int, []byte, error) { return clientB.Delete(productionRunPath(runID)) },
		"add batches": func() (int, []byte, error) {
			return clientB.Post(productionRunBatchesPath(runID), map[string]any{"batches": []any{plannedBatch("1")}}, newIdempotencyKey())
		},
		"list batches": func() (int, []byte, error) {
			return clientB.GetListRaw(productionRunBatchesPath(runID), url.Values{"scope": {"run"}})
		},
	}
	for name, call := range checks {
		t.Run(name, func(t *testing.T) {
			status, body, err := call()
			require.NoError(t, err)
			assert.Equal(t, 404, status, "tenant B %s on tenant A's run: %s", name, string(body))
		})
	}

	// Tenant B's list does not include it.
	status, body, err := clientB.GetListRaw(productionRunsPath, url.Values{"status": {"all"}, "limit": {"100"}, "q": {runID}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	for _, raw := range jsonArray(parseJSON(body), "data") {
		assert.NotEqual(t, runID, jsonField(raw.(map[string]any), "id"))
	}
	mustGet(t, productionRunPath(runID))
}

func TestProductionRuns_RequiresPermission(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(apiKeysPath, map[string]any{
		"name":    uniqueName("e2e-pr-salesrep"),
		"role_id": SeedSalesRepRoleID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	created := parseJSON(body)
	keyID := jsonField(jsonObject(created, "api_key_info"), "id")
	defer apiClient.Delete(apiKeysPath + "/" + keyID)
	salesRep := apiClient.WithBearerToken(jsonField(created, "api_key_secret"), SeedAccountID)

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")

	checks := map[string]func() (int, []byte, error){
		"list": func() (int, []byte, error) { return salesRep.GetListRaw(productionRunsPath, nil) },
		"get":  func() (int, []byte, error) { return salesRep.GetListRaw(productionRunPath(runID), nil) },
		"create": func() (int, []byte, error) {
			return salesRep.Post(productionRunsPath, map[string]any{"responsible_user_id": SeedUserID}, newIdempotencyKey())
		},
		"patch": func() (int, []byte, error) {
			return salesRep.Patch(productionRunPath(runID), map[string]any{"number": uniqueName("x")}, newIdempotencyKey())
		},
		"delete": func() (int, []byte, error) { return salesRep.Delete(productionRunPath(runID)) },
		"add batches": func() (int, []byte, error) {
			return salesRep.Post(productionRunBatchesPath(runID), map[string]any{"batches": []any{plannedBatch("1")}}, newIdempotencyKey())
		},
		"list batches": func() (int, []byte, error) { return salesRep.GetListRaw(productionRunBatchesPath(runID), nil) },
	}
	for name, call := range checks {
		t.Run(name, func(t *testing.T) {
			status, body, err := call()
			require.NoError(t, err)
			requireStatus(t, 403, status, body)
			requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")
		})
	}
	mustGet(t, productionRunPath(runID))
}

// ──────────────────────────────────────────────
// Large runs and batch flows
// ──────────────────────────────────────────────

const (
	seedSewLargeStationID = "sgsn_01k0a8201zev8vyp148804tqa4"
)

func plannedBatches(n int) []any {
	out := make([]any, n)
	for i := range out {
		b := plannedBatch(fmt.Sprintf("%d", i+1))
		b["machine_ids"] = []any{SeedMachineID}
		out[i] = b
	}
	return out
}

// walkRunBatchIDs pages through a run's batches and fails on a repeated batch.
func walkRunBatchIDs(t *testing.T, runID, scope string, limit int) []string {
	t.Helper()
	params := url.Values{"scope": {scope}, "limit": {fmt.Sprintf("%d", limit)}}
	list, status, err := apiClient.GetList(productionRunBatchesPath(runID), params)
	require.NoError(t, err)
	require.Equal(t, 200, status)
	seen := map[string]bool{}
	var ids []string
	for {
		for _, item := range list.Data {
			id := DataItemField(item, "id")
			require.False(t, seen[id], "batch %s returned twice while paging", id)
			seen[id] = true
			ids = append(ids, id)
		}
		if !list.PageInfo.HasNextPage || list.PageInfo.NextPageURL == nil {
			return ids
		}
		list, _, err = apiClient.GetListFromPageURL(list.PageInfo.NextPageURL)
		require.NoError(t, err)
	}
}

// A run sized like a real week of knitting: created in one request, extended in another, and
// read back whole across pages.
func TestProductionRuns_LargeRun(t *testing.T) {
	t.Parallel()

	run := createProductionRun(t, map[string]any{"responsible_user_id": SeedUserID, "batches": plannedBatches(250)})
	runID := jsonField(run, "id")
	assert.Equal(t, "250", jsonField(run, "batch_count"))
	// 1 + 2 + ... + 250
	assertBatchSummary(t, run, SeedGreigeItemID, "31375", "250")

	status, body, err := apiClient.Post(productionRunBatchesPath(runID), map[string]any{"batches": plannedBatches(250)}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	// The whole response arrives: it is well over the size the idempotency layer once cut at.
	added := jsonArray(parseJSON(body), "data")
	require.Len(t, added, 250)
	for i, raw := range added {
		b := raw.(map[string]any)
		assert.Equal(t, fmt.Sprintf("%d", i+1), jsonField(jsonObject(b, "quantity"), "value"), "batches come back in request order")
		assert.Equal(t, runID, jsonField(jsonObject(b, "production_run"), "id"))
	}

	got := parseJSON(mustGet(t, productionRunPath(runID)))
	assert.Equal(t, "500", jsonField(got, "batch_count"))

	for _, scope := range []string{"run", "flow"} {
		ids := walkRunBatchIDs(t, runID, scope, 100)
		assert.Len(t, ids, 500, "scope=%s pages over every batch exactly once", scope)
	}

	// Machines are linked for every batch in the bulk insert.
	b := runBatches(t, runID, "run")[0]
	machines := jsonArray(jsonObject(b, "machines"), "data")
	require.Len(t, machines, 1)
	assert.Equal(t, SeedMachineID, jsonField(machines[0].(map[string]any), "id"))
}

func TestProductionRuns_BatchCountCap(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(productionRunsPath, map[string]any{"responsible_user_id": SeedUserID, "batches": plannedBatches(501)}, newIdempotencyKey())
	require.NoError(t, err)
	if status == 201 {
		apiClient.Delete(productionRunPath(jsonField(parseJSON(body), "id")))
	}
	requireStatus(t, 400, status, body)

	runID := jsonField(createRunWithBatches(t, plannedBatch("1")), "id")
	status, body, err = apiClient.Post(productionRunBatchesPath(runID), map[string]any{"batches": plannedBatches(501)}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// The flow scope follows a run's batches downstream into the batches made from them; the run
// scope does not.
func TestProductionRuns_FlowScopeFollowsDownstream(t *testing.T) {
	t.Parallel()

	runID := jsonField(createRunWithBatches(t, plannedBatch("10"), plannedBatch("10")), "id")
	planned := runBatches(t, runID, "run")
	require.Len(t, planned, 2)
	scanned := jsonField(planned[0], "id")
	untouched := jsonField(planned[1], "id")

	scanBatch(t, scanned)
	status, body, err := apiClient.Post("/v1/operations/batches/actions/move", map[string]any{
		"batch_ids":           []any{scanned},
		"production_step_id":  SeedSewLargeProductionStepID,
		"scanning_station_id": seedSewLargeStationID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	downstream := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, downstream)
	t.Cleanup(func() {
		apiClient.Delete("/v1/operations/batches/" + downstream)
		apiClient.Delete("/v1/operations/batches/" + scanned)
	})

	run := walkRunBatchIDs(t, runID, "run", 1)
	assert.ElementsMatch(t, []string{scanned, untouched}, run)

	flow := walkRunBatchIDs(t, runID, "flow", 1)
	assert.ElementsMatch(t, []string{scanned, untouched, downstream}, flow)

	// The downstream batch names its upstream, and is found by searching the flow.
	for _, b := range runBatches(t, runID, "flow") {
		if jsonField(b, "id") == downstream {
			inputs := jsonArray(jsonObject(b, "input_batches"), "data")
			require.Len(t, inputs, 1)
			assert.Equal(t, scanned, jsonField(inputs[0].(map[string]any), "id"))
		}
	}
	status, body, err = apiClient.GetListRaw(productionRunBatchesPath(runID), url.Values{"q": {downstream}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	data := jsonArray(parseJSON(body), "data")
	require.Len(t, data, 1)
	assert.Equal(t, downstream, jsonField(data[0].(map[string]any), "id"))
}
