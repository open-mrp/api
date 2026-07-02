package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var serviceLevelSvcTracer = tracing.GetTracer("core-service.service_level_service")

type serviceLevelSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ServiceLevelSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ServiceLevelSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("service level service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("service level service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("service level service: tx manager is required")
	}
	return nil
}

func NewServiceLevelSvc(config *ServiceLevelSvcConfig) domain.ServiceLevelSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &serviceLevelSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *serviceLevelSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *serviceLevelSvcImpl) withTx(ctx context.Context, fn func(context.Context, *serviceLevelSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &serviceLevelSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *serviceLevelSvcImpl) ListServiceLevels(ctx context.Context, params domain.ListServiceLevelsParams) (*domain.ListServiceLevelsResult, *apierror.APIError) {
	ctx, span := serviceLevelSvcTracer.Start(ctx, "service.service_level.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkCarrierReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewServiceLevelRepo().List(ctx, params)
}

func (s *serviceLevelSvcImpl) GetServiceLevel(ctx context.Context, carrierID, serviceLevelID string) (*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := serviceLevelSvcTracer.Start(ctx, "service.service_level.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkCarrierReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	accountID := identity.Target.AccountID

	// Validate service level belongs to carrier
	isInCarrier, apiErr := s.repos.NewServiceLevelRepo().IsInCarrier(ctx, serviceLevelID, carrierID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !isInCarrier {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Service level not found."))
	}

	return s.repos.NewServiceLevelRepo().Get(ctx, accountID, serviceLevelID)
}

func (s *serviceLevelSvcImpl) CreateServiceLevel(ctx context.Context, params domain.CreateServiceLevelParams) (*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := serviceLevelSvcTracer.Start(ctx, "service.service_level.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	optionID, apiErr := id.GenID(id.ServiceLevelIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ServiceLevel](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ServiceLevel
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *serviceLevelSvcImpl) *apierror.APIError {
			// Validate carrier exists
			_, apiErr := txSvc.repos.NewCarrierRepo().Get(txCtx, domain.GetCarrierParams{AccountID: params.AccountID, CarrierID: params.CarrierID})
			if apiErr != nil {
				return apiErr
			}

			// Check code uniqueness within carrier
			serviceLevelRepo := txSvc.repos.NewServiceLevelRepo()
			exists, apiErr := serviceLevelRepo.ExistsByCodeInCarrier(txCtx, params.CarrierID, params.Code, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A service level with this code already exists in this carrier.", "code")
			}

			if params.IsDefault {
				if apiErr := serviceLevelRepo.ClearDefaultsForCarrier(txCtx, params.AccountID, params.CarrierID); apiErr != nil {
					return apiErr
				}
			}

			created, apiErr := serviceLevelRepo.Create(txCtx, optionID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeServiceLevel,
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

func (s *serviceLevelSvcImpl) UpdateServiceLevel(ctx context.Context, params domain.UpdateServiceLevelParams) (*domain.ServiceLevel, *apierror.APIError) {
	ctx, span := serviceLevelSvcTracer.Start(ctx, "service.service_level.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ServiceLevel](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ServiceLevel
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *serviceLevelSvcImpl) *apierror.APIError {
			serviceLevelRepo := txSvc.repos.NewServiceLevelRepo()

			// Validate service level belongs to carrier
			isInCarrier, apiErr := serviceLevelRepo.IsInCarrier(txCtx, params.ServiceLevelID, params.CarrierID)
			if apiErr != nil {
				return apiErr
			}
			if !isInCarrier {
				wasDeleted, deletedCheckErr := txSvc.repos.NewDeletedRecordRepo().Exists(txCtx, constants.DeletedRecordResourceTypeServiceLevel, params.ServiceLevelID)
				if deletedCheckErr != nil {
					return deletedCheckErr
				}
				if wasDeleted {
					return apierror.NewAlreadyDeletedError("This service level has already been deleted and can no longer be modified.")
				}
				return apierror.NewResourceNotFoundError("Service level not found.")
			}

			existing, apiErr := serviceLevelRepo.Get(txCtx, params.AccountID, params.ServiceLevelID)
			if apiErr != nil {
				return apiErr
			}

			if existing.AccountID == nil {
				return apierror.NewAuthorizationError("System-owned service levels cannot be updated.")
			}
			if *existing.AccountID != params.AccountID {
				return apierror.NewAuthorizationError("This service level is owned by another account and cannot be updated.")
			}

			// Check code uniqueness if code is being changed
			if params.Code != nil {
				exists, apiErr := serviceLevelRepo.ExistsByCodeInCarrier(txCtx, params.CarrierID, *params.Code, &params.ServiceLevelID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A service level with this code already exists in this carrier.", "code")
				}
			}

			if params.IsDefault != nil && *params.IsDefault {
				if apiErr := serviceLevelRepo.ClearDefaultsForCarrier(txCtx, params.AccountID, params.CarrierID); apiErr != nil {
					return apiErr
				}
			}

			updated, apiErr := serviceLevelRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(existing, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeServiceLevel,
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

func (s *serviceLevelSvcImpl) DeleteServiceLevel(ctx context.Context, carrierID, serviceLevelID string) *apierror.APIError {
	ctx, span := serviceLevelSvcTracer.Start(ctx, "service.service_level.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainCarriers, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	serviceLevelRepo := s.repos.NewServiceLevelRepo()

	// Validate service level belongs to carrier
	isInCarrier, apiErr := serviceLevelRepo.IsInCarrier(ctx, serviceLevelID, carrierID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isInCarrier {
		wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeServiceLevel, serviceLevelID)
		if deletedCheckErr != nil {
			return tracing.Trace(span, deletedCheckErr)
		}
		if wasDeleted {
			return tracing.Trace(span, apierror.NewAlreadyDeletedError("This service level has already been deleted and can no longer be modified."))
		}
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Service level not found."))
	}

	// Check if service level is a default — defaults cannot be deleted
	existing, apiErr := serviceLevelRepo.Get(ctx, accountID, serviceLevelID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeServiceLevel, serviceLevelID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This service level has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if existing.AccountID == nil {
		return tracing.Trace(span, apierror.NewAuthorizationError("System-owned service levels cannot be deleted."))
	}
	if *existing.AccountID != accountID {
		return tracing.Trace(span, apierror.NewAuthorizationError("This service level is owned by another account and cannot be deleted."))
	}
	if existing.IsDefault {
		return tracing.Trace(span, apierror.NewValidationError("Default service levels cannot be deleted."))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *serviceLevelSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeServiceLevel, existing.ID, existing); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewServiceLevelRepo().Delete(txCtx, accountID, serviceLevelID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(existing, (*domain.ServiceLevel)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeServiceLevel,
			ResourceID:   existing.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}
