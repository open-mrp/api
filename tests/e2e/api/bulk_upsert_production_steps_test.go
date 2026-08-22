//go:build e2e

package api_test

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/open-mrp/api/shared/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const productionStepsBulkUpsertPath = productionStepsPath + "/actions/bulk-upsert"
const jobsPath = "/v1/core/jobs"

func bulkUpsertProductionSteps(t *testing.T, steps ...map[string]any) (int, []byte) {
	t.Helper()
	rows := make([]any, len(steps))
	for i, s := range steps {
		rows[i] = s
	}
	status, body, err := apiClient.Post(productionStepsBulkUpsertPath, map[string]any{"production_steps": rows}, newIdempotencyKey())
	require.NoError(t, err)
	return status, body
}

// pollJobUntilTerminal GETs the job until it reaches a terminal status and returns it.
func pollJobUntilTerminal(t *testing.T, jobID string) map[string]any {
	t.Helper()
	var job map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		status, body, err := apiClient.GetListRaw(jobsPath+"/"+jobID, nil)
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("job %s GET returned status %d", jobID, status)
		}
		job = parseJSON(body)
		switch jsonField(job, "status") {
		case "completed", "failed", "cancelled":
			return nil
		}
		return fmt.Errorf("job %s not terminal yet (status %q)", jobID, jsonField(job, "status"))
	})
	return job
}

// acceptBulkUpsertSteps posts a bulk upsert, requires the 202 acknowledgment, follows
// the job to completion, and returns the created/updated step IDs from its results.
func acceptBulkUpsertSteps(t *testing.T, steps ...map[string]any) (createdIDs, updatedIDs []string) {
	t.Helper()
	status, body := bulkUpsertProductionSteps(t, steps...)
	requireStatus(t, 202, status, body)

	m := parseJSON(body)
	assert.Equal(t, "job", jsonField(m, "object"), "202 returns the canonical job resource")
	jobID := jsonField(m, "id")
	require.NotEmpty(t, jobID, "202 must name the job to poll")

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)

	require.NotEmpty(t, jobResults(job), "a completed job must carry results")
	return jobResultIDs(job)
}

// acceptBulkUpsertStepsJob posts a bulk upsert and returns the completed job. Bulk
// upsert is partial-success: a row whose validation can only be decided against existing
// rows (e.g. the create-only department rule) is recorded in the job's `errors` field,
// not failed — the job completes.
func acceptBulkUpsertStepsJob(t *testing.T, steps ...map[string]any) map[string]any {
	t.Helper()
	status, body := bulkUpsertProductionSteps(t, steps...)
	requireStatus(t, 202, status, body)
	jobID := jsonField(parseJSON(body), "id")
	require.NotEmpty(t, jobID)

	job := pollJobUntilTerminal(t, jobID)
	require.Equal(t, "completed", jsonField(job, "status"), "job should complete: %v", job)
	return job
}

func cleanupStepIDs(ids ...string) {
	for _, id := range ids {
		if id != "" {
			apiClient.Delete(productionStepsPath + "/" + id)
		}
	}
}

// Fuzzy-reference builders: units resolve by abbreviation, items by SKU, departments
// and scanning stations by name.
func refAbbr(a string) map[string]any { return map[string]any{"abbreviation": a} }
func refName(n string) map[string]any { return map[string]any{"name": n} }
func refSKU(s string) map[string]any  { return map[string]any{"sku": s} }

// bulkStepRow builds a minimal valid production step row producing a seeded item.
func bulkStepRow(name string) map[string]any {
	return map[string]any{
		"name":          name,
		"labor_rate":    map[string]any{"value": "25.00", "numerator_unit": refAbbr("$"), "denominator_unit": refAbbr("hr")},
		"labor_time":    map[string]any{"value": "1.5", "numerator_unit": refAbbr("hr"), "denominator_unit": refAbbr("ea")},
		"overhead_rate": map[string]any{"value": "15.00", "numerator_unit": refAbbr("$"), "denominator_unit": refAbbr("hr")},
		"production":    map[string]any{"item": refSKU(SeedItemSKU), "quantity_value": "100", "quantity_unit": refAbbr("ea")},
	}
}

// assertDecimalEqual compares decimal strings numerically — the API returns DB
// decimals at full scale (e.g. "1.100000000000000000000000000000").
func assertDecimalEqual(t *testing.T, expected, actual string, msgAndArgs ...any) {
	t.Helper()
	want, err := strconv.ParseFloat(expected, 64)
	require.NoError(t, err, "expected value %q is not a decimal", expected)
	got, err := strconv.ParseFloat(actual, 64)
	require.NoError(t, err, "actual value %q is not a decimal", actual)
	assert.InDelta(t, want, got, 1e-9, msgAndArgs...)
}

func getStep(t *testing.T, id string, includes ...string) map[string]any {
	t.Helper()
	var q url.Values
	if len(includes) > 0 {
		q = url.Values{"include": includes}
	}
	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+id, q)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func TestProductionSteps_BulkUpsert_AllCreates(t *testing.T) {
	t.Parallel()

	created, updated := acceptBulkUpsertSteps(t,
		bulkStepRow(uniqueName("e2e-bup-step-a")),
		bulkStepRow(uniqueName("e2e-bup-step-b")),
	)
	defer cleanupStepIDs(created...)

	require.Len(t, created, 2)
	for _, createdID := range created {
		assertIDFormat(t, createdID, id.ProductionStepIDPrefix)
	}
	assert.Empty(t, updated)
}

// TestProductionSteps_BulkUpsert_CreateWithAllFields exercises the full create branch:
// notes, leveling factor, allowances, case-insensitive station and department
// resolution, consumptions with waste and instructions, and rate values.
func TestProductionSteps_BulkUpsert_CreateWithAllFields(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-step-full")
	row := bulkStepRow(name)
	row["notes"] = "full step notes"
	row["leveling_factor"] = "1.1"
	row["allowances"] = "0.05"
	row["scanning_station"] = refName("knitting station") // by name, case-insensitive
	row["department"] = refName("knitting")               // by name, case-insensitive
	row["consumptions"] = []any{
		map[string]any{
			"item":                 refSKU("LKN"),
			"quantity_value":       "2",
			"quantity_unit":        refAbbr("ea"),
			"waste_quantity_value": "0.1",
			"instructions":         "handle with care",
		},
	}
	created, _ := acceptBulkUpsertSteps(t, row)
	defer cleanupStepIDs(created...)

	require.Len(t, created, 1)
	stepID := created[0]

	got := getStep(t, stepID, "production.produced_item", "consumptions", "scanning_station", "department")
	assert.Equal(t, name, jsonField(got, "name"))
	assert.Equal(t, "full step notes", jsonField(got, "notes"))
	assertDecimalEqual(t, "1.1", jsonField(got, "leveling_factor"))
	assertDecimalEqual(t, "0.05", jsonField(got, "allowances"))
	assertDecimalEqual(t, "25.00", jsonField(jsonObject(got, "labor_rate"), "value"))
	assertDecimalEqual(t, "1.5", jsonField(jsonObject(got, "labor_time"), "value"))
	assertDecimalEqual(t, "15.00", jsonField(jsonObject(got, "overhead_rate"), "value"))

	station := jsonObject(got, "scanning_station")
	require.NotNil(t, station, "scanning_station should be populated with include")
	assert.Equal(t, "Knitting Station", jsonField(station, "name"))
	dept := jsonObject(got, "department")
	require.NotNil(t, dept, "department should be populated with include")
	assert.Equal(t, SeedDepartmentID, jsonField(dept, "id"))

	production := jsonObject(got, "production")
	require.NotNil(t, production, "production should be populated with include")
	producedItem := jsonObject(production, "produced_item")
	require.NotNil(t, producedItem)
	assert.Equal(t, SeedItemID, jsonField(producedItem, "id"))

	consumptions := jsonListData(got, "consumptions")
	require.Len(t, consumptions, 1)
	consumption, ok := consumptions[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "handle with care", jsonField(consumption, "instructions"))
}

// --- Synchronous rejections: validation and reference resolution happen in the accept
// phase, so a bad request is a 400 before any job is raised. ---

func TestProductionSteps_BulkUpsert_RejectsDuplicateNameInRequest(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-step-dup")
	status, body := bulkUpsertProductionSteps(t,
		bulkStepRow(name),
		bulkStepRow(strings.ToUpper(name)), // duplicate differing only by casing
	)
	requireStatus(t, 400, status, body)
	assert.Equal(t, "invalid_request_error", jsonField(jsonObject(parseJSON(body), "error"), "type"))
}

func TestProductionSteps_BulkUpsert_EmptyRejected(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Post(productionStepsBulkUpsertPath, map[string]any{"production_steps": []any{}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 400, status, body)
}

// TestProductionSteps_BulkUpsert_RejectsUnknownSKU: a row referencing an item SKU that
// does not resolve fails synchronously with a clear validation error naming the row.
func TestProductionSteps_BulkUpsert_RejectsUnknownSKU(t *testing.T) {
	t.Parallel()

	badRow := bulkStepRow(uniqueName("e2e-bup-step-badsku"))
	badRow["production"] = map[string]any{"item": refSKU(uniqueName("no-such-sku")), "quantity_value": "1", "quantity_unit": refAbbr("ea")}
	status, body := bulkUpsertProductionSteps(t,
		bulkStepRow(uniqueName("e2e-bup-step-ok")),
		badRow,
	)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_steps[1].production.item")
}

// TestProductionSteps_BulkUpsert_RejectsDuplicateConsumptionSKU: each consumed item is
// one line within a step — the same SKU twice (any casing) is rejected.
func TestProductionSteps_BulkUpsert_RejectsDuplicateConsumptionSKU(t *testing.T) {
	t.Parallel()

	row := bulkStepRow(uniqueName("e2e-bup-step-dupcons"))
	row["consumptions"] = []any{
		map[string]any{"item": refSKU("LKN"), "quantity_value": "1", "quantity_unit": refAbbr("ea")},
		map[string]any{"item": refSKU("lkn"), "quantity_value": "2", "quantity_unit": refAbbr("ea")}, // duplicate differing only by casing
	}
	status, body := bulkUpsertProductionSteps(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_steps[0].consumptions[1].item")
}

// TestProductionSteps_BulkUpsert_RejectsUnknownConsumptionSKU: consumption SKUs are a
// separate resolution branch from the production SKU and get their own row-indexed
// param.
func TestProductionSteps_BulkUpsert_RejectsUnknownConsumptionSKU(t *testing.T) {
	t.Parallel()

	row := bulkStepRow(uniqueName("e2e-bup-step-badcsku"))
	row["consumptions"] = []any{
		map[string]any{"item": refSKU(uniqueName("no-such-sku")), "quantity_value": "1", "quantity_unit": refAbbr("ea")},
	}
	status, body := bulkUpsertProductionSteps(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_steps[0].consumptions[0].item")
}

func TestProductionSteps_BulkUpsert_RejectsUnknownDepartment(t *testing.T) {
	t.Parallel()

	row := bulkStepRow(uniqueName("e2e-bup-step-baddept"))
	row["department"] = refName(uniqueName("no-such-department"))
	status, body := bulkUpsertProductionSteps(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_steps[0].department")
}

// TestProductionSteps_BulkUpsert_RejectsUnknownWasteUnit: the waste quantity unit is
// resolved independently of the consumption quantity unit.
func TestProductionSteps_BulkUpsert_RejectsUnknownWasteUnit(t *testing.T) {
	t.Parallel()

	row := bulkStepRow(uniqueName("e2e-bup-step-badwaste"))
	row["consumptions"] = []any{
		map[string]any{
			"item":                 refSKU("LKN"),
			"quantity_value":       "1",
			"quantity_unit":        refAbbr("ea"),
			"waste_quantity_value": "0.1",
			"waste_quantity_unit":  refAbbr("parsec"),
		},
	}
	status, body := bulkUpsertProductionSteps(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_steps[0].consumptions[0].waste_quantity_unit")
}

func TestProductionSteps_BulkUpsert_RejectsUnknownUnit(t *testing.T) {
	t.Parallel()

	row := bulkStepRow(uniqueName("e2e-bup-step-badunit"))
	row["labor_time"] = map[string]any{"value": "1.5", "numerator_unit": refAbbr("lightyear"), "denominator_unit": refAbbr("ea")}
	status, body := bulkUpsertProductionSteps(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_steps[0].labor_time.numerator_unit")
}

func TestProductionSteps_BulkUpsert_RejectsUnknownStation(t *testing.T) {
	t.Parallel()

	row := bulkStepRow(uniqueName("e2e-bup-step-badstation"))
	row["scanning_station"] = refName(uniqueName("no-such-station"))
	status, body := bulkUpsertProductionSteps(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_steps[0].scanning_station")
}

// TestProductionSteps_BulkUpsert_RejectsNonCurrencyRate: labor_rate and overhead_rate
// must have a currency numerator and a non-currency denominator.
func TestProductionSteps_BulkUpsert_RejectsNonCurrencyRate(t *testing.T) {
	t.Parallel()

	row := bulkStepRow(uniqueName("e2e-bup-step-badrate"))
	row["labor_rate"] = map[string]any{"value": "25.00", "numerator_unit": refAbbr("ea"), "denominator_unit": refAbbr("hr")}
	status, body := bulkUpsertProductionSteps(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_steps[0].labor_rate.numerator_unit")
}

// TestProductionSteps_BulkUpsert_RejectsCurrencyRateDenominator: the denominator of a
// cost-typed rate must not be a currency unit — the inverse of the numerator rule.
func TestProductionSteps_BulkUpsert_RejectsCurrencyRateDenominator(t *testing.T) {
	t.Parallel()

	row := bulkStepRow(uniqueName("e2e-bup-step-badden"))
	row["overhead_rate"] = map[string]any{"value": "15.00", "numerator_unit": refAbbr("$"), "denominator_unit": refAbbr("$")}
	status, body := bulkUpsertProductionSteps(t, row)
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "production_steps[0].overhead_rate.denominator_unit")
}

// --- Asynchronous rejections: the create-only department rule is decided against
// existing rows, which is read in the execute phase — so it is accepted synchronously
// (202) and the job fails. ---

// TestProductionSteps_BulkUpsert_RejectsAddingDepartmentToStepWithoutOne: the
// department is create-only even when the existing step has none — stating one on
// update is a contradiction, not a backfill.
func TestProductionSteps_BulkUpsert_RejectsAddingDepartmentToStepWithoutOne(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-step-deptadd")
	created, _ := acceptBulkUpsertSteps(t, bulkStepRow(name)) // no department
	defer cleanupStepIDs(created...)

	row := bulkStepRow(name)
	row["department"] = refName(seedDepartmentName)
	job := acceptBulkUpsertStepsJob(t, row)
	rowErrs := jobErrors(job)
	require.Len(t, rowErrs, 1, "the rejected row is recorded in errors")
	msg := jobRowErrorMessage(rowErrs[0])
	assert.Contains(t, msg, "department")
}

// TestProductionSteps_BulkUpsert_RejectsDepartmentChange: the department is create-only
// — a row matching an existing step but stating a different department is recorded as a
// per-row failure rather than silently changing it (the job still completes).
func TestProductionSteps_BulkUpsert_RejectsDepartmentChange(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-step-deptchg")
	seedRow := bulkStepRow(name)
	seedRow["department"] = refName(seedDepartmentName) // Knitting
	created, _ := acceptBulkUpsertSteps(t, seedRow)
	defer cleanupStepIDs(created...)

	row := bulkStepRow(name)
	row["department"] = refName("Washing")
	job := acceptBulkUpsertStepsJob(t, row)
	rowErrs := jobErrors(job)
	require.Len(t, rowErrs, 1)
	msg := jobRowErrorMessage(rowErrs[0])
	assert.Contains(t, msg, "department")
}

// TestProductionSteps_BulkUpsert_UpdatesEveryField creates a step, then upserts the
// same name (different casing) changing notes, factors, rates, productions, and
// consumptions — asserting the match is case-insensitive, the ID is stable, the name
// adopts the new casing, rates are freshly written, productions are replaced
// wholesale, and omitted fields are preserved.
func TestProductionSteps_BulkUpsert_UpdatesEveryField(t *testing.T) {
	t.Parallel()

	name := uniqueName("e2e-bup-step-upd")
	createRow := bulkStepRow(name)
	createRow["notes"] = "original notes"
	createRow["leveling_factor"] = "1"
	created, _ := acceptBulkUpsertSteps(t, createRow)
	defer cleanupStepIDs(created...)
	require.Len(t, created, 1)
	stepID := created[0]

	upper := strings.ToUpper(name)
	updRow := bulkStepRow(upper)
	updRow["notes"] = "updated notes"
	updRow["leveling_factor"] = "2"
	updRow["labor_rate"] = map[string]any{"value": "30.00", "numerator_unit": refAbbr("$"), "denominator_unit": refAbbr("hr")}
	updRow["production"] = map[string]any{"item": refSKU("SCK-002"), "quantity_value": "50", "quantity_unit": refAbbr("ea")}
	updRow["consumptions"] = []any{
		map[string]any{"item": refSKU("LKN"), "quantity_value": "3", "quantity_unit": refAbbr("ea")},
	}
	updCreated, updUpdated := acceptBulkUpsertSteps(t, updRow)
	assert.Empty(t, updCreated, "existing name must update, not create")
	require.Len(t, updUpdated, 1)
	assert.Equal(t, stepID, updUpdated[0])

	got := getStep(t, stepID, "production.produced_item", "consumptions")
	assert.Equal(t, upper, jsonField(got, "name"), "name adopts the request's casing")
	assert.Equal(t, "updated notes", jsonField(got, "notes"))
	assertDecimalEqual(t, "2", jsonField(got, "leveling_factor"))
	assertDecimalEqual(t, "30.00", jsonField(jsonObject(got, "labor_rate"), "value"), "rate is freshly written on update")
	production := jsonObject(got, "production")
	require.NotNil(t, production)
	assert.Equal(t, "SCK-002", jsonField(jsonObject(production, "produced_item"), "sku"), "production is replaced on update")
	require.Len(t, jsonListData(got, "consumptions"), 1)

	// Omitted notes and leveling factor are preserved on a subsequent update.
	reCreated, _ := acceptBulkUpsertSteps(t, bulkStepRow(upper))
	assert.Empty(t, reCreated)
	got2 := getStep(t, stepID)
	assert.Equal(t, "updated notes", jsonField(got2, "notes"))
	assertDecimalEqual(t, "2", jsonField(got2, "leveling_factor"))
}

func TestProductionSteps_BulkUpsert_MixedCreateAndUpdate(t *testing.T) {
	t.Parallel()

	existingName := uniqueName("e2e-bup-step-mix-exist")
	newName := uniqueName("e2e-bup-step-mix-new")

	seedCreated, _ := acceptBulkUpsertSteps(t, bulkStepRow(existingName))
	defer cleanupStepIDs(seedCreated...)

	updRow := bulkStepRow(existingName)
	updRow["notes"] = "touched"
	created, updated := acceptBulkUpsertSteps(t, updRow, bulkStepRow(newName))
	defer cleanupStepIDs(created...)

	assert.Len(t, created, 1)
	assert.Len(t, updated, 1)
}

// TestProductionSteps_BulkUpsert_AutoDerivesFlowEdges: flow DAG edges are not part of
// the input — a step consuming an item is linked under the step producing it. Two
// fresh part items are created so the chain is fully isolated from seed steps.
func TestProductionSteps_BulkUpsert_AutoDerivesFlowEdges(t *testing.T) {
	t.Parallel()

	skuA := uniqueName("e2e-bup-step-dag-p1")
	skuB := uniqueName("e2e-bup-step-dag-p2")
	partIDs, _ := bulkUpsertPartIDs(t,
		map[string]any{"sku": skuA, "category": map[string]any{"id": SeedItemCategoryID}},
		map[string]any{"sku": skuB, "category": map[string]any{"id": SeedItemCategoryID}},
	)
	defer cleanupPartIDs(partIDs)

	nameA := uniqueName("e2e-bup-step-dag-a")
	nameB := uniqueName("e2e-bup-step-dag-b")
	rowA := bulkStepRow(nameA)
	rowA["production"] = map[string]any{"item": refSKU(skuA), "quantity_value": "10", "quantity_unit": refAbbr("ea")}
	rowB := bulkStepRow(nameB)
	rowB["production"] = map[string]any{"item": refSKU(skuB), "quantity_value": "10", "quantity_unit": refAbbr("ea")}
	rowB["consumptions"] = []any{
		map[string]any{"item": refSKU(skuA), "quantity_value": "1", "quantity_unit": refAbbr("ea")},
	}

	created, _ := acceptBulkUpsertSteps(t, rowA, rowB)
	defer cleanupStepIDs(created...)
	require.Len(t, created, 2)
	stepAID := created[0]
	stepBID := created[1]

	// Flow edges are derived after the job commits (a post-commit side effect), so a job
	// that reads "completed" may not have linked them yet. They are also relinked one step
	// at a time, each relink clearing then rebuilding that step's edges, so a concurrent
	// read can briefly see an edge missing mid-relink — poll both directions until the
	// A→B edge is stably present.
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		inSteps := jsonListData(getStep(t, stepBID, "in_steps"), "in_steps")
		if len(inSteps) != 1 {
			return fmt.Errorf("step B not yet auto-linked under step A (in_steps: %d)", len(inSteps))
		}
		inStep, ok := inSteps[0].(map[string]any)
		if !ok || jsonField(inStep, "id") != stepAID {
			return fmt.Errorf("step B's in_step is not step A")
		}

		outSteps := jsonListData(getStep(t, stepAID, "out_steps"), "out_steps")
		if len(outSteps) != 1 {
			return fmt.Errorf("step A does not yet list step B as an output step (out_steps: %d)", len(outSteps))
		}
		outStep, ok := outSteps[0].(map[string]any)
		if !ok || jsonField(outStep, "id") != stepBID {
			return fmt.Errorf("step A's out_step is not step B")
		}
		return nil
	})
}
