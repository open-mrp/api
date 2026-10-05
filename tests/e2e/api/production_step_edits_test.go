//go:build e2e

package api_test

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const costBasisChangedKey = "core.event.item_cost_basis_changed"

// costBasisReasons lists the reasons on the cost-basis events the outbox holds for itemID, oldest first.
func costBasisReasons(t *testing.T, itemID string) []string {
	t.Helper()
	rows, err := authDB(t).Query(`
		SELECT CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR)
		FROM message_outbox
		WHERE routing_key = ?
		AND CAST(FROM_BASE64(JSON_UNQUOTE(JSON_EXTRACT(payload, '$.data'))) AS CHAR) LIKE ?
		ORDER BY id`,
		costBasisChangedKey, "%"+itemID+"%")
	require.NoError(t, err)
	defer rows.Close()

	var reasons []string
	for rows.Next() {
		var data string
		require.NoError(t, rows.Scan(&data))
		evt := parseJSON([]byte(data))
		require.Equal(t, itemID, jsonField(evt, "item_id"))
		require.Equal(t, SeedAccountID, jsonField(evt, "account_id"))
		reasons = append(reasons, jsonField(evt, "reason"))
	}
	require.NoError(t, rows.Err())
	return reasons
}

// requireNewCostBasisReason asserts the outbox gained exactly one cost-basis event for itemID since
// `before` events, carrying reason.
func requireNewCostBasisReason(t *testing.T, itemID string, before int, reason string) int {
	t.Helper()
	reasons := costBasisReasons(t, itemID)
	require.Len(t, reasons, before+1, "one cost-basis event per edit; got %v", reasons)
	assert.Equal(t, reason, reasons[before])
	return len(reasons)
}

type editableStep struct {
	fixture      laborCostFixture
	id           string
	productionID string
	laborRateID  string
	consumption  string
}

// createEditableStep makes a step producing a fresh part from a fresh material, so the cost-basis events
// it causes name items no other test touches.
func createEditableStep(t *testing.T) editableStep {
	t.Helper()
	f := createLaborCostFixture(t)

	status, body, err := apiClient.Post(productionStepsPath+"?include=production&include=consumptions", map[string]any{
		"name":            uniqueName("e2e-step-edits"),
		"leveling_factor": "0",
		"allowances":      "0",
		"labor_time":      map[string]any{"value": "30", "numerator_unit_id": unitSecond, "denominator_unit_id": unitEach},
		"labor_rate":      map[string]any{"value": "25", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
		"overhead_rate":   map[string]any{"value": "20", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
		"production":      map[string]any{"item_id": f.partItemID, "quantity_value": "1", "quantity_unit_id": unitEach},
		"consumptions": []map[string]any{
			{"item_id": f.materialItemID, "quantity_value": "1", "quantity_unit_id": unitEach,
				"waste_quantity_value": "0", "waste_quantity_unit_id": unitEach},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	step := parseJSON(body)
	id := jsonField(step, "id")
	t.Cleanup(func() { cleanupStepIDs(id) })

	production := jsonObject(step, "production")
	require.NotNil(t, production, "created step carries its production: %s", body)
	consumptions := jsonArray(jsonObject(step, "consumptions"), "data")
	require.Len(t, consumptions, 1, "created step carries its consumption: %s", body)

	return editableStep{
		fixture:      f,
		id:           id,
		productionID: jsonField(production, "id"),
		laborRateID:  jsonField(jsonObject(step, "labor_rate"), "id"),
		consumption:  jsonField(consumptions[0].(map[string]any), "id"),
	}
}

// Everything a step is made of feeds what its item costs, so every edit has to tell costing to
// recompute. The dashboard published this event on each edit; leaving any of them out leaves item
// costs stale until something else happens to touch the item.
func TestProductionSteps_EditsRestateWhatTheItemCosts(t *testing.T) {
	t.Parallel()
	s := createEditableStep(t)
	part := s.fixture.partItemID

	n := requireNewCostBasisReason(t, part, 0, "production_step_created")

	status, body, err := apiClient.Patch(productionStepsPath+"/"+s.id, map[string]any{"leveling_factor": "0.1"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	n = requireNewCostBasisReason(t, part, n, "production_step_updated")

	status, body, err = apiClient.Patch("/v1/operations/rates/"+s.laborRateID, map[string]any{
		"value": "30", "object_id": s.id, "object_type": "production_step",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	n = requireNewCostBasisReason(t, part, n, "rate_updated")

	consumptionPath := fmt.Sprintf("%s/%s/consumptions/%s", productionStepsPath, s.id, s.consumption)
	status, body, err = apiClient.Patch(consumptionPath, map[string]any{"quantity_value": "2"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	n = requireNewCostBasisReason(t, part, n, "consumption_updated")

	status, body, err = apiClient.Post(fmt.Sprintf("%s/%s/consumptions", productionStepsPath, s.id), map[string]any{
		"item_id": createLaborCostFixture(t).materialItemID, "quantity_value": "1", "quantity_unit_id": unitEach,
		"waste_quantity_value": "0", "waste_quantity_unit_id": unitEach,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	added := jsonField(parseJSON(body), "id")
	n = requireNewCostBasisReason(t, part, n, "consumption_created")

	status, body, err = apiClient.Delete(fmt.Sprintf("%s/%s/consumptions/%s", productionStepsPath, s.id, added))
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	n = requireNewCostBasisReason(t, part, n, "consumption_deleted")

	status, body, err = apiClient.Patch(fmt.Sprintf("%s/%s/productions/%s", productionStepsPath, s.id, s.productionID),
		map[string]any{"quantity_value": "2"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	n = requireNewCostBasisReason(t, part, n, "production_updated")

	// A material's unit cost restates the material, and costing walks downstream from there.
	material := s.fixture.materialItemID
	before := len(costBasisReasons(t, material))
	status, body, err = apiClient.Patch(materialsPath+"/"+s.fixture.materialID, map[string]any{
		"unit_cost": map[string]any{"value": "0.25", "numerator_unit_id": unitDollar, "denominator_unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	requireNewCostBasisReason(t, material, before, "unit_cost_updated")

	status, body, err = apiClient.Delete(productionStepsPath + "/" + s.id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	requireNewCostBasisReason(t, part, n, "production_step_deleted")
}

// The consumer recosts the part once the edits land: labor follows the step's rate and time.
func TestProductionSteps_RateEditRecostsTheItem(t *testing.T) {
	t.Parallel()
	s := createEditableStep(t)
	part := s.fixture.partItemID

	// 30 s an each at $25 an hour is $0.208333… an each, $0.416666… a pair.
	waitForDecimal(t, part, "direct_labor_cost", "0.416666666666666667")

	status, body, err := apiClient.Patch("/v1/operations/rates/"+s.laborRateID, map[string]any{
		"value": "50", "object_id": s.id, "object_type": "production_step",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	waitForDecimal(t, part, "direct_labor_cost", "0.833333333333333333")
}

func decimalClose(want, got string) bool {
	w, err1 := strconv.ParseFloat(want, 64)
	g, err2 := strconv.ParseFloat(got, 64)
	return err1 == nil && err2 == nil && math.Abs(w-g) < 1e-9
}

func waitForDecimal(t *testing.T, itemID, field, want string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		got := jsonField(getItemCosts(t, itemID), field)
		if decimalClose(want, got) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s of %s is %s, want %s", field, itemID, got, want)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// The step page costs a step client-side, so the units every rate and quantity is counted in, and the
// unit group and unit cost of every item, have to come back as objects.
func TestProductionSteps_IncludeUnitsForCosting(t *testing.T) {
	t.Parallel()
	s := createEditableStep(t)

	includes := url.Values{"include": {
		"labor_rate.numerator_unit", "labor_rate.denominator_unit",
		"labor_time.numerator_unit", "labor_time.denominator_unit",
		"overhead_rate.numerator_unit", "overhead_rate.denominator_unit",
		"production.quantity.unit",
		"production.produced_item.category.unit_group.base_unit",
		"production.produced_item.category.unit_group.associated_units.unit",
		"consumptions.quantity.unit", "consumptions.waste_quantity.unit",
		"consumptions.consumed_item.unit_cost",
		"consumptions.consumed_item.category.unit_group.base_unit",
		"machines.department",
	}}
	listed := url.Values{"item_ids": {s.fixture.partItemID}, "include": includes["include"]}
	for path, query := range map[string]url.Values{productionStepsPath + "/" + s.id: includes, productionStepsPath: listed} {
		status, body, err := apiClient.GetListRaw(path, query)
		require.NoError(t, err)
		requireStatus(t, 200, status, body)

		step := parseJSON(body)
		if data, ok := step["data"]; ok {
			list := data.([]any)
			require.Len(t, list, 1, "the part is made by one step: %s", body)
			step = list[0].(map[string]any)
		}

		for _, rate := range []string{"labor_rate", "labor_time", "overhead_rate"} {
			r := jsonObject(step, rate)
			require.NotNil(t, r, "%s on %s", rate, path)
			assert.Equal(t, "unit", jsonField(jsonObject(r, "numerator_unit"), "object"), "%s.numerator_unit on %s", rate, path)
			assert.Equal(t, "unit", jsonField(jsonObject(r, "denominator_unit"), "object"), "%s.denominator_unit on %s", rate, path)
		}
		assert.Equal(t, "hour", jsonField(jsonObject(jsonObject(step, "labor_rate"), "denominator_unit"), "id"))

		production := jsonObject(step, "production")
		assert.Equal(t, unitEach, jsonField(jsonObject(jsonObject(production, "quantity"), "unit"), "id"), "production.quantity.unit on %s", path)
		group := jsonObject(jsonObject(jsonObject(production, "produced_item"), "category"), "unit_group")
		require.NotNil(t, group, "produced item unit group on %s: %s", path, body)
		assert.Equal(t, "unit", jsonField(jsonObject(group, "base_unit"), "object"))
		assert.NotEmpty(t, jsonArray(jsonObject(group, "associated_units"), "data"))

		consumption := jsonArray(jsonObject(step, "consumptions"), "data")[0].(map[string]any)
		assert.Equal(t, unitEach, jsonField(jsonObject(jsonObject(consumption, "quantity"), "unit"), "id"))
		assert.Equal(t, unitEach, jsonField(jsonObject(jsonObject(consumption, "waste_quantity"), "unit"), "id"))
		unitCost := jsonObject(jsonObject(consumption, "consumed_item"), "unit_cost")
		require.NotNil(t, unitCost, "consumed item unit cost on %s: %s", path, body)
		assertDecimalEqual(t, "0.10", jsonField(unitCost, "value"), "consumed item unit cost")
	}
}

func createMachineForStep(t *testing.T) string {
	t.Helper()
	status, body, err := apiClient.Post(machinesPath, map[string]any{
		"name":          uniqueName("e2e-step-machine"),
		"serial_number": uniqueName("SN"),
		"department_id": SeedDepartmentID,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { apiClient.Delete(machinesPath + "/" + id) })
	return id
}

func stepMachineIDs(t *testing.T, stepID string) []string {
	t.Helper()
	status, body, err := apiClient.GetListRaw(productionStepsPath+"/"+stepID, url.Values{"include": {"machines.department"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	var ids []string
	for _, m := range jsonArray(jsonObject(parseJSON(body), "machines"), "data") {
		machine := m.(map[string]any)
		assert.Equal(t, "department", jsonField(jsonObject(machine, "department"), "object"), "machines.department expands")
		assert.NotEmpty(t, jsonField(machine, "serial_number"), "a step's machine carries its serial number")
		ids = append(ids, jsonField(machine, "id"))
	}
	return ids
}

// The step page edits notes and machines. The dashboard silently dropped both; they must persist.
func TestProductionSteps_UpdateNotesAndMachines(t *testing.T) {
	t.Parallel()
	s := createEditableStep(t)
	other := createEditableStep(t)
	m1, m2 := createMachineForStep(t), createMachineForStep(t)

	patch := func(stepID string, body map[string]any) map[string]any {
		t.Helper()
		status, resp, err := apiClient.Patch(productionStepsPath+"/"+stepID, body, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, status, resp)
		return parseJSON(resp)
	}

	got := patch(s.id, map[string]any{"notes": "Check tension first."})
	assert.Equal(t, "Check tension first.", jsonField(got, "notes"))
	got = patch(s.id, map[string]any{"name": jsonField(got, "name")})
	assert.Equal(t, "Check tension first.", jsonField(got, "notes"), "omitting notes keeps them")
	got = patch(s.id, map[string]any{"notes": nil})
	assert.Nil(t, got["notes"], "null clears the notes")

	patch(s.id, map[string]any{"machine_ids": []string{m1, m2}})
	assert.ElementsMatch(t, []string{m1, m2}, stepMachineIDs(t, s.id))

	patch(s.id, map[string]any{"leveling_factor": "0.2"})
	assert.ElementsMatch(t, []string{m1, m2}, stepMachineIDs(t, s.id), "omitting machine_ids keeps them")

	// A machine runs one step: assigning m2 elsewhere takes it off this step.
	patch(other.id, map[string]any{"machine_ids": []string{m2}})
	assert.ElementsMatch(t, []string{m1}, stepMachineIDs(t, s.id))
	assert.ElementsMatch(t, []string{m2}, stepMachineIDs(t, other.id))

	patch(s.id, map[string]any{"machine_ids": []string{}})
	assert.Empty(t, stepMachineIDs(t, s.id), "an empty list unassigns every machine")
	assert.ElementsMatch(t, []string{m2}, stepMachineIDs(t, other.id), "other steps keep theirs")

	status, body, err := apiClient.Patch(productionStepsPath+"/"+s.id, map[string]any{"machine_ids": []string{"mchn_doesnotexist000"}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 404, status, body)
}

// Batches outlive the step they were scanned at: deleting the step unlinks them, as the dashboard did,
// rather than leaving them pointing at a step that is gone.
func TestProductionSteps_DeleteUnlinksBatches(t *testing.T) {
	t.Parallel()
	s := createEditableStep(t)

	db := authDB(t)
	quantityID, batchID := uniqueName("qu_e2estepdel"), uniqueName("bt_e2estepdel")
	_, err := db.Exec(`INSERT INTO quantity (id, value, unit_id) VALUES (?, 1, ?)`, quantityID, unitEach)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO batch (id, account_id, item_id, quantity_id, production_step_id) VALUES (?, ?, ?, ?, ?)`,
		batchID, SeedAccountID, s.fixture.partItemID, quantityID, s.id)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Exec(`DELETE FROM batch WHERE id = ?`, batchID)
		db.Exec(`DELETE FROM quantity WHERE id = ?`, quantityID)
	})

	status, body, err := apiClient.Delete(productionStepsPath + "/" + s.id)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	var stepID *string
	require.NoError(t, db.QueryRow(`SELECT production_step_id FROM batch WHERE id = ?`, batchID).Scan(&stepID))
	assert.Nil(t, stepID, "the batch no longer points at the deleted step")
}

// The list search matches every word, in any order, as the dashboard's search did. Words under the
// fulltext index's three-character minimum match nothing there too.
func TestProductionSteps_SearchMatchesEveryWord(t *testing.T) {
	t.Parallel()
	s := createEditableStep(t)
	token := uniqueName("zqx")
	status, body, err := apiClient.Patch(productionStepsPath+"/"+s.id, map[string]any{"name": "Sew " + token + " hem"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	search := func(q string) []string {
		t.Helper()
		status, body, err := apiClient.GetListRaw(productionStepsPath, url.Values{"q": {q}})
		require.NoError(t, err)
		requireStatus(t, 200, status, body)
		var ids []string
		for _, d := range jsonArray(parseJSON(body), "data") {
			ids = append(ids, jsonField(d.(map[string]any), "id"))
		}
		return ids
	}

	assert.Contains(t, search("hem "+token), s.id, "words in another order")
	assert.Contains(t, search(token[:len(token)-2]), s.id, "a word prefix")
	assert.NotContains(t, search(token+" zipper"), s.id, "every word must match")
}

func TestConsumptions_InstructionsClearWithNull(t *testing.T) {
	t.Parallel()
	s := createEditableStep(t)
	path := fmt.Sprintf("%s/%s/consumptions/%s", productionStepsPath, s.id, s.consumption)

	status, body, err := apiClient.Patch(path, map[string]any{"instructions": "Trim loose ends."}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "Trim loose ends.", jsonField(parseJSON(body), "instructions"))

	status, body, err = apiClient.Patch(path, map[string]any{"quantity_value": "3"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Equal(t, "Trim loose ends.", jsonField(parseJSON(body), "instructions"), "omitting instructions keeps them")

	status, body, err = apiClient.Patch(path, map[string]any{"instructions": nil}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)
	assert.Nil(t, parseJSON(body)["instructions"], "null clears the instructions")
}

func createPartItem(t *testing.T, prefix string) string {
	t.Helper()
	status, body, err := apiClient.Post(partsPath+"?include=item", validPartBody(uniqueName(prefix)), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	part := parseJSON(body)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + jsonField(part, "id")) })
	return jsonField(jsonObject(part, "item"), "id")
}

// createLinkedStep makes a step producing one each of produced from one each of consumed. Creating it
// links it to the steps around it by item, as the dashboard did.
func createLinkedStep(t *testing.T, name, produced, consumed string) string {
	t.Helper()
	rate := func(value, num, den string) map[string]any {
		return map[string]any{"value": value, "numerator_unit_id": num, "denominator_unit_id": den}
	}
	status, body, err := apiClient.Post(productionStepsPath, map[string]any{
		"name":            uniqueName(name),
		"leveling_factor": "0",
		"allowances":      "0",
		"labor_time":      rate("30", unitSecond, unitEach),
		"labor_rate":      rate("25", unitDollar, unitHour),
		"overhead_rate":   rate("20", unitDollar, unitHour),
		"production":      map[string]any{"item_id": produced, "quantity_value": "1", "quantity_unit_id": unitEach},
		"consumptions": []map[string]any{{"item_id": consumed, "quantity_value": "1", "quantity_unit_id": unitEach,
			"waste_quantity_value": "0", "waste_quantity_unit_id": unitEach}},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { cleanupStepIDs(id) })
	return id
}

// An item's flow is how it is made: its producing step and everything feeding it. A sibling product
// built from the same part is another item's flow. In and out steps are the flow's own steps, real
// names and all, and machines are the real machines.
func TestProductionFlows_AreUpstreamOfTheItem(t *testing.T) {
	t.Parallel()
	material := createLaborCostFixture(t).materialItemID
	part, product, sibling := createPartItem(t, "e2e-flow-part"), createPartItem(t, "e2e-flow-product"), createPartItem(t, "e2e-flow-sibling")

	partStep := createLinkedStep(t, "e2e-flow-make-part", part, material)
	productStep := createLinkedStep(t, "e2e-flow-make-product", product, part)
	siblingStep := createLinkedStep(t, "e2e-flow-make-sibling", sibling, part)
	machine := createMachineForStep(t)
	status, body, err := apiClient.Patch(productionStepsPath+"/"+partStep, map[string]any{"machine_ids": []string{machine}}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	status, body, err = apiClient.GetListRaw("/v1/operations/production-flows/by-item/"+product, url.Values{"include": {
		"steps", "steps.in_steps", "steps.out_steps", "steps.machines", "steps.production",
	}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	steps := map[string]map[string]any{}
	var order []string
	for _, raw := range jsonArray(jsonObject(parseJSON(body), "steps"), "data") {
		step := raw.(map[string]any)
		steps[jsonField(step, "id")] = step
		order = append(order, jsonField(step, "id"))
	}
	require.ElementsMatch(t, []string{productStep, partStep}, order, "the sibling's step is not part of the product's flow")
	assert.Equal(t, productStep, order[0], "the item's producer comes first")
	assert.NotContains(t, steps, siblingStep)

	linked := func(step map[string]any, key string) map[string]string {
		names := map[string]string{}
		for _, raw := range jsonArray(jsonObject(step, key), "data") {
			s := raw.(map[string]any)
			names[jsonField(s, "id")] = jsonField(s, "name")
		}
		return names
	}
	assert.Equal(t, map[string]string{partStep: jsonField(steps[partStep], "name")}, linked(steps[productStep], "in_steps"))
	assert.Empty(t, linked(steps[productStep], "out_steps"), "the producer feeds nothing in this flow")
	assert.Equal(t, map[string]string{productStep: jsonField(steps[productStep], "name")}, linked(steps[partStep], "out_steps"),
		"out steps are restricted to the flow, so the sibling's step is not listed")

	machines := jsonArray(jsonObject(steps[partStep], "machines"), "data")
	require.Len(t, machines, 1)
	m := machines[0].(map[string]any)
	assert.Equal(t, machine, jsonField(m, "id"))
	assert.NotEqual(t, "Machine", jsonField(m, "name"), "the machine is the real one, not a placeholder")
	assert.NotEqual(t, "—", jsonField(m, "serial_number"))
}
