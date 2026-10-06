package service

import (
	"context"
	"sort"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// productionCostDivisionScale is the decimal places a production cost report divides to: the scale every quantity and rate is stored at.
const productionCostDivisionScale = 30

// AnalyzeProductionCosts costs the batches scanned at production steps over a window, as the dashboard's production costs report did, with each step's labor costed in its labor time's own units.
func (s *analyticsSvcImpl) AnalyzeProductionCosts(ctx context.Context, params domain.AnalyzeProductionCostsParams) (*domain.ProductionCostReport, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_production_costs")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCosts, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	switch {
	case params.StartDate.IsZero():
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Required.", "starts_at"))
	case params.EndDate.IsZero():
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Required.", "ends_at"))
	case params.EndDate.Before(params.StartDate):
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Must not be before starts_at.", "ends_at"))
	}

	params.AccountID = identity.Target.AccountID

	report, apiErr := cachedReport(ctx, s.reportCache().productionCost, analyticsReport{
		accountID: params.AccountID,
		family:    analyticsFamilyProduction,
		method:    "production_costs",
		params:    params,
		ttl:       s.reportCache().ttlForWindow(params.EndDate),
	}, func(ctx context.Context) (*domain.ProductionCostReport, *apierror.APIError) {
		return s.buildProductionCostReport(ctx, params)
	})
	return report, tracing.Trace(span, apiErr)
}

func (s *analyticsSvcImpl) buildProductionCostReport(ctx context.Context, params domain.AnalyzeProductionCostsParams) (*domain.ProductionCostReport, *apierror.APIError) {
	repo := s.reports().NewAnalyticsRepo()
	rows, apiErr := repo.GetProductionCostRows(ctx, params)
	if apiErr != nil {
		return nil, apiErr
	}
	stepIDs := make([]string, len(rows))
	for i, row := range rows {
		stepIDs[i] = row.ProductionStepID
	}
	steps, apiErr := repo.GetProductionCostSteps(ctx, params.AccountID, dedupeStrings(stepIDs))
	if apiErr != nil {
		return nil, apiErr
	}
	baseUnits, apiErr := repo.GetBaseUnitIDsByDimension(ctx)
	if apiErr != nil {
		return nil, apiErr
	}

	report, apiErr := assembleProductionCostReport(rows, steps, baseUnits)
	if apiErr != nil {
		return nil, apiErr
	}
	unitIDs := []string{report.CurrencyUnitID, report.TimeUnitID}
	for _, p := range report.Totals.Total.Produced {
		unitIDs = append(unitIDs, p.UnitID)
	}
	if report.Units, apiErr = s.unitsByID(ctx, params.AccountID, unitIDs); apiErr != nil {
		return nil, apiErr
	}
	return report, nil
}

// assembleProductionCostReport costs each row's batches against its step and rolls the rows up by department and category. Rows whose step is not the account's, or has no production, are left out, as the dashboard left them out.
//
// A step is costed per run (calculateStepCost) and each kind of output is charged the runs its batches amount to: their quantity in the production's unit over the quantity one run produces.
func assembleProductionCostReport(rows []domain.ProductionCostRow, steps map[string]domain.ProductionCostStep, baseUnits map[string]string) (*domain.ProductionCostReport, *apierror.APIError) {
	currencyUnitID, timeUnitID := baseUnits[string(constants.UnitTypeCurrency)], baseUnits[string(constants.UnitTypeTime)]
	if currencyUnitID == "" || timeUnitID == "" {
		return nil, apierror.NewInvariantViolationError("No base unit is defined for currency or time.")
	}

	type groupKey struct{ department, category string }
	byDepartmentCategory := map[groupKey]*productionCostGroupSum{}
	for _, row := range rows {
		step, ok := steps[row.ProductionStepID]
		if !ok {
			continue
		}
		producedUnitID := baseUnits[step.Step.Production.Quantity.Unit.Type]
		if producedUnitID == "" {
			return nil, apierror.NewInvariantViolationError("No base unit is defined for the " + step.Step.Production.Quantity.Unit.Type + " dimension.")
		}
		perRun := calculateStepCost(&step.Step, step.Consumptions)

		key := groupKey{category: row.Category.ID}
		if row.Department != nil {
			key.department = row.Department.ID
		}
		group, ok := byDepartmentCategory[key]
		if !ok {
			category := row.Category
			group = newProductionCostGroupSum(row.Department, &category)
			byDepartmentCategory[key] = group
		}
		group.productive.add(chargeRuns(perRun, step.Step.Production.Quantity, row.Productive, producedUnitID))
		group.seconds.add(chargeRuns(perRun, step.Step.Production.Quantity, row.Seconds, producedUnitID))
		group.waste.add(chargeRuns(perRun, step.Step.Production.Quantity, row.Waste, producedUnitID))
	}

	totals := newProductionCostGroupSum(nil, nil)
	byDepartment := map[string]*productionCostGroupSum{}
	byCategory := map[string]*productionCostGroupSum{}
	departmentCategories := make([]*productionCostGroupSum, 0, len(byDepartmentCategory))
	for key, group := range byDepartmentCategory {
		departmentCategories = append(departmentCategories, group)
		totals.addGroup(group)
		if _, ok := byDepartment[key.department]; !ok {
			byDepartment[key.department] = newProductionCostGroupSum(group.department, nil)
		}
		byDepartment[key.department].addGroup(group)
		if _, ok := byCategory[key.category]; !ok {
			byCategory[key.category] = newProductionCostGroupSum(nil, group.category)
		}
		byCategory[key.category].addGroup(group)
	}

	return &domain.ProductionCostReport{
		CurrencyUnitID:       currencyUnitID,
		TimeUnitID:           timeUnitID,
		Totals:               totals.set(),
		Departments:          sortedProductionCostGroups(mapValues(byDepartment)),
		Categories:           sortedProductionCostGroups(mapValues(byCategory)),
		DepartmentCategories: sortedProductionCostGroups(departmentCategories),
	}, nil
}

// chargeRuns is what a kind of output costs: the per-run cost times the runs its batches amount to, and what they produced in base units.
//
// The runs are the batches' quantity carried into the production's unit, over the quantity one run produces. A production with no quantity cannot be costed per run, so its batches are charged nothing but still count as produced.
func chargeRuns(perRun *itemStepCost, production domain.BatchQuantity, batches domain.ProductionCostBatches, producedUnitID string) productionCostSum {
	ratio := unitRatioExact(production.Unit.RatioNumerator, production.Unit.RatioDenominator)
	offset := unitRatioExact(production.Unit.OffsetNumerator, production.Unit.OffsetDenominator)
	inProductionBase := batches.BaseQuantity.Sub(offset.Mul(decimal.NewFromInt(batches.Count)))

	runs := decimal.Zero
	if perRunQuantity := ratio.Mul(production.Measure); !perRunQuantity.IsZero() {
		runs = inProductionBase.DivRound(perRunQuantity, productionCostDivisionScale)
	}
	return productionCostSum{
		materials:  perRun.material.Mul(runs),
		labor:      perRun.labor.Mul(runs),
		overhead:   perRun.overhead.Mul(runs),
		total:      perRun.total.Mul(runs),
		laborHours: perRun.laborHours.Mul(runs),
		// The dashboard reported the production's quantity per run times the runs: the batches' own base quantity whenever the production's unit has no offset.
		produced: map[string]decimal.Decimal{producedUnitID: inProductionBase.Add(offset.Mul(runs))},
	}
}

// unitRatioExact is numerator over denominator, or zero when either is missing or the denominator is zero.
func unitRatioExact(numerator, denominator string) decimal.Decimal {
	num, numErr := decimal.NewFromString(numerator)
	den, denErr := decimal.NewFromString(denominator)
	if numErr != nil || denErr != nil || den.IsZero() {
		return decimal.Zero
	}
	return num.DivRound(den, productionCostDivisionScale)
}

// productionCostSum accumulates a ProductionCost; produced is keyed by unit id.
type productionCostSum struct {
	materials, labor, overhead, total, laborHours decimal.Decimal
	produced                                      map[string]decimal.Decimal
}

func (a *productionCostSum) add(b productionCostSum) {
	a.materials = a.materials.Add(b.materials)
	a.labor = a.labor.Add(b.labor)
	a.overhead = a.overhead.Add(b.overhead)
	a.total = a.total.Add(b.total)
	a.laborHours = a.laborHours.Add(b.laborHours)
	if a.produced == nil {
		a.produced = map[string]decimal.Decimal{}
	}
	for unitID, v := range b.produced {
		a.produced[unitID] = a.produced[unitID].Add(v)
	}
}

func (a productionCostSum) cost() domain.ProductionCost {
	produced := make([]domain.ProducedQuantity, 0, len(a.produced))
	for unitID, v := range a.produced {
		produced = append(produced, domain.ProducedQuantity{UnitID: unitID, Value: v})
	}
	sort.Slice(produced, func(i, j int) bool { return produced[i].UnitID < produced[j].UnitID })
	return domain.ProductionCost{
		Materials:  a.materials,
		Labor:      a.labor,
		Overhead:   a.overhead,
		Total:      a.total,
		LaborHours: a.laborHours,
		Produced:   produced,
	}
}

type productionCostGroupSum struct {
	department, category       *domain.ProductionCostRef
	productive, seconds, waste productionCostSum
}

func newProductionCostGroupSum(department, category *domain.ProductionCostRef) *productionCostGroupSum {
	return &productionCostGroupSum{department: department, category: category}
}

func (g *productionCostGroupSum) addGroup(o *productionCostGroupSum) {
	g.productive.add(o.productive)
	g.seconds.add(o.seconds)
	g.waste.add(o.waste)
}

func (g *productionCostGroupSum) set() domain.ProductionCostSet {
	var total productionCostSum
	total.add(g.productive)
	total.add(g.seconds)
	total.add(g.waste)
	return domain.ProductionCostSet{
		Productive: g.productive.cost(),
		Seconds:    g.seconds.cost(),
		Waste:      g.waste.cost(),
		Total:      total.cost(),
	}
}

func mapValues[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// sortedProductionCostGroups orders groups by department then category name, batches with no department last.
func sortedProductionCostGroups(groups []*productionCostGroupSum) []domain.ProductionCostGroup {
	sort.Slice(groups, func(i, j int) bool {
		if c := compareProductionCostRefs(groups[i].department, groups[j].department); c != 0 {
			return c < 0
		}
		return compareProductionCostRefs(groups[i].category, groups[j].category) < 0
	})
	out := make([]domain.ProductionCostGroup, len(groups))
	for i, g := range groups {
		out[i] = domain.ProductionCostGroup{Department: g.department, Category: g.category, Costs: g.set()}
	}
	return out
}

func compareProductionCostRefs(a, b *domain.ProductionCostRef) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	case a.Name != b.Name:
		if a.Name < b.Name {
			return -1
		}
		return 1
	case a.ID < b.ID:
		return -1
	case a.ID > b.ID:
		return 1
	}
	return 0
}
