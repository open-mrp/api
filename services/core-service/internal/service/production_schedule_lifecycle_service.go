package service

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

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

// lineSnapshot is what a deviation records before and after a change. It is a flat copy rather than a reference so the log stays readable after the line is deleted — the whole point of an append-only record is that it outlives its subject.
type lineSnapshot struct {
	ID            string  `json:"id"`
	WeekIndex     int32   `json:"week_index"`
	MachineID     string  `json:"machine_id"`
	ItemID        string  `json:"item_id"`
	Quantity      float64 `json:"planned_quantity"`
	Lots          int32   `json:"planned_lots"`
	RunHours      float64 `json:"planned_run_hours"`
	SequenceIndex int32   `json:"sequence_index"`
	StatusCode    string  `json:"status_code"`
	SourceCode    string  `json:"source_code"`
	IsFrozen      bool    `json:"is_frozen"`
}

func snapshotLine(line *domain.ProductionScheduleLine) *lineSnapshot {
	if line == nil {
		return nil
	}
	return &lineSnapshot{
		ID:            line.ID,
		WeekIndex:     line.WeekIndex,
		MachineID:     line.MachineID,
		ItemID:        line.ItemID,
		Quantity:      line.PlannedQuantity,
		Lots:          line.PlannedLots,
		RunHours:      line.PlannedRunHours,
		SequenceIndex: line.SequenceIndex,
		StatusCode:    line.StatusCode,
		SourceCode:    line.SourceCode,
		IsFrozen:      line.IsFrozen,
	}
}

func marshalSnapshot(snapshot *lineSnapshot) []byte {
	if snapshot == nil {
		return nil
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		// A snapshot of a flat struct of scalars cannot fail to marshal; losing the snapshot is better than losing the deviation.
		return nil
	}
	return encoded
}

// classifyDeviation names the most significant thing that changed. An edit that moves a campaign to another machine AND changes its quantity is recorded as the machine change, because that is the change a planner has to react to first.
func classifyDeviation(before, after *lineSnapshot) string {
	switch {
	case before == nil:
		return domain.DeviationTypeLineAdded
	case after == nil:
		return domain.DeviationTypeLineRemoved
	case before.MachineID != after.MachineID:
		return domain.DeviationTypeMachineChanged
	case before.WeekIndex != after.WeekIndex:
		return domain.DeviationTypeWeekMoved
	case before.Quantity != after.Quantity:
		return domain.DeviationTypeQuantityChanged
	case before.SequenceIndex != after.SequenceIndex:
		return domain.DeviationTypeResequenced
	default:
		return domain.DeviationTypeQuantityChanged
	}
}

// isWeekFrozen reports whether a week falls inside the schedule's frozen window as it stands right now. A draft has no frozen_through_date, so nothing is frozen until the version is published — freezing is what publishing means.
func isWeekFrozen(schedule *domain.ProductionSchedule, weekStart time.Time) bool {
	if schedule.FrozenThroughDate == nil {
		return false
	}
	return !weekStart.After(*schedule.FrozenThroughDate)
}

// weekStartFor returns the horizon start plus whole weeks, matching how the solver lays the horizon out.
func weekStartFor(schedule *domain.ProductionSchedule, weekIndex int32) time.Time {
	return schedule.HorizonStartDate.AddDate(0, 0, int(weekIndex)*7)
}

func (s *productionScheduleSvcImpl) writeIdentity(ctx context.Context, action types.Action) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, action); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

func actorIDOf(identity *types.Identity) string {
	if identity.Actor != nil {
		return identity.Actor.ID
	}
	return ""
}

// recordDeviation writes one append-only log row.
//
// isFrozenWeek is materialized here, from the schedule as it stands at this instant. Deriving it at read time would let a later publish retroactively reclassify this edit, and frozen-week adherence would drift under its own history.
func (s *productionScheduleSvcImpl) recordDeviation(
	ctx context.Context,
	schedule *domain.ProductionSchedule,
	before, after *lineSnapshot,
	weekStart time.Time,
	reasonCode, reasonNote *string,
	actorID string,
) *apierror.APIError {
	deviationID, apiErr := id.GenID(id.ProductionScheduleDeviationIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	deviation := &domain.ProductionScheduleDeviation{
		AccountID:            schedule.AccountID,
		ProductionScheduleID: schedule.ID,
		DeviationTypeCode:    classifyDeviation(before, after),
		IsFrozenWeek:         isWeekFrozen(schedule, weekStart),
		BeforeJSON:           marshalSnapshot(before),
		AfterJSON:            marshalSnapshot(after),
		ReasonCode:           reasonCode,
		ReasonNote:           reasonNote,
		ActorID:              actorID,
	}

	// Deltas are signed and extracted here so adherence can SUM a column rather than parse JSON per row.
	var beforeQty, beforeHours, afterQty, afterHours float64
	if before != nil {
		beforeQty, beforeHours = before.Quantity, before.RunHours
		deviation.WeekIndex = &before.WeekIndex
		deviation.MachineID = &before.MachineID
		deviation.ItemID = &before.ItemID
	}
	if after != nil {
		afterQty, afterHours = after.Quantity, after.RunHours
		deviation.WeekIndex = &after.WeekIndex
		deviation.MachineID = &after.MachineID
		deviation.ItemID = &after.ItemID
		deviation.ProductionScheduleLineID = &after.ID
	}
	deviation.DeltaQuantity = afterQty - beforeQty
	deviation.DeltaRunHours = afterHours - beforeHours

	return s.repos.NewProductionScheduleRepo().CreateDeviation(ctx, deviationID, deviation)
}

// requireEditable rejects edits to a version that is no longer a live plan. A superseded or archived version is history, and history that can still be edited is not history.
func requireEditable(schedule *domain.ProductionSchedule) *apierror.APIError {
	switch schedule.StatusCode {
	case domain.ScheduleStatusDraft, domain.ScheduleStatusPublished:
		return nil
	default:
		return apierror.NewValidationError("Only draft and published schedules can be edited.")
	}
}

// requireFrozenReason enforces the one rule that makes the deviation log worth keeping: a change inside the frozen week has to say why.
func requireFrozenReason(frozen bool, reasonCode *string) *apierror.APIError {
	if !frozen {
		return nil
	}
	if reasonCode == nil || *reasonCode == "" {
		return apierror.NewValidationErrorWithParam("A reason is required to change a frozen week.", "reason_code")
	}
	return nil
}

// ListScheduleDeviationTypes returns the global taxonomy of what a hand change can be.
func (s *productionScheduleSvcImpl) ListScheduleDeviationTypes(ctx context.Context) ([]*domain.ScheduleDeviationType, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_deviation_types")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewProductionScheduleRepo().ListScheduleDeviationTypes(ctx)
}

// ListProductionScheduleDeviations returns the append-only log of hand changes.
func (s *productionScheduleSvcImpl) ListProductionScheduleDeviations(ctx context.Context, params domain.ListProductionScheduleDeviationsParams) (*domain.ListProductionScheduleDeviationsResult, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_deviations")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	return s.repos.NewProductionScheduleRepo().ListDeviations(ctx, params)
}

// estimatedRunHours prices a hand-added campaign in constraint time.
//
// The version's own solved rate first, so a campaign added next to a solved one is priced the same way.
//
// A SKU the version holds no policy for is the whole reason the planner is adding it by hand — it had no scan in the measurement window, so the solver never saw it — and falling to zero there would book a campaign that claims no machine time at all, leaving the week reading idle for work somebody has to do.
//
// So the item's own scans are read next, ignoring any window. A run rate here is the labor time of the production step a batch was produced at, which is what MeasureItems takes off the solver's own batches: a sample from two years ago therefore yields exactly the rate the solver would have derived had the scan been recent. Ties break toward the machine the campaign is going on, since that is the machine whose hours are being booked.
//
// The order below is item-specific before machine-specific, deliberately. A step belongs to an item, not to a machine — one knitting machine runs a ten-minute step for one SKU and an eight-minute step for another — so the machine's own production step is a per-machine default, and the weakest evidence of how long this SKU will occupy it. It is also the source most often absent: a machine is not required to name a step at all.
//
// Zero only when none of them is known, which is a SKU that has never run and whose flow configures no labor time.
func (s *productionScheduleSvcImpl) estimatedRunHours(
	ctx context.Context,
	accountID, scheduleID, itemID, machineID string,
	quantity float64,
	productionStepID *string,
) (float64, *apierror.APIError) {
	policies, apiErr := s.repos.NewProductionScheduleRepo().ListItemPolicies(ctx, accountID, scheduleID)
	if apiErr != nil {
		return 0, apiErr
	}
	for _, policy := range policies {
		if policy.ItemID == itemID && policy.SecondsPerUnit > 0 {
			return quantity * policy.SecondsPerUnit / 3600, nil
		}
	}

	measured, apiErr := s.measuredSecondsPerUnit(ctx, accountID, itemID, machineID)
	if apiErr != nil {
		return 0, apiErr
	}
	if measured > 0 {
		return quantity * measured / 3600, nil
	}

	// Never run, so fall back to what the flow was configured to do: the step that makes this item, then the machine's own.
	stepIDs, apiErr := s.repos.NewProductionFlowRepo().FindStepsByProducedItem(ctx, accountID, itemID)
	if apiErr != nil {
		return 0, apiErr
	}
	if productionStepID != nil {
		stepIDs = append(stepIDs, *productionStepID)
	}

	for _, stepID := range stepIDs {
		secondsPerUnit, apiErr := s.stepSecondsPerUnit(ctx, accountID, stepID)
		if apiErr != nil {
			return 0, apiErr
		}
		if secondsPerUnit > 0 {
			return quantity * secondsPerUnit / 3600, nil
		}
	}
	return 0, nil
}

// runRateHistorySamples caps how far back a hand-added campaign is priced from. A rarely-made SKU has a handful of scans in total, so this is a guard against reading a common SKU's whole history rather than a real horizon.
const runRateHistorySamples = 50

// measuredSecondsPerUnit prices an item from its own scan history, preferring the machine the campaign runs on. Zero when nothing it ever ran at carries a labor time.
func (s *productionScheduleSvcImpl) measuredSecondsPerUnit(ctx context.Context, accountID, itemID, machineID string) (float64, *apierror.APIError) {
	samples, apiErr := s.repos.NewProductionScheduleInputRepo().GetItemRunRateHistory(ctx, accountID, itemID, runRateHistorySamples)
	if apiErr != nil {
		return 0, apiErr
	}

	var anyMachine float64
	for _, sample := range samples {
		if sample.LaborTimeValue <= 0 {
			continue
		}
		secondsPerUnit := scheduling.SecondsPerUnitFromLaborTime(sample.LaborTimeValue, sample.LaborTimeUnit)
		if secondsPerUnit <= 0 {
			continue
		}
		// Samples arrive newest first, so the first hit on this machine is the most recent thing it actually ran.
		if machineID != "" && sample.MachineID == machineID {
			return secondsPerUnit, nil
		}
		if anyMachine == 0 {
			anyMachine = secondsPerUnit
		}
	}
	// No scan on this machine: the item has run somewhere, and the rate it ran at beats booking no time at all.
	return anyMachine, nil
}

// stepSecondsPerUnit reads a step's configured labor time as seconds per unit, or zero when it has none.
func (s *productionScheduleSvcImpl) stepSecondsPerUnit(ctx context.Context, accountID, stepID string) (float64, *apierror.APIError) {
	step, apiErr := s.repos.NewProductionStepRepo().Get(ctx, accountID, stepID)
	if apiErr != nil {
		return 0, apiErr
	}
	if step == nil || step.LaborTime == nil {
		return 0, nil
	}
	value, err := strconv.ParseFloat(step.LaborTime.Value, 64)
	if err != nil || value <= 0 {
		return 0, nil
	}
	return scheduling.SecondsPerUnitFromLaborTime(value, step.LaborTime.NumeratorUnit.Abbreviation), nil
}

// CreateProductionScheduleLine adds a campaign by hand and logs a deviation.
//
// The flow, in order: 1) editability check on the version, 2) frozen-week reason enforcement, 3) the insert plus its deviation-log row in one transaction, 4) an audit event when the change breaks a frozen commitment.
func (s *productionScheduleSvcImpl) CreateProductionScheduleLine(ctx context.Context, params domain.CreateProductionScheduleLineParams) (*domain.ProductionScheduleLine, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.create_line")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	lineID, apiErr := id.GenID(id.ProductionScheduleLineIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductionScheduleLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		result, apiErr := s.createProductionScheduleLineTx(ctx, identity, accountID, lineID, idempotencyKey.TypeID, params)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// createProductionScheduleLineTx is the started-phase body of CreateProductionScheduleLine: the insert, its deviation row, the frozen-week audit event and the idempotency cache commit together.
func (s *productionScheduleSvcImpl) createProductionScheduleLineTx(
	ctx context.Context,
	identity *types.Identity,
	accountID, lineID, idempotencyKeyTypeID string,
	params domain.CreateProductionScheduleLineParams,
) (*domain.ProductionScheduleLine, *apierror.APIError) {
	var result *domain.ProductionScheduleLine
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		schedule, apiErr := repo.Get(txCtx, domain.GetProductionScheduleParams{
			AccountID:  accountID,
			ScheduleID: params.ScheduleID,
		})
		if apiErr != nil {
			return apiErr
		}
		if apiErr := requireEditable(schedule); apiErr != nil {
			return apiErr
		}
		if params.WeekIndex < 0 || params.WeekIndex >= schedule.HorizonWeeks {
			return apierror.NewValidationErrorWithParam("The week is outside this schedule's horizon.", "week_index")
		}

		weekStart := weekStartFor(schedule, params.WeekIndex)
		frozen := isWeekFrozen(schedule, weekStart)
		if apiErr := requireFrozenReason(frozen, params.ReasonCode); apiErr != nil {
			return apiErr
		}

		machine, apiErr := txSvc.repos.NewMachineRepo().Get(txCtx, domain.GetMachineParams{
			AccountID: accountID,
			MachineID: params.MachineID,
		})
		if apiErr != nil {
			return apierror.NewValidationErrorWithParam("Unknown machine.", "machine_id")
		}

		sequenceIndex, apiErr := repo.NextSequenceIndex(txCtx, accountID, params.ScheduleID, params.WeekIndex)
		if apiErr != nil {
			return apiErr
		}

		line := &domain.ProductionScheduleLine{
			ID:                   lineID,
			ProductionScheduleID: params.ScheduleID,
			WeekIndex:            params.WeekIndex,
			WeekStartDate:        weekStart,
			MachineID:            params.MachineID,
			ProductionStepID:     machine.ProductionStepID,
			DepartmentID:         machine.DepartmentID,
			ItemID:               params.ItemID,
			PlannedQuantity:      params.Quantity,
			SequenceIndex:        sequenceIndex,
			StatusCode:           domain.ScheduleLineStatusPlanned,
			// Hand-added lines are manual from birth; nothing about them came from the solver, and a regenerate must be able to tell the difference.
			SourceCode: domain.ScheduleLineSourceManual,
			ReasonCode: params.ReasonCode,
			IsFrozen:   frozen,
		}
		// A hand-added campaign runs in the same lots as a solved one. Without this it would release to the floor as a single undifferentiated batch, however large.
		//
		// Resolved through the item's own lot chain rather than straight off the account default, so a line whose product line knits in sixties is planned in sixties. Falling to the account default here was how a campaign could be released in a lot size nobody had configured for that item.
		itemUnitID, apiErr := txSvc.itemCountingUnitID(txCtx, accountID, params.ItemID)
		if apiErr != nil {
			return apiErr
		}
		lot, apiErr := resolveItemLotDefault(txCtx, txSvc.repos, accountID, params.ItemID, itemUnitID)
		if apiErr != nil {
			return apiErr
		}
		if lot != nil && lot.Quantity > 0 {
			line.PlannedLotUnits = lot.Quantity
			if lot.UnitID != "" {
				line.PlannedUnitID = &lot.UnitID
			}
		}

		if params.Lots != nil {
			line.PlannedLots = *params.Lots
		} else if line.PlannedLotUnits > 0 {
			// Counted the way the release will actually split the campaign, not by dividing and rounding. A quantity below one lot rounds to zero lots, which reads as a campaign that issues no tickets while the release issues one anyway; a quantity that is not a lot multiple lands short of the batch the remainder becomes.
			line.PlannedLots = safeconv.IntToInt32(scheduling.CountLots(params.Quantity, line.PlannedLotUnits))
		}
		if params.RunHours != nil {
			line.PlannedRunHours = *params.RunHours
		} else {
			// Derived from the version's own measured rate for this SKU rather than left at zero. A campaign that claims no constraint time makes the week's utilisation read low, which is exactly the number a planner adds a campaign against.
			runHours, apiErr := txSvc.estimatedRunHours(txCtx, accountID, params.ScheduleID, params.ItemID, params.MachineID, params.Quantity, machine.ProductionStepID)
			if apiErr != nil {
				return apiErr
			}
			line.PlannedRunHours = runHours
		}

		if apiErr := repo.CreateLines(txCtx, accountID, params.ScheduleID, []*domain.ProductionScheduleLine{line}); apiErr != nil {
			return apiErr
		}

		created, apiErr := repo.GetLine(txCtx, accountID, lineID)
		if apiErr != nil {
			return apiErr
		}
		result = created

		if apiErr := txSvc.recordDeviation(txCtx, schedule, nil, snapshotLine(created), weekStart,
			params.ReasonCode, params.ReasonNote, actorIDOf(identity)); apiErr != nil {
			return apiErr
		}

		// A frozen-week change is a commitment being broken, so it always audits.
		if frozen {
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionCreate,
				ResourceType:     constants.ObjectTypeProductionScheduleLine,
				ResourceID:       created.ID,
				RootResourceType: constants.ObjectTypeProductionSchedule,
				RootResourceID:   schedule.ID,
				Changes:          audit.ComputeChanges(nil, created),
			}); apiErr != nil {
				return apiErr
			}
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKeyTypeID, result)
	})
	if apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}

// UpdateProductionScheduleLine edits a campaign and logs a deviation.
//
// The flow, in order: 1) editability check on the version, 2) frozen-week reason enforcement (against both the week the line is in and the week it moves to), 3) the update plus its deviation-log row in one transaction, 4) an audit event when the change breaks a frozen commitment.
func (s *productionScheduleSvcImpl) UpdateProductionScheduleLine(ctx context.Context, params domain.UpdateProductionScheduleLineParams) (*domain.ProductionScheduleLine, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.update_line")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductionScheduleLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		result, apiErr := s.updateProductionScheduleLineTx(ctx, identity, accountID, idempotencyKey.TypeID, params)
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}
		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// updateProductionScheduleLineTx is the started-phase body of UpdateProductionScheduleLine: the update, its deviation row, the frozen-week audit event and the idempotency cache commit together.
func (s *productionScheduleSvcImpl) updateProductionScheduleLineTx(
	ctx context.Context,
	identity *types.Identity,
	accountID, idempotencyKeyTypeID string,
	params domain.UpdateProductionScheduleLineParams,
) (*domain.ProductionScheduleLine, *apierror.APIError) {
	var result *domain.ProductionScheduleLine
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		schedule, apiErr := repo.Get(txCtx, domain.GetProductionScheduleParams{
			AccountID:  accountID,
			ScheduleID: params.ScheduleID,
		})
		if apiErr != nil {
			return apiErr
		}
		if apiErr := requireEditable(schedule); apiErr != nil {
			return apiErr
		}

		existing, apiErr := repo.GetLine(txCtx, accountID, params.LineID)
		if apiErr != nil {
			return apiErr
		}
		if existing.ProductionScheduleID != params.ScheduleID {
			return apierror.NewResourceNotFoundError("Schedule line not found.")
		}

		// A campaign builds something by definition. Zeroing one would leave a job on the plan that produces nothing while still holding its machine hours, its lots and its slot against a regenerate — the plan would read as busy for work nobody is doing.
		if params.Quantity != nil && *params.Quantity <= 0 {
			return apierror.NewValidationErrorWithParam("A campaign must build at least one unit. Delete the campaign to take it off the plan.", "quantity")
		}

		before := snapshotLine(existing)

		// The frozen test uses the week the line is in NOW and the week it is moving to: dragging a campaign out of the frozen week is as much a broken commitment as changing it in place.
		frozen := isWeekFrozen(schedule, existing.WeekStartDate)
		repoParams := domain.UpdateLineRepoParams{
			AccountID:       accountID,
			LineID:          params.LineID,
			MachineID:       params.MachineID,
			PlannedQuantity: params.Quantity,
			PlannedLots:     params.Lots,
			PlannedRunHours: params.RunHours,
			SequenceIndex:   params.SequenceIndex,
			StatusCode:      params.StatusCode,
			ReasonCode:      params.ReasonCode.ValuePtr(),
			ClearReasonCode: params.ReasonCode.IsClear(),
		}
		if params.WeekIndex != nil {
			if *params.WeekIndex < 0 || *params.WeekIndex >= schedule.HorizonWeeks {
				return apierror.NewValidationErrorWithParam("The week is outside this schedule's horizon.", "week_index")
			}
			targetWeekStart := weekStartFor(schedule, *params.WeekIndex)
			frozen = frozen || isWeekFrozen(schedule, targetWeekStart)
			repoParams.WeekIndex = params.WeekIndex
			repoParams.WeekStartDate = &targetWeekStart
		}

		if apiErr := requireFrozenReason(frozen, params.ReasonCode.ValuePtr()); apiErr != nil {
			return apiErr
		}

		if params.MachineID != nil {
			if _, apiErr := txSvc.repos.NewMachineRepo().Get(txCtx, domain.GetMachineParams{
				AccountID: accountID,
				MachineID: *params.MachineID,
			}); apiErr != nil {
				return apierror.NewValidationErrorWithParam("Unknown machine.", "machine_id")
			}
		}

		// Machine time and lot count follow the quantity unless the caller prices the campaign itself. Left alone, a resized campaign keeps the hours it was originally sized at, and the week's utilisation — the number a planner resizes a campaign against — goes on reporting work that is no longer planned.
		if params.Quantity != nil {
			if params.RunHours == nil {
				// The machine this edit is moving the campaign to, not the one it is leaving: the hours being booked are the new machine's.
				machineID := existing.MachineID
				if params.MachineID != nil {
					machineID = *params.MachineID
				}
				runHours, apiErr := txSvc.estimatedRunHours(txCtx, accountID, params.ScheduleID, existing.ItemID, machineID, *params.Quantity, existing.ProductionStepID)
				if apiErr != nil {
					return apiErr
				}
				// An item this version holds no rate for estimates to zero, which would throw away a rate somebody set by hand. Scaling what is on the line keeps it in proportion instead.
				if runHours == 0 && existing.PlannedRunHours > 0 && existing.PlannedQuantity > 0 {
					runHours = existing.PlannedRunHours * *params.Quantity / existing.PlannedQuantity
				}
				repoParams.PlannedRunHours = &runHours
			}
			if params.Lots == nil && existing.PlannedLotUnits > 0 {
				lots := safeconv.IntToInt32(scheduling.CountLots(*params.Quantity, existing.PlannedLotUnits))
				repoParams.PlannedLots = &lots
			}
		}

		updated, apiErr := repo.UpdateLine(txCtx, repoParams)
		if apiErr != nil {
			return apiErr
		}
		result = updated

		if apiErr := txSvc.recordDeviation(txCtx, schedule, before, snapshotLine(updated), existing.WeekStartDate,
			params.ReasonCode.ValuePtr(), params.ReasonNote, actorIDOf(identity)); apiErr != nil {
			return apiErr
		}

		if frozen {
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeProductionScheduleLine,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeProductionSchedule,
				RootResourceID:   schedule.ID,
				Changes:          audit.ComputeChanges(existing, updated),
			}); apiErr != nil {
				return apiErr
			}
		}

		return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKeyTypeID, result)
	})
	if apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}

// DeleteProductionScheduleLine removes a campaign and logs a deviation.
func (s *productionScheduleSvcImpl) DeleteProductionScheduleLine(ctx context.Context, params domain.DeleteProductionScheduleLineParams) *apierror.APIError {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.delete_line")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		schedule, apiErr := repo.Get(txCtx, domain.GetProductionScheduleParams{
			AccountID:  accountID,
			ScheduleID: params.ScheduleID,
		})
		if apiErr != nil {
			return apiErr
		}
		if apiErr := requireEditable(schedule); apiErr != nil {
			return apiErr
		}

		existing, apiErr := repo.GetLine(txCtx, accountID, params.LineID)
		if apiErr != nil {
			return apiErr
		}
		if existing.ProductionScheduleID != params.ScheduleID {
			return apierror.NewResourceNotFoundError("Schedule line not found.")
		}

		frozen := isWeekFrozen(schedule, existing.WeekStartDate)
		if apiErr := requireFrozenReason(frozen, params.ReasonCode); apiErr != nil {
			return apiErr
		}

		before := snapshotLine(existing)

		if apiErr := repo.DeleteLine(txCtx, accountID, params.LineID); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.recordDeviation(txCtx, schedule, before, nil, existing.WeekStartDate,
			params.ReasonCode, params.ReasonNote, actorIDOf(identity)); apiErr != nil {
			return apiErr
		}

		if frozen {
			return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionDelete,
				ResourceType:     constants.ObjectTypeProductionScheduleLine,
				ResourceID:       params.LineID,
				RootResourceType: constants.ObjectTypeProductionSchedule,
				RootResourceID:   schedule.ID,
				Changes:          audit.ComputeChanges(existing, nil),
			})
		}
		return nil
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// PublishProductionSchedule freezes the first weeks, snapshots the frozen counts, and supersedes whatever it replaces.
func (s *productionScheduleSvcImpl) PublishProductionSchedule(ctx context.Context, scheduleID string) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.publish")
	defer span.End()

	// Publish is an update, not its own action: it is the same authority as editing the plan, and a separate action would have to be granted everywhere separately.
	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	publishedBy := actorIDOf(identity)
	var publishedByID *string
	if publishedBy != "" {
		publishedByID = &publishedBy
	}

	return s.publishSchedule(ctx, identity.Target.AccountID, scheduleID, publishedByID)
}

// publishSchedule is the identity-free publish core. The RPC path resolves its actor and account through writeIdentity above; the cadence's auto-publish calls this directly with the trusted account ID carried by the outbox message and no human actor.
func (s *productionScheduleSvcImpl) publishSchedule(ctx context.Context, accountID, scheduleID string, publishedByID *string) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.publish_core")
	defer span.End()

	var result *domain.ProductionSchedule
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		schedule, apiErr := repo.Get(txCtx, domain.GetProductionScheduleParams{
			AccountID:  accountID,
			ScheduleID: scheduleID,
		})
		if apiErr != nil {
			return apiErr
		}
		if schedule.StatusCode != domain.ScheduleStatusDraft {
			return apierror.NewValidationError("Only a draft schedule can be published.")
		}

		// frozen_through_date is the last day of the frozen window. With frozen_weeks=1 that is the end of week 1, so every week-1 line freezes and nothing later does.
		frozenThrough := schedule.HorizonStartDate.AddDate(0, 0, int(schedule.FrozenWeeks)*7-1)

		if apiErr := repo.FreezeLines(txCtx, accountID, scheduleID, frozenThrough); apiErr != nil {
			return apiErr
		}

		// The counts are captured here and never recomputed: adherence has to keep the denominator it was committed to, even after lines are added or removed.
		totals, apiErr := repo.SumFrozenLines(txCtx, accountID, scheduleID, frozenThrough)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := repo.Publish(txCtx, accountID, scheduleID, frozenThrough, totals, publishedByID); apiErr != nil {
			return apiErr
		}

		// Anything else published over this horizon becomes history, pointed at its replacement rather than rewritten.
		supersededIDs, apiErr := repo.ListPublishedOverlapping(txCtx, accountID, scheduleID,
			schedule.HorizonStartDate, schedule.HorizonEndDate)
		if apiErr != nil {
			return apiErr
		}
		for _, oldID := range supersededIDs {
			if apiErr := repo.Supersede(txCtx, accountID, oldID, scheduleID); apiErr != nil {
				return apiErr
			}
		}

		published, apiErr := repo.Get(txCtx, domain.GetProductionScheduleParams{
			AccountID:  accountID,
			ScheduleID: scheduleID,
		})
		if apiErr != nil {
			return apiErr
		}
		result = published

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeProductionSchedule,
			ResourceID:   scheduleID,
			Changes:      audit.ComputeChanges(schedule, published),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// ArchiveProductionSchedule retires a version without deleting its history.
func (s *productionScheduleSvcImpl) ArchiveProductionSchedule(ctx context.Context, scheduleID string) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.archive")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	var result *domain.ProductionSchedule
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		schedule, apiErr := repo.Get(txCtx, domain.GetProductionScheduleParams{
			AccountID:  accountID,
			ScheduleID: scheduleID,
		})
		if apiErr != nil {
			return apiErr
		}
		if schedule.StatusCode == domain.ScheduleStatusArchived {
			return apierror.NewValidationError("This schedule is already archived.")
		}

		if apiErr := repo.SetStatus(txCtx, accountID, scheduleID, domain.ScheduleStatusArchived); apiErr != nil {
			return apiErr
		}

		archived, apiErr := repo.Get(txCtx, domain.GetProductionScheduleParams{
			AccountID:  accountID,
			ScheduleID: scheduleID,
		})
		if apiErr != nil {
			return apiErr
		}
		result = archived

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeProductionSchedule,
			ResourceID:   scheduleID,
			Changes:      audit.ComputeChanges(schedule, archived),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

// DeleteProductionSchedule removes a draft. Published versions must be archived.
func (s *productionScheduleSvcImpl) DeleteProductionSchedule(ctx context.Context, scheduleID string) *apierror.APIError {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.delete")
	defer span.End()

	identity, apiErr := s.writeIdentity(ctx, types.ActionDelete)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionScheduleSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionScheduleRepo()

		schedule, apiErr := repo.Get(txCtx, domain.GetProductionScheduleParams{
			AccountID:  accountID,
			ScheduleID: scheduleID,
		})
		if apiErr != nil {
			return apiErr
		}
		// A published version is the baseline attainment is measured against. Deleting it would silently erase the history of what was promised; archive instead.
		if schedule.StatusCode != domain.ScheduleStatusDraft {
			return apierror.NewValidationError("Only a draft schedule can be deleted. Archive published versions instead.")
		}

		if apiErr := repo.Delete(txCtx, accountID, scheduleID); apiErr != nil {
			return apiErr
		}

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProductionSchedule,
			ResourceID:   scheduleID,
			Changes:      audit.ComputeChanges(schedule, nil),
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// itemCountingUnitID is the unit an item is counted in, from its category's unit group.
//
// Empty rather than an error when the item has no category or the category no base unit: a lot with no unit is still a usable lot size, and the release resolves the unit again from the item before it writes a batch.
func (s *productionScheduleSvcImpl) itemCountingUnitID(ctx context.Context, accountID, itemID string) (string, *apierror.APIError) {
	item, apiErr := s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{AccountID: accountID, ItemID: itemID})
	if apiErr != nil {
		return "", apiErr
	}
	if item.ItemCategoryID == "" {
		return "", nil
	}
	unitID, _, apiErr := s.repos.NewItemRepo().GetCategoryBaseUnitID(ctx, item.ItemCategoryID)
	if apiErr != nil {
		return "", apiErr
	}
	return unitID, nil
}
