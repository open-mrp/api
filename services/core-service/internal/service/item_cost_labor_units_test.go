package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Count units as the unit group records them: an each is the base.
var (
	countEach       = domain.LightUnit{ID: "un_ea", Type: "quantity", RatioNumerator: "1", RatioDenominator: "1"}
	countPair       = domain.LightUnit{ID: "un_pr", Type: "quantity", RatioNumerator: "2", RatioDenominator: "1"}
	countCaseOfFive = domain.LightUnit{ID: "un_cs50ea", Type: "quantity", RatioNumerator: "50", RatioDenominator: "1"}
	weightPound     = domain.LightUnit{ID: "un_lbs", Type: "mass", RatioNumerator: "453.59237", RatioDenominator: "1"}
)

// costTolerance is a hundred-millionth of a cent: tight enough that any unit slip fails, loose enough
// for decimal's 16-digit division.
var costTolerance = decimal.RequireFromString("0.0000000001")

func requireCostEqual(t *testing.T, label string, got decimal.Decimal, want string) {
	t.Helper()
	w := decimal.RequireFromString(want)
	if got.Sub(w).Abs().GreaterThan(costTolerance) {
		t.Errorf("%s = %s, want %s (off by a factor of %s)", label, got, w, got.Div(w))
	}
}

// laborTimeIn is a labor time of value seconds per one quantityUnit.
func laborTimeIn(value string, quantityUnit domain.LightUnit) *domain.FlowRate {
	return &domain.FlowRate{
		Value:               value,
		NumeratorRatio:      ratioSecond,
		DenominatorRatio:    unitRatio(quantityUnit.RatioNumerator, quantityUnit.RatioDenominator).String(),
		DenominatorUnitType: quantityUnit.Type,
	}
}

func perHour(value string) *domain.FlowRate {
	return &domain.FlowRate{Value: value, DenominatorRatio: ratioHour}
}

func laborStep(batch string, producedIn domain.LightUnit, laborTime *domain.FlowRate, laborRate, overheadRate string) *domain.ProductionFlowStep {
	return &domain.ProductionFlowStep{
		Production: domain.StepProduction{Quantity: domain.BatchQuantity{
			Measure: decimal.RequireFromString(batch),
			Unit:    producedIn,
		}},
		LevelingFactor: "0",
		Allowances:     "0",
		LaborTime:      laborTime,
		LaborRate:      perHour(laborRate),
		OverheadRate:   perHour(overheadRate),
	}
}

func perEachMaterial(qty, costPerEach string) domain.CostFlowConsumption {
	return consumedAgainstItsOwnUnit(qty, costPerEach, "1")
}

func TestLaborTimeUnitsPerProducedUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		producedIn domain.LightUnit
		laborTime  *domain.FlowRate
		want       string
	}{
		{"labor per each, producing eaches", countEach, laborTimeIn("1", countEach), "1"},
		{"labor per pair, producing pairs", countPair, laborTimeIn("1", countPair), "1"},
		{"labor per case of fifty, producing eaches", countEach, laborTimeIn("1", countCaseOfFive), "0.02"},
		{"labor per each, producing cases of fifty", countCaseOfFive, laborTimeIn("1", countEach), "50"},
		{"labor per pair, producing eaches", countEach, laborTimeIn("1", countPair), "0.5"},
		{"labor per each, producing pairs", countPair, laborTimeIn("1", countEach), "2"},
		{"labor per case of fifty, producing pairs", countPair, laborTimeIn("1", countCaseOfFive), "0.04"},
		{"labor per pound, producing eaches: no conversion exists", countEach, laborTimeIn("1", weightPound), "1"},
		{"labor with no quantity unit recorded", countCaseOfFive, &domain.FlowRate{Value: "1"}, "50"},
		{"produced unit with no ratio recorded", domain.LightUnit{Type: "quantity"}, laborTimeIn("1", countCaseOfFive), "0.02"},
		{"produced unit with a zero ratio", domain.LightUnit{Type: "quantity", RatioNumerator: "0", RatioDenominator: "1"}, laborTimeIn("1", countEach), "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			step := laborStep("1", tt.producedIn, tt.laborTime, "0", "0")
			got := laborTimeUnitsPerProducedUnit(step)
			if !got.Equal(decimal.RequireFromString(tt.want)) {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

// A boxing step timed at 733.2 seconds per case of fifty, at $25.18 and $24.14 an hour, producing one
// each. Reading the time as per each charges fifty times the labor: $10.05 an each.
func TestCalculateStepCost_LaborTimePerCaseIsSpreadAcrossTheCase(t *testing.T) {
	t.Parallel()

	step := laborStep("1", countEach, laborTimeIn("733.2", countCaseOfFive), "25.18", "24.14")
	got := calculateStepCost(step, []domain.CostFlowConsumption{
		perEachMaterial("0.04", "3.26"),
		perEachMaterial("0.02", "1.458"),
	})

	requireCostEqual(t, "material", got.material, "0.15956")
	requireCostEqual(t, "labor", got.labor, "0.1025665333333333333")
	requireCostEqual(t, "overhead", got.overhead, "0.0983302666666666667")
	requireCostEqual(t, "total", got.total, "0.3604568")
}

// The same work written every way a person might enter it has to cost the same. 14.664 seconds an
// each is 733.2 a case of fifty, 29.328 a pair, and 0.2444 minutes an each.
func TestCalculateStepCost_LaborCostIsIndependentOfTheUnitsItIsEnteredIn(t *testing.T) {
	t.Parallel()

	minutesPerEach := laborTimeIn("0.2444", countEach)
	minutesPerEach.NumeratorRatio = ratioMinute

	laborTimes := map[string]*domain.FlowRate{
		"seconds per each":          laborTimeIn("14.664", countEach),
		"seconds per case of fifty": laborTimeIn("733.2", countCaseOfFive),
		"seconds per pair":          laborTimeIn("29.328", countPair),
		"minutes per each":          minutesPerEach,
	}
	batches := []struct {
		name       string
		qty        string
		producedIn domain.LightUnit
	}{
		{"one each", "1", countEach},
		{"a hundred eaches", "100", countEach},
		{"fifty pairs", "50", countPair},
		{"two cases of fifty", "2", countCaseOfFive},
	}

	for ltName, lt := range laborTimes {
		for _, b := range batches {
			t.Run(ltName+"/"+b.name, func(t *testing.T) {
				t.Parallel()
				step := laborStep(b.qty, b.producedIn, lt, "25.18", "24.14")
				got := calculateStepCost(step, nil)

				eaches := decimal.RequireFromString(b.qty).Mul(unitRatio(b.producedIn.RatioNumerator, b.producedIn.RatioDenominator))
				perEach := got.total.Div(eaches)
				requireCostEqual(t, "labor and overhead per each", perEach, "0.2008968")
			})
		}
	}
}

// A step whose labor time is in the unit it produces in is untouched: the conversion is the identity.
// Nearly every step in production has this shape.
func TestCalculateStepCost_LaborInTheProducedUnitIsUnchanged(t *testing.T) {
	t.Parallel()

	step := laborStep("24", countPair, laborTimeIn("410", countPair), "2.51", "5.12")
	got := calculateStepCost(step, nil)

	// 24 pairs × 410 s × ($2.51 + $5.12)/h
	requireCostEqual(t, "labor", got.labor, "6.860666666666666667")
	requireCostEqual(t, "overhead", got.overhead, "13.994666666666666667")
}

// Labor per pair on a step producing eaches is half a pair's time per each, so half the cost.
func TestCalculateStepCost_LaborPerPairOnAStepProducingEaches(t *testing.T) {
	t.Parallel()

	perEach := calculateStepCost(laborStep("24", countEach, laborTimeIn("205", countEach), "2.51", "5.12"), nil)
	perPair := calculateStepCost(laborStep("24", countEach, laborTimeIn("410", countPair), "2.51", "5.12"), nil)

	requireCostEqual(t, "labor per pair", perPair.labor, perEach.labor.String())
	requireCostEqual(t, "overhead per pair", perPair.overhead, perEach.overhead.String())
	requireCostEqual(t, "labor", perPair.labor, "3.4303333333333333333")
}

// Leveling factor and allowances scale the converted time, not the raw entry.
func TestCalculateStepCost_CorrectiveFactorAppliesAfterUnitConversion(t *testing.T) {
	t.Parallel()

	step := laborStep("1", countEach, laborTimeIn("733.2", countCaseOfFive), "25.18", "24.14")
	step.LevelingFactor = "0.1"
	step.Allowances = "0.15"
	got := calculateStepCost(step, nil)

	// (1.1 × 1.15) × 14.664 s × $49.32/h
	requireCostEqual(t, "labor and overhead", got.labor.Add(got.overhead), "0.254134452")
}

// A sewn part, then a boxing step timed per case of fifty. Reading that time as per each costs the
// boxed item $12.566307278888… an each; it is $2.722364078888….
func TestComputeItemCosts_SewnThenBoxedItemTimedPerCase(t *testing.T) {
	t.Parallel()

	const (
		accountID  = "ac_cost"
		itemID     = "itm_boxed"
		sewnID     = "itm_sewn"
		boxStepID  = "ps_box"
		sewStepID  = "ps_sew"
		currencyID = "un_usd"
	)
	ctrl := gomock.NewController(t)

	sew := laborStep("1", countEach, laborTimeIn("40.9142", countEach), "25.45", "22.77")
	sew.ID = sewStepID
	sew.Production.ProducedItem = domain.LightItem{ID: sewnID}

	box := laborStep("1", countEach, laborTimeIn("733.2", countCaseOfFive), "25.18", "24.14")
	box.ID = boxStepID
	box.Production.ProducedItem = domain.LightItem{ID: itemID}

	steps := map[string]*domain.ProductionFlowStep{sewStepID: sew, boxStepID: box}
	consumptions := map[string][]domain.CostFlowConsumption{
		sewStepID: {
			perEachMaterial("1", "0.69"),
			perEachMaterial("1", "0.57"),
			perEachMaterial("1", "0.52"),
			perEachMaterial("0.167", "0.2029"),
		},
		boxStepID: {
			perEachMaterial("0.04", "3.26"),
			perEachMaterial("0.02", "1.458"),
			{ConsumedItemType: "part", ConsumptionQuantity: decimal.NewFromInt(1), ConsumptionUnitRatio: decimal.NewFromInt(1),
				WasteQuantity: decimal.Zero, WasteUnitRatio: decimal.NewFromInt(1),
				UnitCost: decimal.RequireFromString("2.36"), UnitCostDenominatorRatio: decimal.NewFromInt(1)},
		},
	}
	details := map[string]*domain.ProductionStepDetail{
		sewStepID: {ID: sewStepID},
		boxStepID: {ID: boxStepID, Consumptions: []domain.StepConsumption{{
			ConsumedItem: domain.LightItem{ID: sewnID},
			Quantity:     domain.BatchQuantity{Measure: decimal.NewFromInt(1), Unit: countEach},
		}}},
	}

	flowRepo := repositorymock.NewMockProductionFlowRepo(ctrl)
	flowRepo.EXPECT().FindStepsByProducedItem(gomock.Any(), accountID, itemID).Return([]string{boxStepID}, nil)
	flowRepo.EXPECT().GetAllStepEdgesForAccount(gomock.Any(), accountID).
		Return([]domain.StepEdge{{ParentStepID: sewStepID, ChildStepID: boxStepID}}, nil)
	flowRepo.EXPECT().GetFlowStep(gomock.Any(), accountID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _, id string) (*domain.ProductionFlowStep, *apierror.APIError) {
			return steps[id], nil
		}).Times(2)

	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	itemRepo.EXPECT().GetCostFlowConsumptions(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string) ([]domain.CostFlowConsumption, *apierror.APIError) {
			return consumptions[id], nil
		}).Times(2)
	itemRepo.EXPECT().GetStockingUnit(gomock.Any(), accountID, itemID).
		Return(&domain.ItemStockingUnit{UnitGroupID: "ug_each", BaseUnitID: countEach.ID, CostNumeratorUnitID: currencyID}, nil)

	stepQueryRepo := repositorymock.NewMockProductionStepQueryRepo(ctrl)
	stepQueryRepo.EXPECT().Find(gomock.Any(), accountID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _, id string) (*domain.ProductionStepDetail, *apierror.APIError) {
			return details[id], nil
		}).Times(2)

	unitRepo := repositorymock.NewMockUnitRepo(ctrl)
	unitRepo.EXPECT().IsUnitInGroup(gomock.Any(), "ug_each", countEach.ID).Return(true, nil)

	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewProductionFlowRepo().Return(flowRepo).AnyTimes()
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	repos.EXPECT().NewProductionStepQueryRepo().Return(stepQueryRepo).AnyTimes()
	repos.EXPECT().NewUnitRepo().Return(unitRepo).AnyTimes()
	repos.EXPECT().NewUnitConversionRepo().Return(repositorymock.NewMockUnitConversionRepo(ctrl)).AnyTimes()

	svc := &itemSvcImpl{repos: repos}
	costs, apiErr := svc.ComputeItemCosts(context.Background(), accountID, itemID)
	if apiErr != nil {
		t.Fatalf("ComputeItemCosts: %v", apiErr)
	}

	requireCostEqual(t, "direct material", decimal.RequireFromString(costs.DirectMaterialCost), "1.9734443")
	requireCostEqual(t, "direct labor", decimal.RequireFromString(costs.DirectLaborCost), "0.3918071972222222222")
	requireCostEqual(t, "overhead", decimal.RequireFromString(costs.OverheadCost), "0.3571125816666666667")
	requireCostEqual(t, "total", decimal.RequireFromString(costs.TotalCost), "2.7223640788888888889")
	if costs.UnitID != countEach.ID {
		t.Errorf("unit = %s, want %s", costs.UnitID, countEach.ID)
	}
	if costs.NumeratorUnitID != currencyID {
		t.Errorf("currency = %s, want %s", costs.NumeratorUnitID, currencyID)
	}
}
