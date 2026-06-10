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

var transactionSvcTracer = tracing.GetTracer("core-service.transaction_service")

type transactionSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type TransactionSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *TransactionSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("transaction service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("transaction service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("transaction service: tx manager is required")
	}
	return nil
}

func NewTransactionSvc(config *TransactionSvcConfig) domain.TransactionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &transactionSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *transactionSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *transactionSvcImpl) withTx(ctx context.Context, fn func(context.Context, *transactionSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &transactionSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *transactionSvcImpl) ListTransactions(ctx context.Context, params domain.ListTransactionsParams) (*domain.ListTransactionsResult, *apierror.APIError) {
	ctx, span := transactionSvcTracer.Start(ctx, "service.transaction.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTransactions, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewTransactionRepo().List(ctx, params)
}

func (s *transactionSvcImpl) GetTransaction(ctx context.Context, params domain.GetTransactionParams) (*domain.Transaction, *apierror.APIError) {
	ctx, span := transactionSvcTracer.Start(ctx, "service.transaction.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTransactions, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewTransactionRepo()

	transaction, apiErr := repo.Get(ctx, params.AccountID, params.TransactionID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for _, include := range params.Includes {
		switch include {
		case "allocations":
			allocations, apiErr := repo.GetAllocations(ctx, params.TransactionID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			transaction.Allocations = allocations
		}
	}

	return transaction, nil
}

func (s *transactionSvcImpl) CreateTransaction(ctx context.Context, params domain.CreateTransactionParams) (*domain.Transaction, *apierror.APIError) {
	ctx, span := transactionSvcTracer.Start(ctx, "service.transaction.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTransactions, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	txID, apiErr := id.GenID(id.TransactionIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Transaction](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Transaction
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *transactionSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewTransactionRepo()

			// Auto-populate responsible user from identity if not explicitly provided.
			if params.ResponsibleUserID == nil {
				accountUserRepo := txSvc.repos.NewAccountUserRepo()
				accountUser, apiErr := accountUserRepo.FindByAccountAndUserID(txCtx, identity.Actor.ID, params.AccountID)
				if apiErr != nil {
					return apiErr
				}
				params.ResponsibleUserID = &accountUser.ID
			}

			// Generate transaction number
			number, apiErr := txRepo.FetchAndIncrementTransactionNumber(txCtx, params.AccountID)
			if apiErr != nil {
				return apiErr
			}

			// Get dollar unit ID for the amount
			dollarUnitID, apiErr := txRepo.GetDollarUnitID(txCtx)
			if apiErr != nil {
				return apiErr
			}

			// Create the transaction
			if apiErr := txRepo.Create(txCtx, txID, number, params.TransactionTypeCode, params.AccountID, params.CustomerID, params.StripePaymentID, params.TransactionMethodCode, params.AdjustmentTypeCode, params.ResponsibleUserID, params.Note, params.Amount, dollarUnitID); apiErr != nil {
				return apiErr
			}

			// Fetch the created transaction
			created, apiErr := txRepo.Get(txCtx, params.AccountID, txID)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeTransaction,
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

func (s *transactionSvcImpl) UpdateTransaction(ctx context.Context, params domain.UpdateTransactionParams) (*domain.Transaction, *apierror.APIError) {
	ctx, span := transactionSvcTracer.Start(ctx, "service.transaction.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTransactions, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Transaction](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Transaction
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *transactionSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewTransactionRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.TransactionID)
			if apiErr != nil {
				return apiErr
			}

			if params.Number != nil {
				exists, apiErr := txRepo.ExistsByNumber(txCtx, params.AccountID, *params.Number, &params.TransactionID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A transaction with this number already exists.", "number")
				}
			}

			if params.ResponsibleUserID != nil {
				resolvedID, apiErr := txRepo.ResolveResponsibleUserID(txCtx, params.AccountID, *params.ResponsibleUserID)
				if apiErr != nil {
					return apierror.NewResourceNotFoundError("Account user not found.")
				}
				params.ResponsibleUserID = &resolvedID
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
				ResourceType: constants.ObjectTypeTransaction,
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

func (s *transactionSvcImpl) DeleteTransaction(ctx context.Context, params domain.DeleteTransactionParams) (*domain.Transaction, *apierror.APIError) {
	ctx, span := transactionSvcTracer.Start(ctx, "service.transaction.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTransactions, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewTransactionRepo()

	// Fetch the transaction before deleting
	transaction, apiErr := repo.Get(ctx, params.AccountID, params.TransactionID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeTransaction, params.TransactionID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewAlreadyDeletedError("This transaction has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *transactionSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewTransactionRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeTransaction, transaction.ID, transaction); apiErr != nil {
			return apiErr
		}

		// Delete allocations first
		if apiErr := txRepo.DeleteAllocations(txCtx, params.TransactionID); apiErr != nil {
			return apiErr
		}

		// Delete the quantity
		if apiErr := txRepo.DeleteQuantity(txCtx, transaction.AmountID); apiErr != nil {
			return apiErr
		}

		// Delete the transaction
		if apiErr := txRepo.Delete(txCtx, params.TransactionID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(transaction, (*domain.Transaction)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeTransaction,
			ResourceID:   transaction.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return transaction, nil
}

func (s *transactionSvcImpl) ListAccountTransactions(ctx context.Context, params domain.ListAccountTransactionsParams) (*domain.ListAccountTransactionsResult, *apierror.APIError) {
	ctx, span := transactionSvcTracer.Start(ctx, "service.transaction.list_by_customer")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTransactions, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewTransactionRepo()

	result, apiErr := repo.ListByCustomer(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Fetch allocations for each transaction to match legacy Dashboard behavior.
	for _, tx := range result.Transactions {
		allocations, apiErr := repo.GetAllocations(ctx, tx.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		tx.Allocations = allocations
	}

	return result, nil
}
