package service

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

var productionScheduleSvcTracer = tracing.GetTracer("core-service.production_schedule_service")

type productionScheduleSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	enqueuer        domain.ProductionScheduleEnqueuer
}

type ProductionScheduleSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps the multi-table schedule write.
	TxManager TransactionManager

	// Enqueuer (optional; default: nil) queues cadence-driven generations. Nil disables the cadence, which is how a deployment without messaging still serves every read path.
	Enqueuer domain.ProductionScheduleEnqueuer
}

func (c *ProductionScheduleSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("production schedule service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("production schedule service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("production schedule service: tx manager is required")
	}
	return nil
}

func NewProductionScheduleSvc(config *ProductionScheduleSvcConfig) domain.ProductionScheduleSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &productionScheduleSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		enqueuer:        config.Enqueuer,
	}
}

func (s *productionScheduleSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productionScheduleSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productionScheduleSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		return fn(txCtx, &productionScheduleSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
			enqueuer:        s.enqueuer,
		})
	})
}

// PreviewProductionSchedule runs the solver and returns the plan without persisting it.
//
// This is the parity and inspection surface: it is the same code path a generated schedule will take, minus the write, so a plan can be diffed against the TS script before anything depends on it.
func (s *productionScheduleSvcImpl) PreviewProductionSchedule(
	ctx context.Context,
	params domain.PreviewProductionScheduleParams,
) (*scheduling.SolverOutput, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.preview")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	effective, apiErr := s.loadEffectiveSettings(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Request values are a preview override on top of the merchant's settings, so a planner can try "what if the horizon were 26 weeks" without saving anything.
	if params.HorizonWeeks > 0 {
		effective.Settings.HorizonWeeks = params.HorizonWeeks
	}
	if params.DemandBasis != "" {
		effective.DemandBasisCode = params.DemandBasis
	}

	planningAsOf := params.PlanningAsOf
	if planningAsOf.IsZero() {
		planningAsOf = time.Now().UTC()
	}

	input, apiErr := s.loadSolverInput(ctx, domain.LoadSolverInputParams{
		AccountID:                accountID,
		PlanningAsOf:             planningAsOf,
		Settings:                 effective.Settings,
		DemandWindowMonths:       effective.DemandWindowMonths,
		ForecastHistoryMonths:    effective.ForecastHistoryMonths,
		ForecastMonths:           effective.ForecastMonths,
		DemandBasisCode:          effective.DemandBasisCode,
		ForecastZ:                effective.ForecastZ,
		ConstraintDepartmentID:   effective.ConstraintDepartmentID,
		ItemSettings:             effective.ItemSettings,
		DefaultFulfillmentPolicy: effective.DefaultFulfillmentPolicy,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	output := scheduling.Solve(*input)
	return &output, nil
}

// solveFor runs the solver for an account, returning the plan alongside the settings it used. Preview and Generate both go through here so a preview can never disagree with what generation would persist.
func (s *productionScheduleSvcImpl) solveFor(
	ctx context.Context,
	accountID string,
	planningAsOf time.Time,
	horizonWeeks int,
	demandBasis string,
	pinnedCampaigns []scheduling.PinnedCampaign,
) (*scheduling.SolverOutput, *domain.EffectiveScheduleSettings, *apierror.APIError) {
	output, _, effective, apiErr := s.solveForWithInput(ctx, accountID, planningAsOf, horizonWeeks, demandBasis, pinnedCampaigns)
	return output, effective, apiErr
}

// solveForWithInput is solveFor, additionally handing back the assembled solver input so a caller that needs a second solve of the same plant can re-run the solver without paying for the input loads again. The regenerate preview uses it to solve the replace_all counterfactual.
func (s *productionScheduleSvcImpl) solveForWithInput(
	ctx context.Context,
	accountID string,
	planningAsOf time.Time,
	horizonWeeks int,
	demandBasis string,
	pinnedCampaigns []scheduling.PinnedCampaign,
) (*scheduling.SolverOutput, *scheduling.SolverInput, *domain.EffectiveScheduleSettings, *apierror.APIError) {
	effective, apiErr := s.loadEffectiveSettings(ctx, accountID)
	if apiErr != nil {
		return nil, nil, nil, apiErr
	}
	if horizonWeeks > 0 {
		effective.Settings.HorizonWeeks = horizonWeeks
	}
	if demandBasis != "" {
		effective.DemandBasisCode = demandBasis
	}

	input, apiErr := s.loadSolverInput(ctx, domain.LoadSolverInputParams{
		AccountID:                accountID,
		PlanningAsOf:             planningAsOf,
		Settings:                 effective.Settings,
		DemandWindowMonths:       effective.DemandWindowMonths,
		ForecastHistoryMonths:    effective.ForecastHistoryMonths,
		ForecastMonths:           effective.ForecastMonths,
		DemandBasisCode:          effective.DemandBasisCode,
		ForecastZ:                effective.ForecastZ,
		ConstraintDepartmentID:   effective.ConstraintDepartmentID,
		ItemSettings:             effective.ItemSettings,
		DefaultFulfillmentPolicy: effective.DefaultFulfillmentPolicy,
		PinnedCampaigns:          pinnedCampaigns,
	})
	if apiErr != nil {
		return nil, nil, nil, apiErr
	}

	output := scheduling.Solve(*input)
	return &output, input, effective, nil
}

// loadEffectiveSettings returns the account's planning assumptions, falling back to code defaults for an account that has never configured scheduling. Callers always get usable settings rather than having to handle "not configured".
//
//  1. Starts from the code defaults.
//  2. Reads the account's stored settings row; absent means never configured, and the defaults stand.
//  3. Overlays every stored assumption onto the defaults.
//  4. Loads the per-item overrides.
func (s *productionScheduleSvcImpl) loadEffectiveSettings(
	ctx context.Context,
	accountID string,
) (*domain.EffectiveScheduleSettings, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.load_effective_settings")
	defer span.End()

	effective := &domain.EffectiveScheduleSettings{
		Settings:                    scheduling.DefaultSettings(),
		DemandWindowMonths:          12,
		ForecastHistoryMonths:       24,
		ForecastMonths:              12,
		DemandBasisCode:             scheduling.DemandBasisTrailing12,
		ForecastZ:                   1,
		ItemSettings:                map[string]domain.ProductionScheduleItemSetting{},
		DefaultFulfillmentPolicy:    scheduling.PolicyMakeToStock,
		RecommendationThresholds:    scheduling.DefaultRecommendationThresholds(),
		DefaultCustomerLeadTimeDays: scheduling.DefaultSettings().DefaultCustomerLeadTimeDays,
	}

	// Hold a physical greige buffer at the constraint. Off in DefaultSettings so the parity gate reproduces the script, which pools the buffer into finished goods; on for every real solve, because a plant that keeps no undifferentiated greige cannot rebalance its finished mix when demand shifts to a colorway it is not already sitting on.
	effective.Settings.GreigeBufferEnabled = true

	repo := s.repos.NewProductionScheduleInputRepo()

	row, apiErr := repo.GetAccountScheduleSettings(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if row == nil {
		return effective, nil
	}

	effective.Settings.HorizonWeeks = row.PlanningHorizonWeeks
	effective.Settings.FrozenWeeks = row.FrozenWeeks
	effective.Settings.WeekStartDay = row.WeekStartDay
	effective.Settings.ShiftsPerDay = row.ShiftsPerDay
	effective.Settings.HoursPerShift = row.HoursPerShift
	effective.Settings.WorkDaysPerWeek = row.WorkDaysPerWeek
	effective.Settings.WeeksPerYear = row.WeeksPerYear
	effective.Settings.CapacityHeadroomPct = row.CapacityHeadroomPct
	effective.Settings.DefaultLotUnits = row.DefaultLotUnits
	effective.Settings.DefaultCustomerLeadTimeDays = int(row.DefaultCustomerLeadTimeDays)
	effective.Settings.ChangeoverAvgMinutes = row.ChangeoverAvgMinutes
	effective.Settings.ChangeoverMinMinutes = row.ChangeoverMinMinutes
	effective.Settings.ChangeoverMaxMinutes = row.ChangeoverMaxMinutes
	effective.Settings.ChangeoverLaborRate = row.ChangeoverLaborRate
	effective.Settings.HoldingRatePct = row.HoldingRatePct
	effective.Settings.ServiceLevelZ = row.ServiceLevelZ
	effective.Settings.FinishLeadTimeWeeks = row.FinishLeadTimeWeeks
	effective.Settings.DefaultConstraintLeadTimeWeeks = row.DefaultConstraintLeadTimeWeeks
	effective.Settings.MaxWeeksSupply = row.MaxWeeksSupply
	effective.Settings.MaxFlowDepth = row.MaxFlowDepth

	effective.DemandWindowMonths = row.DemandWindowMonths
	effective.ForecastHistoryMonths = row.ForecastHistoryMonths
	effective.ForecastMonths = row.ForecastMonths
	effective.DemandBasisCode = row.DemandBasisCode
	effective.ForecastZ = row.ForecastZ
	effective.ConstraintDepartmentID = row.ConstraintDepartmentID
	if row.DefaultFulfillmentPolicyCode != "" {
		effective.DefaultFulfillmentPolicy = row.DefaultFulfillmentPolicyCode
	}
	effective.RecommendationThresholds = row.RecommendationThresholds
	effective.DefaultCustomerLeadTimeDays = row.DefaultCustomerLeadTimeDays

	// The constraint department's own labor rate prices its changeovers; the account-wide setting above is only the fallback for departments that have none. Rates are conventionally $/hr, matching how the setting is expressed.
	if effective.ConstraintDepartmentID != "" {
		departmentRate, apiErr := repo.GetConstraintDepartmentLaborRate(ctx, accountID, effective.ConstraintDepartmentID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if departmentRate != nil && *departmentRate > 0 {
			effective.Settings.ChangeoverLaborRate = *departmentRate
		}
	}

	itemRows, apiErr := repo.ListScheduleItemSettings(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, item := range itemRows {
		effective.ItemSettings[item.ItemID] = item
	}

	return effective, nil
}

// loadSolverInput assembles everything the solver needs in one pass.
//
// The solver is pure, so every read happens here, through the input repository's thin queries. Ordering is normalized on the way out: anything the solver iterates must be sorted, or the plan changes between runs.
func (s *productionScheduleSvcImpl) loadSolverInput(
	ctx context.Context,
	params domain.LoadSolverInputParams,
) (*scheduling.SolverInput, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.load_solver_input")
	defer span.End()

	repo := s.repos.NewProductionScheduleInputRepo()

	in := &scheduling.SolverInput{
		AccountID:    params.AccountID,
		PlanningAsOf: params.PlanningAsOf,
		// The order book is dated against the same week boundary the plan is written to, so a requirement's due week and a campaign's week index mean the same thing.
		HorizonStart:       scheduleWeekStart(params.PlanningAsOf, params.Settings.WeekStartDay),
		Settings:           params.Settings,
		DemandBasisCode:    params.DemandBasisCode,
		ForecastZ:          params.ForecastZ,
		ForecastMonths:     params.ForecastMonths,
		StepInputs:         map[string]map[string]bool{},
		MonthlyByItem:      map[string][]scheduling.MonthlyDemand{},
		DownstreamByItem:   map[string][]scheduling.FinishedGood{},
		OnHandByItem:       map[string]float64{},
		GreigeOnHandByItem: map[string]float64{},
		ItemsByProductLine: map[string][]string{},
		ItemLotUnits:       map[string]float64{},
		LotDefaultByItem:   map[string]scheduling.LotDefault{},
		ExcludedItemIDs:    map[string]bool{},
		PinnedCampaigns:    params.PinnedCampaigns,

		ItemPolicyOverrides:      map[string]string{},
		ProductLineByItem:        map[string]string{},
		PolicyByProductLine:      map[string]string{},
		DefaultFulfillmentPolicy: params.DefaultFulfillmentPolicy,
	}

	// 1. The machines that constitute the constraint. The constraint is a department, not a list of machines: the knitting room sets the pace of the factory, and every machine in it is planned without anyone having to remember to opt one in.
	if params.ConstraintDepartmentID == "" {
		return nil, tracing.Trace(span, apierror.NewValidationError(
			"No constraint department is configured. Choose the department that sets the pace of the factory in production schedule settings."))
	}

	machines, apiErr := repo.GetConstraintMachines(ctx, params.AccountID, params.ConstraintDepartmentID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	machineIDs := make([]string, 0, len(machines))
	for _, m := range machines {
		in.Machines = append(in.Machines, m)
		machineIDs = append(machineIDs, m.ID)
	}
	if len(machineIDs) == 0 {
		// A department with no planned machines is a configuration problem, not an empty result, so it is named rather than returned as a blank plan.
		return nil, tracing.Trace(span, apierror.NewValidationError(
			"The constraint department has no machines to plan. Add machines to it, or clear the exclusions on the ones it has."))
	}

	// A campaign explodes downstream through its machine's production step, so machines without one derive no department work. That is worth reporting rather than silently dropping half the floor's work list.
	machinesWithoutStep, apiErr := repo.CountConstraintMachinesWithoutStep(ctx, params.AccountID, params.ConstraintDepartmentID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	in.MachinesWithoutStep = machinesWithoutStep

	// 2. Batch history over the demand window: run rates, costs, affinity, lead times.
	windowStart := params.PlanningAsOf.AddDate(0, -params.DemandWindowMonths, 0)
	batchRows, apiErr := repo.GetConstraintBatchMeasurements(ctx, domain.GetConstraintBatchMeasurementsParams{
		AccountID:              params.AccountID,
		WindowStart:            windowStart,
		WindowEnd:              params.PlanningAsOf,
		MachineIDs:             machineIDs,
		ConstraintDepartmentID: params.ConstraintDepartmentID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	constraintItemIDs := map[string]bool{}
	stepIDs := map[string]bool{}
	// The unit each item is actually scanned in, which is what the account-wide lot fallback counts in when no product line supplies one.
	unitByItem := map[string]string{}
	// The scan unit's ratio to its base unit. Every quantity the solver sees is expressed in the item's scan unit: run rates, lot sizes and costs are all denominated in it (a 60-unit doff is 60 pairs, labor time is per pair), so leaving demand or stock in base units would double every pair-counted number.
	nativeRatioByItem := map[string]float64{}
	for _, row := range batchRows {
		if row.QuantityUnitID != nil && *row.QuantityUnitID != "" {
			unitByItem[row.Measurement.ItemID] = *row.QuantityUnitID
			if row.QuantityUnitRatio > 0 {
				nativeRatioByItem[row.Measurement.ItemID] = row.QuantityUnitRatio
			}
		}
		if row.ProductionStepID != nil {
			stepIDs[*row.ProductionStepID] = true
		}
		in.Batches = append(in.Batches, row.Measurement)
		constraintItemIDs[row.Measurement.ItemID] = true
	}
	nativeRatioOf := func(itemID string) float64 {
		if ratio, ok := nativeRatioByItem[itemID]; ok && ratio > 0 {
			return ratio
		}
		return 1
	}
	// Batch quantities arrive normalized to base units; bring each back into its item's scan unit so mixed-unit scan history still sums coherently.
	for i := range in.Batches {
		in.Batches[i].Quantity /= nativeRatioOf(in.Batches[i].ItemID)
	}

	if len(constraintItemIDs) == 0 {
		return in, nil // nothing produced in the window; an empty plan is the honest answer
	}

	// 3. Input BOM per step, for the changeover model.
	if len(stepIDs) > 0 {
		consumptionRows, apiErr := repo.GetStepConsumptionItems(ctx, scheduleSortedKeys(stepIDs))
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range consumptionRows {
			if in.StepInputs[row.ProductionStepID] == nil {
				in.StepInputs[row.ProductionStepID] = map[string]bool{}
			}
			in.StepInputs[row.ProductionStepID][row.ItemID] = true
		}
	}

	itemIDs := scheduleSortedKeys(constraintItemIDs)

	// 4. Walk the batch genealogy forward to the finished goods each item becomes.
	descendantItemsByItem, apiErr := s.walkDescendants(ctx, repo, params.AccountID, itemIDs, windowStart, params.PlanningAsOf, params.Settings.MaxFlowDepthOrDefault())
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 5. Echelon on hand: the constraint item plus everything downstream of it. The buffer is pooled at the constraint, so stock held as finished goods still counts against the decision to build more.
	allInventoryItems := map[string]bool{}
	for _, id := range itemIDs {
		allInventoryItems[id] = true
	}
	for _, descendants := range descendantItemsByItem {
		for _, id := range descendants {
			allInventoryItems[id] = true
		}
	}
	onHandByItem, apiErr := repo.GetEchelonOnHand(ctx, params.AccountID, scheduleSortedKeys(allInventoryItems))
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, itemID := range itemIDs {
		// The greige stage on its own is kept as well as rolled up. Once the echelon total is summed it cannot be decomposed back, and "how much greige is there" is exactly the question the pooled figure hides.
		// Stock arrives in base units; the whole echelon is expressed in the constraint item's scan unit (downstream eaches count as halves of a pair, not as whole pairs).
		greige := onHandByItem[itemID] / nativeRatioOf(itemID)
		total := greige
		for _, descendantID := range descendantItemsByItem[itemID] {
			total += onHandByItem[descendantID] / nativeRatioOf(itemID)
		}
		in.GreigeOnHandByItem[itemID] = greige
		in.OnHandByItem[itemID] = total
	}

	// 6. Pool finished-goods order demand back onto the constraint item.
	if apiErr := s.loadDemand(ctx, repo, params, itemIDs, descendantItemsByItem, onHandByItem, nativeRatioOf, in); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 7. Overrides in force at the planning date.
	overrides, apiErr := repo.GetActiveDemandOverrides(ctx, params.AccountID, params.PlanningAsOf)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	productLineIDs := map[string]bool{}
	for _, override := range overrides {
		in.Overrides = append(in.Overrides, override)

		if override.ScopeCode == scheduling.OverrideScopeProductLine {
			productLineIDs[override.ScopeRefID] = true
		}
	}

	// 8. Resolve product-line overrides onto items.
	if len(productLineIDs) > 0 {
		lineRows, apiErr := repo.GetItemsForProductLines(ctx, params.AccountID, scheduleSortedKeys(productLineIDs))
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range lineRows {
			in.ItemsByProductLine[row.ProductLineID] = append(in.ItemsByProductLine[row.ProductLineID], row.ItemID)
		}
	}

	// 9. Per-item settings overrides.
	for itemID, setting := range params.ItemSettings {
		if setting.IsExcluded {
			in.ExcludedItemIDs[itemID] = true
		}
		if setting.LotMultipleUnits > 0 {
			in.ItemLotUnits[itemID] = setting.LotMultipleUnits
		}
	}

	// 10. Stage two: the rest of the factory, which decides how many of which finished good to make from what stage one knits.
	if apiErr := s.loadFinishingInput(ctx, repo, params, in, nativeRatioOf); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// 11. What lot each planned item is made in, resolved once through the whole chain.
	//
	// This runs last because it depends on the downstream map built in step 8: greige is not sold and has no product line, so it takes its lot from the finished goods it becomes.
	lotInput, apiErr := s.loadLotResolutionInput(ctx, repo, params.AccountID, in, unitByItem)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for itemID := range constraintItemIDs {
		if lot, ok := scheduling.ResolveLotDefault(itemID, lotInput); ok {
			in.LotDefaultByItem[itemID] = lot
		}
	}

	return in, nil
}

// loadFinishingInput loads everything stage two needs: the machines outside the constraint, the production history that gives them run rates, and where each finished good is made.
//
// Everything here is best-effort. A plant that has never scanned a finishing step still gets a knit plan; it gets a finishing plan with every SKU reported as unrateable, which is a diagnosis rather than a failure. Refusing to solve at all because the second stage is unmeasured would take away the plan that does work.
func (s *productionScheduleSvcImpl) loadFinishingInput(
	ctx context.Context,
	repo domain.ProductionScheduleInputRepo,
	params domain.LoadSolverInputParams,
	in *scheduling.SolverInput,
	nativeRatioOf func(itemID string) float64,
) *apierror.APIError {
	in.FinishingRateScaleByItem = map[string]float64{}
	in.FinishingStepByItem = map[string]scheduling.FinishingStep{}
	in.FinishingLotByItem = map[string]scheduling.LotDefault{}

	machines, apiErr := repo.GetFinishingMachines(ctx, params.AccountID, params.ConstraintDepartmentID)
	if apiErr != nil {
		return apiErr
	}
	in.FinishingMachines = machines

	// Which greige each finished good comes from, so its rate can be expressed in the unit the plan is denominated in.
	greigeByFinished := map[string]string{}
	finishedItemIDs := map[string]bool{}
	for greigeItemID, finishedGoods := range in.DownstreamByItem {
		for _, finished := range finishedGoods {
			if finished.ItemID == "" {
				continue
			}
			finishedItemIDs[finished.ItemID] = true
			if _, taken := greigeByFinished[finished.ItemID]; !taken {
				// First wins, matching how the genealogy walk attributes a descendant reachable from two constraint items.
				greigeByFinished[finished.ItemID] = greigeItemID
			}
		}
	}
	if len(finishedItemIDs) == 0 {
		return nil
	}

	windowStart := params.PlanningAsOf.AddDate(0, -params.DemandWindowMonths, 0)
	batchRows, apiErr := repo.GetFinishingBatchMeasurements(ctx, domain.GetFinishingBatchMeasurementsParams{
		AccountID:              params.AccountID,
		WindowStart:            windowStart,
		WindowEnd:              params.PlanningAsOf,
		ItemIDs:                scheduleSortedKeys(finishedItemIDs),
		ConstraintDepartmentID: params.ConstraintDepartmentID,
	})
	if apiErr != nil {
		return apiErr
	}

	finishedRatio := map[string]float64{}
	finishedUnitID := map[string]string{}
	for _, row := range batchRows {
		itemID := row.Measurement.ItemID
		if row.QuantityUnitRatio > 0 {
			finishedRatio[itemID] = row.QuantityUnitRatio
		}
		if row.QuantityUnitID != nil && *row.QuantityUnitID != "" {
			finishedUnitID[itemID] = *row.QuantityUnitID
		}
		// The last step a finished good was scanned at is where it is made. Rows arrive ordered by scan time, so this settles on the most recent one, which is the room to put the work in front of today.
		if row.ProductionStepID != nil && *row.ProductionStepID != "" {
			step := scheduling.FinishingStep{ProductionStepID: *row.ProductionStepID}
			if row.StepDepartmentID != nil {
				step.DepartmentID = *row.StepDepartmentID
			}
			in.FinishingStepByItem[itemID] = step
		}
		in.FinishingBatches = append(in.FinishingBatches, row.Measurement)
	}

	// Batch quantities arrive normalized to base units; bring each back into its own scan unit, the same correction the constraint history gets.
	for i := range in.FinishingBatches {
		if ratio, ok := finishedRatio[in.FinishingBatches[i].ItemID]; ok && ratio > 0 {
			in.FinishingBatches[i].Quantity /= ratio
		}
	}

	// The plan is denominated in the greige's scan unit. A finished good measured in its own unit has to be rescaled, or a sock scanned in eaches and knitted in pairs is costed at half the hours it takes.
	for itemID := range finishedItemIDs {
		greigeItemID, ok := greigeByFinished[itemID]
		if !ok {
			continue
		}
		fgRatio, ok := finishedRatio[itemID]
		if !ok || fgRatio <= 0 {
			continue
		}
		if scale := nativeRatioOf(greigeItemID) / fgRatio; scale > 0 {
			in.FinishingRateScaleByItem[itemID] = scale
		}
	}

	// The lot a finished good is made in, through the same chain the constraint items use. A finished good is sellable, so its own product line answers in one step where greige needs the downstream walk.
	lotInput, apiErr := s.loadLotResolutionInput(ctx, repo, params.AccountID, in, finishedUnitID)
	if apiErr != nil {
		return apiErr
	}
	for itemID := range finishedItemIDs {
		if lot, ok := scheduling.ResolveLotDefault(itemID, lotInput); ok {
			in.FinishingLotByItem[itemID] = lot
		}
	}

	return nil
}

// loadLotResolutionInput gathers the product-line lot conventions and the item-to-line mapping the resolution chain needs.
func (s *productionScheduleSvcImpl) loadLotResolutionInput(
	ctx context.Context,
	repo domain.ProductionScheduleInputRepo,
	accountID string,
	in *scheduling.SolverInput,
	unitByItem map[string]string,
) (scheduling.LotResolutionInput, *apierror.APIError) {
	lotInput := scheduling.LotResolutionInput{
		ItemOverrides:     in.ItemLotUnits,
		ProductLineByItem: map[string]string{},
		LotByProductLine:  map[string]scheduling.ProductLineLot{},
		DownstreamByItem:  in.DownstreamByItem,
		AccountLotUnits:   in.Settings.DefaultLotUnits,
		UnitByItem:        unitByItem,
	}

	lines, apiErr := repo.ListProductLineLotDefaults(ctx, accountID)
	if apiErr != nil {
		return lotInput, apiErr
	}
	for _, line := range lines {
		if line.Quantity <= 0 {
			continue
		}
		lotInput.LotByProductLine[line.ProductLineID] = line
	}

	// A constraint item can itself be sellable, so its own line is checked before the downstream inheritance that greige depends on.
	itemIDs := make([]string, 0, len(in.OnHandByItem))
	for itemID := range in.OnHandByItem {
		itemIDs = append(itemIDs, itemID)
	}
	for _, downstream := range in.DownstreamByItem {
		for _, finished := range downstream {
			itemIDs = append(itemIDs, finished.ItemID)
		}
	}
	if len(itemIDs) == 0 {
		return lotInput, nil
	}
	sort.Strings(itemIDs)

	rows, apiErr := repo.ListItemProductLines(ctx, accountID, itemIDs)
	if apiErr != nil {
		return lotInput, apiErr
	}
	for _, row := range rows {
		lotInput.ProductLineByItem[row.ItemID] = row.ProductLineID
	}

	return lotInput, nil
}

// walkDescendants follows the batch genealogy forward, one level at a time with the whole frontier batched into each query. Returns the finished-good item ids reachable from each constraint item.
//
// Attribution is first-wins when a descendant is reachable from more than one constraint item, and the walk starts from SKU-sorted items so that choice is stable. The script left this to JavaScript map ordering, which is why the same input could produce different echelon stock between runs.
func (s *productionScheduleSvcImpl) walkDescendants(
	ctx context.Context,
	repo domain.ProductionScheduleInputRepo,
	accountID string,
	itemIDs []string,
	windowStart, windowEnd time.Time,
	maxDepth int,
) (map[string][]string, *apierror.APIError) {
	// Every scan in the demand window seeds the walk. Sampling recent batches misses finished goods that only older batches flowed to, and stock held that way still has to count against the decision to build more.
	seedRows, apiErr := repo.GetSeedBatchesForItems(ctx, domain.GetSeedBatchesParams{
		AccountID:   accountID,
		ItemIDs:     itemIDs,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	// Batch -> the constraint item it descends from. First writer wins.
	rootByBatch := map[string]string{}
	frontier := make([]string, 0, len(seedRows))

	for _, row := range seedRows {
		if _, taken := rootByBatch[row.BatchID]; taken {
			continue
		}
		rootByBatch[row.BatchID] = row.ItemID
		frontier = append(frontier, row.BatchID)
	}

	descendants := map[string]map[string]bool{}
	claimedItem := map[string]string{} // descendant item -> owning constraint item

	for depth := 0; depth < maxDepth && len(frontier) > 0; depth++ {
		sort.Strings(frontier)

		childRows, apiErr := repo.GetBatchFlowChildren(ctx, accountID, frontier)
		if apiErr != nil {
			return nil, apiErr
		}

		next := make([]string, 0, len(childRows))
		for _, row := range childRows {
			rootItemID, ok := rootByBatch[row.ParentBatchID]
			if !ok {
				continue
			}
			if _, seen := rootByBatch[row.BatchID]; seen {
				continue // already reached by another path
			}
			rootByBatch[row.BatchID] = rootItemID
			next = append(next, row.BatchID)

			// Attribute the item to the first constraint item that reaches it, so a shared finished good is not double-counted into two echelons.
			if owner, taken := claimedItem[row.ItemID]; taken && owner != rootItemID {
				continue
			}
			claimedItem[row.ItemID] = rootItemID
			if descendants[rootItemID] == nil {
				descendants[rootItemID] = map[string]bool{}
			}
			descendants[rootItemID][row.ItemID] = true
		}
		frontier = next
	}

	out := make(map[string][]string, len(descendants))
	for rootItemID, items := range descendants {
		out[rootItemID] = scheduleSortedKeys(items)
	}
	return out, nil
}

// loadDemand pools finished-goods order demand back onto the constraint item that produces it, keeping the per-finished-good detail so the two safety-stock echelons can be computed separately.
func (s *productionScheduleSvcImpl) loadDemand(
	ctx context.Context,
	repo domain.ProductionScheduleInputRepo,
	params domain.LoadSolverInputParams,
	itemIDs []string,
	descendantItemsByItem map[string][]string,
	onHandByItem map[string]float64,
	nativeRatioOf func(itemID string) float64,
	in *scheduling.SolverInput,
) *apierror.APIError {
	// Only items with a product carry order demand.
	sellableCandidates := map[string]bool{}
	for _, itemID := range itemIDs {
		sellableCandidates[itemID] = true
		for _, descendantID := range descendantItemsByItem[itemID] {
			sellableCandidates[descendantID] = true
		}
	}

	productRows, apiErr := repo.GetProductsForItems(ctx, params.AccountID, scheduleSortedKeys(sellableCandidates))
	if apiErr != nil {
		return apiErr
	}

	itemByProduct := map[string]string{}
	skuByItem := map[string]string{}
	productLineByItem := map[string]string{}
	productIDs := make([]string, 0, len(productRows))
	for _, row := range productRows {
		itemByProduct[row.ProductID] = row.ItemID
		skuByItem[row.ItemID] = row.SKU
		if row.ProductLineID != nil {
			productLineByItem[row.ItemID] = *row.ProductLineID
		}
		productIDs = append(productIDs, row.ProductID)
	}
	if len(productIDs) == 0 {
		return nil
	}

	windowStart := params.PlanningAsOf.AddDate(0, -params.ForecastHistoryMonths, 0)
	demandRows, apiErr := repo.GetPooledOrderDemandByProduct(ctx, domain.GetPooledOrderDemandParams{
		AccountID:   params.AccountID,
		WindowStart: windowStart,
		WindowEnd:   params.PlanningAsOf,
		ProductIDs:  productIDs,
	})
	if apiErr != nil {
		return apiErr
	}

	// Monthly demand per finished-good item.
	byFinishedItem := map[string]map[time.Time]float64{}
	for _, row := range demandRows {
		itemID, ok := itemByProduct[row.ProductID]
		if !ok {
			continue
		}
		monthStart := time.Date(row.Year, time.Month(row.Month), 1, 0, 0, 0, 0, time.UTC)
		if byFinishedItem[itemID] == nil {
			byFinishedItem[itemID] = map[time.Time]float64{}
		}
		byFinishedItem[itemID][monthStart] += row.Quantity
	}

	for itemID, setting := range params.ItemSettings {
		if setting.FulfillmentPolicyCode != "" {
			in.ItemPolicyOverrides[itemID] = setting.FulfillmentPolicyCode
		}
	}
	for itemID, lineID := range productLineByItem {
		in.ProductLineByItem[itemID] = lineID
	}

	linePolicies, apiErr := repo.ListProductLineFulfillmentPolicies(ctx, params.AccountID)
	if apiErr != nil {
		return apiErr
	}
	maps.Copy(in.PolicyByProductLine, linePolicies)

	// A make-to-order finished good contributes nothing to its constraint item's forecast or to its pooled variability: it is built when it is ordered, and forecasting it as well would build it twice. Resolved here, before pooling, because pooling is where that contribution would otherwise be made.
	forecastExcluded := map[string]bool{}
	for finishedItemID := range byFinishedItem {
		resolved := scheduling.ResolveFulfillmentPolicy(finishedItemID, scheduling.PolicyResolutionInput{
			ItemOverrides:       in.ItemPolicyOverrides,
			ProductLineByItem:   in.ProductLineByItem,
			PolicyByProductLine: in.PolicyByProductLine,
			AccountDefault:      in.DefaultFulfillmentPolicy,
		})
		if resolved.Policy == scheduling.PolicyMakeToOrder {
			forecastExcluded[finishedItemID] = true
		}
	}

	for _, constraintItemID := range itemIDs {
		// The constraint item's own demand plus every finished good it becomes. One unit of finished good consumes one unit of the constraint item, matching the script's assumption.
		contributors := append([]string{constraintItemID}, descendantItemsByItem[constraintItemID]...)

		// Sold quantities arrive normalized to base units; the whole family is expressed in the constraint item's scan unit so demand lines up with the run rates and lot sizes denominated in it.
		ratio := nativeRatioOf(constraintItemID)

		pooled := map[time.Time]float64{}
		var downstream []scheduling.FinishedGood

		for _, contributorID := range contributors {
			months := byFinishedItem[contributorID]
			if len(months) == 0 {
				continue
			}
			// Still listed as a downstream good below — the plan has to know what this greige becomes — but its history does not enter the forecast or the pooled sigma.
			if forecastExcluded[contributorID] {
				continue
			}
			for monthStart, quantity := range months {
				pooled[monthStart] += quantity / ratio
			}
			if contributorID != constraintItemID {
				// Identity is kept alongside the series: once these are pooled into the greige buffer the family total cannot name which SKU it came from, and that is exactly what the finished targets have to report.
				downstream = append(downstream, scheduling.FinishedGood{
					ItemID:        contributorID,
					SKU:           skuByItem[contributorID],
					ProductLineID: productLineByItem[contributorID],
					Monthly:       toMonthlySeriesScaled(months, ratio),
					OnHand:        onHandByItem[contributorID] / ratio,
				})
			}
		}

		// Sorted so the pooled sigma and the finished rows are built in a fixed order; Go randomizes the map iteration that fed this.
		sort.SliceStable(downstream, func(i, j int) bool {
			if downstream[i].SKU != downstream[j].SKU {
				return downstream[i].SKU < downstream[j].SKU
			}
			return downstream[i].ItemID < downstream[j].ItemID
		})

		in.MonthlyByItem[constraintItemID] = toMonthlySeries(pooled)
		if len(downstream) > 0 {
			in.DownstreamByItem[constraintItemID] = downstream
		}
	}

	return loadOpenOrderBook(ctx, repo, params, in, itemIDs, descendantItemsByItem, itemByProduct, nativeRatioOf)
}

// loadOpenOrderBook attaches the outstanding order book to the solve, pooled onto constraint items exactly the way historical demand is.
//
// It walks the same contributor sets and applies the same unit ratio, because the order book and the forecast are compared week by week during leveling: expressing them in different units would make the greater-of rule pick whichever happened to be scaled larger.
func loadOpenOrderBook(
	ctx context.Context,
	repo domain.ProductionScheduleInputRepo,
	params domain.LoadSolverInputParams,
	in *scheduling.SolverInput,
	itemIDs []string,
	descendantItemsByItem map[string][]string,
	itemByProduct map[string]string,
	nativeRatioOf func(string) float64,
) *apierror.APIError {
	// A finished good reachable from two constraint items is claimed by the first, matching the first-wins attribution the echelon on-hand walk already uses. itemIDs is sorted, so the choice is stable.
	constraintByContributor := map[string]string{}
	for _, constraintItemID := range itemIDs {
		for _, contributorID := range append([]string{constraintItemID}, descendantItemsByItem[constraintItemID]...) {
			if _, claimed := constraintByContributor[contributorID]; !claimed {
				constraintByContributor[contributorID] = constraintItemID
			}
		}
	}

	productIDs := make([]string, 0, len(itemByProduct))
	for productID := range itemByProduct {
		productIDs = append(productIDs, productID)
	}
	sort.Strings(productIDs)

	rows, apiErr := repo.GetOpenOrderRequirements(ctx, params.AccountID, productIDs)
	if apiErr != nil {
		return apiErr
	}

	for _, row := range rows {
		finishedItemID, ok := itemByProduct[row.ProductID]
		if !ok {
			continue
		}
		constraintItemID, ok := constraintByContributor[finishedItemID]
		if !ok {
			// Nothing in the plan produces this order. Dropping it is right: the alternative is putting work on a machine for a product it does not make.
			continue
		}
		ratio := nativeRatioOf(constraintItemID)
		if ratio == 0 {
			continue
		}
		in.OpenOrders = append(in.OpenOrders, scheduling.OpenOrderLine{
			SalesOrderID:     row.SalesOrderID,
			SalesOrderNumber: row.SalesOrderNumber,
			SalesOrderLineID: row.SalesOrderLineID,
			FinishedItemID:   finishedItemID,
			ConstraintItemID: constraintItemID,
			Units:            row.OutstandingQty / ratio,
			ShipByDate:       row.ShipByDate,
		})
	}

	return nil
}

// toMonthlySeries converts a month->quantity map to a chronologically sorted series.
func toMonthlySeries(months map[time.Time]float64) []scheduling.MonthlyDemand {
	return toMonthlySeriesScaled(months, 1)
}

// toMonthlySeriesScaled is toMonthlySeries with every quantity divided by the given unit ratio, which is how base-unit sold quantities become constraint-item scan units.
func toMonthlySeriesScaled(months map[time.Time]float64, ratio float64) []scheduling.MonthlyDemand {
	out := make([]scheduling.MonthlyDemand, 0, len(months))
	for monthStart, quantity := range months {
		out = append(out, scheduling.MonthlyDemand{MonthStart: monthStart, Quantity: quantity / ratio})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].MonthStart.Before(out[j].MonthStart) })
	return out
}

func scheduleSortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// weekStart returns the Monday on or before t. Schedule horizons honor the account's configured week start via scheduleWeekStart; this fixed-Monday form is for surfaces like analytics that bucket by ISO-style weeks.
func weekStart(t time.Time) time.Time {
	return scheduleWeekStart(t, 1)
}

// scheduleWeekStart returns the most recent day on or before t that falls on the configured weekday (0 = Sunday through 6 = Saturday), which is where a horizon week begins.
func scheduleWeekStart(t time.Time, weekStartDay int) time.Time {
	if weekStartDay < 0 || weekStartDay > 6 {
		weekStartDay = 1
	}
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(day.Weekday()) - weekStartDay + 7) % 7
	return day.AddDate(0, 0, -offset)
}

// GenerateProductionSchedule solves and persists a new draft version.
//
// The header, its lines and its policy snapshot are written in one transaction: a half-written plan would look like a real schedule with silently missing campaigns, which is worse than no schedule at all.
func (s *productionScheduleSvcImpl) GenerateProductionSchedule(
	ctx context.Context,
	params domain.GenerateProductionScheduleParams,
) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.generate")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	planningAsOf := params.PlanningAsOf
	if planningAsOf.IsZero() {
		planningAsOf = time.Now().UTC()
	}
	sourceCode := params.SourceCode
	if sourceCode == "" {
		sourceCode = domain.ScheduleSourceManual
	}

	var generatedBy *string
	if identity.Actor != nil {
		generatedBy = &identity.Actor.ID
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductionSchedule](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		result, apiErr := s.persistPlan(ctx, persistPlanParams{
			AccountID:     accountID,
			PlanningAsOf:  planningAsOf,
			HorizonWeeks:  params.HorizonWeeks,
			DemandBasis:   params.DemandBasis,
			Name:          params.Name,
			SourceCode:    sourceCode,
			GeneratedByID: generatedBy,
			// Cache the response inside the same transaction that writes the plan, so a retried POST replays this version rather than minting another.
			CacheResponse: func(txCtx context.Context, txSvc *productionScheduleSvcImpl, schedule *domain.ProductionSchedule) *apierror.APIError {
				return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, schedule)
			},
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// GetProductionSchedule returns one version by ID.
func (s *productionScheduleSvcImpl) GetProductionSchedule(ctx context.Context, scheduleID string) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.get")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The permission check is repeated here rather than left to readIdentity alone because the drift guard matches literal checks per handler; a check reachable only through a helper reads as an unprotected endpoint.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repos.NewProductionScheduleRepo().Get(ctx, domain.GetProductionScheduleParams{
		AccountID:  identity.Target.AccountID,
		ScheduleID: scheduleID,
	})
}

// GetCurrentProductionSchedule returns the published version covering today, or nil when there is none.
func (s *productionScheduleSvcImpl) GetCurrentProductionSchedule(ctx context.Context) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.get_current")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The permission check is repeated here rather than left to readIdentity alone because the drift guard matches literal checks per handler; a check reachable only through a helper reads as an unprotected endpoint.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repos.NewProductionScheduleRepo().GetCurrent(ctx, identity.Target.AccountID, time.Now().UTC())
}

// ListProductionSchedules returns a paginated list of versions.
func (s *productionScheduleSvcImpl) ListProductionSchedules(ctx context.Context, params domain.ListProductionSchedulesParams) (*domain.ListProductionSchedulesResult, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The permission check is repeated here rather than left to readIdentity alone because the drift guard matches literal checks per handler; a check reachable only through a helper reads as an unprotected endpoint.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID
	return s.repos.NewProductionScheduleRepo().List(ctx, params)
}

// ListProductionScheduleLines returns the planned campaigns for a version.
func (s *productionScheduleSvcImpl) ListProductionScheduleLines(ctx context.Context, params domain.ListProductionScheduleLinesParams) ([]*domain.ProductionScheduleLine, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_lines")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The permission check is repeated here rather than left to readIdentity alone because the drift guard matches literal checks per handler; a check reachable only through a helper reads as an unprotected endpoint.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID
	return s.repos.NewProductionScheduleRepo().ListLines(ctx, params)
}

// ListProductionScheduleItemPolicies returns the per-item policy snapshot behind a version.
func (s *productionScheduleSvcImpl) ListProductionScheduleItemPolicies(ctx context.Context, scheduleID string) ([]*domain.ProductionScheduleItemPolicy, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_item_policies")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The permission check is repeated here rather than left to readIdentity alone because the drift guard matches literal checks per handler; a check reachable only through a helper reads as an unprotected endpoint.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repos.NewProductionScheduleRepo().ListItemPolicies(ctx, identity.Target.AccountID, scheduleID)
}

// ListProductionScheduleFinishedPolicies returns the per-finished-SKU decomposition of a version's pooled constraint buffers.
func (s *productionScheduleSvcImpl) ListProductionScheduleFinishedPolicies(ctx context.Context, scheduleID string) ([]*domain.ProductionScheduleFinishedPolicy, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_finished_policies")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The permission check is repeated here rather than left to readIdentity alone because the drift guard matches literal checks per handler; a check reachable only through a helper reads as an unprotected endpoint.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repos.NewProductionScheduleRepo().ListFinishedPolicies(ctx, identity.Target.AccountID, scheduleID)
}

// ListProductionScheduleFinishingLines returns stage two: how many of which finished good to make from the knitted parts.
func (s *productionScheduleSvcImpl) ListProductionScheduleFinishingLines(ctx context.Context, params domain.ListProductionScheduleFinishingLinesParams) ([]*domain.ProductionScheduleFinishingLine, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_finishing_lines")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The permission check is repeated here rather than left to readIdentity alone because the drift guard matches literal checks per handler; a check reachable only through a helper reads as an unprotected endpoint.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	return s.repos.NewProductionScheduleRepo().ListFinishingLines(ctx, params)
}

// readIdentity is the shared read gate for every schedule query.
func (s *productionScheduleSvcImpl) readIdentity(ctx context.Context) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

// machineStepID and machineDepartmentID denormalize the machine's placement onto each line. A missing machine leaves them nil rather than guessing: work attributed to the wrong department is worse than work attributed to none.
func machineStepID(machines map[string]*domain.Machine, machineID string) *string {
	if machine, ok := machines[machineID]; ok {
		return machine.ProductionStepID
	}
	return nil
}

func machineDepartmentID(machines map[string]*domain.Machine, machineID string) *string {
	if machine, ok := machines[machineID]; ok {
		return machine.DepartmentID
	}
	return nil
}

// primaryMachines names, for each planned item, the machine carrying most of its run hours — the machine a planner would hand-extend a campaign on. Ties break to the lexicographically smallest machine ID so the same solve always names the same machine.
func primaryMachines(campaigns []scheduling.Campaign) map[string]string {
	hours := map[string]map[string]float64{}
	for _, c := range campaigns {
		if hours[c.ItemID] == nil {
			hours[c.ItemID] = map[string]float64{}
		}
		hours[c.ItemID][c.MachineID] += c.RunHours
	}

	primary := make(map[string]string, len(hours))
	for itemID, byMachine := range hours {
		best := ""
		for machineID, machineHours := range byMachine {
			if best == "" || machineHours > byMachine[best] ||
				(machineHours == byMachine[best] && machineID < best) {
				best = machineID
			}
		}
		primary[itemID] = best
	}
	return primary
}

// writeSolvedPlanParams drives the write half of a solve.
type writeSolvedPlanParams struct {
	AccountID    string
	ScheduleID   string
	HorizonStart time.Time
	Output       *scheduling.SolverOutput
	Starved      map[string]bool
	Capped       map[string]bool
	// SkipKeys are campaigns a regenerate is keeping from the existing plan because a person edited them. The solver's own answer for those is dropped rather than written alongside, which would leave two campaigns for one machine-item-week.
	SkipKeys map[campaignKey]bool
}

// lotUnitIDOf renders an optional lot unit for the nullable column.
func lotUnitIDOf(unitID string) *string {
	if unitID == "" {
		return nil
	}
	return &unitID
}

// writeSolvedPlan writes the lines, the policy snapshot and the derived department work for one solve.
//
// Shared by the first generate and by a regenerate so the two can never drift into writing a plan differently — a regenerate that quietly disagrees with what a planner saw in preview would be worse than no regenerate.
func (s *productionScheduleSvcImpl) writeSolvedPlan(ctx context.Context, params writeSolvedPlanParams) *apierror.APIError {
	repo := s.repos.NewProductionScheduleRepo()
	output := params.Output
	scheduleID := params.ScheduleID
	horizonStart := params.HorizonStart
	starved := params.Starved
	capped := params.Capped

	// Every campaign runs on a machine, and the machine knows its production step and department. Carrying those onto the line is what lets department work be derived from the plan and what lets attainment roll up by department.
	machineIDs := map[string]bool{}
	for _, c := range output.Campaigns {
		if c.MachineID != "" {
			machineIDs[c.MachineID] = true
		}
	}
	machineList := make([]string, 0, len(machineIDs))
	for machineID := range machineIDs {
		machineList = append(machineList, machineID)
	}
	sort.Strings(machineList)

	machines, apiErr := s.repos.NewMachineRepo().GetByIDs(ctx, params.AccountID, machineList)
	if apiErr != nil {
		return apiErr
	}
	machineByID := make(map[string]*domain.Machine, len(machines))
	for _, machine := range machines {
		machineByID[machine.ID] = machine
	}

	policyByItem := make(map[string]string, len(output.Policies))
	for _, p := range output.Policies {
		policyByItem[p.ItemID] = p.FulfillmentPolicy
	}

	lines := make([]*domain.ProductionScheduleLine, 0, len(output.Campaigns))
	for _, c := range output.Campaigns {
		// A campaign a person edited and a regenerate is keeping already exists; writing the solver's answer for it too would leave two campaigns on one machine-item-week and double what the plan asks for.
		if params.SkipKeys[campaignKey{ItemID: c.ItemID, MachineID: c.MachineID, WeekIndex: safeconv.IntToInt32(c.WeekIndex)}] {
			continue
		}

		lineID, apiErr := id.GenID(id.ProductionScheduleLineIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		projected := output.ProjectedOnHand[c.ItemID]
		var before, after float64
		if c.WeekIndex < len(projected) {
			after = projected[c.WeekIndex]
			if c.WeekIndex > 0 {
				before = projected[c.WeekIndex-1]
			}
		}

		// The reason names what put the campaign on the plan: an order already on the book, or a buffer running down. The column has existed since the schedule shipped and nothing has ever written it.
		reason := string(constants.ScheduleLineReasonReorderPoint)
		if policyByItem[c.ItemID] == scheduling.PolicyMakeToOrder {
			reason = string(constants.ScheduleLineReasonFirmOrder)
		}

		lines = append(lines, &domain.ProductionScheduleLine{
			ID:                   lineID,
			ReasonCode:           &reason,
			ProductionScheduleID: scheduleID,
			WeekIndex:            safeconv.IntToInt32(c.WeekIndex),
			WeekStartDate:        horizonStart.AddDate(0, 0, c.WeekIndex*7),
			MachineID:            c.MachineID,
			ItemID:               c.ItemID,
			PlannedQuantity:      c.Units,
			PlannedLots:          safeconv.IntToInt32(c.Lots),
			PlannedLotUnits:      c.LotUnits,
			// The unit the lot is counted in, so a 60 on the plan and a 60 on the floor mean the same thing. Null when nothing in the chain supplied one.
			PlannedUnitID:         lotUnitIDOf(c.LotUnitID),
			PlannedRunHours:       c.RunHours,
			SequenceIndex:         safeconv.IntToInt32(len(lines)),
			ProjectedOnHandBefore: before,
			ProjectedOnHandAfter:  after,
			StatusCode:            domain.ScheduleLineStatusPlanned,
			SourceCode:            domain.ScheduleLineSourceSolver,
			ProductionStepID:      machineStepID(machineByID, c.MachineID),
			DepartmentID:          machineDepartmentID(machineByID, c.MachineID),
			// Nothing is frozen until publish; a draft is entirely editable.
			IsFrozen: false,
		})
	}
	if apiErr := repo.CreateLines(ctx, params.AccountID, scheduleID, lines); apiErr != nil {
		return apiErr
	}

	// Which campaign is building which order. Written in the same transaction as the lines it points at: a link to a campaign that does not exist is worse than no link, and the at-risk report is read from the same allocation, so the two can never disagree about whether an order is covered.
	if apiErr := repo.ReplaceLineOrders(ctx, params.AccountID, scheduleID, buildLineOrderLinks(output.Allocations, lines)); apiErr != nil {
		return apiErr
	}

	// The policy snapshot describes one solve. A regenerate re-solves the same version, so the previous snapshot goes before the new one lands; a fresh generate clears nothing because there is nothing there.
	if apiErr := repo.DeleteItemPolicies(ctx, params.AccountID, scheduleID); apiErr != nil {
		return apiErr
	}

	primaryMachineByItem := primaryMachines(output.Campaigns)

	policies := make([]*domain.ProductionScheduleItemPolicy, 0, len(output.Policies))
	for _, p := range output.Policies {
		policyID, apiErr := id.GenID(id.ProductionScheduleItemPolicyIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		policy := &domain.ProductionScheduleItemPolicy{
			ID:                      policyID,
			ProductionScheduleID:    scheduleID,
			ItemID:                  p.ItemID,
			SKU:                     p.SKU,
			UnitID:                  lotUnitIDOf(p.UnitID),
			AnnualDemand:            p.AnnualDemand,
			WeeklyDemand:            p.WeeklyDemand,
			SecondsPerUnit:          p.SecondsPerUnit,
			UnitCost:                p.UnitCost,
			SetupCost:               p.SetupCost,
			HoldingCost:             p.HoldingCost,
			EOQUnits:                p.EOQUnits,
			ConstraintLeadTimeWeeks: p.ConstraintLeadTimeWeeks,
			FinishLeadTimeWeeks:     p.FinishLeadTimeWeeks,
			SigmaWeeklyPooled:       p.SigmaWeeklyPooled,
			SigmaDownstreamSum:      p.SigmaDownstreamSum,
			SafetyStockPrimary:      p.SafetyStockPrimary,
			SafetyStockDownstream:   p.SafetyStockDownstream,
			ReorderPoint:            p.ReorderPoint,
			OrderUpTo:               p.OrderUpTo,
			OnHandEchelon:           p.OnHandEchelon,
			OnHandGreige:            p.OnHandGreige,
			AverageGreigeInventory:  p.AverageGreigeInventory,
			MaxGreigeInventory:      p.MaxGreigeInventory,
			ProjectedOnHand:         output.ProjectedOnHand[p.ItemID],
			ProjectedGreigeOnHand:   output.ProjectedGreigeOnHand[p.ItemID],
			WeeksOfCover:            p.WeeksOfCover,
			AnnualRunHours:          p.AnnualRunHours(),
			WasEOQCapped:            capped[p.SKU],
			WasCapacityStarved:      starved[p.SKU],
			FulfillmentPolicyCode:   p.FulfillmentPolicy,
			PolicySourceCode:        p.PolicySource,
			FirmDemandUnits:         p.FirmDemandUnits,
			ForecastDemandUnits:     p.ForecastDemandUnits,
		}
		if p.ABCClass != "" {
			// The scheduling engine mirrors the TS reference implementation and emits uppercase classes; the API contract (constants.ABCClass) is lowercase, so normalize at the boundary before anything is stored or emitted.
			abc := strings.ToLower(p.ABCClass)
			policy.ABCClass = &abc
		}
		if machineID, ok := primaryMachineByItem[p.ItemID]; ok {
			primary := machineID
			policy.PrimaryMachineID = &primary
			policy.ProductionStepID = machineStepID(machineByID, machineID)
		}
		policies = append(policies, policy)
	}
	if apiErr := repo.CreateItemPolicies(ctx, params.AccountID, scheduleID, policies); apiErr != nil {
		return apiErr
	}

	// The finished-goods decomposition of those pooled buffers. Written in the same transaction so a version can never hold a greige target whose finished half is missing — half an inventory picture is worse than none.
	finished := make([]*domain.ProductionScheduleFinishedPolicy, 0, len(output.FinishedPolicies))
	for _, f := range output.FinishedPolicies {
		finishedID, apiErr := id.GenID(id.ProductionScheduleFinishedPolicyIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		row := &domain.ProductionScheduleFinishedPolicy{
			ID:                   finishedID,
			ProductionScheduleID: scheduleID,
			ItemID:               f.ItemID,
			SKU:                  f.SKU,
			GreigeItemID:         f.GreigeItemID,
			GreigeSKU:            f.GreigeSKU,
			AnnualDemand:         f.AnnualDemand,
			WeeklyDemand:         f.WeeklyDemand,
			SigmaWeekly:          f.SigmaWeekly,
			SafetyStock:          f.SafetyStock,
			ReorderPoint:         f.ReorderPoint,
			OnHand:               f.OnHand,
			WeeksOfCover:         f.WeeksOfCover,
		}
		if f.ProductLineID != "" {
			lineID := f.ProductLineID
			row.ProductLineID = &lineID
		}
		finished = append(finished, row)
	}
	if apiErr := repo.ReplaceFinishedPolicies(ctx, params.AccountID, scheduleID, finished); apiErr != nil {
		return apiErr
	}

	// Stage two, written in the same transaction as the stage-one plan it draws from. A version holding a knit plan with no finishing plan reads as "nothing to finish" rather than as a half-written record, which is the worse of the two failures.
	if apiErr := s.writeFinishingPlan(ctx, params); apiErr != nil {
		return apiErr
	}

	// Department work is derived inside the same transaction as the plan it follows from. Deriving afterwards would leave a window where a schedule exists but the departments downstream of it have nothing to do.
	return s.deriveDepartmentWork(ctx, params.AccountID, scheduleID)
}

// writeFinishingPlan persists stage two: how many of which finished good to make from the knitted parts.
//
// Rewritten wholesale rather than patched, like the item policies: the mix is a pure function of the knit plan, the order book and each SKU's position, so a partial update could leave a week holding a line for a campaign the re-solve no longer produces.
func (s *productionScheduleSvcImpl) writeFinishingPlan(ctx context.Context, params writeSolvedPlanParams) *apierror.APIError {
	repo := s.repos.NewProductionScheduleRepo()

	lines := make([]*domain.ProductionScheduleFinishingLine, 0, len(params.Output.FinishingLines))
	for _, f := range params.Output.FinishingLines {
		lineID, apiErr := id.GenID(id.ProductionScheduleFinishingLineIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		line := &domain.ProductionScheduleFinishingLine{
			ID:                    lineID,
			ProductionScheduleID:  params.ScheduleID,
			WeekIndex:             safeconv.IntToInt32(f.WeekIndex),
			WeekStartDate:         params.HorizonStart.AddDate(0, 0, f.WeekIndex*7),
			ItemID:                f.ItemID,
			SKU:                   f.SKU,
			GreigeItemID:          f.GreigeItemID,
			GreigeSKU:             f.GreigeSKU,
			PlannedQuantity:       f.Quantity,
			PlannedUnitID:         lotUnitIDOf(f.LotUnitID),
			PlannedLots:           safeconv.IntToInt32(f.Lots),
			PlannedLotUnits:       f.LotUnits,
			PlannedRunHours:       f.RunHours,
			GreigeConsumed:        f.GreigeConsumed,
			FirmUnits:             f.FirmUnits,
			ProjectedOnHandBefore: f.ProjectedOnHandBefore,
			ProjectedOnHandAfter:  f.ProjectedOnHandAfter,
			StatusCode:            domain.ScheduleLineStatusPlanned,
			SourceCode:            domain.ScheduleLineSourceSolver,
			// Nothing is frozen until publish; a draft is entirely editable.
			IsFrozen: false,
		}
		// Where the SKU is made, denormalized so a department rollup needs no join. Absent for a finished good nobody has scanned a finishing step for, which is a data gap rather than a reason to drop the line.
		if f.ProductionStepID != "" {
			stepID := f.ProductionStepID
			line.ProductionStepID = &stepID
		}
		if f.DepartmentID != "" {
			departmentID := f.DepartmentID
			line.DepartmentID = &departmentID
		}

		lines = append(lines, line)
	}

	return repo.ReplaceFinishingLines(ctx, params.AccountID, params.ScheduleID, lines)
}

// persistPlanParams drives the one shared solve-and-write path.
type persistPlanParams struct {
	AccountID    string
	PlanningAsOf time.Time
	HorizonWeeks int
	DemandBasis  string
	Name         *string
	SourceCode   string
	// GeneratedByID is nil for a cadence run, which has no human actor.
	GeneratedByID *string
	// ExistingScheduleID fills a row already created in `generating`. Empty creates a new version.
	ExistingScheduleID string
	// CacheResponse, when set, caches the stored schedule as the idempotent success response inside the same transaction that writes the plan. Nil for the cadence path, which carries no user idempotency key.
	CacheResponse func(txCtx context.Context, txSvc *productionScheduleSvcImpl, schedule *domain.ProductionSchedule) *apierror.APIError
}

// persistPlan solves and writes one complete version.
//
// Manual generation and the generation cadence both come through here so there is exactly one solve-and-persist path in the system.
func (s *productionScheduleSvcImpl) persistPlan(ctx context.Context, params persistPlanParams) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.persist_plan")
	defer span.End()

	output, effective, apiErr := s.solveFor(ctx, params.AccountID, params.PlanningAsOf, params.HorizonWeeks, params.DemandBasis, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	scheduleID := params.ExistingScheduleID
	if scheduleID == "" {
		generated, apiErr := id.GenID(id.ProductionScheduleIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		scheduleID = generated
	}

	horizonStart := scheduleWeekStart(params.PlanningAsOf, effective.Settings.WeekStartDay)
	horizonEnd := horizonStart.AddDate(0, 0, effective.Settings.HorizonWeeks*7-1)

	// Snapshot the assumptions and the diagnostics so the plan stays explainable after settings, costs or demand move underneath it.
	settingsSnapshot, err := json.Marshal(effective.Settings)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Could not snapshot schedule settings."))
	}
	diagnostics, err := json.Marshal(output.Diagnostics)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Could not snapshot schedule diagnostics."))
	}

	schedule := &domain.ProductionSchedule{
		ID:                   scheduleID,
		AccountID:            params.AccountID,
		StatusCode:           domain.ScheduleStatusDraft,
		Name:                 params.Name,
		PlanningAsOf:         params.PlanningAsOf,
		HorizonStartDate:     horizonStart,
		HorizonEndDate:       horizonEnd,
		HorizonWeeks:         safeconv.IntToInt32(effective.Settings.HorizonWeeks),
		FrozenWeeks:          safeconv.IntToInt32(effective.Settings.FrozenWeeks),
		DemandBasisCode:      effective.DemandBasisCode,
		GenerationSourceCode: params.SourceCode,
		SolverVersion:        output.SolverVersion,
		SettingsSnapshot:     settingsSnapshot,
		Diagnostics:          diagnostics,
	}
	schedule.GeneratedByID = params.GeneratedByID

	starved := map[string]bool{}
	for _, sku := range output.Diagnostics.CapacityStarvedSKUs {
		starved[sku] = true
	}
	capped := map[string]bool{}
	for _, sku := range output.Diagnostics.EOQCappedSKUs {
		capped[sku] = true
	}

	var stored *domain.ProductionSchedule
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		// Two ways in: a manual generate creates the header here, while a cadence solve fills a row that was already created in `generating` when the work was queued. Both then write identical lines and policies, so the two paths cannot drift into planning differently — a cadence that quietly disagrees with what a planner sees in preview would be worse than no cadence.
		if params.ExistingScheduleID == "" {
			version, apiErr := repo.NextVersion(txCtx, params.AccountID)
			if apiErr != nil {
				return apiErr
			}
			schedule.Version = version

			if apiErr := repo.Create(txCtx, schedule); apiErr != nil {
				return apiErr
			}
		} else if apiErr := repo.FillGeneratedSchedule(txCtx, schedule); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.writeSolvedPlan(txCtx, writeSolvedPlanParams{
			AccountID:    params.AccountID,
			ScheduleID:   scheduleID,
			HorizonStart: horizonStart,
			Output:       output,
			Starved:      starved,
			Capped:       capped,
		}); apiErr != nil {
			return apiErr
		}

		// Re-read inside the transaction so the caller (and the idempotency cache) sees the row as stored, including the version the fill path did not set.
		got, apiErr := repo.Get(txCtx, domain.GetProductionScheduleParams{
			AccountID:  params.AccountID,
			ScheduleID: scheduleID,
		})
		if apiErr != nil {
			return apiErr
		}
		stored = got

		if params.CacheResponse != nil {
			return params.CacheResponse(txCtx, txSvc, stored)
		}
		return nil
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return stored, nil
}

// buildLineOrderLinks maps the solver's allocations onto the lines that were actually written.
//
// The solver identifies a campaign by (item, machine, week) because it has no line IDs; this resolves those back to the rows just persisted. A campaign the write skipped — one a regenerate is keeping by hand — has no line to point at, and its allocation is dropped rather than pointed at somebody else's.
func buildLineOrderLinks(
	allocations []scheduling.OrderAllocation,
	lines []*domain.ProductionScheduleLine,
) []domain.CreateLineOrderParams {
	type slot struct {
		itemID    string
		machineID string
		week      int32
	}
	lineBySlot := make(map[slot]string, len(lines))
	for _, l := range lines {
		lineBySlot[slot{itemID: l.ItemID, machineID: l.MachineID, week: l.WeekIndex}] = l.ID
	}

	out := make([]domain.CreateLineOrderParams, 0, len(allocations))
	for _, a := range allocations {
		lineID, ok := lineBySlot[slot{itemID: a.ItemID, machineID: a.CampaignMachineID, week: safeconv.IntToInt32(a.CampaignWeek)}]
		if !ok {
			continue
		}
		// The order line is what a shipment is measured against; without one the link cannot be reconciled later, so it is dropped rather than stored half-formed.
		if a.SalesOrderLineID == "" {
			continue
		}
		out = append(out, domain.CreateLineOrderParams{
			ProductionScheduleLineID: lineID,
			SalesOrderID:             a.SalesOrderID,
			SalesOrderLineID:         a.SalesOrderLineID,
			AllocatedQuantity:        a.Units,
		})
	}
	return out
}
