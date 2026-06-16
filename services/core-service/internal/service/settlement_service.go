package service

import (
	"context"
	"fmt"
	"strconv"

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

var settlementSvcTracer = tracing.GetTracer("core-service.settlement_service")

type settlementSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type SettlementSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *SettlementSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("settlement service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("settlement service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("settlement service: tx manager is required")
	}
	return nil
}

func NewSettlementSvc(config *SettlementSvcConfig) domain.SettlementSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &settlementSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *settlementSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *settlementSvcImpl) withTx(ctx context.Context, fn func(context.Context, *settlementSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &settlementSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *settlementSvcImpl) ListSettlements(ctx context.Context, params domain.ListSettlementsParams) (*domain.ListSettlementsResult, *apierror.APIError) {
	ctx, span := settlementSvcTracer.Start(ctx, "service.settlement.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSettlements, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewSettlementRepo().List(ctx, params)
}

func (s *settlementSvcImpl) GetSettlement(ctx context.Context, params domain.GetSettlementParams) (*domain.Settlement, *apierror.APIError) {
	ctx, span := settlementSvcTracer.Start(ctx, "service.settlement.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSettlements, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewSettlementRepo()

	settlement, apiErr := repo.Get(ctx, params.AccountID, params.SettlementID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for _, include := range params.Includes {
		switch include {
		case "allocations":
			allocations, apiErr := repo.GetAllocations(ctx, params.SettlementID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			settlement.Allocations = allocations
		}
	}

	return settlement, nil
}

func (s *settlementSvcImpl) CreateSettlement(ctx context.Context, params domain.CreateSettlementParams) (*domain.Settlement, *apierror.APIError) {
	ctx, span := settlementSvcTracer.Start(ctx, "service.settlement.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSettlements, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	settlementID, apiErr := id.GenID(id.SettlementIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Validate responsible user exists in account and resolve to the
	// account_user ID. The client may send either an account_user id or a
	// user id (the latter matching the legacy Dashboard behavior).
	accountUserRepo := s.repos.NewAccountUserRepo()
	resolvedID, apiErr := accountUserRepo.ResolveAccountUserID(ctx, params.AccountID, params.ResponsibleUserID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account user not found."))
	}
	params.ResponsibleUserID = resolvedID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Settlement](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Settlement
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *settlementSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSettlementRepo()

			// Generate settlement number
			nextNumber, apiErr := txRepo.GetNextSettlementNumber(txCtx, params.AccountID)
			if apiErr != nil {
				return apiErr
			}
			number := strconv.FormatInt(nextNumber, 10)

			// Generate sys property ID for upsert
			sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}

			// Update the next number
			if apiErr := txRepo.UpdateNextSettlementNumber(txCtx, sysPropertyID, params.AccountID, nextNumber); apiErr != nil {
				return apiErr
			}

			// Insert the settlement
			if apiErr := txRepo.InsertSettlement(txCtx, settlementID, number, params); apiErr != nil {
				return apiErr
			}

			// Get dollar unit ID for quantity records
			dollarUnitID, apiErr := txRepo.GetDollarUnitID(txCtx)
			if apiErr != nil {
				return apiErr
			}

			// Create allocations
			for _, alloc := range params.Allocations {
				allocationID, apiErr := id.GenID(id.TransactionAllocationIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				if apiErr := txRepo.CreateAllocation(txCtx, allocationID, quantityID, settlementID, dollarUnitID, alloc); apiErr != nil {
					return apiErr
				}
			}

			// Fetch the created settlement
			created, apiErr := txRepo.Get(txCtx, params.AccountID, settlementID)
			if apiErr != nil {
				return apiErr
			}

			// Fetch allocations
			allocations, apiErr := txRepo.GetAllocations(txCtx, settlementID)
			if apiErr != nil {
				return apiErr
			}
			created.Allocations = allocations

			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeSettlement,
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

		// After transaction: update payment statuses on affected invoices and transactions
		s.updatePaymentStatuses(ctx, settlementID, params.AccountID)

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *settlementSvcImpl) UpdateSettlement(ctx context.Context, params domain.UpdateSettlementParams) (*domain.Settlement, *apierror.APIError) {
	ctx, span := settlementSvcTracer.Start(ctx, "service.settlement.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSettlements, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// If the responsible user is being updated, validate existence and resolve
	// to the account_user ID. The client may send either an account_user id or
	// a user id.
	if params.ResponsibleUserID != nil {
		resolvedID, apiErr := s.repos.NewAccountUserRepo().ResolveAccountUserID(ctx, params.AccountID, *params.ResponsibleUserID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account user not found."))
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Settlement](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Settlement
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *settlementSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSettlementRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.SettlementID)
			if apiErr != nil {
				return apiErr
			}

			if params.Number != nil {
				isDuplicate, apiErr := txRepo.IsDuplicateNumber(txCtx, params.AccountID, *params.Number, &params.SettlementID)
				if apiErr != nil {
					return apiErr
				}
				if isDuplicate {
					return apierror.NewConflictErrorWithParam("A settlement with this number already exists.", "number")
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
				ResourceType: constants.ObjectTypeSettlement,
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

func (s *settlementSvcImpl) DeleteSettlement(ctx context.Context, params domain.DeleteSettlementParams) (*domain.Settlement, *apierror.APIError) {
	ctx, span := settlementSvcTracer.Start(ctx, "service.settlement.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSettlements, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewSettlementRepo()

	// Fetch the settlement before deleting
	settlement, apiErr := repo.Get(ctx, params.AccountID, params.SettlementID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeSettlement, params.SettlementID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewAlreadyDeletedError("This settlement has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	// Get allocations for the response
	allocations, apiErr := repo.GetAllocations(ctx, params.SettlementID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	settlement.Allocations = allocations

	// Get affected transaction and invoice IDs before deletion
	transactionIDs, apiErr := repo.GetAllocationTransactionIDs(ctx, params.SettlementID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	invoiceIDs, apiErr := repo.GetAllocationInvoiceIDs(ctx, params.SettlementID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *settlementSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewSettlementRepo()

		// Delete orphaned adjustment transactions
		if apiErr := txRepo.DeleteOrphanedAdjustmentTransactions(txCtx, params.SettlementID); apiErr != nil {
			return apiErr
		}

		// Delete allocations (quantities + allocation records)
		if _, apiErr := txRepo.DeleteAllocations(txCtx, params.SettlementID); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeSettlement, settlement.ID, settlement); apiErr != nil {
			return apiErr
		}

		// Delete the settlement
		if apiErr := txRepo.Delete(txCtx, params.AccountID, params.SettlementID); apiErr != nil {
			return apiErr
		}

		// Update transactions: set is_fully_allocated = false
		if apiErr := txRepo.UpdateTransactionsFullyAllocated(txCtx, transactionIDs, false); apiErr != nil {
			return apiErr
		}

		// Update invoices: set is_paid_in_full = false, is_over_paid = false
		for _, invoiceID := range invoiceIDs {
			if apiErr := txRepo.UpdateInvoicePaymentStatus(txCtx, invoiceID, false, false); apiErr != nil {
				return apiErr
			}
		}

		changes := audit.ComputeChanges(settlement, (*domain.Settlement)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeSettlement,
			ResourceID:   settlement.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return settlement, nil
}

// updatePaymentStatuses updates payment-related flags on affected invoices and transactions.
// This runs after the settlement creation transaction has committed.
func (s *settlementSvcImpl) updatePaymentStatuses(ctx context.Context, settlementID, _ string) {
	// This is a best-effort operation after the main transaction.
	// In the future, this could be replaced with an outbox message.
	// For now, we just log and continue if it fails.
	repo := s.repos.NewSettlementRepo()

	transactionIDs, apiErr := repo.GetAllocationTransactionIDs(ctx, settlementID)
	if apiErr != nil {
		return
	}

	invoiceIDs, apiErr := repo.GetAllocationInvoiceIDs(ctx, settlementID)
	if apiErr != nil {
		return
	}

	// Mark transactions as fully allocated where balance <= 0
	// For now, mark all as fully allocated since we just created allocations
	_ = repo.UpdateTransactionsFullyAllocated(ctx, transactionIDs, true)

	// Recompute each affected invoice's paid-in-full / over-paid flags from the
	// full set of allocations against the invoice's invoiced total (allocations
	// from any settlement count, not just this one).
	flags, apiErr := repo.GetInvoicePaymentFlags(ctx, invoiceIDs)
	if apiErr != nil {
		return
	}
	for _, f := range flags {
		_ = repo.UpdateInvoicePaymentStatus(ctx, f.InvoiceID, f.IsPaidInFull, f.IsOverPaid)
	}
}
