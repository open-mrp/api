package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

var productionCostBaseUnits = map[string]string{"currency": "dollar", "time": "hour", "quantity": "each", "mass": "gram"}

var massGram = domain.LightUnit{ID: "gram", Type: "mass", RatioNumerator: "1", RatioDenominator: "1"}

func minutesPer(value string, quantityUnit domain.LightUnit) *domain.FlowRate {
	return &domain.FlowRate{
		Value:               value,
		NumeratorRatio:      ratioMinute,
		DenominatorRatio:    unitRatio(quantityUnit.RatioNumerator, quantityUnit.RatioDenominator).String(),
		DenominatorUnitType: quantityUnit.Type,
	}
}

type costStepSpec struct {
	produce          string
	unit             domain.LightUnit
	laborTime        *domain.FlowRate
	laborRate        string
	overheadRate     string
	leveling, allows string
	consumptions     []domain.CostFlowConsumption
}

func costStep(s costStepSpec) domain.ProductionCostStep {
	unit := s.unit
	unit.OffsetNumerator, unit.OffsetDenominator = "0", "1"
	leveling, allows := s.leveling, s.allows
	if leveling == "" {
		leveling = "0"
	}
	if allows == "" {
		allows = "0"
	}
	return domain.ProductionCostStep{
		Step: domain.ProductionFlowStep{
			Production:     domain.StepProduction{Quantity: domain.BatchQuantity{Measure: decimal.RequireFromString(s.produce), Unit: unit}},
			LevelingFactor: leveling,
			Allowances:     allows,
			LaborTime:      s.laborTime,
			LaborRate:      perHour(s.laborRate),
			OverheadRate:   perHour(s.overheadRate),
		},
		Consumptions: s.consumptions,
	}
}

func batches(base string, count int64) domain.ProductionCostBatches {
	return domain.ProductionCostBatches{BaseQuantity: decimal.RequireFromString(base), Count: count}
}

func ref(id, name string) *domain.ProductionCostRef {
	return &domain.ProductionCostRef{ID: id, Name: name}
}

type wantCost struct {
	materials, labor, overhead, total, hours string
	produced                                 map[string]string
}

func requireProductionCost(t *testing.T, label string, got domain.ProductionCost, want wantCost) {
	t.Helper()
	requireCostEqual(t, label+" materials", got.Materials, want.materials)
	requireCostEqual(t, label+" labor", got.Labor, want.labor)
	requireCostEqual(t, label+" overhead", got.Overhead, want.overhead)
	requireCostEqual(t, label+" total", got.Total, want.total)
	requireCostEqual(t, label+" labor hours", got.LaborHours, want.hours)
	require.Len(t, got.Produced, len(want.produced), label+" produced")
	for _, p := range got.Produced {
		w, ok := want.produced[p.UnitID]
		require.True(t, ok, "%s produced an unexpected unit %s", label, p.UnitID)
		requireCostEqual(t, label+" produced "+p.UnitID, p.Value, w)
	}
}

// One run makes 10 ea from 20 ea of a $0.50 material with 2 ea of waste ($11), plus a part that is not
// material; it takes 6 min an each stretched by 1.155 (0.1 leveling, 0.05 allowances): 1.155 h at $25
// and $20 an hour. Each kind of output is charged the runs it amounts to.
func TestAssembleProductionCostReport_ChargesEachKindOfOutputItsRuns(t *testing.T) {
	t.Parallel()

	material := perEachMaterial("20", "0.50")
	material.WasteQuantity = decimal.RequireFromString("2")
	part := perEachMaterial("10", "3")
	part.ConsumedItemType = "part"
	steps := map[string]domain.ProductionCostStep{"ps_cut": costStep(costStepSpec{
		produce: "10", unit: countEach, laborTime: minutesPer("6", countEach), laborRate: "25", overheadRate: "20",
		leveling: "0.1", allows: "0.05", consumptions: []domain.CostFlowConsumption{material, part},
	})}
	rows := []domain.ProductionCostRow{{
		ProductionStepID: "ps_cut",
		Department:       ref("dp_a", "Assembly"),
		Category:         domain.ProductionCostRef{ID: "ic_x", Name: "Socks"},
		Productive:       batches("25", 3),
		Seconds:          batches("5", 1),
		Waste:            batches("2.5", 1),
	}}

	report, apiErr := assembleProductionCostReport(rows, steps, productionCostBaseUnits)
	require.Nil(t, apiErr)
	require.Equal(t, "dollar", report.CurrencyUnitID)
	require.Equal(t, "hour", report.TimeUnitID)

	productive := wantCost{"27.5", "72.1875", "57.75", "157.4375", "2.8875", map[string]string{"each": "25"}}
	seconds := wantCost{"5.5", "14.4375", "11.55", "31.4875", "0.5775", map[string]string{"each": "5"}}
	waste := wantCost{"2.75", "7.21875", "5.775", "15.74375", "0.28875", map[string]string{"each": "2.5"}}
	total := wantCost{"35.75", "93.84375", "75.075", "204.66875", "3.75375", map[string]string{"each": "32.5"}}
	for _, set := range []domain.ProductionCostSet{report.Totals, report.Departments[0].Costs, report.Categories[0].Costs, report.DepartmentCategories[0].Costs} {
		requireProductionCost(t, "productive", set.Productive, productive)
		requireProductionCost(t, "seconds", set.Seconds, seconds)
		requireProductionCost(t, "waste", set.Waste, waste)
		requireProductionCost(t, "total", set.Total, total)
	}
}

// A step producing 1 pr whose labor time is 2 min an each takes 4 minutes a run: $0.67 at $10 an hour.
// The dashboard multiplied in base units and read the hours it got as minutes, so it reported 1/60 of
// that: $0.0111 of labor and 0.0667 "minutes".
func TestAssembleProductionCostReport_LaborTimeIsReadInItsOwnUnits(t *testing.T) {
	t.Parallel()

	steps := map[string]domain.ProductionCostStep{"ps_pair": costStep(costStepSpec{
		produce: "1", unit: countPair, laborTime: minutesPer("2", countEach), laborRate: "10", overheadRate: "6",
	})}
	rows := []domain.ProductionCostRow{{
		ProductionStepID: "ps_pair",
		Category:         domain.ProductionCostRef{ID: "ic_x", Name: "Socks"},
		// One pair, as two eaches in the base unit.
		Productive: batches("2", 1),
	}}

	report, apiErr := assembleProductionCostReport(rows, steps, productionCostBaseUnits)
	require.Nil(t, apiErr)
	requireProductionCost(t, "productive", report.Totals.Productive, wantCost{
		materials: "0", labor: "0.666666666666666667", overhead: "0.4", total: "1.066666666666666667", hours: "0.066666666666666667",
		produced: map[string]string{"each": "2"},
	})
	legacyLabor := decimal.RequireFromString("0.0111111111")
	require.False(t, report.Totals.Productive.Labor.Sub(legacyLabor).Abs().LessThan(decimal.RequireFromString("0.001")), "labor must not be scaled by the labor time unit's size in hours")
}

// Batches are counted in the base unit, whatever they were recorded in: 3 dz is 36 ea, which is three runs
// of a step producing 6 pr. Labor of 30 s a pair is 3 minutes a run.
func TestAssembleProductionCostReport_RunsAreBatchesInTheProductionsUnit(t *testing.T) {
	t.Parallel()

	steps := map[string]domain.ProductionCostStep{"ps_knit": costStep(costStepSpec{
		produce: "6", unit: countPair, laborTime: laborTimeIn("30", countPair), laborRate: "20", overheadRate: "0",
		consumptions: []domain.CostFlowConsumption{perEachMaterial("12", "0.25")},
	})}
	rows := []domain.ProductionCostRow{{
		ProductionStepID: "ps_knit",
		Category:         domain.ProductionCostRef{ID: "ic_x", Name: "Socks"},
		Productive:       batches("36", 1),
	}}

	report, apiErr := assembleProductionCostReport(rows, steps, productionCostBaseUnits)
	require.Nil(t, apiErr)
	requireProductionCost(t, "productive", report.Totals.Productive, wantCost{
		materials: "9", labor: "3", overhead: "0", total: "12", hours: "0.15",
		produced: map[string]string{"each": "36"},
	})
}

// Groups are ordered by name, batches with no department last, and every rollup is the sum of the rows
// under it. A row whose step did not load (another account's, or one with no production) is left out.
func TestAssembleProductionCostReport_RollsUpByDepartmentAndCategory(t *testing.T) {
	t.Parallel()

	// $1 of material a run of one each; no labor.
	steps := map[string]domain.ProductionCostStep{
		"ps_one": costStep(costStepSpec{produce: "1", unit: countEach, laborRate: "0", overheadRate: "0",
			consumptions: []domain.CostFlowConsumption{perEachMaterial("1", "1")}}),
		"ps_yarn": costStep(costStepSpec{produce: "1000", unit: massGram, laborRate: "0", overheadRate: "0",
			consumptions: []domain.CostFlowConsumption{perEachMaterial("1", "4")}}),
	}
	knitting, assembly := ref("dp_k", "Knitting"), ref("dp_a", "Assembly")
	socks := domain.ProductionCostRef{ID: "ic_s", Name: "Socks"}
	yarn := domain.ProductionCostRef{ID: "ic_y", Name: "Yarn"}
	rows := []domain.ProductionCostRow{
		{ProductionStepID: "ps_one", Department: knitting, Category: socks, Productive: batches("2", 1)},
		{ProductionStepID: "ps_one", Department: assembly, Category: socks, Productive: batches("3", 1), Waste: batches("1", 1)},
		{ProductionStepID: "ps_one", Category: socks, Productive: batches("5", 2)},
		{ProductionStepID: "ps_yarn", Department: knitting, Category: yarn, Productive: batches("500", 1)},
		{ProductionStepID: "ps_missing", Department: knitting, Category: yarn, Productive: batches("100", 1)},
	}

	report, apiErr := assembleProductionCostReport(rows, steps, productionCostBaseUnits)
	require.Nil(t, apiErr)

	names := func(groups []domain.ProductionCostGroup, pick func(domain.ProductionCostGroup) *domain.ProductionCostRef) []string {
		out := []string{}
		for _, g := range groups {
			if r := pick(g); r != nil {
				out = append(out, r.Name)
			} else {
				out = append(out, "-")
			}
		}
		return out
	}
	department := func(g domain.ProductionCostGroup) *domain.ProductionCostRef { return g.Department }
	category := func(g domain.ProductionCostGroup) *domain.ProductionCostRef { return g.Category }

	require.Equal(t, []string{"Assembly", "Knitting", "-"}, names(report.Departments, department))
	require.Equal(t, []string{"-", "-", "-"}, names(report.Departments, category))
	require.Equal(t, []string{"Socks", "Yarn"}, names(report.Categories, category))
	require.Equal(t, []string{"-", "-"}, names(report.Categories, department))
	require.Equal(t, []string{"Assembly", "Knitting", "Knitting", "-"}, names(report.DepartmentCategories, department))
	require.Equal(t, []string{"Socks", "Socks", "Yarn", "Socks"}, names(report.DepartmentCategories, category))

	requireCostEqual(t, "assembly", report.Departments[0].Costs.Total.Materials, "4")
	requireCostEqual(t, "knitting", report.Departments[1].Costs.Total.Materials, "4")
	requireCostEqual(t, "no department", report.Departments[2].Costs.Total.Materials, "5")
	requireCostEqual(t, "socks", report.Categories[0].Costs.Total.Materials, "11")
	requireCostEqual(t, "yarn", report.Categories[1].Costs.Total.Materials, "2")
	requireCostEqual(t, "waste", report.Totals.Waste.Materials, "1")
	requireProductionCost(t, "totals", report.Totals.Total, wantCost{
		materials: "13", labor: "0", overhead: "0", total: "13", hours: "0",
		produced: map[string]string{"each": "11", "gram": "500"},
	})
	require.Equal(t, "each", report.Totals.Total.Produced[0].UnitID, "produced is ordered by unit")
}

// A production with no quantity cannot be costed per run; its batches cost nothing but still count as produced.
func TestAssembleProductionCostReport_AStepProducingNothingCostsNothing(t *testing.T) {
	t.Parallel()

	steps := map[string]domain.ProductionCostStep{"ps_zero": costStep(costStepSpec{
		produce: "0", unit: countEach, laborTime: minutesPer("6", countEach), laborRate: "25", overheadRate: "20",
		consumptions: []domain.CostFlowConsumption{perEachMaterial("1", "1")},
	})}
	rows := []domain.ProductionCostRow{{ProductionStepID: "ps_zero", Category: domain.ProductionCostRef{ID: "ic_x", Name: "Socks"}, Productive: batches("7", 1)}}

	report, apiErr := assembleProductionCostReport(rows, steps, productionCostBaseUnits)
	require.Nil(t, apiErr)
	requireProductionCost(t, "productive", report.Totals.Productive, wantCost{
		materials: "0", labor: "0", overhead: "0", total: "0", hours: "0", produced: map[string]string{"each": "7"},
	})
}

func TestAssembleProductionCostReport_NoRowsIsAnEmptyReport(t *testing.T) {
	t.Parallel()

	report, apiErr := assembleProductionCostReport(nil, nil, productionCostBaseUnits)
	require.Nil(t, apiErr)
	require.Empty(t, report.Departments)
	require.Empty(t, report.Categories)
	require.Empty(t, report.DepartmentCategories)
	require.True(t, report.Totals.Total.Total.IsZero())
	require.Empty(t, report.Totals.Total.Produced)
}

func TestAssembleProductionCostReport_NeedsTheCurrencyAndTimeBaseUnits(t *testing.T) {
	t.Parallel()

	_, apiErr := assembleProductionCostReport(nil, nil, map[string]string{"time": "hour"})
	require.NotNil(t, apiErr)
}

// The report is batches:read for internal users, validates its window, attaches the units its figures
// are counted in, and is computed once per window and filters.
func TestAnalyzeProductionCosts_ChecksAccessValidatesAndCaches(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	ctrl := gomock.NewController(t)
	analytics := repositorymock.NewMockAnalyticsRepo(ctrl)
	units := repositorymock.NewMockUnitRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAnalyticsRepo().Return(analytics).AnyTimes()
	repos.EXPECT().NewUnitRepo().Return(units).AnyTimes()
	reportCache, _ := newTestAnalyticsCache(t, now)
	svc := &analyticsSvcImpl{repos: repos, cache: reportCache}

	params := domain.AnalyzeProductionCostsParams{
		AccountID:     "ac_spoofed",
		StartDate:     time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC),
		DepartmentIDs: []string{"dp_a"},
	}
	var seen []domain.AnalyzeProductionCostsParams
	analytics.EXPECT().GetProductionCostRows(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p domain.AnalyzeProductionCostsParams) ([]domain.ProductionCostRow, *apierror.APIError) {
		seen = append(seen, p)
		return []domain.ProductionCostRow{
			{ProductionStepID: "ps_1", Category: domain.ProductionCostRef{ID: "ic_x", Name: "Socks"}, Productive: batches("4", 1)},
			{ProductionStepID: "ps_1", Department: ref("dp_a", "Assembly"), Category: domain.ProductionCostRef{ID: "ic_x", Name: "Socks"}, Productive: batches("2", 1)},
		}, nil
	}).Times(1)
	analytics.EXPECT().GetProductionCostSteps(gomock.Any(), "ac_1", []string{"ps_1"}).Return(map[string]domain.ProductionCostStep{
		"ps_1": costStep(costStepSpec{produce: "2", unit: countEach, laborRate: "0", overheadRate: "0", consumptions: []domain.CostFlowConsumption{perEachMaterial("2", "1.5")}}),
	}, nil).Times(1)
	analytics.EXPECT().GetBaseUnitIDsByDimension(gomock.Any()).Return(productionCostBaseUnits, nil).Times(1)
	units.EXPECT().GetByIDs(gomock.Any(), "ac_1", gomock.InAnyOrder([]string{"dollar", "hour", "each"})).
		Return([]*domain.Unit{{ID: "dollar"}, {ID: "hour"}, {ID: "each"}}, nil).Times(1)

	reader := salesCtx("ac_1", string(constants.RoleTypeCustom), map[string]bool{"batches:read": true})
	report, apiErr := svc.AnalyzeProductionCosts(reader, params)
	require.Nil(t, apiErr)
	require.Len(t, seen, 1)
	require.Equal(t, "ac_1", seen[0].AccountID, "the report reads the caller's account, whatever the params say")
	requireCostEqual(t, "total", report.Totals.Total.Materials, "9")
	require.Len(t, report.Departments, 2)
	require.Contains(t, report.Units, "dollar")
	require.Contains(t, report.Units, "each")

	_, apiErr = svc.AnalyzeProductionCosts(reader, params)
	require.Nil(t, apiErr, "the same window and filters are served from the cache")

	_, apiErr = svc.AnalyzeProductionCosts(salesCtx("ac_1", string(constants.RoleTypeCustom), map[string]bool{"items:read": true}), params)
	require.NotNil(t, apiErr)
	require.Equal(t, 403, apierror.GetHTTPStatusCode(apiErr.Code))

	backwards := params
	backwards.StartDate, backwards.EndDate = params.EndDate, params.StartDate
	_, apiErr = svc.AnalyzeProductionCosts(reader, backwards)
	require.NotNil(t, apiErr)
	require.Equal(t, "ends_at", apiErr.Param)

	_, apiErr = svc.AnalyzeProductionCosts(reader, domain.AnalyzeProductionCostsParams{EndDate: params.EndDate})
	require.NotNil(t, apiErr)
	require.Equal(t, "starts_at", apiErr.Param)
}
