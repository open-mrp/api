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

var scanningStationSvcTracer = tracing.GetTracer("core-service.scanning_station_service")

type scanningStationSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ScanningStationSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ScanningStationSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("scanning station service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("scanning station service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("scanning station service: tx manager is required")
	}
	return nil
}

func NewScanningStationSvc(config *ScanningStationSvcConfig) domain.ScanningStationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &scanningStationSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *scanningStationSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *scanningStationSvcImpl) withTx(ctx context.Context, fn func(context.Context, *scanningStationSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &scanningStationSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *scanningStationSvcImpl) BatchGetScanningStationsByIDs(ctx context.Context, ids []string) ([]*domain.ScanningStation, *apierror.APIError) {
	ctx, span := scanningStationSvcTracer.Start(ctx, "service.scanning_station.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	return s.repos.NewScanningStationRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

func (s *scanningStationSvcImpl) ListScanningStations(ctx context.Context, params domain.ListScanningStationsParams) (*domain.ListScanningStationsResult, *apierror.APIError) {
	ctx, span := scanningStationSvcTracer.Start(ctx, "service.scanning_station.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewScanningStationRepo().List(ctx, params)
}

func (s *scanningStationSvcImpl) GetScanningStation(ctx context.Context, params domain.GetScanningStationParams) (*domain.ScanningStation, *apierror.APIError) {
	ctx, span := scanningStationSvcTracer.Start(ctx, "service.scanning_station.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewScanningStationRepo().Get(ctx, params)
}

func (s *scanningStationSvcImpl) CreateScanningStation(ctx context.Context, params domain.CreateScanningStationParams) (*domain.ScanningStation, *apierror.APIError) {
	ctx, span := scanningStationSvcTracer.Start(ctx, "service.scanning_station.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !constants.ScanningStationType(params.Type).IsValid() {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Invalid scanning station type.", "type"))
	}

	scanningStationID, apiErr := id.GenID(id.ScanningStationIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ScanningStation](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ScanningStation
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *scanningStationSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewScanningStationRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A scanning station with this name already exists.", "name")
			}

			created, apiErr := txRepo.Create(txCtx, scanningStationID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeScanningStation,
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

func (s *scanningStationSvcImpl) UpdateScanningStation(ctx context.Context, params domain.UpdateScanningStationParams) (*domain.ScanningStation, *apierror.APIError) {
	ctx, span := scanningStationSvcTracer.Start(ctx, "service.scanning_station.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ScanningStation](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ScanningStation
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *scanningStationSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewScanningStationRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetScanningStationParams{
				AccountID:         params.AccountID,
				ScanningStationID: params.ScanningStationID,
			})
			if apiErr != nil {
				return apiErr
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.ScanningStationID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A scanning station with this name already exists.", "name")
				}
			}

			params.Notes = params.Notes.BackfillUnsetPtr(old.Notes)

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeScanningStation,
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

func (s *scanningStationSvcImpl) DeleteScanningStation(ctx context.Context, scanningStationID string) *apierror.APIError {
	ctx, span := scanningStationSvcTracer.Start(ctx, "service.scanning_station.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	scanningStation, apiErr := s.repos.NewScanningStationRepo().Get(ctx, domain.GetScanningStationParams{
		AccountID:         accountID,
		ScanningStationID: scanningStationID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeScanningStation, scanningStationID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(
					span,
					apierror.NewAlreadyDeletedError("This scanning station has already been deleted and can no longer be modified."),
				)
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *scanningStationSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeScanningStation, scanningStation.ID, scanningStation); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewScanningStationRepo().Delete(txCtx, domain.DeleteScanningStationParams{
			AccountID:         accountID,
			ScanningStationID: scanningStationID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(scanningStation, (*domain.ScanningStation)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeScanningStation,
			ResourceID:   scanningStation.ID,
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

func (s *scanningStationSvcImpl) ConnectProductionStepsByName(ctx context.Context, params domain.ConnectProductionStepsByNameParams) *apierror.APIError {
	ctx, span := scanningStationSvcTracer.Start(ctx, "service.scanning_station.connect_production_steps_by_name")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainScanningStations, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	old, apiErr := s.repos.NewScanningStationRepo().Get(ctx, domain.GetScanningStationParams{
		AccountID:         accountID,
		ScanningStationID: params.ScanningStationID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *scanningStationSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewScanningStationRepo()

		if apiErr := txRepo.ConnectProductionStepsByName(txCtx, accountID, params.ScanningStationID, params.Name); apiErr != nil {
			return apiErr
		}

		updated, apiErr := txRepo.Get(txCtx, domain.GetScanningStationParams{
			AccountID:         accountID,
			ScanningStationID: params.ScanningStationID,
		})
		if apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(old, updated)

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeScanningStation,
			ResourceID:   updated.ID,
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
