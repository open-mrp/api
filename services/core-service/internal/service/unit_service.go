package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var unitSvcTracer = tracing.GetTracer("core-service.unit_service")

type unitSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type UnitSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *UnitSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("unit service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("unit service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("unit service: tx manager is required")
	}
	return nil
}

func NewUnitSvc(config *UnitSvcConfig) domain.UnitSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &unitSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *unitSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *unitSvcImpl) withTx(ctx context.Context, fn func(context.Context, *unitSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &unitSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *unitSvcImpl) ListUnits(ctx context.Context, params domain.ListUnitsParams) (*domain.ListUnitsResult, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainUnits, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = *identity.TargetAccountID

	return s.repos.NewUnitRepo().List(ctx, params)
}

func (s *unitSvcImpl) GetUnit(ctx context.Context, unitID string) (*domain.Unit, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainUnits, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	return s.repos.NewUnitRepo().Get(ctx, domain.GetUnitParams{
		AccountID: *identity.TargetAccountID,
		UnitID:    unitID,
	})
}

func (s *unitSvcImpl) CreateUnit(ctx context.Context, params domain.CreateUnitParams) (*domain.Unit, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainUnits, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	unitID, apiErr := id.GenID(id.UnitIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = *identity.TargetAccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:         identity.Actor.ID,
		IdentityType:    identity.Type,
		TargetAccountID: identity.TargetAccountID,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Unit](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Unit
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewUnitRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A unit with this name already exists.", "name")
			}

			exists, apiErr = txRepo.ExistsByAbbreviation(txCtx, params.AccountID, params.Abbreviation, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A unit with this abbreviation already exists.", "abbreviation")
			}

			created, apiErr := txRepo.Create(txCtx, unitID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

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

func (s *unitSvcImpl) UpdateUnit(ctx context.Context, params domain.UpdateUnitParams) (*domain.Unit, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainUnits, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = *identity.TargetAccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, &domain.RequestIdentity{
		ActorID:         identity.Actor.ID,
		IdentityType:    identity.Type,
		TargetAccountID: identity.TargetAccountID,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Unit](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Unit
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewUnitRepo()

			unit, apiErr := txRepo.Get(txCtx, domain.GetUnitParams{AccountID: params.AccountID, UnitID: params.UnitID})
			if apiErr != nil {
				return apiErr
			}
			if unit.AccountID == nil {
				return apierror.NewValidationError("System units cannot be modified.")
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.UnitID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A unit with this name already exists.", "name")
				}
			}

			if params.Abbreviation != nil {
				exists, apiErr := txRepo.ExistsByAbbreviation(txCtx, params.AccountID, *params.Abbreviation, &params.UnitID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A unit with this abbreviation already exists.", "abbreviation")
				}
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

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

func (s *unitSvcImpl) DeleteUnit(ctx context.Context, unitID string) *apierror.APIError {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := types.CheckIsInternalActor(identity); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := types.CheckHasPermission(identity, types.PermissionDomainUnits, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if identity.TargetAccountID == nil {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountID := *identity.TargetAccountID

	unit, apiErr := s.repos.NewUnitRepo().Get(ctx, domain.GetUnitParams{AccountID: accountID, UnitID: unitID})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if unit.AccountID == nil {
		return tracing.Trace(span, apierror.NewValidationError("System units cannot be deleted."))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitSvcImpl) *apierror.APIError {
		return txSvc.repos.NewUnitRepo().Delete(txCtx, domain.DeleteUnitParams{
			AccountID: accountID,
			UnitID:    unitID,
		})
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
