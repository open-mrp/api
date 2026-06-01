package service

import (
	"context"
	"fmt"
	"slices"

	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

type PropertySvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *PropertySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("property service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("property service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("property service: tx manager is required")
	}
	return nil
}

func NewPropertySvc(config *PropertySvcConfig) domain.PropertySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &propertySvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

type propertySvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

var propertySvcTracer = tracing.GetTracer("core-service.property_service")

func (s *propertySvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *propertySvcImpl) withTx(ctx context.Context, fn func(context.Context, *propertySvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &propertySvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *propertySvcImpl) BatchGetPropertiesByIDs(ctx context.Context, ids []string) ([]*domain.Property, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	meds := s.mediators()
	if apiErr := authorizeCatalogBatchRead(ctx, identity, span, meds, func() *apierror.APIError {
		return identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionRead)
	}); apiErr != nil {
		return nil, apiErr
	}
	if len(ids) == 0 {
		return nil, nil
	}

	properties, apiErr := s.repos.NewPropertyRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if err := s.populatePropertyAttributes(ctx, identity.Target.AccountID, properties); err != nil {
		return nil, tracing.Trace(span, err)
	}

	return properties, nil
}

func (s *propertySvcImpl) ListProperties(ctx context.Context, params domain.ListPropertiesParams, includes []string) (*domain.ListPropertiesResult, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	result, apiErr := s.repos.NewPropertyRepo().List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(includes, "attributes") {
		if err := s.populatePropertyAttributes(ctx, params.AccountID, result.Properties); err != nil {
			return nil, tracing.Trace(span, err)
		}
	}

	return result, nil
}

func (s *propertySvcImpl) GetProperty(ctx context.Context, propertyID string, includes []string) (*domain.Property, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	property, apiErr := s.repos.NewPropertyRepo().Get(ctx, domain.GetPropertyParams{
		PropertyID: propertyID,
		AccountID:  identity.Target.AccountID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(includes, "attributes") {
		if err := s.populatePropertyAttributes(ctx, identity.Target.AccountID, []*domain.Property{property}); err != nil {
			return nil, tracing.Trace(span, err)
		}
	}

	return property, nil
}

func (s *propertySvcImpl) populatePropertyAttributes(ctx context.Context, accountID string, properties []*domain.Property) *apierror.APIError {
	if len(properties) == 0 {
		return nil
	}

	propertyIDs := make([]string, len(properties))
	for i, p := range properties {
		propertyIDs[i] = p.ID
	}

	attributes, apiErr := s.repos.NewAttributeRepo().ListByPropertyIDs(ctx, accountID, propertyIDs)
	if apiErr != nil {
		return apiErr
	}

	attrsByProperty := make(map[string][]*domain.Attribute)
	for _, a := range attributes {
		attrsByProperty[a.PropertyID] = append(attrsByProperty[a.PropertyID], a)
	}

	for _, p := range properties {
		p.Attributes = attrsByProperty[p.ID]
	}

	return nil
}

func (s *propertySvcImpl) CreateProperty(ctx context.Context, params domain.CreatePropertyParams, includes []string) (*domain.Property, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	propertyID, apiErr := id.GenID(id.PropertyIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Property](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Property
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *propertySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPropertyRepo()

			created, apiErr := txRepo.Create(txCtx, propertyID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProperty,
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

		if slices.Contains(includes, "attributes") {
			if err := s.populatePropertyAttributes(ctx, params.AccountID, []*domain.Property{result}); err != nil {
				return nil, tracing.Trace(span, err)
			}
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *propertySvcImpl) UpdateProperty(ctx context.Context, params domain.UpdatePropertyParams, includes []string) (*domain.Property, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Property](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Property
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *propertySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPropertyRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetPropertyParams{
				PropertyID: params.PropertyID,
				AccountID:  params.AccountID,
			})
			if apiErr != nil {
				return apiErr
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
				ResourceType: constants.ObjectTypeProperty,
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

		if slices.Contains(includes, "attributes") {
			if err := s.populatePropertyAttributes(ctx, params.AccountID, []*domain.Property{result}); err != nil {
				return nil, tracing.Trace(span, err)
			}
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *propertySvcImpl) DeleteProperty(ctx context.Context, propertyID string) *apierror.APIError {
	ctx, span := propertySvcTracer.Start(ctx, "service.property.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProperties, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	property, apiErr := s.repos.NewPropertyRepo().Get(ctx, domain.GetPropertyParams{
		PropertyID: propertyID,
		AccountID:  accountID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeProperty, propertyID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This property has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *propertySvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewPropertyRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeProperty, property.ID, property); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.DeleteAttributesByPropertyID(txCtx, propertyID, accountID); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.Delete(txCtx, domain.DeletePropertyParams{
			PropertyID: propertyID,
			AccountID:  accountID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(property, (*domain.Property)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProperty,
			ResourceID:   property.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
}
