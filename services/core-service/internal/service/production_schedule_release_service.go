package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
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
	lots        []float64
}

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

	unitID, apiErr := s.repos.NewItemRepo().GetCategoryBaseUnitID(ctx, item.ItemCategoryID)
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

	for _, line := range lines {
		if line.StatusCode == string(constants.ProductionScheduleLineStatusCancelled) {
			continue
		}
		if line.PlannedQuantity <= 0 {
			continue
		}

		lots := scheduling.SplitIntoLots(line.PlannedQuantity, line.PlannedLotUnits)
		if len(lots) > scheduling.MaxLotsPerCampaign {
			return nil, apierror.NewValidationError(fmt.Sprintf(
				"Releasing %s would create %d batches for one campaign, which is more than the %d allowed. Check the lot size on this item.",
				skuOrItem(skuByItem, line.ItemID), len(lots), scheduling.MaxLotsPerCampaign))
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
		}
		if name, ok := nameByMachine[line.MachineID]; ok {
			linePlan.machineName = &name
		}

		plan.lines = append(plan.lines, linePlan)
		plan.batchCount += safeconv.IntToInt32(len(lots))
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
		batches := make([]domain.ReleaseBatch, 0, len(linePlan.lots))
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

	plan, apiErr := s.buildReleasePlan(ctx, identity.Target.AccountID, scheduleID, weekIndex)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.ReleaseScheduleWeekPreview{
		WeekIndex:               weekIndex,
		WeekStartDate:           plan.weekStartDate,
		LineCount:               safeconv.IntToInt32(len(plan.lines)),
		BatchCount:              plan.batchCount,
		TotalQuantity:           plan.totalQuantity,
		Lines:                   plan.toDomainLines(),
		IsReleasable:            plan.blockedReason == nil,
		BlockedReason:           plan.blockedReason,
		ExistingProductionRunID: plan.existingProductionRunID,
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
		plan, apiErr := txSvc.buildReleasePlan(txCtx, accountID, params.ProductionScheduleID, params.WeekIndex)
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
		var batchCount int32

		for _, linePlan := range plan.lines {
			line := linePlan.line
			batches := make([]domain.ReleaseBatch, 0, len(linePlan.lots))

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
				Batches:                  batches,
			})
		}

		result = &domain.ReleaseScheduleWeekResult{
			ProductionRun:     run,
			WeekIndex:         params.WeekIndex,
			WeekStartDate:     plan.weekStartDate,
			ReleasedLineCount: safeconv.IntToInt32(len(releasedLines)),
			BatchCount:        batchCount,
			TotalQuantity:     plan.totalQuantity,
			Lines:             releasedLines,
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
