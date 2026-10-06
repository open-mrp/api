package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/ledgerlock"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
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

	if apiErr := identity.CheckIsInternalActorForRead(); apiErr != nil {
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
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("The responsible user was not found in this account.", "responsible_user_id"))
	}
	params.ResponsibleUserID = resolvedID

	if apiErr := validateAddBatchInputs(ctx, s.repos, params.AccountID, params.Batches); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

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
			if len(params.Batches) > 0 {
				if _, apiErr := createBatchesForRun(txCtx, txSvc.repos, params.AccountID, productionRunID, params.Batches); apiErr != nil {
					return apiErr
				}
				// Re-read so the batch count and totals include what was just planned.
				created, apiErr = txRepo.Get(txCtx, domain.GetProductionRunParams{ProductionRunID: productionRunID, AccountID: params.AccountID})
				if apiErr != nil {
					return apiErr
				}
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
			return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("The responsible user was not found in this account.", "responsible_user_id"))
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
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeProductionRun, params.ProductionRunID, params.AccountID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This production run has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	// The orders whose reservations this delete releases, and the items those reservations sit on, resolved on the pool so the transaction can take their ordering root as its first statement. See ledgerlock, Corollary A.
	orderIDs, apiErr := s.repos.NewProductionRunRepo().FindOrderIDsByRun(ctx, params.AccountID, params.ProductionRunID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	reservedItemIDs, apiErr := s.repos.NewInventoryReservationRepo().ListReservedItemIDsForOrders(ctx, params.AccountID, orderIDs)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Perform cascading delete in a transaction.
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *productionRunSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewProductionRunRepo()
		txReservationRepo := txSvc.repos.NewInventoryReservationRepo()

		scope, apiErr := ledgerlock.Acquire(txCtx, txReservationRepo, reservedItemIDs)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeProductionRun, productionRun.ID, params.AccountID, productionRun); apiErr != nil {
			return apiErr
		}

		// A scanned batch received its output into inventory, and deleting the row would leave that stock
		// counted with nothing behind it. Deleting a batch undoes its scan; the run has to wait for that.
		hasScanned, apiErr := txRepo.HasScannedBatches(txCtx, params.AccountID, params.ProductionRunID)
		if apiErr != nil {
			return apiErr
		}
		if hasScanned {
			return apierror.NewResourceConflictError("This production run has scanned batches. Delete those batches first to reverse their scans.")
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

		// Delete the production run.
		if apiErr := txRepo.Delete(txCtx, params); apiErr != nil {
			return apiErr
		}

		// Unlink orders from the run.
		if apiErr := txRepo.UnlinkOrdersFromRun(txCtx, params.AccountID, params.ProductionRunID); apiErr != nil {
			return apiErr
		}

		// Release each order's material reservations: the allocations covering them, the receipts those allocations were holding down, and the issues themselves.
		for _, orderID := range orderIDs {
			releasedItemIDs, apiErr := txReservationRepo.ReleaseReservedIssuesForOrder(txCtx, scope, params.AccountID, orderID)
			if apiErr != nil {
				return apiErr
			}

			// Material this run was sitting on is material another order's open demand may now be coverable from.
			if apiErr := mediator.EnqueueAllocateOpenIssues(txCtx, txSvc.repos, params.AccountID, releasedItemIDs...); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txSvc.mediators().ProductionRunActivity.NotifyRunDeleted(txCtx, identity, productionRun); apiErr != nil {
			return apiErr
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
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
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
	run, apiErr := repo.Get(ctx, domain.GetProductionRunParams{
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

	if apiErr := validateAddBatchInputs(ctx, s.repos, params.AccountID, params.Batches); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
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
			// Published to `results` only once built, so a retried transaction cannot double the
			// batches the caller is told it created.
			created, apiErr := createBatchesForRun(txCtx, txSvc.repos, params.AccountID, params.ProductionRunID, params.Batches)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := txSvc.mediators().ProductionRunActivity.NotifyBatchesAdded(txCtx, identity, run, len(created)); apiErr != nil {
				return apiErr
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

	// An unknown run is a 404, not an empty list.
	if _, apiErr := repo.Get(ctx, domain.GetProductionRunParams{
		ProductionRunID: params.ProductionRunID,
		AccountID:       params.AccountID,
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return repo.ListBatchesByRun(ctx, params)
}

// validateAddBatchInputs checks planned batches against the account: a positive quantity,
// non-negative seconds and waste, and every referenced item, unit, step, station, and
// machine belonging to it. The foreign keys alone would accept another account's records.
func validateAddBatchInputs(ctx context.Context, repos domain.RepoFactory, accountID string, inputs []domain.AddBatchInput) *apierror.APIError {
	if len(inputs) == 0 {
		return nil
	}

	var itemIdentifiers []domain.ItemIdentifier
	var unitIdentifiers []domain.UnitIdentifier
	var stationIdentifiers []domain.ObjectIdentifier
	var machineIDs []string
	for _, b := range inputs {
		itemIdentifiers = append(itemIdentifiers, domain.ItemIdentifier{ID: b.ItemID})
		unitIdentifiers = append(unitIdentifiers, domain.UnitIdentifier{ID: b.Quantity.UnitID})
		if b.Seconds != nil {
			unitIdentifiers = append(unitIdentifiers, domain.UnitIdentifier{ID: b.Seconds.UnitID})
		}
		if b.Waste != nil {
			unitIdentifiers = append(unitIdentifiers, domain.UnitIdentifier{ID: b.Waste.UnitID})
		}
		if b.ScanningStationID != nil && *b.ScanningStationID != "" {
			stationIdentifiers = append(stationIdentifiers, domain.ObjectIdentifier{ID: *b.ScanningStationID})
		}
		machineIDs = append(machineIDs, b.MachineIDs...)
	}

	items, apiErr := newItemIdentifierResolver(ctx, repos, accountID, itemIdentifiers)
	if apiErr != nil {
		return apiErr
	}
	units, apiErr := newUnitIdentifierResolver(ctx, repos, accountID, unitIdentifiers)
	if apiErr != nil {
		return apiErr
	}
	stationRepo := repos.NewScanningStationRepo()
	stations, apiErr := newObjectIdentifierResolver(ctx, accountID, "scanning station", stationIdentifiers,
		stationRepo.GetByIDs, stationRepo.FindByNames,
		func(s *domain.ScanningStation) string { return s.ID },
		func(s *domain.ScanningStation) string { return s.Name },
	)
	if apiErr != nil {
		return apiErr
	}

	validMachineIDs := make(map[string]struct{})
	if len(machineIDs) > 0 {
		machines, apiErr := repos.NewMachineRepo().GetByIDs(ctx, accountID, machineIDs)
		if apiErr != nil {
			return apiErr
		}
		for _, m := range machines {
			validMachineIDs[m.ID] = struct{}{}
		}
	}

	stepQueryRepo := repos.NewProductionStepQueryRepo()
	validStepIDs := make(map[string]struct{})
	for j, b := range inputs {
		param := func(field string) string { return fmt.Sprintf("batches[%d].%s", j, field) }

		if !b.Quantity.Measure.IsPositive() {
			return apierror.NewValidationErrorWithParam("Quantity must be greater than zero.", param("quantity_value"))
		}
		if b.Seconds != nil && b.Seconds.Measure.IsNegative() {
			return apierror.NewValidationErrorWithParam("Seconds cannot be negative.", param("seconds_value"))
		}
		if b.Waste != nil && b.Waste.Measure.IsNegative() {
			return apierror.NewValidationErrorWithParam("Waste cannot be negative.", param("waste_value"))
		}

		if _, apiErr := items.resolveOrError(domain.ItemIdentifier{ID: b.ItemID}, param("item_id")); apiErr != nil {
			return apiErr
		}
		if _, apiErr := units.resolveOrError(domain.UnitIdentifier{ID: b.Quantity.UnitID}, param("quantity_unit_id")); apiErr != nil {
			return apiErr
		}
		if b.Seconds != nil {
			if _, apiErr := units.resolveOrError(domain.UnitIdentifier{ID: b.Seconds.UnitID}, param("seconds_unit_id")); apiErr != nil {
				return apiErr
			}
		}
		if b.Waste != nil {
			if _, apiErr := units.resolveOrError(domain.UnitIdentifier{ID: b.Waste.UnitID}, param("waste_unit_id")); apiErr != nil {
				return apiErr
			}
		}
		if b.ScanningStationID != nil && *b.ScanningStationID != "" {
			if _, apiErr := stations.resolveOrError(domain.ObjectIdentifier{ID: *b.ScanningStationID}, param("scanning_station_id")); apiErr != nil {
				return apiErr
			}
		}
		for k, machineID := range b.MachineIDs {
			if _, ok := validMachineIDs[machineID]; !ok {
				return apierror.NewValidationErrorWithParam(
					fmt.Sprintf("Machine %q was not found.", machineID), param(fmt.Sprintf("machine_ids[%d]", k)))
			}
		}
		if b.ProductionStepID != nil && *b.ProductionStepID != "" {
			if _, ok := validStepIDs[*b.ProductionStepID]; !ok {
				inAccount, apiErr := stepQueryRepo.IsInAccount(ctx, accountID, *b.ProductionStepID)
				if apiErr != nil {
					return apiErr
				}
				if !inAccount {
					return apierror.NewValidationErrorWithParam(
						fmt.Sprintf("Production step %q was not found.", *b.ProductionStepID), param("production_step_id"))
				}
				validStepIDs[*b.ProductionStepID] = struct{}{}
			}
		}
	}

	return nil
}

// createBatchesForRun writes planned batches onto a run in a fixed number of statements. It
// consumes the caller's transaction.
func createBatchesForRun(txCtx context.Context, repos domain.RepoFactory, accountID, productionRunID string, inputs []domain.AddBatchInput) ([]*domain.BaseBatch, *apierror.APIError) {
	batches := make([]domain.NewBatch, len(inputs))
	for i, input := range inputs {
		batchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
		if apiErr != nil {
			return nil, apiErr
		}
		batches[i] = domain.NewBatch{
			ID: batchID,
			CreateBatchParams: domain.CreateBatchParams{
				AccountID:         accountID,
				ItemID:            input.ItemID,
				Quantity:          input.Quantity,
				Seconds:           input.Seconds,
				Waste:             input.Waste,
				ProductionStepID:  ptrutil.Deref(input.ProductionStepID),
				ScanningStationID: ptrutil.Deref(input.ScanningStationID),
				ProductionRunID:   productionRunID,
				MachineIDs:        input.MachineIDs,
			},
		}
	}
	return repos.NewBatchRepo().CreateMany(txCtx, batches)
}
