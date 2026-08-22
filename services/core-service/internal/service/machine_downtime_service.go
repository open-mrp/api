package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var machineDowntimeSvcTracer = tracing.GetTracer("core-service.machine_downtime_service")

// maxDowntimeClockSkew allows a shop-floor tablet whose clock runs slightly fast to log "just now" without being rejected.
const maxDowntimeClockSkew = 5 * time.Minute

type machineDowntimeSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type MachineDowntimeSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *MachineDowntimeSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("machine downtime service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("machine downtime service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("machine downtime service: tx manager is required")
	}
	return nil
}

func NewMachineDowntimeSvc(config *MachineDowntimeSvcConfig) domain.MachineDowntimeSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &machineDowntimeSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *machineDowntimeSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *machineDowntimeSvcImpl) withTx(ctx context.Context, fn func(context.Context, *machineDowntimeSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &machineDowntimeSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// downtimeShiftDate derives the business day a stoppage belongs to. Until the shift calendar (production_shift) is wired in, this is the calendar date of the start, which is correct for every shift that does not cross midnight.
func downtimeShiftDate(startedAt time.Time) time.Time {
	return time.Date(startedAt.Year(), startedAt.Month(), startedAt.Day(), 0, 0, 0, 0, time.UTC)
}

// downtimeDuration returns the closed duration in seconds, or nil while the event is still open. Duration is materialized rather than computed on read so that aggregation queries can sum a column instead of a TIMESTAMPDIFF per row.
func downtimeDuration(startedAt time.Time, endedAt *time.Time) *int32 {
	if endedAt == nil {
		return nil
	}
	seconds := int32(endedAt.Sub(startedAt).Seconds())
	return &seconds
}

// maxDowntimeDuration bounds a single stoppage. A machine down for a year is a mistyped unit — "90" entered against days rather than minutes — and storing it would drag every availability figure it touches down with it.
const maxDowntimeDuration = 365 * 24 * time.Hour

// resolveDowntimeEnd turns a duration into the moment the machine started running again.
//
// The unit has to measure time. Accepting any unit would let a lot size be sent as a duration and silently become a number of seconds, which is exactly the class of error carrying the unit is meant to make impossible.
func (s *machineDowntimeSvcImpl) resolveDowntimeEnd(ctx context.Context, accountID string, startedAt time.Time, duration domain.DowntimeDurationInput) (*time.Time, *apierror.APIError) {
	value, err := decimal.NewFromString(strings.TrimSpace(duration.Value))
	if err != nil {
		return nil, apierror.NewValidationErrorWithParam("The downtime duration is not a number.", "duration")
	}
	if value.LessThanOrEqual(decimal.Zero) {
		return nil, apierror.NewValidationErrorWithParam("The downtime duration must be greater than zero.", "duration")
	}

	factors, apiErr := s.repos.NewUnitConversionRepo().GetUnitFactors(ctx, accountID, []string{duration.UnitID})
	if apiErr != nil {
		return nil, apiErr
	}
	unit, ok := factors[duration.UnitID]
	if !ok {
		return nil, apierror.NewValidationErrorWithParam("Unknown unit for the downtime duration.", "duration")
	}
	if unit.DimensionCode != string(constants.UnitTypeTime) {
		return nil, apierror.NewValidationErrorWithParam("The downtime duration must be expressed in a unit of time.", "duration")
	}

	// Seconds is the base unit of the time dimension, which is also what the event stores its duration in.
	seconds := unit.ToBase(value)
	if seconds.LessThanOrEqual(decimal.Zero) {
		return nil, apierror.NewValidationErrorWithParam("The downtime duration must be greater than zero.", "duration")
	}
	elapsed := time.Duration(seconds.InexactFloat64() * float64(time.Second))
	if elapsed > maxDowntimeDuration {
		return nil, apierror.NewValidationErrorWithParam("The downtime duration is longer than a year. Check the unit.", "duration")
	}

	endedAt := startedAt.Add(elapsed)
	return &endedAt, nil
}

func validateDowntimeWindow(startedAt time.Time, endedAt *time.Time) *apierror.APIError {
	if startedAt.After(time.Now().UTC().Add(maxDowntimeClockSkew)) {
		return apierror.NewValidationErrorWithParam("Downtime cannot start in the future.", "started_at")
	}
	if endedAt != nil && !endedAt.After(startedAt) {
		return apierror.NewValidationErrorWithParam("Downtime must end after it starts.", "ended_at")
	}
	return nil
}

// ListDowntimeReasons returns the global downtime reason taxonomy, ordered for display.
func (s *machineDowntimeSvcImpl) ListDowntimeReasons(ctx context.Context) ([]*domain.MachineDowntimeReason, *apierror.APIError) {
	ctx, span := machineDowntimeSvcTracer.Start(ctx, "service.machine_downtime.list_reasons")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachineDowntime, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewMachineDowntimeRepo().ListReasons(ctx)
}

// BatchGetDowntimeEventsByIDs returns downtime events by their IDs for include resolution.
func (s *machineDowntimeSvcImpl) BatchGetDowntimeEventsByIDs(ctx context.Context, ids []string) ([]*domain.MachineDowntimeEvent, *apierror.APIError) {
	ctx, span := machineDowntimeSvcTracer.Start(ctx, "service.machine_downtime.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachineDowntime, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewMachineDowntimeRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

// ListDowntimeEvents returns a paginated list of downtime events for the caller's account.
func (s *machineDowntimeSvcImpl) ListDowntimeEvents(ctx context.Context, params domain.ListMachineDowntimeEventsParams) (*domain.ListMachineDowntimeEventsResult, *apierror.APIError) {
	ctx, span := machineDowntimeSvcTracer.Start(ctx, "service.machine_downtime.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachineDowntime, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewMachineDowntimeRepo().List(ctx, params)
}

// GetDowntimeEvent returns a single downtime event by ID.
func (s *machineDowntimeSvcImpl) GetDowntimeEvent(ctx context.Context, eventID string) (*domain.MachineDowntimeEvent, *apierror.APIError) {
	ctx, span := machineDowntimeSvcTracer.Start(ctx, "service.machine_downtime.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachineDowntime, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewMachineDowntimeRepo().Get(ctx, domain.GetMachineDowntimeEventParams{
		AccountID: identity.Target.AccountID,
		EventID:   eventID,
	})
}

// CreateDowntimeEvent logs a stoppage. Department and production step are resolved from the machine; an open event (no end) records that the machine is still down.
//
// The flow, under an idempotency key so a retried POST cannot double-log the stoppage:
//  1. Validate the downtime window and the reason code against the taxonomy.
//  2. Reject a second open event for a machine that is already down.
//  3. Persist the event with its duration materialized when already closed.
//  4. Publish the audit create event and cache the response in the same transaction.
func (s *machineDowntimeSvcImpl) CreateDowntimeEvent(ctx context.Context, params domain.CreateMachineDowntimeEventParams) (*domain.MachineDowntimeEvent, *apierror.APIError) {
	ctx, span := machineDowntimeSvcTracer.Start(ctx, "service.machine_downtime.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachineDowntime, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// A duration is the other way of saying when the stoppage ended, so it is resolved into one before anything downstream sees the event.
	if params.Duration != nil {
		if params.EndedAt != nil {
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Send either an end time or a duration, not both.", "duration"))
		}
		endedAt, apiErr := s.resolveDowntimeEnd(ctx, params.AccountID, params.StartedAt, *params.Duration)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		params.EndedAt = endedAt
	}

	if apiErr := validateDowntimeWindow(params.StartedAt, params.EndedAt); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	eventID, apiErr := id.GenID(id.MachineDowntimeEventIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.ReportedByID == "" && identity.Actor != nil {
		params.ReportedByID = identity.Actor.ID
	}
	sourceCode := domain.MachineDowntimeSourceManual
	if params.SourceCode != nil && *params.SourceCode != "" {
		sourceCode = *params.SourceCode
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.MachineDowntimeEvent](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.MachineDowntimeEvent
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *machineDowntimeSvcImpl) *apierror.APIError {
			downtimeRepo := txSvc.repos.NewMachineDowntimeRepo()

			// The reason taxonomy is what maps a stoppage onto an OEE term, so an unknown code has to fail loudly rather than silently land outside every bucket.
			if _, apiErr := downtimeRepo.GetReason(txCtx, params.ReasonCode); apiErr != nil {
				return apierror.NewValidationErrorWithParam("Unknown downtime reason.", "reason_code")
			}

			machine, apiErr := txSvc.repos.NewMachineRepo().Get(txCtx, domain.GetMachineParams{
				AccountID: params.AccountID,
				MachineID: params.MachineID,
			})
			if apiErr != nil {
				return apiErr
			}

			// Two concurrent open events for one machine would double-count its downtime against the same wall-clock window and push Availability below zero.
			if params.EndedAt == nil {
				open, apiErr := downtimeRepo.GetOpenForMachine(txCtx, params.AccountID, params.MachineID)
				if apiErr != nil {
					return apiErr
				}
				if open != nil {
					return apierror.NewConflictErrorWithParam("This machine already has an open downtime event. Close it before logging another.", "machine_id")
				}
			}

			event := &domain.MachineDowntimeEvent{
				AccountID:        params.AccountID,
				MachineID:        params.MachineID,
				DepartmentID:     machine.DepartmentID,
				ProductionStepID: machine.ProductionStepID,
				ReasonCode:       params.ReasonCode,
				StartedAt:        params.StartedAt,
				EndedAt:          params.EndedAt,
				DurationSeconds:  downtimeDuration(params.StartedAt, params.EndedAt),
				ShiftDate:        downtimeShiftDate(params.StartedAt),
				ItemID:           params.ItemID,
				ProductionRunID:  params.ProductionRunID,
				BatchID:          params.BatchID,
				Note:             params.Note,
				ReportedByID:     params.ReportedByID,
				SourceCode:       sourceCode,
			}

			created, apiErr := downtimeRepo.Create(txCtx, eventID, event)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeMachineDowntimeEvent,
				ResourceID:   created.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// UpdateDowntimeEvent closes or reclassifies an event.
//
// The flow, under an idempotency key so a retried PATCH replays its cached response:
//  1. Load the event and apply the requested changes, honoring the explicit clear flags for the nullable columns.
//  2. Re-validate the downtime window and re-materialize the duration.
//  3. Reject a reopen that would collide with another open event on the machine.
//  4. Persist, publish the audit update event, and cache the response in the same transaction.
func (s *machineDowntimeSvcImpl) UpdateDowntimeEvent(ctx context.Context, params domain.UpdateMachineDowntimeEventParams) (*domain.MachineDowntimeEvent, *apierror.APIError) {
	ctx, span := machineDowntimeSvcTracer.Start(ctx, "service.machine_downtime.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachineDowntime, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.MachineDowntimeEvent](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.MachineDowntimeEvent
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *machineDowntimeSvcImpl) *apierror.APIError {
			downtimeRepo := txSvc.repos.NewMachineDowntimeRepo()

			existing, apiErr := downtimeRepo.Get(txCtx, domain.GetMachineDowntimeEventParams{
				AccountID: accountID,
				EventID:   params.EventID,
			})
			if apiErr != nil {
				return apiErr
			}

			old := *existing

			if params.ReasonCode != nil {
				if _, apiErr := downtimeRepo.GetReason(txCtx, *params.ReasonCode); apiErr != nil {
					return apierror.NewValidationErrorWithParam("Unknown downtime reason.", "reason_code")
				}
				existing.ReasonCode = *params.ReasonCode
			}
			if params.StartedAt != nil {
				existing.StartedAt = *params.StartedAt
				existing.ShiftDate = downtimeShiftDate(*params.StartedAt)
			}
			// Moving the event re-resolves the department and step from the new machine. Leaving the old ones would charge the stoppage to a room the machine is not in, which is the drilldown OEE is read through.
			if params.MachineID != nil && *params.MachineID != existing.MachineID {
				machine, apiErr := txSvc.repos.NewMachineRepo().Get(txCtx, domain.GetMachineParams{
					AccountID: accountID,
					MachineID: *params.MachineID,
				})
				if apiErr != nil {
					return apiErr
				}
				existing.MachineID = machine.ID
				existing.DepartmentID = machine.DepartmentID
				existing.ProductionStepID = machine.ProductionStepID
			}
			// A cleared EndedAt reopens an event closed by mistake; clear has to be distinguishable from unset because an unset EndedAt means "leave unchanged". The other Clearable fields null their columns for the same reason.
			if params.EndedAt.IsClear() {
				existing.EndedAt = nil
			} else if endedAt := params.EndedAt.ValuePtr(); endedAt != nil {
				existing.EndedAt = endedAt
			}
			// A duration restates the end relative to the start, so it is applied after any new start and refuses to compete with an end time sent in the same request.
			if params.Duration.IsClear() {
				if params.EndedAt.WasProvided() {
					return apierror.NewValidationErrorWithParam("Send either an end time or a duration, not both.", "duration")
				}
				existing.EndedAt = nil
			} else if duration := params.Duration.ValuePtr(); duration != nil {
				if params.EndedAt.WasProvided() {
					return apierror.NewValidationErrorWithParam("Send either an end time or a duration, not both.", "duration")
				}
				endedAt, apiErr := txSvc.resolveDowntimeEnd(txCtx, accountID, existing.StartedAt, *duration)
				if apiErr != nil {
					return apiErr
				}
				existing.EndedAt = endedAt
			}
			if params.ItemID.IsClear() {
				existing.ItemID = nil
			} else if itemID := params.ItemID.ValuePtr(); itemID != nil {
				existing.ItemID = itemID
			}
			if params.ProductionRunID.IsClear() {
				existing.ProductionRunID = nil
			} else if runID := params.ProductionRunID.ValuePtr(); runID != nil {
				existing.ProductionRunID = runID
			}
			if params.BatchID.IsClear() {
				existing.BatchID = nil
			} else if batchID := params.BatchID.ValuePtr(); batchID != nil {
				existing.BatchID = batchID
			}
			if params.Note.IsClear() {
				existing.Note = nil
			} else if note := params.Note.ValuePtr(); note != nil {
				existing.Note = note
			}

			if apiErr := validateDowntimeWindow(existing.StartedAt, existing.EndedAt); apiErr != nil {
				return apiErr
			}
			existing.DurationSeconds = downtimeDuration(existing.StartedAt, existing.EndedAt)

			// An open event must be the only open event on its machine — whether it got there by being reopened or by being moved onto a machine that is already down. Two would double-count the same wall-clock window and drive Availability below zero.
			if existing.EndedAt == nil && (old.EndedAt != nil || existing.MachineID != old.MachineID) {
				open, apiErr := downtimeRepo.GetOpenForMachine(txCtx, accountID, existing.MachineID)
				if apiErr != nil {
					return apiErr
				}
				if open != nil && open.ID != existing.ID {
					return apierror.NewConflictErrorWithParam("This machine already has an open downtime event.", "machine_id")
				}
			}

			updated, apiErr := downtimeRepo.Update(txCtx, existing)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(&old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeMachineDowntimeEvent,
				ResourceID:   updated.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// DeleteDowntimeEvent removes a mis-logged event, publishing an audit delete event with the full change set in the same transaction.
func (s *machineDowntimeSvcImpl) DeleteDowntimeEvent(ctx context.Context, eventID string) *apierror.APIError {
	ctx, span := machineDowntimeSvcTracer.Start(ctx, "service.machine_downtime.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachineDowntime, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *machineDowntimeSvcImpl) *apierror.APIError {
		downtimeRepo := txSvc.repos.NewMachineDowntimeRepo()

		existing, apiErr := downtimeRepo.Get(txCtx, domain.GetMachineDowntimeEventParams{
			AccountID: accountID,
			EventID:   eventID,
		})
		if apiErr != nil {
			return apiErr
		}

		if apiErr := downtimeRepo.Delete(txCtx, domain.DeleteMachineDowntimeEventParams{
			AccountID: accountID,
			EventID:   eventID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(existing, nil)

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeMachineDowntimeEvent,
			ResourceID:   eventID,
			Changes:      changes,
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
