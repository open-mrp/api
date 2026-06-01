package service

import (
	"context"
	"fmt"
	"strings"

	types "github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

type AttributeSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *AttributeSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("attribute service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("attribute service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("attribute service: tx manager is required")
	}
	return nil
}

func NewAttributeSvc(config *AttributeSvcConfig) domain.AttributeSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &attributeSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

type attributeSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

var attributeSvcTracer = tracing.GetTracer("core-service.attribute_service")

func (s *attributeSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *attributeSvcImpl) withTx(ctx context.Context, fn func(context.Context, *attributeSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &attributeSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *attributeSvcImpl) checkPropertyInAccount(ctx context.Context, accountID, propertyID string) *apierror.APIError {
	inAccount, apiErr := s.repos.NewPropertyRepo().IsInAccount(ctx, accountID, propertyID)
	if apiErr != nil {
		return apiErr
	}
	if !inAccount {
		return apierror.NewResourceNotFoundError("Property not found.")
	}
	return nil
}

func (s *attributeSvcImpl) ListAttributes(ctx context.Context, params domain.ListAttributesParams) (*domain.ListAttributesResult, *apierror.APIError) {
	ctx, span := attributeSvcTracer.Start(ctx, "service.attribute.list")
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

	params.AccountID = identity.Target.AccountID

	if apiErr := s.checkPropertyInAccount(ctx, params.AccountID, params.PropertyID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewAttributeRepo().List(ctx, params)
}

func (s *attributeSvcImpl) BatchGetAttributesByIDs(ctx context.Context, ids []string) ([]*domain.Attribute, *apierror.APIError) {
	ctx, span := attributeSvcTracer.Start(ctx, "service.attribute.batch_get_by_ids")
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
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewAttributeRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

func (s *attributeSvcImpl) GetAttribute(ctx context.Context, propertyID, attributeID string) (*domain.Attribute, *apierror.APIError) {
	ctx, span := attributeSvcTracer.Start(ctx, "service.attribute.get")
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

	accountID := identity.Target.AccountID

	if apiErr := s.checkPropertyInAccount(ctx, accountID, propertyID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewAttributeRepo().Get(ctx, domain.GetAttributeParams{
		AttributeID: attributeID,
		PropertyID:  propertyID,
		AccountID:   accountID,
	})
}

func (s *attributeSvcImpl) CreateAttribute(ctx context.Context, params domain.CreateAttributeParams) (*domain.Attribute, *apierror.APIError) {
	ctx, span := attributeSvcTracer.Start(ctx, "service.attribute.create")
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

	attributeID, apiErr := id.GenID(id.AttributeIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	params.Value = strings.TrimSpace(params.Value)

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Attribute](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Attribute
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *attributeSvcImpl) *apierror.APIError {
			if apiErr := txSvc.checkPropertyInAccount(txCtx, params.AccountID, params.PropertyID); apiErr != nil {
				return apiErr
			}

			txRepo := txSvc.repos.NewAttributeRepo()

			exists, apiErr := txRepo.ExistsByValueInAccount(txCtx, params.AccountID, params.Value, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam(fmt.Sprintf("Attribute %s already exists.", params.Value), "value")
			}

			count, apiErr := txRepo.CountByProperty(txCtx, params.PropertyID, params.AccountID)
			if apiErr != nil {
				return apiErr
			}
			nextOrder := safeconv.Int64ToInt32(count) + 1

			if params.SortOrder <= 0 {
				params.SortOrder = nextOrder
			} else if params.SortOrder > nextOrder {
				return apierror.NewValidationErrorWithParam(
					fmt.Sprintf("Order %d is out of range. The maximum allowed order is %d.", params.SortOrder, nextOrder),
					"sort_order",
				)
			} else {
				if apiErr := txRepo.ShiftOrdersUp(txCtx, params.PropertyID, params.AccountID, params.SortOrder); apiErr != nil {
					return apiErr
				}
			}

			created, apiErr := txRepo.Create(txCtx, attributeID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeAttribute,
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

func (s *attributeSvcImpl) UpdateAttribute(ctx context.Context, params domain.UpdateAttributeParams) (*domain.Attribute, *apierror.APIError) {
	ctx, span := attributeSvcTracer.Start(ctx, "service.attribute.update")
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

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Attribute](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		if params.Value != nil {
			trimmed := strings.TrimSpace(*params.Value)
			if trimmed == "" {
				return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Attribute must have a name.", "value"))
			}
			params.Value = &trimmed
		}

		var result *domain.Attribute
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *attributeSvcImpl) *apierror.APIError {
			if apiErr := txSvc.checkPropertyInAccount(txCtx, params.AccountID, params.PropertyID); apiErr != nil {
				return apiErr
			}

			txRepo := txSvc.repos.NewAttributeRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetAttributeParams{
				AttributeID: params.AttributeID,
				PropertyID:  params.PropertyID,
				AccountID:   params.AccountID,
			})
			if apiErr != nil {
				return apiErr
			}

			if params.Value != nil {
				exists, apiErr := txRepo.ExistsByValueInAccount(txCtx, params.AccountID, *params.Value, &params.AttributeID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam(fmt.Sprintf("Attribute name \"%s\" already taken.", *params.Value), "value")
				}
			}

			if params.SortOrder != nil && *params.SortOrder != old.SortOrder {
				newOrder := *params.SortOrder
				count, apiErr := txRepo.CountByProperty(txCtx, params.PropertyID, params.AccountID)
				if apiErr != nil {
					return apiErr
				}
				maxOrder := safeconv.Int64ToInt32(count)

				if newOrder > maxOrder {
					return apierror.NewValidationErrorWithParam(
						fmt.Sprintf("Order %d is out of range. The maximum allowed order is %d.", newOrder, maxOrder),
						"sort_order",
					)
				}

				if newOrder < old.SortOrder {
					if apiErr := txRepo.ShiftOrdersUpBounded(txCtx, params.PropertyID, params.AccountID, newOrder, old.SortOrder); apiErr != nil {
						return apiErr
					}
				} else {
					if apiErr := txRepo.ShiftOrdersDownBounded(txCtx, params.PropertyID, params.AccountID, old.SortOrder, newOrder); apiErr != nil {
						return apiErr
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
				ResourceType: constants.ObjectTypeAttribute,
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

func (s *attributeSvcImpl) DeleteAttribute(ctx context.Context, params domain.DeleteAttributeParams) *apierror.APIError {
	ctx, span := attributeSvcTracer.Start(ctx, "service.attribute.delete")
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

	params.AccountID = identity.Target.AccountID

	if apiErr := s.checkPropertyInAccount(ctx, params.AccountID, params.PropertyID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	attribute, apiErr := s.repos.NewAttributeRepo().Get(ctx, domain.GetAttributeParams(params))
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeAttribute, params.AttributeID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This attribute has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *attributeSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeAttribute, attribute.ID, attribute); apiErr != nil {
			return apiErr
		}

		txRepo := txSvc.repos.NewAttributeRepo()

		if apiErr := txRepo.Delete(txCtx, params); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.ShiftOrdersDown(txCtx, params.PropertyID, params.AccountID, attribute.SortOrder); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(attribute, (*domain.Attribute)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeAttribute,
			ResourceID:   attribute.ID,
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
