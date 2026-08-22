package service

import (
	"context"
	"fmt"
	"time"

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

var demandOverrideSvcTracer = tracing.GetTracer("core-service.demand_override_service")

type demandOverrideSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type DemandOverrideSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *DemandOverrideSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("demand override service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("demand override service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("demand override service: tx manager is required")
	}
	return nil
}

func NewDemandOverrideSvc(config *DemandOverrideSvcConfig) domain.DemandOverrideSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &demandOverrideSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *demandOverrideSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *demandOverrideSvcImpl) withTx(ctx context.Context, fn func(context.Context, *demandOverrideSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &demandOverrideSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// dayStart truncates to the calendar day. Override periods are whole days on both ends, so a time component would make an equality filter behave differently depending on which hour the row happened to be written at.
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func validateOverridePeriod(start, end time.Time) *apierror.APIError {
	if end.Before(start) {
		return apierror.NewValidationErrorWithParam("The period must end on or after it starts.", "period_ends_at")
	}
	return nil
}

func validateOverrideValue(typeCode string, value float64) *apierror.APIError {
	switch typeCode {
	case string(constants.DemandOverrideAdjustmentAbsolute):
		if value < 0 {
			return apierror.NewValidationErrorWithParam("An absolute override cannot be negative.", "value")
		}
	case string(constants.DemandOverrideAdjustmentDeltaPercent):
		// -100% zeroes the demand; anything below that would make it negative, which the solver would read as supply and plan against.
		if value < -100 {
			return apierror.NewValidationErrorWithParam("A percent override cannot reduce demand by more than 100%.", "value")
		}
	case string(constants.DemandOverrideAdjustmentDeltaUnits):
		// Negative unit deltas are legitimate: a cancelled program removes demand.
	default:
		return apierror.NewValidationErrorWithParam("Unknown override type.", "override_type_code")
	}
	return nil
}

func validateOverrideScope(scopeCode, typeCode string) *apierror.APIError {
	switch scopeCode {
	case string(constants.DemandOverrideScopeItem), string(constants.DemandOverrideScopeProductLine):
		return nil
	case string(constants.DemandOverrideScopeAccount):
		// An absolute value fanned out to every item would set the whole plan to one number; only relative adjustments make sense account-wide.
		if typeCode == string(constants.DemandOverrideAdjustmentAbsolute) {
			return apierror.NewValidationErrorWithParam("An account-wide override must be a delta, not an absolute value.", "override_type_code")
		}
		return nil
	default:
		return apierror.NewValidationErrorWithParam("Unknown override scope.", "scope_code")
	}
}

// ListDemandOverrideTypes returns the global override type taxonomy.
func (s *demandOverrideSvcImpl) ListDemandOverrideTypes(ctx context.Context) ([]*domain.DemandOverrideType, *apierror.APIError) {
	ctx, span := demandOverrideSvcTracer.Start(ctx, "service.demand_override.list_types")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDemandOverrides, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewDemandOverrideRepo().ListTypes(ctx)
}

// BatchGetDemandOverridesByIDs returns demand overrides by their IDs for include resolution.
func (s *demandOverrideSvcImpl) BatchGetDemandOverridesByIDs(ctx context.Context, ids []string) ([]*domain.DemandOverride, *apierror.APIError) {
	ctx, span := demandOverrideSvcTracer.Start(ctx, "service.demand_override.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDemandOverrides, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewDemandOverrideRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

// ListDemandOverrides returns a paginated list of demand overrides for the caller's account.
func (s *demandOverrideSvcImpl) ListDemandOverrides(ctx context.Context, params domain.ListDemandOverridesParams) (*domain.ListDemandOverridesResult, *apierror.APIError) {
	ctx, span := demandOverrideSvcTracer.Start(ctx, "service.demand_override.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDemandOverrides, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewDemandOverrideRepo().List(ctx, params)
}

// GetDemandOverride returns a single demand override by ID.
func (s *demandOverrideSvcImpl) GetDemandOverride(ctx context.Context, overrideID string) (*domain.DemandOverride, *apierror.APIError) {
	ctx, span := demandOverrideSvcTracer.Start(ctx, "service.demand_override.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDemandOverrides, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewDemandOverrideRepo().Get(ctx, domain.GetDemandOverrideParams{
		AccountID:  identity.Target.AccountID,
		OverrideID: overrideID,
	})
}

// CreateDemandOverride records demand the forecast cannot see. The scope reference is validated so an override can never silently match nothing.
//
// 1. Check identity, internal-actor status and the demand-overrides create permission.
// 2. Validate the scope code, type/value pair and period bounds.
// 3. Idempotently, within a transaction: validate the scope reference exists, insert the override, publish the audit event and cache the response.
func (s *demandOverrideSvcImpl) CreateDemandOverride(ctx context.Context, params domain.CreateDemandOverrideParams) (*domain.DemandOverride, *apierror.APIError) {
	ctx, span := demandOverrideSvcTracer.Start(ctx, "service.demand_override.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDemandOverrides, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := validateOverrideScope(params.ScopeCode, params.OverrideTypeCode); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := validateOverrideValue(params.OverrideTypeCode, params.Value); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	periodStart := dayStart(params.PeriodStartDate)
	periodEnd := dayStart(params.PeriodEndDate)
	if apiErr := validateOverridePeriod(periodStart, periodEnd); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	overrideID, apiErr := id.GenID(id.DemandOverrideIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	// An account-wide override has nothing to point at, so the ref is stamped here rather than asked of the caller.
	if params.ScopeCode == string(constants.DemandOverrideScopeAccount) && params.ScopeRefID == "" {
		params.ScopeRefID = accountID
	}

	effectiveFrom := time.Now().UTC()
	if params.EffectiveFrom != nil {
		effectiveFrom = *params.EffectiveFrom
	}
	isActive := true
	if params.IsActive != nil {
		isActive = *params.IsActive
	}
	createdByID := params.CreatedByID
	if createdByID == "" && identity.Actor != nil {
		createdByID = identity.Actor.ID
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.DemandOverride](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.DemandOverride
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *demandOverrideSvcImpl) *apierror.APIError {
			overrideRepo := txSvc.repos.NewDemandOverrideRepo()

			// An override pointing at an item or product line that does not exist would be silently ignored by the solver, so it is rejected at write time instead.
			if apiErr := txSvc.validateScopeRef(txCtx, accountID, params.ScopeCode, params.ScopeRefID); apiErr != nil {
				return apiErr
			}

			override := &domain.DemandOverride{
				AccountID:        accountID,
				ScopeCode:        params.ScopeCode,
				ScopeRefID:       params.ScopeRefID,
				PeriodStartDate:  periodStart,
				PeriodEndDate:    periodEnd,
				OverrideTypeCode: params.OverrideTypeCode,
				Value:            params.Value,
				UnitID:           params.UnitID,
				ReasonCode:       params.ReasonCode,
				Note:             params.Note,
				CreatedByID:      createdByID,
				EffectiveFrom:    effectiveFrom,
				ExpiresAt:        params.ExpiresAt,
				IsActive:         isActive,
			}

			created, apiErr := overrideRepo.Create(txCtx, overrideID, override)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeDemandOverride,
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

// validateScopeRef confirms the override actually points at something. Overrides are the only lever management has over the forecast, so one that silently matches nothing is worse than an error: the plan looks adjusted and is not.
func (s *demandOverrideSvcImpl) validateScopeRef(ctx context.Context, accountID, scopeCode, scopeRefID string) *apierror.APIError {
	switch scopeCode {
	case string(constants.DemandOverrideScopeItem):
		if _, apiErr := s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
			AccountID: accountID,
			ItemID:    scopeRefID,
		}); apiErr != nil {
			return apierror.NewValidationErrorWithParam("Unknown item.", "scope_ref_id")
		}
	case string(constants.DemandOverrideScopeProductLine):
		if _, apiErr := s.repos.NewProductLineRepo().Get(ctx, domain.GetProductLineParams{
			AccountID:     accountID,
			ProductLineID: scopeRefID,
		}); apiErr != nil {
			return apierror.NewValidationErrorWithParam("Unknown product line.", "scope_ref_id")
		}
	case string(constants.DemandOverrideScopeAccount):
		// The account is its own referent; the ref is stamped server-side so nothing else needs pointing at.
		if scopeRefID != accountID {
			return apierror.NewValidationErrorWithParam("An account-wide override cannot reference anything.", "scope_ref_id")
		}
	}
	return nil
}

// UpdateDemandOverride partially updates an override. Type and value are validated as a pair against the resulting row.
//
// 1. Check identity, internal-actor status and the demand-overrides update permission.
// 2. Idempotently, within a transaction: load the existing row, validate the effective type/value pair and period bounds, apply the update, publish the audit event and cache the response.
func (s *demandOverrideSvcImpl) UpdateDemandOverride(ctx context.Context, params domain.UpdateDemandOverrideParams) (*domain.DemandOverride, *apierror.APIError) {
	ctx, span := demandOverrideSvcTracer.Start(ctx, "service.demand_override.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDemandOverrides, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	params.AccountID = accountID

	if params.PeriodStartDate != nil {
		start := dayStart(*params.PeriodStartDate)
		params.PeriodStartDate = &start
	}
	if params.PeriodEndDate != nil {
		end := dayStart(*params.PeriodEndDate)
		params.PeriodEndDate = &end
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.DemandOverride](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.DemandOverride
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *demandOverrideSvcImpl) *apierror.APIError {
			overrideRepo := txSvc.repos.NewDemandOverrideRepo()

			existing, apiErr := overrideRepo.Get(txCtx, domain.GetDemandOverrideParams{
				AccountID:  accountID,
				OverrideID: params.OverrideID,
			})
			if apiErr != nil {
				return apiErr
			}

			old := *existing

			// Type and value validate against the resulting pair, not the supplied one: switching an existing +5000 units override to delta_percent has to be checked as a percent even though only the type was sent.
			effectiveType := existing.OverrideTypeCode
			if params.OverrideTypeCode != nil {
				effectiveType = *params.OverrideTypeCode
			}
			effectiveValue := existing.Value
			if params.Value != nil {
				effectiveValue = *params.Value
			}
			if apiErr := validateOverrideValue(effectiveType, effectiveValue); apiErr != nil {
				return apiErr
			}

			effectiveStart := existing.PeriodStartDate
			if params.PeriodStartDate != nil {
				effectiveStart = *params.PeriodStartDate
			}
			effectiveEnd := existing.PeriodEndDate
			if params.PeriodEndDate != nil {
				effectiveEnd = *params.PeriodEndDate
			}
			if apiErr := validateOverridePeriod(effectiveStart, effectiveEnd); apiErr != nil {
				return apiErr
			}

			updated, apiErr := overrideRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(&old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeDemandOverride,
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

// DeleteDemandOverride removes an override.
//
// 1. Check identity, internal-actor status and the demand-overrides delete permission.
// 2. Within a transaction: load the existing row, delete it and publish the audit event.
func (s *demandOverrideSvcImpl) DeleteDemandOverride(ctx context.Context, overrideID string) *apierror.APIError {
	ctx, span := demandOverrideSvcTracer.Start(ctx, "service.demand_override.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDemandOverrides, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *demandOverrideSvcImpl) *apierror.APIError {
		overrideRepo := txSvc.repos.NewDemandOverrideRepo()

		existing, apiErr := overrideRepo.Get(txCtx, domain.GetDemandOverrideParams{
			AccountID:  accountID,
			OverrideID: overrideID,
		})
		if apiErr != nil {
			return apiErr
		}

		if apiErr := overrideRepo.Delete(txCtx, domain.DeleteDemandOverrideParams{
			AccountID:  accountID,
			OverrideID: overrideID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(existing, nil)

		return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeDemandOverride,
			ResourceID:   overrideID,
			Changes:      changes,
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
