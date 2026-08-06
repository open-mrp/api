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

var productionStepSvcTracer = tracing.GetTracer("core-service.production_step_service")

type productionStepSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

type ProductionStepSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// JobSvcFactory (required) builds the job service used to raise and settle the
	// jobs behind the async bulk upsert.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ProductionStepSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("production step service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("production step service: mediator factory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("production step service: job service factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("production step service: tx manager is required")
	}
	return nil
}

func NewProductionStepSvc(config *ProductionStepSvcConfig) domain.ProductionStepSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &productionStepSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		jobSvcFactory:   config.JobSvcFactory,
		txManager:       config.TxManager,
	}
}

func (s *productionStepSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productionStepSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productionStepSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &productionStepSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			jobSvcFactory:   s.jobSvcFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *productionStepSvcImpl) ListProductionSteps(ctx context.Context, params domain.ListProductionStepsParams) (*domain.ListProductionStepsResult, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.list")
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

	params.AccountID = identity.Target.AccountID

	return s.repos.NewProductionStepRepo().List(ctx, params)
}

func (s *productionStepSvcImpl) GetProductionStep(ctx context.Context, stepID string) (*domain.ProductionStep, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.get")
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

	return s.repos.NewProductionStepRepo().Get(ctx, identity.Target.AccountID, stepID)
}

func (s *productionStepSvcImpl) CreateProductionStep(ctx context.Context, params domain.CreateProductionStepParams) (*domain.ProductionStep, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.create")
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

	// Generate IDs.
	stepID, apiErr := id.GenID(id.ProductionStepIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	laborRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	laborTimeID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	overheadRateID, apiErr := id.GenID(id.RateIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productionID, apiErr := id.GenID(id.ProductionIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductionStep](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductionStep
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductionStepRepo()
			txUnitRepo := txSvc.repos.NewUnitRepo()

			// Validate cost-typed rates: numerator must be currency, denominator must not. labor_time is a time-per-unit rate (not money), so it's exempt.
			if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.LaborRate.NumeratorUnitID, params.LaborRate.DenominatorUnitID, "labor_rate"); apiErr != nil {
				return apiErr
			}
			if apiErr := ValidateCostRateUnits(txCtx, txUnitRepo, params.OverheadRate.NumeratorUnitID, params.OverheadRate.DenominatorUnitID, "overhead_rate"); apiErr != nil {
				return apiErr
			}

			// Insert rates.
			if apiErr := txRepo.InsertRate(txCtx, laborRateID, params.LaborRate); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.InsertRate(txCtx, laborTimeID, params.LaborTime); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.InsertRate(txCtx, overheadRateID, params.OverheadRate); apiErr != nil {
				return apiErr
			}

			// Insert the production step.
			if apiErr := txRepo.InsertStep(txCtx, stepID, params.Name, params.Notes, params.LevelingFactor, params.Allowances, laborRateID, laborTimeID, overheadRateID, params.ScanningStationID, params.DepartmentID, params.AccountID); apiErr != nil {
				return apiErr
			}

			// Insert production quantity and production output.
			if apiErr := txRepo.InsertQuantity(txCtx, quantityID, params.Production.QuantityValue, params.Production.QuantityUnitID); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.InsertProduction(txCtx, productionID, params.Production.ItemID, quantityID, stepID); apiErr != nil {
				return apiErr
			}

			// Create consumptions.
			consumptionRepo := txSvc.repos.NewConsumptionRepo()
			for _, cp := range params.Consumptions {
				consumptionID, apiErr := id.GenID(id.ConsumptionIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				cQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				wasteQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				_, apiErr = consumptionRepo.Create(txCtx, consumptionID, cQuantityID, wasteQuantityID, domain.CreateConsumptionParams{
					AccountID:           params.AccountID,
					ProductionStepID:    stepID,
					ItemID:              cp.ItemID,
					QuantityValue:       cp.QuantityValue,
					QuantityUnitID:      cp.QuantityUnitID,
					WasteQuantityValue:  cp.WasteQuantityValue,
					WasteQuantityUnitID: cp.WasteQuantityUnitID,
					Instructions:        cp.Instructions,
				})
				if apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch the full step to return.
			fetched, apiErr := txRepo.Get(txCtx, params.AccountID, stepID)
			if apiErr != nil {
				return apiErr
			}
			result = fetched

			changes := audit.ComputeChanges(nil, result)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProductionStep,
				ResourceID:   result.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		// Link the production flow outside the main transaction.
		if apiErr := meds.ProductionFlow.LinkFlow(ctx, stepID, params.AccountID); apiErr != nil {
			// Non-fatal: log but don't fail the create.
			_ = apiErr
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

// validLaborTimeUnits are the accepted abbreviations for the labor time numerator unit.
var validLaborTimeUnits = map[string]bool{
	"hr":     true,
	"minute": true,
	"min":    true,
	"second": true,
	"sec":    true,
	"day":    true,
}

func (s *productionStepSvcImpl) UpdateProductionStep(ctx context.Context, params domain.UpdateProductionStepParams) (*domain.ProductionStep, *apierror.APIError) {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.update")
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

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductionStep](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductionStep
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductionStepRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.ProductionStepID)
			if apiErr != nil {
				return apiErr
			}

			// Check for name conflicts if name is changing.
			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.ProductionStepID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A production step with this name already exists.", "name")
				}
			}

			if apiErr := txRepo.Update(txCtx, params); apiErr != nil {
				return apiErr
			}

			fetched, apiErr := txRepo.Get(txCtx, params.AccountID, params.ProductionStepID)
			if apiErr != nil {
				return apiErr
			}
			result = fetched

			changes := audit.ComputeChanges(old, result)
			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeProductionStep,
				ResourceID:   result.ID,
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

func (s *productionStepSvcImpl) DeleteProductionStep(ctx context.Context, stepID string) *apierror.APIError {
	ctx, span := productionStepSvcTracer.Start(ctx, "service.production_step.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSteps, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	// Fetch the step before deletion for the audit diff.
	step, apiErr := s.repos.NewProductionStepRepo().Get(ctx, accountID, stepID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeProductionStep, stepID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This production step has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	// Disconnect parent-child links, then delete — atomically.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionStepSvcImpl) *apierror.APIError {
		repo := txSvc.repos.NewProductionStepRepo()

		if apiErr := repo.DeleteParentChildLinks(txCtx, stepID); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeProductionStep, step.ID, step); apiErr != nil {
			return apiErr
		}

		if apiErr := repo.Delete(txCtx, accountID, stepID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(step, (*domain.ProductionStep)(nil))
		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProductionStep,
			ResourceID:   step.ID,
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
