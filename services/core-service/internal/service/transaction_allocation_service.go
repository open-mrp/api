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
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var transactionAllocationSvcTracer = tracing.GetTracer("core-service.transaction_allocation_service")

type transactionAllocationSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type TransactionAllocationSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *TransactionAllocationSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("transaction allocation service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("transaction allocation service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("transaction allocation service: tx manager is required")
	}
	return nil
}

func NewTransactionAllocationSvc(config *TransactionAllocationSvcConfig) domain.TransactionAllocationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &transactionAllocationSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *transactionAllocationSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *transactionAllocationSvcImpl) withTx(ctx context.Context, fn func(context.Context, *transactionAllocationSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &transactionAllocationSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *transactionAllocationSvcImpl) ListAllocationEntries(ctx context.Context, params domain.ListAllocationEntriesParams) (*domain.ListAllocationEntriesResult, *apierror.APIError) {
	ctx, span := transactionAllocationSvcTracer.Start(ctx, "service.transaction_allocation.list_entries")
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

	return s.repos.NewTransactionAllocationRepo().ListEntries(ctx, params)
}

func (s *transactionAllocationSvcImpl) UpdateTransactionAllocation(ctx context.Context, params domain.UpdateTransactionAllocationParams) (*domain.TransactionAllocation, *apierror.APIError) {
	ctx, span := transactionAllocationSvcTracer.Start(ctx, "service.transaction_allocation.update")
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

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.TransactionAllocation](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.TransactionAllocation
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *transactionAllocationSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewTransactionAllocationRepo()

			// Verify the allocation exists and belongs to this account
			existing, apiErr := txRepo.GetByID(txCtx, params.AccountID, params.AllocationID)
			if apiErr != nil {
				return apiErr
			}

			// Update amount if provided
			if params.Amount != nil {
				if apiErr := txRepo.UpdateAmount(txCtx, existing.AmountID, *params.Amount); apiErr != nil {
					return apiErr
				}
			}

			// Fetch updated allocation
			updated, apiErr := txRepo.GetByID(txCtx, params.AccountID, params.AllocationID)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(existing, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeTransactionAllocation,
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

func (s *transactionAllocationSvcImpl) DeleteTransactionAllocation(ctx context.Context, params domain.DeleteTransactionAllocationParams) *apierror.APIError {
	ctx, span := transactionAllocationSvcTracer.Start(ctx, "service.transaction_allocation.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSettlements, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewTransactionAllocationRepo()

	// Verify the allocation exists and belongs to this account
	allocation, apiErr := repo.GetByID(ctx, params.AccountID, params.AllocationID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeTransactionAllocation, params.AllocationID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This transaction allocation has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *transactionAllocationSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeTransactionAllocation, allocation.ID, allocation); apiErr != nil {
			return apiErr
		}

		// Delete the allocation (quantity is deleted first inside repo)
		if apiErr := txSvc.repos.NewTransactionAllocationRepo().Delete(txCtx, params.AccountID, params.AllocationID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(allocation, (*domain.TransactionAllocation)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeTransactionAllocation,
			ResourceID:   allocation.ID,
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

func (s *transactionAllocationSvcImpl) ListOpenCredits(ctx context.Context, params domain.ListOpenCreditsParams) (*domain.ListOpenCreditsResult, *apierror.APIError) {
	ctx, span := transactionAllocationSvcTracer.Start(ctx, "service.transaction_allocation.list_open_credits")
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

	return s.repos.NewTransactionAllocationRepo().ListOpenCredits(ctx, params)
}
