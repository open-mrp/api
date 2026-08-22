package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
)

var productionRunSvcTracer = tracing.GetTracer("core-service.production_run_service")

type productionRunSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

type ProductionRunSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// JobSvcFactory (required) builds the job service used to raise and settle the
	// jobs behind this service's asynchronous operations.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ProductionRunSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("production run service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("production run service: mediator factory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("production run service: job service factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("production run service: tx manager is required")
	}
	return nil
}

func NewProductionRunSvc(config *ProductionRunSvcConfig) domain.ProductionRunSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productionRunSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		jobSvcFactory:   config.JobSvcFactory,
		txManager:       config.TxManager,
	}
}

func (s *productionRunSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *productionRunSvcImpl) withTx(ctx context.Context, fn func(context.Context, *productionRunSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &productionRunSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			jobSvcFactory:   s.jobSvcFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *productionRunSvcImpl) ListProductionRuns(ctx context.Context, params domain.ListProductionRunsParams) (*domain.ListProductionRunsResult, *apierror.APIError) {
	ctx, span := productionRunSvcTracer.Start(ctx, "service.production_run.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionRuns, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductionRunRepo()
	return repo.List(ctx, params)
}

func (s *productionRunSvcImpl) GetProductionRun(ctx context.Context, params domain.GetProductionRunParams) (*domain.ProductionRun, *apierror.APIError) {
	ctx, span := productionRunSvcTracer.Start(ctx, "service.production_run.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionRuns, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductionRunRepo()
	return repo.Get(ctx, params)
}

func (s *productionRunSvcImpl) CreateProductionRun(ctx context.Context, params domain.CreateProductionRunParams) (*domain.ProductionRun, *apierror.APIError) {
	ctx, span := productionRunSvcTracer.Start(ctx, "service.production_run.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionRuns, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Validate that the responsible user exists in the account. The client may send either an account_user id or a user id; store the account_user id.
	accountUserRepo := s.repos.NewAccountUserRepo()
	resolvedID, apiErr := accountUserRepo.ResolveAccountUserID(ctx, params.AccountID, params.ResponsibleUserID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("The responsible user was not found in this account."))
	}
	params.ResponsibleUserID = resolvedID

	productionRunID, apiErr := id.GenID(id.ProductionRunIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductionRun](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductionRun
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionRunSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductionRunRepo()

			number, apiErr := txRepo.GetNextNumber(txCtx, params.AccountID)
			if apiErr != nil {
				return apiErr
			}

			created, apiErr := txRepo.Create(txCtx, productionRunID, params, number)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeProductionRun,
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

func (s *productionRunSvcImpl) UpdateProductionRun(ctx context.Context, params domain.UpdateProductionRunParams) (*domain.ProductionRun, *apierror.APIError) {
	ctx, span := productionRunSvcTracer.Start(ctx, "service.production_run.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionRuns, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductionRunRepo()

	// Verify the run exists and is not completed.
	isCompleted, apiErr := repo.IsCompleted(ctx, params.AccountID, params.ProductionRunID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if isCompleted {
		return nil, tracing.Trace(span, apierror.NewValidationError("Cannot update a completed production run."))
	}

	// If number is being updated, check uniqueness.
	if params.Number != nil {
		exists, apiErr := repo.ExistsByNumber(ctx, params.AccountID, *params.Number, &params.ProductionRunID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if exists {
			return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("A production run with this number already exists.", "number"))
		}
	}

	// If responsible user is being updated, validate existence. The client may send either an account_user id or a user id; store the account_user id.
	if params.ResponsibleUserID != nil {
		accountUserRepo := s.repos.NewAccountUserRepo()
		resolvedID, apiErr := accountUserRepo.ResolveAccountUserID(ctx, params.AccountID, *params.ResponsibleUserID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("The responsible user was not found in this account."))
		}
		params.ResponsibleUserID = &resolvedID
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ProductionRun](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ProductionRun
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionRunSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewProductionRunRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetProductionRunParams{
				ProductionRunID: params.ProductionRunID,
				AccountID:       params.AccountID,
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
				ResourceType: constants.ObjectTypeProductionRun,
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

func (s *productionRunSvcImpl) DeleteProductionRun(ctx context.Context, params domain.DeleteProductionRunParams) *apierror.APIError {
	ctx, span := productionRunSvcTracer.Start(ctx, "service.production_run.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionRuns, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductionRunRepo()

	// Verify the run exists (Get will 404 if not found).
	productionRun, apiErr := repo.Get(ctx, domain.GetProductionRunParams(params))
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeProductionRun, params.ProductionRunID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This production run has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	// Perform cascading delete in a transaction.
	return s.withTx(ctx, func(txCtx context.Context, txSvc *productionRunSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewProductionRunRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeProductionRun, productionRun.ID, productionRun); apiErr != nil {
			return apiErr
		}

		// Delete all batches for this run.
		if apiErr := txRepo.DeleteBatchesByRun(txCtx, params.AccountID, params.ProductionRunID); apiErr != nil {
			return apiErr
		}

		// A schedule week released as this run goes back to planned, so it can be issued
		// again rather than being stuck looking released with no run behind it.
		if apiErr := txSvc.repos.NewProductionScheduleRepo().
			UnreleaseLinesForRun(txCtx, params.AccountID, params.ProductionRunID); apiErr != nil {
			return apiErr
		}

		// Find linked order IDs before deleting the run.
		orderIDs, apiErr := txRepo.FindOrderIDsByRun(txCtx, params.AccountID, params.ProductionRunID)
		if apiErr != nil {
			return apiErr
		}

		// Delete the production run.
		if apiErr := txRepo.Delete(txCtx, params); apiErr != nil {
			return apiErr
		}

		// Unlink orders from the run.
		if apiErr := txRepo.UnlinkOrdersFromRun(txCtx, params.AccountID, params.ProductionRunID); apiErr != nil {
			return apiErr
		}

		// For each order, delete reserved inventory issues.
		for _, orderID := range orderIDs {
			if apiErr := txRepo.DeleteReservedInventoryIssuesByOrder(txCtx, params.AccountID, orderID); apiErr != nil {
				return apiErr
			}
		}

		changes := audit.ComputeChanges(productionRun, (*domain.ProductionRun)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeProductionRun,
			ResourceID:   productionRun.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
}

func (s *productionRunSvcImpl) AddBatchesToProductionRun(ctx context.Context, params domain.AddBatchesToProductionRunParams) ([]*domain.BaseBatch, *apierror.APIError) {
	ctx, span := productionRunSvcTracer.Start(ctx, "service.production_run.add_batches")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionRuns, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductionRunRepo()

	// Verify run exists.
	_, apiErr := repo.Get(ctx, domain.GetProductionRunParams{
		ProductionRunID: params.ProductionRunID,
		AccountID:       params.AccountID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Verify run is not completed.
	isCompleted, apiErr := repo.IsCompleted(ctx, params.AccountID, params.ProductionRunID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if isCompleted {
		return nil, tracing.Trace(span, apierror.NewValidationError("Cannot add batches to a completed production run."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[[]*domain.BaseBatch](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return *cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var results []*domain.BaseBatch
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionRunSvcImpl) *apierror.APIError {
			batchRepo := txSvc.repos.NewBatchRepo()
			prRepo := txSvc.repos.NewProductionRunRepo()

			// Accumulated inside the transaction and published to `results` only once it has
			// been built. Appending to the outer slice directly would double the batches if the
			// transaction were ever run twice, and the caller would be told it created each one
			// of them.
			created := make([]*domain.BaseBatch, 0, len(params.Batches))

			for _, input := range params.Batches {
				batchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				batch, apiErr := batchRepo.Create(txCtx, batchID, domain.CreateBatchParams{
					AccountID:         params.AccountID,
					ItemID:            input.ItemID,
					Quantity:          input.Quantity,
					Seconds:           input.Seconds,
					Waste:             input.Waste,
					ProductionStepID:  ptrutil.Deref(input.ProductionStepID),
					ScanningStationID: ptrutil.Deref(input.ScanningStationID),
				})
				if apiErr != nil {
					return apiErr
				}

				// Connect the batch to the production run.
				if apiErr := prRepo.SetBatchProductionRunID(txCtx, params.AccountID, batchID, params.ProductionRunID); apiErr != nil {
					return apiErr
				}
				batch.ProductionRunID = &params.ProductionRunID

				created = append(created, batch)
			}

			results = created

			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, results)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return results, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *productionRunSvcImpl) ListBatchesByProductionRun(ctx context.Context, params domain.ListBatchesByProductionRunParams) (*domain.ListBatchesByProductionRunResult, *apierror.APIError) {
	ctx, span := productionRunSvcTracer.Start(ctx, "service.production_run.list_batches")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionRuns, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewProductionRunRepo()
	return repo.ListBatchesByRun(ctx, params)
}
