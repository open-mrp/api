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

var consumptionSvcTracer = tracing.GetTracer("core-service.consumption_service")

type consumptionSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ConsumptionSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *ConsumptionSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("consumption service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("consumption service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("consumption service: tx manager is required")
	}
	return nil
}

func NewConsumptionSvc(config *ConsumptionSvcConfig) domain.ConsumptionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &consumptionSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *consumptionSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *consumptionSvcImpl) withTx(ctx context.Context, fn func(context.Context, *consumptionSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &consumptionSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// GetConsumption returns a single consumption by ID within a production step.
//
// 1. Extract and validate the caller's identity, actor type, and production_steps:read permission.
// 2. Require the Augno-Account header to scope the query.
// 3. Verify the production step belongs to the account.
// 4. Fetch the consumption from the repository.
func (s *consumptionSvcImpl) GetConsumption(ctx context.Context, productionStepID, consumptionID string) (*domain.Consumption, *apierror.APIError) {
	ctx, span := consumptionSvcTracer.Start(ctx, "service.consumption.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	isInAccount, apiErr := s.repos.NewProductionStepQueryRepo().IsInAccount(ctx, accountID, productionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !isInAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}

	return s.repos.NewConsumptionRepo().Get(ctx, accountID, productionStepID, consumptionID)
}

// CreateConsumption creates a new consumption within a production step, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type, and production_steps:create permission.
// 2. Verify the production step and item belong to the account.
// 3. Generate IDs for the consumption, quantity, and waste quantity.
// 4. Upsert an idempotency key; if already finished, return the cached response.
// 5. Within a transaction, create the consumption and link the production flow.
// 6. On error, cache the error response for idempotent replay.
func (s *consumptionSvcImpl) CreateConsumption(ctx context.Context, params domain.CreateConsumptionParams) (*domain.Consumption, *apierror.APIError) {
	ctx, span := consumptionSvcTracer.Start(ctx, "service.consumption.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	isInAccount, apiErr := s.repos.NewProductionStepQueryRepo().IsInAccount(ctx, params.AccountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !isInAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}

	_, apiErr = s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
		AccountID: params.AccountID,
		ItemID:    params.ItemID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	consumptionID, apiErr := id.GenID(id.ConsumptionIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	wasteQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Consumption](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Consumption
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *consumptionSvcImpl) *apierror.APIError {
			created, apiErr := txSvc.repos.NewConsumptionRepo().Create(txCtx, consumptionID, quantityID, wasteQuantityID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			if apiErr := txSvc.mediators().ProductionFlow.LinkFlow(txCtx, params.ProductionStepID, params.AccountID); apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeConsumption,
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

// UpdateConsumption partially updates a consumption, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type, and production_steps:update permission.
// 2. Verify the production step belongs to the account.
// 3. Upsert an idempotency key; if already finished, return the cached response.
// 4. If ItemID is changing, disconnect the old downstream step and re-link the flow.
// 5. Update quantities and instructions as needed.
// 6. On error, cache the error response for idempotent replay.
func (s *consumptionSvcImpl) UpdateConsumption(ctx context.Context, params domain.UpdateConsumptionParams) (*domain.Consumption, *apierror.APIError) {
	ctx, span := consumptionSvcTracer.Start(ctx, "service.consumption.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	isInAccount, apiErr := s.repos.NewProductionStepQueryRepo().IsInAccount(ctx, params.AccountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !isInAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Consumption](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Consumption

		if params.ItemID != nil {
			// Validate the new item belongs to the account.
			_, apiErr = s.repos.NewItemRepo().Get(ctx, domain.GetItemParams{
				AccountID: params.AccountID,
				ItemID:    *params.ItemID,
			})
			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
			}

			// Get the old item ID to find the downstream step to disconnect.
			var oldItemID string
			oldItemID, apiErr = s.repos.NewConsumptionRepo().GetItemID(ctx, params.ConsumptionID)
			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
			}

			var downstreamStepID *string
			downstreamStepID, apiErr = meds.ProductionFlow.FindDownstreamStepByItem(ctx, params.ProductionStepID, oldItemID, params.AccountID)
			if apiErr != nil {
				return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, tracing.Trace(span, apiErr))
			}

			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *consumptionSvcImpl) *apierror.APIError {
				txMeds := txSvc.mediators()

				old, apiErr := txSvc.repos.NewConsumptionRepo().Get(txCtx, params.AccountID, params.ProductionStepID, params.ConsumptionID)
				if apiErr != nil {
					return apiErr
				}

				if downstreamStepID != nil {
					if apiErr := txMeds.ProductionFlow.DisconnectSteps(txCtx, *downstreamStepID, params.ProductionStepID); apiErr != nil {
						return apiErr
					}
				}

				// Preserve existing instructions when not explicitly provided.
				instructions := params.Instructions
				if instructions == nil {
					currentInstructions, apiErr := txSvc.repos.NewConsumptionRepo().GetInstructions(txCtx, params.ConsumptionID)
					if apiErr != nil {
						return apiErr
					}
					instructions = currentInstructions
				}

				if apiErr := txSvc.repos.NewConsumptionRepo().UpdateItem(txCtx, params.AccountID, params.ConsumptionID, *params.ItemID, instructions); apiErr != nil {
					return apiErr
				}

				// Update quantities if provided.
				if err := updateConsumptionQuantities(txCtx, txSvc.repos.NewConsumptionRepo(), params); err != nil {
					return err
				}

				if apiErr := txMeds.ProductionFlow.LinkFlow(txCtx, params.ProductionStepID, params.AccountID); apiErr != nil {
					return apiErr
				}

				fetched, apiErr := txSvc.repos.NewConsumptionRepo().Get(txCtx, params.AccountID, params.ProductionStepID, params.ConsumptionID)
				if apiErr != nil {
					return apiErr
				}
				result = fetched

				changes := audit.ComputeChanges(old, result)

				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionUpdate,
					ResourceType: constants.ObjectTypeConsumption,
					ResourceID:   result.ID,
					Changes:      changes,
				}); apiErr != nil {
					return apiErr
				}

				return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
			})
		} else {
			// ItemID is not changing — just update quantities and/or instructions.
			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *consumptionSvcImpl) *apierror.APIError {
				consumptionRepo := txSvc.repos.NewConsumptionRepo()

				old, apiErr := consumptionRepo.Get(txCtx, params.AccountID, params.ProductionStepID, params.ConsumptionID)
				if apiErr != nil {
					return apiErr
				}

				if err := updateConsumptionQuantities(txCtx, consumptionRepo, params); err != nil {
					return err
				}

				if params.Instructions != nil {
					// UpdateItem sets item_id unconditionally, so we must pass the current item.
					currentItemID, apiErr := consumptionRepo.GetItemID(txCtx, params.ConsumptionID)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := consumptionRepo.UpdateItem(txCtx, params.AccountID, params.ConsumptionID, currentItemID, params.Instructions); apiErr != nil {
						return apiErr
					}
				}

				fetched, apiErr := consumptionRepo.Get(txCtx, params.AccountID, params.ProductionStepID, params.ConsumptionID)
				if apiErr != nil {
					return apiErr
				}
				result = fetched

				changes := audit.ComputeChanges(old, result)

				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionUpdate,
					ResourceType: constants.ObjectTypeConsumption,
					ResourceID:   result.ID,
					Changes:      changes,
				}); apiErr != nil {
					return apiErr
				}

				return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
			})
		}

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// DeleteConsumption deletes a consumption from a production step and returns it.
//
// 1. Extract and validate the caller's identity, actor type, and production_steps:delete permission.
// 2. Verify the production step belongs to the account.
// 3. Fetch the consumption (to return it after deletion).
// 4. Find source steps connected via this consumption.
// 5. Within a transaction, disconnect source steps, delete quantity records, delete the consumption, and re-link the flow.
func (s *consumptionSvcImpl) DeleteConsumption(ctx context.Context, params domain.DeleteConsumptionParams) (*domain.Consumption, *apierror.APIError) {
	ctx, span := consumptionSvcTracer.Start(ctx, "service.consumption.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	isInAccount, apiErr := s.repos.NewProductionStepQueryRepo().IsInAccount(ctx, accountID, params.ProductionStepID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !isInAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Production step not found."))
	}

	consumption, apiErr := s.repos.NewConsumptionRepo().Get(ctx, accountID, params.ProductionStepID, params.ConsumptionID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeConsumption, params.ConsumptionID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewAlreadyDeletedError("This consumption has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	sourceStepIDs, apiErr := meds.ProductionFlow.FindSourceStepsByConsumption(ctx, params.ProductionStepID, params.ConsumptionID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *consumptionSvcImpl) *apierror.APIError {
		txMeds := txSvc.mediators()

		for _, sourceStepID := range sourceStepIDs {
			if apiErr := txMeds.ProductionFlow.DisconnectSteps(txCtx, sourceStepID, params.ProductionStepID); apiErr != nil {
				return apiErr
			}
		}

		consumptionRepo := txSvc.repos.NewConsumptionRepo()

		quantityID, wasteQuantityID, apiErr := consumptionRepo.GetQuantityIDs(txCtx, params.ConsumptionID)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeConsumption, consumption.ID, consumption); apiErr != nil {
			return apiErr
		}

		if apiErr := consumptionRepo.Delete(txCtx, accountID, params.ConsumptionID); apiErr != nil {
			return apiErr
		}

		if apiErr := consumptionRepo.DeleteQuantity(txCtx, quantityID); apiErr != nil {
			return apiErr
		}

		if apiErr := consumptionRepo.DeleteQuantity(txCtx, wasteQuantityID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(consumption, (*domain.Consumption)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeConsumption,
			ResourceID:   consumption.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		if apiErr := txMeds.ProductionFlow.LinkFlow(txCtx, params.ProductionStepID, accountID); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return consumption, nil
}

// updateConsumptionQuantities updates the main quantity and/or waste quantity
// independently when their respective value+unit pairs are provided.
func updateConsumptionQuantities(ctx context.Context, repo domain.ConsumptionRepo, params domain.UpdateConsumptionParams) *apierror.APIError {
	needsQuantity := params.QuantityValue != nil && params.QuantityUnitID != nil
	needsWaste := params.WasteQuantityValue != nil && params.WasteQuantityUnitID != nil

	if !needsQuantity && !needsWaste {
		return nil
	}

	quantityID, wasteQuantityID, apiErr := repo.GetQuantityIDs(ctx, params.ConsumptionID)
	if apiErr != nil {
		return apiErr
	}

	if needsQuantity {
		if apiErr := repo.UpdateQuantity(ctx, quantityID, *params.QuantityValue, *params.QuantityUnitID); apiErr != nil {
			return apiErr
		}
	}

	if needsWaste {
		if apiErr := repo.UpdateQuantity(ctx, wasteQuantityID, *params.WasteQuantityValue, *params.WasteQuantityUnitID); apiErr != nil {
			return apiErr
		}
	}

	return nil
}
