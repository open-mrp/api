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

var shippingTermSvcTracer = tracing.GetTracer("core-service.shipping_term_service")

type shippingTermSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ShippingTermSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *ShippingTermSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("shipping term service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("shipping term service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("shipping term service: tx manager is required")
	}
	return nil
}

func NewShippingTermSvc(config *ShippingTermSvcConfig) domain.ShippingTermSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shippingTermSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *shippingTermSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *shippingTermSvcImpl) withTx(ctx context.Context, fn func(context.Context, *shippingTermSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &shippingTermSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *shippingTermSvcImpl) ListShippingTerms(ctx context.Context, params domain.ListShippingTermsParams) (*domain.ListShippingTermsResult, *apierror.APIError) {
	ctx, span := shippingTermSvcTracer.Start(ctx, "service.shipping_term.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShippingTerms, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewShippingTermRepo().List(ctx, params)
}

func (s *shippingTermSvcImpl) GetShippingTerm(ctx context.Context, shippingTermID string) (*domain.ShippingTerm, *apierror.APIError) {
	ctx, span := shippingTermSvcTracer.Start(ctx, "service.shipping_term.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShippingTerms, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewShippingTermRepo().Get(ctx, domain.GetShippingTermParams{
		AccountID:      identity.Target.AccountID,
		ShippingTermID: shippingTermID,
	})
}

func (s *shippingTermSvcImpl) CreateShippingTerm(ctx context.Context, params domain.CreateShippingTermParams) (*domain.ShippingTerm, *apierror.APIError) {
	ctx, span := shippingTermSvcTracer.Start(ctx, "service.shipping_term.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShippingTerms, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	shippingTermID, apiErr := id.GenID(id.ShippingTermIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ShippingTerm](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ShippingTerm
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shippingTermSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewShippingTermRepo()

			// Insert flat rate quantity if provided
			if params.FlatRate != nil {
				flatRateID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := txRepo.InsertQuantity(txCtx, flatRateID, params.FlatRate.Value, params.FlatRate.UnitID); apiErr != nil {
					return apiErr
				}
				params.FlatRateID = &flatRateID
			}

			// Insert minimum order quantity if provided
			if params.MinimumOrderValue != nil {
				minimumOrderID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := txRepo.InsertQuantity(txCtx, minimumOrderID, params.MinimumOrderValue.Value, params.MinimumOrderValue.UnitID); apiErr != nil {
					return apiErr
				}
				params.MinimumOrderID = &minimumOrderID
			}

			// Insert shipping term
			if _, apiErr := txRepo.Create(txCtx, shippingTermID, params); apiErr != nil {
				return apiErr
			}

			// Insert free shipping rules
			for _, serviceLevelID := range params.FreeShippingServiceLevelIDs {
				ruleID, apiErr := id.GenID(id.FreeShippingRuleIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := txRepo.InsertFreeShippingRule(txCtx, ruleID, shippingTermID, serviceLevelID); apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch to get complete data
			created, apiErr := txRepo.Get(txCtx, domain.GetShippingTermParams{
				AccountID:      params.AccountID,
				ShippingTermID: shippingTermID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeShippingTerm,
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

func (s *shippingTermSvcImpl) UpdateShippingTerm(ctx context.Context, params domain.UpdateShippingTermParams) (*domain.ShippingTerm, *apierror.APIError) {
	ctx, span := shippingTermSvcTracer.Start(ctx, "service.shipping_term.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShippingTerms, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ShippingTerm](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ShippingTerm
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shippingTermSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewShippingTermRepo()

			shippingTerm, apiErr := txRepo.Get(txCtx, domain.GetShippingTermParams{
				AccountID:      params.AccountID,
				ShippingTermID: params.ShippingTermID,
			})
			if apiErr != nil {
				return apiErr
			}
			if shippingTerm.AccountID == nil {
				return apierror.NewAuthorizationError("Default shipping term cannot be updated.")
			}

			// Handle flat rate quantity upsert/delete
			if params.FlatRate != nil {
				if shippingTerm.FlatRate != nil {
					// Update existing quantity
					if apiErr := txRepo.UpdateQuantity(txCtx, shippingTerm.FlatRate.ID, params.FlatRate.Value, params.FlatRate.UnitID); apiErr != nil {
						return apiErr
					}
					params.FlatRateID = &shippingTerm.FlatRate.ID
				} else {
					// Create new quantity
					flatRateID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := txRepo.InsertQuantity(txCtx, flatRateID, params.FlatRate.Value, params.FlatRate.UnitID); apiErr != nil {
						return apiErr
					}
					params.FlatRateID = &flatRateID
				}
			} else if params.HasFlatRate {
				// Explicitly sent null — clear the flat rate (params.FlatRateID stays nil → SQL sets column to NULL)
			} else if shippingTerm.FlatRate != nil {
				// Not provided at all — keep existing flat rate ID
				params.FlatRateID = &shippingTerm.FlatRate.ID
			}

			// Handle minimum order quantity upsert/delete
			if params.MinimumOrderValue != nil {
				if shippingTerm.MinimumOrderValue != nil {
					// Update existing quantity
					if apiErr := txRepo.UpdateQuantity(txCtx, shippingTerm.MinimumOrderValue.ID, params.MinimumOrderValue.Value, params.MinimumOrderValue.UnitID); apiErr != nil {
						return apiErr
					}
					params.MinimumOrderID = &shippingTerm.MinimumOrderValue.ID
				} else {
					// Create new quantity
					minimumOrderID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := txRepo.InsertQuantity(txCtx, minimumOrderID, params.MinimumOrderValue.Value, params.MinimumOrderValue.UnitID); apiErr != nil {
						return apiErr
					}
					params.MinimumOrderID = &minimumOrderID
				}
			} else if params.HasMinimumOrderValue {
				// Explicitly sent null — clear the minimum order value (params.MinimumOrderID stays nil → SQL sets column to NULL)
			} else if shippingTerm.MinimumOrderValue != nil {
				// Not provided at all — keep existing minimum order ID
				params.MinimumOrderID = &shippingTerm.MinimumOrderValue.ID
			}

			// Update shipping term
			if _, apiErr := txRepo.Update(txCtx, params); apiErr != nil {
				return apiErr
			}

			// Sync free shipping rules (delete all + re-insert)
			if params.HasFreeShippingServiceLevelIDs {
				if apiErr := txRepo.DeleteFreeShippingRulesByShippingTermID(txCtx, params.ShippingTermID); apiErr != nil {
					return apiErr
				}
				for _, serviceLevelID := range params.FreeShippingServiceLevelIDs {
					ruleID, apiErr := id.GenID(id.FreeShippingRuleIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := txRepo.InsertFreeShippingRule(txCtx, ruleID, params.ShippingTermID, serviceLevelID); apiErr != nil {
						return apiErr
					}
				}
			}

			// Re-fetch to get complete data
			updated, apiErr := txRepo.Get(txCtx, domain.GetShippingTermParams{
				AccountID:      params.AccountID,
				ShippingTermID: params.ShippingTermID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(shippingTerm, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeShippingTerm,
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

func (s *shippingTermSvcImpl) DeleteShippingTerm(ctx context.Context, shippingTermID string) *apierror.APIError {
	ctx, span := shippingTermSvcTracer.Start(ctx, "service.shipping_term.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShippingTerms, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	shippingTerm, apiErr := s.repos.NewShippingTermRepo().Get(ctx, domain.GetShippingTermParams{
		AccountID:      accountID,
		ShippingTermID: shippingTermID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeShippingTerm, shippingTermID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This shipping term has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if shippingTerm.AccountID == nil {
		return tracing.Trace(span, apierror.NewValidationError("Default shipping terms cannot be deleted."))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shippingTermSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewShippingTermRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeShippingTerm, shippingTerm.ID, shippingTerm); apiErr != nil {
			return apiErr
		}

		// Delete free shipping rules
		if apiErr := txRepo.DeleteFreeShippingRulesByShippingTermID(txCtx, shippingTermID); apiErr != nil {
			return apiErr
		}

		// Delete shipping term
		if apiErr := txRepo.Delete(txCtx, domain.DeleteShippingTermParams{
			AccountID:      accountID,
			ShippingTermID: shippingTermID,
		}); apiErr != nil {
			return apiErr
		}

		// Delete orphaned quantity records
		if shippingTerm.FlatRate != nil {
			if apiErr := txRepo.DeleteQuantity(txCtx, shippingTerm.FlatRate.ID); apiErr != nil {
				return apiErr
			}
		}
		if shippingTerm.MinimumOrderValue != nil {
			if apiErr := txRepo.DeleteQuantity(txCtx, shippingTerm.MinimumOrderValue.ID); apiErr != nil {
				return apiErr
			}
		}

		changes := audit.ComputeChanges(shippingTerm, (*domain.ShippingTerm)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeShippingTerm,
			ResourceID:   shippingTerm.ID,
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
