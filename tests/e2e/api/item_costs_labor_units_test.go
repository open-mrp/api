//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Item costs price a step's labor from its labor time, which is entered per whatever count of output
// suited whoever entered it. These build a part stocked by the pair, made by a step that consumes one
// $0.10 packaging each per each produced and takes 30 seconds an each at $25 and $20 an hour, and write
// that same work in every unit combination a person might. The cost of a pair must not move:
// $0.20 of material, $0.41666… of labor and $0.33333… of overhead.

const (
	unitSecond  = "second"
	unitMinute  = "minute"
	unitHour    = "hour"
	unitDollar  = "dollar"
	unitEach    = "each"
	unitDozen   = "un_01seeddozen00000000"
	unitCarton  = "un_e2ecarton12pr000" // twelve pairs: 24 eaches
	packagingID = "itcg_01seedpackaging00"
)

type laborCostFixture struct {
	materialItemID string
	partItemID     string
}

func createLaborCostFixture(t *testing.T) laborCostFixture {
	t.Helper()

	status, body, err := apiClient.Post(materialsPath+"?include=item", map[string]any{
		"sku":         uniqueName("e2e-labor-units-box"),
		"category_id": packagingID,
		"unit_cost":   map[string]any{"value": "0.10", "numerator_unit_id": unitDollar, "denominator_unit_id": unitEach},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	material := parseJSON(body)
	t.Cleanup(func() { apiClient.Delete(materialsPath + "/" + jsonField(material, "id")) })

	status, body, err = apiClient.Post(partsPath+"?include=item", validPartBody(uniqueName("e2e-labor-units-part")), newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	part := parseJSON(body)
	t.Cleanup(func() { apiClient.Delete(partsPath + "/" + jsonField(part, "id")) })

	materialItem := jsonObject(material, "item")
	partItem := jsonObject(part, "item")
	require.NotNil(t, materialItem, "material must carry its item: %s", body)
	require.NotNil(t, partItem, "part must carry its item: %s", body)
	return laborCostFixture{materialItemID: jsonField(materialItem, "id"), partItemID: jsonField(partItem, "id")}
}

type laborEntry struct {
	value, timeUnit, perUnit string
}

type laborStepSpec struct {
	labor                     laborEntry
	produceQty, produceUnit   string
	eachesProduced            string
	levelingFactor, allowance string
}

func createLaborStep(t *testing.T, f laborCostFixture, spec laborStepSpec) {
	t.Helper()
	leveling, allowance := spec.levelingFactor, spec.allowance
	if leveling == "" {
		leveling = "0"
	}
	if allowance == "" {
		allowance = "0"
	}

	status, body, err := apiClient.Post(productionStepsPath, map[string]any{
		"name":            uniqueName("e2e-labor-units-step"),
		"leveling_factor": leveling,
		"allowances":      allowance,
		"labor_time":      map[string]any{"value": spec.labor.value, "numerator_unit_id": spec.labor.timeUnit, "denominator_unit_id": spec.labor.perUnit},
		"labor_rate":      map[string]any{"value": "25", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
		"overhead_rate":   map[string]any{"value": "20", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
		"production":      map[string]any{"item_id": f.partItemID, "quantity_value": spec.produceQty, "quantity_unit_id": spec.produceUnit},
		"consumptions": []map[string]any{
			{"item_id": f.materialItemID, "quantity_value": spec.eachesProduced, "quantity_unit_id": unitEach,
				"waste_quantity_value": "0", "waste_quantity_unit_id": unitEach},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	stepID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { cleanupStepIDs(stepID) })
}

func getItemCosts(t *testing.T, itemID string) map[string]any {
	t.Helper()
	status, body, err := apiClient.GetListRaw(itemsPath+"/"+itemID+"/costs", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "costs must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func TestItemCosts_LaborIsTheSameWhateverUnitsItIsEnteredIn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		spec laborStepSpec
	}{
		{"seconds per each, producing an each", laborStepSpec{labor: laborEntry{"30", unitSecond, unitEach}, produceQty: "1", produceUnit: unitEach, eachesProduced: "1"}},
		{"seconds per dozen, producing an each", laborStepSpec{labor: laborEntry{"360", unitSecond, unitDozen}, produceQty: "1", produceUnit: unitEach, eachesProduced: "1"}},
		{"seconds per pair, producing an each", laborStepSpec{labor: laborEntry{"60", unitSecond, SeedPairUnitID}, produceQty: "1", produceUnit: unitEach, eachesProduced: "1"}},
		{"minutes per each, producing a dozen", laborStepSpec{labor: laborEntry{"0.5", unitMinute, unitEach}, produceQty: "1", produceUnit: unitDozen, eachesProduced: "12"}},
		{"seconds per carton, producing three pairs", laborStepSpec{labor: laborEntry{"720", unitSecond, unitCarton}, produceQty: "3", produceUnit: SeedPairUnitID, eachesProduced: "6"}},
		{"hours per dozen, producing a carton", laborStepSpec{labor: laborEntry{"0.1", unitHour, unitDozen}, produceQty: "1", produceUnit: unitCarton, eachesProduced: "24"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := createLaborCostFixture(t)
			createLaborStep(t, f, tc.spec)

			costs := getItemCosts(t, f.partItemID)
			assertObjectField(t, costs, "item_costs")
			assertDecimalEqual(t, "0.20", jsonField(costs, "direct_material_cost"), "material per pair")
			assertDecimalEqual(t, "0.416666666666666667", jsonField(costs, "direct_labor_cost"), "labor per pair")
			assertDecimalEqual(t, "0.333333333333333333", jsonField(costs, "overhead_cost"), "overhead per pair")
			assertDecimalEqual(t, "0.95", jsonField(costs, "total_cost"), "total per pair")
		})
	}
}

// Leveling factor and allowances stretch the converted time, not the figure as entered:
// (1.1 × 1.15) × 30 s an each, at $45 an hour, is $0.94875 a pair of labor and overhead.
func TestItemCosts_CorrectiveFactorAppliesAfterLaborUnitConversion(t *testing.T) {
	t.Parallel()

	f := createLaborCostFixture(t)
	createLaborStep(t, f, laborStepSpec{
		labor:      laborEntry{"360", unitSecond, unitDozen},
		produceQty: "1", produceUnit: unitEach, eachesProduced: "1",
		levelingFactor: "0.1", allowance: "0.15",
	})

	costs := getItemCosts(t, f.partItemID)
	assertDecimalEqual(t, "0.527083333333333333", jsonField(costs, "direct_labor_cost"))
	assertDecimalEqual(t, "0.421666666666666667", jsonField(costs, "overhead_cost"))
	assertDecimalEqual(t, "1.14875", jsonField(costs, "total_cost"))
}

// Reading costs computes them; it must not store them, so a second read sees what the first did.
func TestItemCosts_ReadingCostsTwiceAgrees(t *testing.T) {
	t.Parallel()

	f := createLaborCostFixture(t)
	createLaborStep(t, f, laborStepSpec{labor: laborEntry{"360", unitSecond, unitDozen}, produceQty: "1", produceUnit: unitEach, eachesProduced: "1"})

	first := getItemCosts(t, f.partItemID)
	second := getItemCosts(t, f.partItemID)
	for _, field := range []string{"direct_material_cost", "direct_labor_cost", "overhead_cost", "total_cost"} {
		require.Equal(t, jsonField(first, field), jsonField(second, field), field)
	}
}

// A consumption's waste is required like its quantity. Leaving it out is the caller's mistake to fix,
// so it is a 400 naming the field, never a database error.
func TestProductionSteps_CreateRejectsAConsumptionWithoutWaste(t *testing.T) {
	t.Parallel()

	f := createLaborCostFixture(t)
	status, body, err := apiClient.Post(productionStepsPath, map[string]any{
		"name":            uniqueName("e2e-step-no-waste"),
		"leveling_factor": "0",
		"allowances":      "0",
		"labor_time":      map[string]any{"value": "30", "numerator_unit_id": unitSecond, "denominator_unit_id": unitEach},
		"labor_rate":      map[string]any{"value": "25", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
		"overhead_rate":   map[string]any{"value": "20", "numerator_unit_id": unitDollar, "denominator_unit_id": unitHour},
		"production":      map[string]any{"item_id": f.partItemID, "quantity_value": "1", "quantity_unit_id": unitEach},
		"consumptions": []map[string]any{
			{"item_id": f.materialItemID, "quantity_value": "1", "quantity_unit_id": unitEach},
		},
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "a missing field must not reach the database: %s", string(body))
	requireStatus(t, 400, status, body)
	errObj := requireErrorResponse(t, body, "validation_failed", "invalid_request_error")
	assertErrorParam(t, errObj, "consumptions[0].waste_quantity_value")
}
