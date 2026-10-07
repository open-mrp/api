package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var addressSvcTracer = tracing.GetTracer("core-service.address_service")

type addressSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type AddressSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *AddressSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("address service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("address service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("address service: tx manager is required")
	}
	return nil
}

func NewAddressSvc(config *AddressSvcConfig) domain.AddressSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &addressSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *addressSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *addressSvcImpl) withTx(ctx context.Context, fn func(context.Context, *addressSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &addressSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *addressSvcImpl) ListAddresses(ctx context.Context, params domain.ListAddressesParams) (*domain.ListAddressesResult, *apierror.APIError) {
	ctx, span := addressSvcTracer.Start(ctx, "service.address.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAddressReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID, apiErr := resolveAddressAccountScope(identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.IsInternalActor() && identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = accountID

	return s.repos.NewAddressRepo().List(ctx, params)
}

func (s *addressSvcImpl) GetAddress(ctx context.Context, params domain.GetAddressParams) (*domain.Address, *apierror.APIError) {
	ctx, span := addressSvcTracer.Start(ctx, "service.address.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAddressReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID, apiErr := resolveAddressAccountScope(identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.IsInternalActor() && identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = accountID

	return s.repos.NewAddressRepo().Get(ctx, params)
}

// BatchGetAddressesByIDs returns addresses matching the input IDs that the caller's account is authorized to read. Addresses are always account-scoped via the account_address junction.
func (s *addressSvcImpl) BatchGetAddressesByIDs(ctx context.Context, ids []string) ([]*domain.Address, *apierror.APIError) {
	ctx, span := addressSvcTracer.Start(ctx, "service.address.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAddressReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID, apiErr := resolveAddressAccountScope(identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.IsInternalActor() && identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewAddressRepo().GetByIDs(ctx, accountID, ids)
}

func (s *addressSvcImpl) CreateAddress(ctx context.Context, params domain.CreateAddressParams) (*domain.Address, *apierror.APIError) {
	ctx, span := addressSvcTracer.Start(ctx, "service.address.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAddressWritePermission(identity, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID, apiErr := resolveAddressAccountScope(identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Only internal actors reach across accounts (a merchant creating an address in
	// a customer/supplier account they own); relation actors are scoped to their own
	// account above, so no cross-account edit check applies to them.
	if identity.IsInternalActor() && identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = accountID

	if _, apiErr := mediator.NormalizeAddressName(params.Name); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Address](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Address
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *addressSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()
			created, apiErr := txMeds.Address.Create(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = created
			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *addressSvcImpl) UpdateAddress(ctx context.Context, params domain.UpdateAddressParams) (*domain.Address, *apierror.APIError) {
	ctx, span := addressSvcTracer.Start(ctx, "service.address.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkAddressWritePermission(identity, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID, apiErr := resolveAddressAccountScope(identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if identity.IsInternalActor() && identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = accountID

	if _, apiErr := mediator.NormalizeOptionalAddressName(params.Name); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Address](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Address
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *addressSvcImpl) *apierror.APIError {
			txMeds := txSvc.mediators()
			updated, apiErr := txMeds.Address.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated
			return txMeds.Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, result)
		})

		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return result, nil

	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *addressSvcImpl) DeleteAddress(ctx context.Context, params domain.DeleteAddressParams) *apierror.APIError {
	ctx, span := addressSvcTracer.Start(ctx, "service.address.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := checkAddressWritePermission(identity, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID, apiErr := resolveAddressAccountScope(identity)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if identity.IsInternalActor() && identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.EditAccess.CheckEditAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = accountID

	repo := s.repos.NewAddressRepo()

	// Verify address is in account
	inAccount, apiErr := repo.IsInAccount(ctx, params.AccountID, params.AddressID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !inAccount {
		wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeAddress, params.AddressID, params.AccountID)
		if deletedCheckErr != nil {
			return tracing.Trace(span, deletedCheckErr)
		}
		if wasDeleted {
			return tracing.Trace(span, apierror.NewAlreadyDeletedError("This address has already been deleted and can no longer be modified."))
		}
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Address not found."))
	}

	// Check not in use
	if apiErr := repo.CheckAddressNotInUse(ctx, params.AddressID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Fetch address before deletion for audit
	address, apiErr := repo.Get(ctx, domain.GetAddressParams(params))
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Delete in transaction
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *addressSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeAddress, address.ID, params.AccountID, address); apiErr != nil {
			return apiErr
		}

		txRepo := txSvc.repos.NewAddressRepo()

		// A non-active account may still default to this address (CheckAddressNotInUse only blocks
		// active-account defaults). Switch those defaults over to the account-relation defaults first
		// so the account keeps a valid default instead of a pointer to the row we're about to delete.
		if apiErr := txRepo.SwitchAccountDefaultAddressToRelation(txCtx, params.AddressID); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.Delete(txCtx, params); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(address, (*domain.Address)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeAddress,
			ResourceID:   address.ID,
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

// resolveAddressAccountScope determines which account an address operation is
// scoped to, mirroring the Dashboard's assertCanActOnAccount /
// resolveAccountForAddressOwnership:
//   - Internal actors operate on the target account (their own account, or a
//     customer/supplier account they own). Cross-account access is separately
//     guarded by EditAccess (writes) / ReadAccess (reads) at the call site.
//   - Customer/supplier relation actors operate only on their OWN account. Portal
//     and order addresses live in the actor's (buyer) account, not the
//     counterparty account they are transacting with, so the target account (e.g.
//     the merchant the customer is ordering from) must not be used as the scope.
func resolveAddressAccountScope(identity *types.Identity) (string, *apierror.APIError) {
	if identity.IsInternalActor() {
		if !identity.IsTargetAccountSet() {
			return "", apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required.")
		}
		return identity.Target.AccountID, nil
	}
	actorAccountID := identity.ActorAccountID()
	if actorAccountID == nil {
		return "", apierror.NewAuthorizationError("You must be assigned to an account to manage addresses.")
	}
	return *actorAccountID, nil
}

// checkAddressReadPermission checks the appropriate read permission based on the identity context: addresses:read in the actor's own account, customers:read in a customer's account and suppliers:read in a supplier's.
func checkAddressReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	// TODO: implement a proper default permission strategy for users without roles.
	// For now, users with no role are granted full address access.
	if !identity.IsRoleSet() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainAddresses, types.ActionRead)
}

// checkAddressWritePermission checks the appropriate write permission based on the identity context: addresses:{action} in the actor's own account, customers:update in a customer's account and suppliers:update in a supplier's.
func checkAddressWritePermission(identity *types.Identity, action types.Action) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	// TODO: implement a proper default permission strategy for users without roles.
	if !identity.IsRoleSet() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionUpdate)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate)
	}
	return identity.CheckHasPermission(types.PermissionDomainAddresses, action)
}
