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

var locationSvcTracer = tracing.GetTracer("core-service.location_service")

type locationSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type LocationSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *LocationSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("location service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("location service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("location service: tx manager is required")
	}
	return nil
}

func NewLocationSvc(config *LocationSvcConfig) domain.LocationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &locationSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *locationSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *locationSvcImpl) withTx(ctx context.Context, fn func(context.Context, *locationSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &locationSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *locationSvcImpl) ListLocations(ctx context.Context, params domain.ListLocationsParams) (*domain.ListLocationsResult, *apierror.APIError) {
	ctx, span := locationSvcTracer.Start(ctx, "service.location.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainLocations, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewLocationRepo().List(ctx, params)
}

func (s *locationSvcImpl) GetLocation(ctx context.Context, params domain.GetLocationParams) (*domain.Location, *apierror.APIError) {
	ctx, span := locationSvcTracer.Start(ctx, "service.location.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainLocations, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewLocationRepo().Get(ctx, params)
}

func (s *locationSvcImpl) CreateLocation(ctx context.Context, params domain.CreateLocationParams) (*domain.Location, *apierror.APIError) {
	ctx, span := locationSvcTracer.Start(ctx, "service.location.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainLocations, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	locationID, apiErr := id.GenID(id.LocationIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Location](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Location
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *locationSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewLocationRepo()

			// Validate parent exists in account if provided
			if params.ParentID != nil {
				inAccount, apiErr := txRepo.IsInAccount(txCtx, params.AccountID, *params.ParentID)
				if apiErr != nil {
					return apiErr
				}
				if !inAccount {
					return apierror.NewValidationErrorWithParam("Parent location not found.", "parent_id")
				}
			}

			// Validate all child IDs exist in account
			for _, childID := range params.ChildIDs {
				inAccount, apiErr := txRepo.IsInAccount(txCtx, params.AccountID, childID)
				if apiErr != nil {
					return apiErr
				}
				if !inAccount {
					return apierror.NewValidationErrorWithParam("Child location not found.", "child_ids")
				}
			}

			created, apiErr := txRepo.Create(txCtx, locationID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeLocation,
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

func (s *locationSvcImpl) UpdateLocation(ctx context.Context, params domain.UpdateLocationParams) (*domain.Location, *apierror.APIError) {
	ctx, span := locationSvcTracer.Start(ctx, "service.location.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainLocations, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Location](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Location
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *locationSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewLocationRepo()

			// Verify location is in account
			inAccount, apiErr := txRepo.IsInAccount(txCtx, params.AccountID, params.LocationID)
			if apiErr != nil {
				return apiErr
			}
			if !inAccount {
				return apierror.NewResourceNotFoundError("Location not found.")
			}

			old, apiErr := txRepo.Get(txCtx, domain.GetLocationParams{
				AccountID:  params.AccountID,
				LocationID: params.LocationID,
			})
			if apiErr != nil {
				return apiErr
			}

			params.ParentID = params.ParentID.BackfillUnsetPtr(old.ParentID)

			if params.ParentID.IsSet() {
				parentID, _ := params.ParentID.Value()
				if parentID == params.LocationID {
					return apierror.NewValidationErrorWithParam("A location cannot be its own parent.", "parent_id")
				}
				inAccount, apiErr := txRepo.IsInAccount(txCtx, params.AccountID, parentID)
				if apiErr != nil {
					return apiErr
				}
				if !inAccount {
					return apierror.NewValidationErrorWithParam("Parent location not found.", "parent_id")
				}
			}

			if params.ChildIDs.WasProvided() {
				childIDs := []string{}
				if params.ChildIDs.IsSet() {
					childIDs, _ = params.ChildIDs.Value()
				}
				for _, childID := range childIDs {
					inAccount, apiErr := txRepo.IsInAccount(txCtx, params.AccountID, childID)
					if apiErr != nil {
						return apiErr
					}
					if !inAccount {
						return apierror.NewValidationErrorWithParam("Child location not found.", "child_ids")
					}
				}
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeLocation,
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

func (s *locationSvcImpl) DeleteLocation(ctx context.Context, params domain.DeleteLocationParams) *apierror.APIError {
	ctx, span := locationSvcTracer.Start(ctx, "service.location.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainLocations, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewLocationRepo()

	// Verify location is in account and fetch for audit
	location, apiErr := repo.Get(ctx, domain.GetLocationParams{AccountID: params.AccountID, LocationID: params.LocationID})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeLocation, params.LocationID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This location has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	// Check for children
	childCount, apiErr := repo.CountChildren(ctx, params.AccountID, params.LocationID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if childCount > 0 {
		return tracing.Trace(span, apierror.NewValidationError("Cannot delete a location that has child locations. Please remove or reassign child locations first."))
	}

	// Delete in transaction
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *locationSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeLocation, location.ID, location); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewLocationRepo().Delete(txCtx, params); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(location, (*domain.Location)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeLocation,
			ResourceID:   location.ID,
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

func (s *locationSvcImpl) ListLocationTypes(ctx context.Context, params domain.ListLocationTypesParams) (*domain.ListLocationTypesResult, *apierror.APIError) {
	ctx, span := locationSvcTracer.Start(ctx, "service.location.list_types")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainLocations, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewLocationRepo().ListTypes(ctx, params)
}

func (s *locationSvcImpl) GetLocationType(ctx context.Context, params domain.GetLocationTypeParams) (*domain.LocationType, *apierror.APIError) {
	ctx, span := locationSvcTracer.Start(ctx, "service.location.get_type")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainLocations, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewLocationRepo().GetType(ctx, params.Identifier)
}

func (s *locationSvcImpl) BatchGetLocationsByIDs(ctx context.Context, ids []string) ([]*domain.Location, *apierror.APIError) {
	ctx, span := locationSvcTracer.Start(ctx, "service.location.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainLocations, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewLocationRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}
