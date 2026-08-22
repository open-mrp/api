package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

// MaxBatchesPerRelease bounds one release, independently of the per-campaign cap.
//
// Thirteen SKUs at 500 lots each is a production run no supervisor can work, and it is far more likely to be a misconfigured lot size than a real week.
const MaxBatchesPerRelease = 2000

// releasePlan is the shared work of previewing and performing a release: everything up to the point where rows would be written.
type releasePlan struct {
	schedule      *domain.ProductionSchedule
	weekStartDate time.Time
	lines         []releaseLinePlan
	batchCount    int32
	carriedCount  int32
	totalQuantity float64
	// blockedReason is set when the week cannot be released. The plan is still returned so a preview can show what *would* have been created alongside the reason.
	blockedReason           *string
	existingProductionRunID *string
}

type releaseLinePlan struct {
	line        *domain.ProductionScheduleLine
	sku         string
	machineName *string
	lotUnits    float64
	unitID      string
	// lots are the batches this release would create, after whatever an earlier week already issued has been counted against the campaign.
	lots []float64
	// carried are tickets an earlier week issued for this item and the floor never worked. They are moved into the new run rather than reprinted.
	carried []domain.CarryForwardBatch
}

// carriedQuantity is how much of a campaign is already covered by printed tickets.
func (p releaseLinePlan) carriedQuantity() float64 {
	var total float64
	for _, batch := range p.carried {
		total += batch.Quantity
	}
	return total
}

// carryForwardEpsilon absorbs the float noise a subtraction chain leaves behind, so a campaign fully covered by carried tickets does not trail a batch of a millionth of a unit — the same tolerance SplitIntoLots applies.
const carryForwardEpsilon = 1e-6

// resolveLineUnitID decides what unit a released batch is counted in.
//
// A campaign carries its own unit when the plan recorded one. Otherwise it falls back to the item's category base unit, which is how the rest of the system derives an item's unit — a batch with no unit is not a quantity, so this fails loudly rather than writing one.
func (s *productionScheduleSvcImpl) resolveLineUnitID(
	ctx context.Context,
	accountID string,
	line *domain.ProductionScheduleLine,
	unitByItem map[string]string,
) (string, *apierror.APIError) {
	if line.PlannedUnitID != nil && *line.PlannedUnitID != "" {
		return *line.PlannedUnitID, nil
	}
	if unitID, ok := unitByItem[line.ItemID]; ok {
		return unitID, nil
	}

	item, apiErr := s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
		AccountID: accountID,
		ItemID:    line.ItemID,
	})
	if apiErr != nil {
		return "", apiErr
	}
	if item.ItemCategoryID == "" {
		return "", apierror.NewValidationError("This item has no category, so the quantity to produce has no unit.")
	}

	unitID, _, apiErr := s.repos.NewItemRepo().GetCategoryBaseUnitID(ctx, item.ItemCategoryID)
	if apiErr != nil {
		return "", apiErr
	}
	if unitID == "" {
		return "", apierror.NewValidationError("This item's category has no base unit, so the quantity to produce has no unit.")
	}

	unitByItem[line.ItemID] = unitID
	return unitID, nil
}

// buildReleasePlan resolves one week into the batches a release would create.
//
// It never writes. Both the preview and the release itself go through here, so what the planner is shown and what the floor receives cannot drift apart.
func (s *productionScheduleSvcImpl) buildReleasePlan(
	ctx context.Context,
	accountID, scheduleID string,
	weekIndex int32,
	skipCarryForward bool,
) (*releasePlan, *apierror.APIError) {
	repo := s.repos.NewProductionScheduleRepo()

	schedule, apiErr := repo.Get(ctx, domain.GetProductionScheduleParams{
		AccountID:  accountID,
		ScheduleID: scheduleID,
	})
	if apiErr != nil {
		return nil, apiErr
	}
	if weekIndex < 0 || weekIndex >= schedule.HorizonWeeks {
		return nil, apierror.NewValidationErrorWithParam(
			fmt.Sprintf("This schedule covers weeks 0 to %d.", schedule.HorizonWeeks-1), "week_index")
	}

	plan := &releasePlan{
		schedule:      schedule,
		weekStartDate: schedule.HorizonStartDate.AddDate(0, 0, int(weekIndex)*7),
	}

	state, apiErr := repo.CountReleasedLinesForWeek(ctx, accountID, scheduleID, weekIndex)
	if apiErr != nil {
		return nil, apiErr
	}
	plan.existingProductionRunID = state.ExistingProductionRunID

	lines, apiErr := repo.ListLines(ctx, domain.ListProductionScheduleLinesParams{
		AccountID:  accountID,
		ScheduleID: scheduleID,
		WeekIndex:  &weekIndex,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	// SKUs come off the version's own policy snapshot rather than from the item table, so a release names the SKU the plan was built against even if it has been renamed.
	policies, apiErr := repo.ListItemPolicies(ctx, accountID, scheduleID)
	if apiErr != nil {
		return nil, apiErr
	}
	skuByItem := make(map[string]string, len(policies))
	for _, policy := range policies {
		skuByItem[policy.ItemID] = policy.SKU
	}

	machineIDs := map[string]bool{}
	for _, line := range lines {
		if line.MachineID != "" {
			machineIDs[line.MachineID] = true
		}
	}
	nameByMachine := map[string]string{}
	if len(machineIDs) > 0 {
		ids := make([]string, 0, len(machineIDs))
		for machineID := range machineIDs {
			ids = append(ids, machineID)
		}
		sort.Strings(ids)

		machines, apiErr := s.repos.NewMachineRepo().GetByIDs(ctx, accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, machine := range machines {
			nameByMachine[machine.ID] = machine.Name
		}
	}

	unitByItem := map[string]string{}
	// Candidates are read once per item and claimed as they are taken, so two campaigns splitting one item across two machines cannot both move the same ticket.
	candidatesByItem := map[string][]*domain.CarryForwardBatch{}
	claimed := map[string]bool{}

	for _, line := range lines {
		if line.StatusCode == string(constants.ProductionScheduleLineStatusCancelled) {
			continue
		}
		if line.PlannedQuantity <= 0 {
			continue
		}

		// Tickets an earlier week issued and nobody worked cover part of this campaign already. They are counted against it before it is split, so the release creates batches only for what is genuinely not yet on the floor.
		carried, apiErr := s.claimCarryForwardBatches(ctx, accountID, line, plan.weekStartDate, skipCarryForward, candidatesByItem, claimed)
		if apiErr != nil {
			return nil, apiErr
		}

		remaining := line.PlannedQuantity
		for _, batch := range carried {
			remaining -= batch.Quantity
		}
		if remaining < carryForwardEpsilon {
			remaining = 0
		}

		lots := scheduling.SplitIntoLots(remaining, line.PlannedLotUnits)
		if len(lots)+len(carried) > scheduling.MaxLotsPerCampaign {
			return nil, apierror.NewValidationError(fmt.Sprintf(
				"Releasing %s would create %d batches for one campaign, which is more than the %d allowed. Check the lot size on this item.",
				skuOrItem(skuByItem, line.ItemID), len(lots)+len(carried), scheduling.MaxLotsPerCampaign))
		}

		// Resolved while planning rather than while writing, so a missing unit blocks the preview too instead of failing halfway through a release.
		unitID, apiErr := s.resolveLineUnitID(ctx, accountID, line, unitByItem)
		if apiErr != nil {
			return nil, apiErr
		}

		linePlan := releaseLinePlan{
			line:     line,
			sku:      skuOrItem(skuByItem, line.ItemID),
			lotUnits: line.PlannedLotUnits,
			unitID:   unitID,
			lots:     lots,
			carried:  carried,
		}
		if name, ok := nameByMachine[line.MachineID]; ok {
			linePlan.machineName = &name
		}

		plan.lines = append(plan.lines, linePlan)
		plan.batchCount += safeconv.IntToInt32(len(lots) + len(carried))
		plan.carriedCount += safeconv.IntToInt32(len(carried))
		plan.totalQuantity += line.PlannedQuantity
	}

	if plan.batchCount > MaxBatchesPerRelease {
		return nil, apierror.NewValidationError(fmt.Sprintf(
			"Releasing this week would create %d batches, which is more than the %d allowed. Check the lot sizes on these items.",
			plan.batchCount, MaxBatchesPerRelease))
	}

	switch {
	case state.ExistingProductionRunID != nil:
		plan.blockedReason = strPtr("This week has already been released to the floor.")
	case len(plan.lines) == 0:
		plan.blockedReason = strPtr("There is nothing planned in this week to release.")
	}

	return plan, nil
}

// claimCarryForwardBatches takes as many of an item's unworked tickets as this campaign can absorb, and marks them taken.
//
// Taking stops as soon as the campaign is covered, so a ticket beyond what this week asks for is left on the run holding it rather than pulled into a week that has no work for it. The last ticket taken may overshoot: a printed 60-doff covering a 50-unit shortfall is still one ticket, and splitting it would mean reprinting the very thing this exists to avoid.
func (s *productionScheduleSvcImpl) claimCarryForwardBatches(
	ctx context.Context,
	accountID string,
	line *domain.ProductionScheduleLine,
	weekStartDate time.Time,
	skipCarryForward bool,
	candidatesByItem map[string][]*domain.CarryForwardBatch,
	claimed map[string]bool,
) ([]domain.CarryForwardBatch, *apierror.APIError) {
	if skipCarryForward {
		return nil, nil
	}

	candidates, ok := candidatesByItem[line.ItemID]
	if !ok {
		var apiErr *apierror.APIError
		candidates, apiErr = s.repos.NewProductionScheduleRepo().ListCarryForwardBatches(ctx, domain.ListCarryForwardBatchesParams{
			AccountID:     accountID,
			ItemID:        line.ItemID,
			WeekStartDate: weekStartDate,
		})
		if apiErr != nil {
			return nil, apiErr
		}
		candidatesByItem[line.ItemID] = candidates
	}

	return takeCarryForwardBatches(candidates, line.PlannedQuantity, claimed), nil
}

// takeCarryForwardBatches is the choosing itself, separated from the fetching so the rule can be tested without a database.
//
// Taking stops as soon as the campaign is covered, so a ticket beyond what this week asks for stays on the run holding it. The last ticket taken may overshoot: a printed 60-doff covering a 50-unit shortfall is still one ticket, and splitting it would mean reprinting the very thing this exists to avoid.
func takeCarryForwardBatches(candidates []*domain.CarryForwardBatch, plannedQuantity float64, claimed map[string]bool) []domain.CarryForwardBatch {
	var carried []domain.CarryForwardBatch
	remaining := plannedQuantity
	for _, candidate := range candidates {
		if remaining < carryForwardEpsilon {
			break
		}
		if candidate == nil || candidate.Quantity <= 0 || claimed[candidate.BatchID] {
			continue
		}
		claimed[candidate.BatchID] = true
		carried = append(carried, *candidate)
		remaining -= candidate.Quantity
	}
	return carried
}

func skuOrItem(skuByItem map[string]string, itemID string) string {
	if sku, ok := skuByItem[itemID]; ok && sku != "" {
		return sku
	}
	return itemID
}

func strPtr(v string) *string {
	return &v
}

func (p *releasePlan) toDomainLines() []domain.ReleasedScheduleLine {
	out := make([]domain.ReleasedScheduleLine, 0, len(p.lines))
	for _, linePlan := range p.lines {
		batches := make([]domain.ReleaseBatch, 0, len(linePlan.lots)+len(linePlan.carried))
		// Carried tickets lead: they are the doffs already on the floor, and a supervisor reading the confirmation needs to see what they are being asked to reuse before what they are being asked to print.
		for _, batch := range linePlan.carried {
			batches = append(batches, domain.ReleaseBatch{
				ItemID:             linePlan.line.ItemID,
				SKU:                linePlan.sku,
				Quantity:           batch.Quantity,
				BatchID:            batch.BatchID,
				CarriedForwardFrom: batch.ProductionRunNumber,
			})
		}
		for _, quantity := range linePlan.lots {
			batches = append(batches, domain.ReleaseBatch{
				ItemID:   linePlan.line.ItemID,
				SKU:      linePlan.sku,
				Quantity: quantity,
			})
		}
		out = append(out, domain.ReleasedScheduleLine{
			ProductionScheduleLineID: linePlan.line.ID,
			ItemID:                   linePlan.line.ItemID,
			SKU:                      linePlan.sku,
			MachineID:                linePlan.line.MachineID,
			MachineName:              linePlan.machineName,
			PlannedQuantity:          linePlan.line.PlannedQuantity,
			LotUnits:                 linePlan.lotUnits,
			Unit:                     unitAbbreviationOf(linePlan.line),
			CarriedForwardQuantity:   linePlan.carriedQuantity(),
			Batches:                  batches,
		})
	}
	return out
}

// unitAbbreviationOf is what a line's quantities are counted in, empty when the plan records none.
func unitAbbreviationOf(line *domain.ProductionScheduleLine) string {
	if line.PlannedUnitAbbreviation == nil {
		return ""
	}
	return *line.PlannedUnitAbbreviation
}

// PreviewReleaseProductionScheduleWeek says what releasing a week would create, without creating it.
func (s *productionScheduleSvcImpl) PreviewReleaseProductionScheduleWeek(
	ctx context.Context,
	scheduleID string,
	weekIndex int32,
	skipCarryForward bool,
) (*domain.ReleaseScheduleWeekPreview, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.preview_release_week")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// A preview is a read of the plan, so it asks for read rather than the update a release needs. A planner without release rights can still see what it would do. Written out rather than routed through writeIdentity because this endpoint is agent callable, and the permission guard matches literal checks at the call site.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	plan, apiErr := s.buildReleasePlan(ctx, identity.Target.AccountID, scheduleID, weekIndex, skipCarryForward)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.ReleaseScheduleWeekPreview{
		WeekIndex:                weekIndex,
		WeekStartDate:            plan.weekStartDate,
		LineCount:                safeconv.IntToInt32(len(plan.lines)),
		BatchCount:               plan.batchCount,
		CarriedForwardBatchCount: plan.carriedCount,
		TotalQuantity:            plan.totalQuantity,
		Lines:                    plan.toDomainLines(),
		IsReleasable:             plan.blockedReason == nil,
		BlockedReason:            plan.blockedReason,
		ExistingProductionRunID:  plan.existingProductionRunID,
	}, nil
}

// ReleaseProductionScheduleWeek turns one planned week into a production run.
//
// Each campaign becomes one batch per planned lot — six batches of 60 for a 360-pair week — so the run the floor receives is the plan broken into the doffs it will actually knit, not one undifferentiated 360-pair instruction.
//
// The whole release is one transaction. A run holding half a week's batches is worse than no run: the missing half looks like work nobody was asked to do, and attainment would report it as unplanned production.
func (s *productionScheduleSvcImpl) ReleaseProductionScheduleWeek(
	ctx context.Context,
	params domain.ReleaseScheduleWeekParams,
) (*domain.ReleaseScheduleWeekResult, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.release_week")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Releasing creates a production run, so it needs the authority to create one. Without this check the schedule permission would be a way around production-run permissions.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionRuns, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ReleaseScheduleWeekResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		result, apiErr := s.releaseProductionScheduleWeekTx(ctx, accountID, idempotencyKey.TypeID, params)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// releaseProductionScheduleWeekTx is the started-phase body of ReleaseProductionScheduleWeek: the run, its batches, the line status flips, the audit event and the idempotency cache commit together.
func (s *productionScheduleSvcImpl) releaseProductionScheduleWeekTx(
	ctx context.Context,
	accountID, idempotencyKeyTypeID string,
	params domain.ReleaseScheduleWeekParams,
) (*domain.ReleaseScheduleWeekResult, *apierror.APIError) {
	var result *domain.ReleaseScheduleWeekResult
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		// Built inside the transaction so the already-released check and the writes that depend on it see the same snapshot.
		plan, apiErr := txSvc.buildReleasePlan(txCtx, accountID, params.ProductionScheduleID, params.WeekIndex, params.SkipCarryForward)
		if apiErr != nil {
			return apiErr
		}
		if plan.blockedReason != nil {
			if plan.existingProductionRunID != nil {
				return apierror.NewValidationError(fmt.Sprintf(
					"Week %d has already been released as production run %s.",
					params.WeekIndex+1, *plan.existingProductionRunID))
			}
			return apierror.NewValidationError(*plan.blockedReason)
		}

		accountUserRepo := txSvc.repos.NewAccountUserRepo()
		responsibleID, apiErr := accountUserRepo.ResolveAccountUserID(txCtx, accountID, params.ResponsibleUserID)
		if apiErr != nil {
			return apierror.NewResourceNotFoundError("The responsible user was not found in this account.")
		}

		runRepo := txSvc.repos.NewProductionRunRepo()
		number, apiErr := runRepo.GetNextNumber(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}

		runID, apiErr := id.GenID(id.ProductionRunIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}

		run, apiErr := runRepo.Create(txCtx, runID, domain.CreateProductionRunParams{
			AccountID:         accountID,
			ResponsibleUserID: responsibleID,
		}, number)
		if apiErr != nil {
			return apiErr
		}

		batchRepo := txSvc.repos.NewBatchRepo()
		releasedLines := make([]domain.ReleasedScheduleLine, 0, len(plan.lines))
		var batchCount, carriedCount int32

		for _, linePlan := range plan.lines {
			line := linePlan.line
			batches := make([]domain.ReleaseBatch, 0, len(linePlan.lots)+len(linePlan.carried))

			// A carried ticket is moved, not remade. Its id, its number and the paper on the floor all stay as they are; only what it belongs to changes.
			for _, carried := range linePlan.carried {
				if apiErr := runRepo.SetBatchProductionRunID(txCtx, accountID, carried.BatchID, runID); apiErr != nil {
					return apiErr
				}
				if line.ProductionStepID != nil && *line.ProductionStepID != "" {
					if apiErr := batchRepo.ConnectProductionStep(txCtx, accountID, carried.BatchID, *line.ProductionStepID); apiErr != nil {
						return apiErr
					}
				}
				// The campaign that absorbed this ticket may run on a different machine from the one that was going to. Attainment attributes production through the batch-machine link, so it follows the work.
				if line.MachineID != "" {
					if apiErr := batchRepo.ReassignMachine(txCtx, accountID, carried.BatchID, line.MachineID); apiErr != nil {
						return apiErr
					}
				}

				batches = append(batches, domain.ReleaseBatch{
					ItemID:             line.ItemID,
					SKU:                linePlan.sku,
					Quantity:           carried.Quantity,
					BatchID:            carried.BatchID,
					CarriedForwardFrom: carried.ProductionRunNumber,
				})
				batchCount++
				carriedCount++
			}

			for _, quantity := range linePlan.lots {
				batchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				createParams := domain.CreateBatchParams{
					AccountID: accountID,
					ItemID:    line.ItemID,
					Quantity: domain.CreateQuantityParams{
						Measure: decimal.NewFromFloat(quantity),
						UnitID:  linePlan.unitID,
					},
				}
				if line.ProductionStepID != nil {
					createParams.ProductionStepID = *line.ProductionStepID
				}
				// The campaign names the machine, and attainment attributes production through the batch-machine link rather than through the schedule.
				if line.MachineID != "" {
					createParams.MachineIDs = []string{line.MachineID}
				}
				if params.ScanningStationID != nil {
					createParams.ScanningStationID = *params.ScanningStationID
				}

				if _, apiErr := batchRepo.Create(txCtx, batchID, createParams); apiErr != nil {
					return apiErr
				}
				if apiErr := runRepo.SetBatchProductionRunID(txCtx, accountID, batchID, runID); apiErr != nil {
					return apiErr
				}

				batches = append(batches, domain.ReleaseBatch{
					ItemID:   line.ItemID,
					SKU:      linePlan.sku,
					Quantity: quantity,
					BatchID:  batchID,
				})
				batchCount++
			}

			if apiErr := txSvc.repos.NewProductionScheduleRepo().
				MarkLineReleased(txCtx, accountID, line.ID, runID); apiErr != nil {
				return apiErr
			}

			releasedLines = append(releasedLines, domain.ReleasedScheduleLine{
				ProductionScheduleLineID: line.ID,
				ItemID:                   line.ItemID,
				SKU:                      linePlan.sku,
				MachineID:                line.MachineID,
				MachineName:              linePlan.machineName,
				PlannedQuantity:          line.PlannedQuantity,
				LotUnits:                 linePlan.lotUnits,
				Unit:                     unitAbbreviationOf(line),
				CarriedForwardQuantity:   linePlan.carriedQuantity(),
				Batches:                  batches,
			})
		}

		result = &domain.ReleaseScheduleWeekResult{
			ProductionRun:            run,
			WeekIndex:                params.WeekIndex,
			WeekStartDate:            plan.weekStartDate,
			ReleasedLineCount:        safeconv.IntToInt32(len(releasedLines)),
			BatchCount:               batchCount,
			CarriedForwardBatchCount: carriedCount,
			TotalQuantity:            plan.totalQuantity,
			Lines:                    releasedLines,
		}

		// Audited against the schedule rather than the run: the question this record answers later is "who committed this week to the floor", and the run is the consequence, not the subject. The changes are computed values with no struct snapshot behind them, so they are built with NewFieldChange rather than ComputeChanges (which only reads audit-tagged struct fields).
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeProductionSchedule,
			ResourceID:   params.ProductionScheduleID,
			Changes: []audit.FieldChange{
				audit.NewFieldChange("released_week_index", nil, params.WeekIndex),
				audit.NewFieldChange("production_run_id", nil, runID),
				audit.NewFieldChange("released_line_count", nil, len(releasedLines)),
				audit.NewFieldChange("released_batch_count", nil, batchCount),
				audit.NewFieldChange("carried_forward_batch_count", nil, carriedCount),
			},
		}); apiErr != nil {
			return apiErr
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKeyTypeID, result)
	})
	if apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}
