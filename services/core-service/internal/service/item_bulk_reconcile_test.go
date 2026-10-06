package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// A yarn-like item stocked by the pound, whose group also carries grains, and the units an import
// might name against it.
const (
	recItemID  = "it_yarn"
	recGroupID = "ug_yarn"
	recPound   = "un_lbs"
	recGrain   = "un_gr"
	recGram    = "un_g"
	recEach    = "un_ea"
	recAcctEA  = "un_acct_ea"
)

func reconcileTestLookups() *reconcileLookups {
	item := domain.ItemSKUInfo{SKU: "YARN", ItemID: recItemID, UnitGroupID: recGroupID, BaseUnitID: recPound}
	return &reconcileLookups{
		itemsBySKU: map[string]domain.ItemSKUInfo{"YARN": item},
		unitsByAbbreviation: map[string][]*domain.Unit{
			"lbs": {{ID: recPound, Abbreviation: "lbs"}},
			"gr":  {{ID: recGrain, Abbreviation: "gr"}},
			"g":   {{ID: recGram, Abbreviation: "g"}},
			// A built-in each and an account's own "EA": neither is in the yarn group.
			"ea": {{ID: recEach, Abbreviation: "ea"}, {ID: recAcctEA, Abbreviation: "EA"}},
		},
		groupUnits: map[string]map[string]bool{recGroupID: {recPound: true, recGrain: true}},
		factors: map[string]domain.UnitFactors{
			recPound: {RatioNum: decimal.RequireFromString("45359237"), RatioDen: decimal.RequireFromString("100000"), DimensionCode: "mass"},
			recGrain: {RatioNum: decimal.RequireFromString("6479891"), RatioDen: decimal.RequireFromString("100000000"), DimensionCode: "mass"},
			recGram:  {RatioNum: decimal.NewFromInt(1), RatioDen: decimal.NewFromInt(1), IsBaseUnit: true, DimensionCode: "mass"},
			recEach:  {RatioNum: decimal.NewFromInt(1), RatioDen: decimal.NewFromInt(1), IsBaseUnit: true, DimensionCode: "quantity"},
		},
	}
}

func reconcileInput(sku, unit, measure string) domain.BulkReconcileItemInput {
	return domain.BulkReconcileItemInput{SKU: sku, Unit: unit, Measure: decimal.RequireFromString(measure)}
}

// A row is booked in the item's base unit whatever unit it names: 7,000 grains is exactly a pound.
func TestPlanBulkReconcile_ConvertsTheRowIntoTheBaseUnit(t *testing.T) {
	t.Parallel()

	rows, result := planBulkReconcile([]domain.BulkReconcileItemInput{
		reconcileInput("YARN", "gr", "7000"),
		reconcileInput("YARN", "LBS", "2.5"),
	}, reconcileTestLookups())

	require.Empty(t, result.Errors)
	require.Empty(t, result.SkippedItems)
	require.Len(t, rows, 2)
	assert.True(t, rows[0].baseMeasure.Equal(decimal.NewFromInt(1)), "7000 gr is 1 lbs, got %s", rows[0].baseMeasure)
	assert.True(t, rows[1].baseMeasure.Equal(decimal.RequireFromString("2.5")), "the base unit itself passes through, matched without case")
	assert.Equal(t, recItemID, rows[0].item.ItemID)
}

// Only the item's unit group says how its units relate, so a unit outside it is an error even when
// it measures the same dimension, and a count unit against a mass item certainly is.
func TestPlanBulkReconcile_UnitOutsideTheGroupIsAnErrorRow(t *testing.T) {
	t.Parallel()

	rows, result := planBulkReconcile([]domain.BulkReconcileItemInput{
		reconcileInput("YARN", "g", "100"),
		reconcileInput("YARN", "ea", "3"),
		reconcileInput("YARN", "furlongs", "3"),
		reconcileInput("NOPE", "lbs", "3"),
	}, reconcileTestLookups())

	assert.Empty(t, rows)
	require.Len(t, result.SkippedItems, 1)
	assert.Equal(t, "NOPE", result.SkippedItems[0].SKU)

	require.Len(t, result.Errors, 3)
	for _, e := range result.Errors {
		assert.Equal(t, recItemID, e.ItemID, "an error names the item its row resolved to")
		assert.Equal(t, "YARN", e.SKU)
	}
	assert.Contains(t, result.Errors[0].Error, "not in this item's unit group")
	assert.Contains(t, result.Errors[1].Error, "not in this item's unit group")
	assert.Contains(t, result.Errors[2].Error, "not found")
}

// An account may define its own unit with a built-in's abbreviation; the one in the item's group wins.
func TestPlanBulkReconcile_PicksTheGroupMemberAmongSameAbbreviationUnits(t *testing.T) {
	t.Parallel()

	lookups := reconcileTestLookups()
	lookups.groupUnits[recGroupID][recAcctEA] = true
	lookups.factors[recAcctEA] = domain.UnitFactors{RatioNum: decimal.NewFromInt(453), RatioDen: decimal.NewFromInt(1), DimensionCode: "mass"}

	rows, result := planBulkReconcile([]domain.BulkReconcileItemInput{reconcileInput("YARN", "ea", "1")}, lookups)

	require.Empty(t, result.Errors)
	require.Len(t, rows, 1)
	want := decimal.NewFromInt(453).Div(decimal.RequireFromString("453.59237"))
	assert.True(t, rows[0].baseMeasure.Equal(want), "converted through the group member's ratio, got %s", rows[0].baseMeasure)
}

func TestConvertToUnit_RefusesAcrossDimensions(t *testing.T) {
	t.Parallel()

	factors := reconcileTestLookups().factors
	_, ok := convertToUnit(decimal.NewFromInt(1), recEach, recPound, factors)
	assert.False(t, ok)
	_, ok = convertToUnit(decimal.NewFromInt(1), "un_unknown", recPound, factors)
	assert.False(t, ok)

	got, ok := convertToUnit(decimal.RequireFromString("453.59237"), recGram, recPound, factors)
	require.True(t, ok)
	assert.True(t, got.Equal(decimal.NewFromInt(1)), "got %s", got)
}

func reconcileRowsFor(n int) []reconcileRow {
	rows := make([]reconcileRow, n)
	for i := range rows {
		rows[i] = reconcileRow{sku: "SKU", item: domain.ItemSKUInfo{ItemID: "it_" + string(rune('a'+i%26))}, baseMeasure: decimal.NewFromInt(1)}
	}
	return rows
}

// Earlier batches have committed by the time a later one fails, so failing the request would invite
// a resubmission that applies them twice. The failed batch's rows are reported and the rest still run.
func TestRunReconcileBatches_AFailedBatchIsReportedRowByRowAndTheRestStillRun(t *testing.T) {
	t.Parallel()

	rows := reconcileRowsFor(2*bulkReconcileBatchSize + 7)
	result := &domain.BulkReconcileItemsResult{}
	var calls int
	failures := runReconcileBatches(rows, result, func(batch []reconcileRow) ([]domain.ReconciledItem, *apierror.APIError) {
		calls++
		if calls == 2 {
			return nil, apierror.NewConflictErrorWithParam("The ledger was busy.", "data")
		}
		out := make([]domain.ReconciledItem, len(batch))
		for i, row := range batch {
			out[i] = domain.ReconciledItem{ItemID: row.item.ItemID, SKU: row.sku}
		}
		return out, nil
	})

	assert.Equal(t, 3, calls, "every batch is attempted")
	require.Len(t, failures, 1)
	assert.Len(t, result.ReconciledItems, bulkReconcileBatchSize+7, "the first and third batches land")
	require.Len(t, result.Errors, bulkReconcileBatchSize, "every row of the failed batch is reported")
	assert.Equal(t, rows[bulkReconcileBatchSize].item.ItemID, result.Errors[0].ItemID)
	assert.Contains(t, result.Errors[0].Error, "The ledger was busy.")
}
