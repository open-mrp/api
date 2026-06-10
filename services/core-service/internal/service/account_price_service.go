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

var accountPriceSvcTracer = tracing.GetTracer("core-service.account_price_service")

type accountPriceSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type AccountPriceSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *AccountPriceSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("account price service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("account price service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("account price service: tx manager is required")
	}
	return nil
}

func NewAccountPriceSvc(config *AccountPriceSvcConfig) domain.AccountPriceSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountPriceSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *accountPriceSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *accountPriceSvcImpl) withTx(ctx context.Context, fn func(context.Context, *accountPriceSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &accountPriceSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// ListAccountPrices returns a paginated list of account prices for the caller's account.
// Customer actors can only see prices where they are the recipient.
//
// 1. Extract and validate the caller's identity via CheckIsAssignedActor.
// 2. For internal actors, require discounts:read permission.
// 3. For customer actors, force the recipient account ID filter to their own account.
// 4. Require the Augno-Account header to scope the query.
// 5. Query the account price repository with the account ID and pagination params.
func (s *accountPriceSvcImpl) ListAccountPrices(ctx context.Context, params domain.ListAccountPricesParams) (*domain.ListAccountPricesResult, *apierror.APIError) {
	ctx, span := accountPriceSvcTracer.Start(ctx, "service.account_price.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAccountPriceReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	if identity.IsCustomerUser() {
		customerAccountID := *identity.Actor.AccountID
		params.RecipientAccountID = &customerAccountID
	}

	return s.repos.NewAccountPriceRepo().List(ctx, params)
}

// GetAccountPrice retrieves a single account price by ID, scoped to the caller's account.
// Customer actors can only see prices where they are the recipient.
//
// 1. Extract and validate the caller's identity via CheckIsAssignedActor.
// 2. For internal actors, require discounts:read permission.
// 3. Require the Augno-Account header.
// 4. Fetch the account price from the repository.
// 5. For customer actors, verify the price's recipient matches their own account.
func (s *accountPriceSvcImpl) GetAccountPrice(ctx context.Context, accountPriceID string) (*domain.AccountPrice, *apierror.APIError) {
	ctx, span := accountPriceSvcTracer.Start(ctx, "service.account_price.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAccountPriceReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	accountPrice, apiErr := s.repos.NewAccountPriceRepo().Get(ctx, identity.Target.AccountID, accountPriceID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.IsCustomerUser() {
		customerAccountID := *identity.Actor.AccountID
		if accountPrice.RecipientAccountID != customerAccountID {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account price not found."))
		}
	}

	return accountPrice, nil
}

// CreateAccountPrice creates a new account price with rate, category, and attribute associations.
//
// 1. Extract and validate the caller's identity, actor type (internal), and discounts:create permission.
// 2. Generate unique account price and rate IDs.
// 3. Upsert an idempotency key; if already finished, return the cached response.
// 4. Within a transaction, insert the account price via the repository.
// 5. On error, cache the error response for idempotent replay.
func (s *accountPriceSvcImpl) CreateAccountPrice(ctx context.Context, params domain.CreateAccountPriceParams) (*domain.AccountPrice, *apierror.APIError) {
	ctx, span := accountPriceSvcTracer.Start(ctx, "service.account_price.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountPriceID, apiErr := id.GenID(id.AccountPriceIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rateID, apiErr := id.GenID(id.RateIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountPrice](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.AccountPrice
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountPriceSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAccountPriceRepo()

			created, apiErr := txRepo.Create(txCtx, accountPriceID, rateID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeAccountPrice,
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

// UpdateAccountPrice partially updates an account price, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type (internal), and discounts:update permission.
// 2. Upsert an idempotency key; if already finished, return the cached response.
// 3. Within a transaction, update the account price via the repository.
// 4. On error, cache the error response for idempotent replay.
func (s *accountPriceSvcImpl) UpdateAccountPrice(ctx context.Context, params domain.UpdateAccountPriceParams) (*domain.AccountPrice, *apierror.APIError) {
	ctx, span := accountPriceSvcTracer.Start(ctx, "service.account_price.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.AccountPrice](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.AccountPrice
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountPriceSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAccountPriceRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.AccountPriceID)
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
				ResourceType: constants.ObjectTypeAccountPrice,
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

// DeleteAccountPrice deletes an account price and cascades to associations and rate.
//
// 1. Extract and validate the caller's identity, actor type (internal), and discounts:delete permission.
// 2. Require the Augno-Account header.
// 3. Within a transaction, delete the account price from the repository.
func (s *accountPriceSvcImpl) DeleteAccountPrice(ctx context.Context, accountPriceID string) *apierror.APIError {
	ctx, span := accountPriceSvcTracer.Start(ctx, "service.account_price.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	accountPrice, apiErr := s.repos.NewAccountPriceRepo().Get(ctx, accountID, accountPriceID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeAccountPrice, accountPriceID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(
					span,
					apierror.NewAlreadyDeletedError("This account price has already been deleted and can no longer be modified."),
				)
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *accountPriceSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeAccountPrice, accountPrice.ID, accountPrice); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewAccountPriceRepo().Delete(txCtx, accountID, accountPriceID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(accountPrice, (*domain.AccountPrice)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeAccountPrice,
			ResourceID:   accountPrice.ID,
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

// checkAccountPriceReadPermission checks the appropriate read permission based on the identity context.
// Internal actors need discounts:read for their own account, or customers:read / suppliers:read for external accounts.
func checkAccountPriceReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead)
}
