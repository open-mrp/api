package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
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

	normalizedName, apiErr := normalizeAddressName(params.Name)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.Name = normalizedName

	addressID, apiErr := id.GenID(id.AddressIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	geolocationID, apiErr := id.GenID(id.GeolocationIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountAddressID, apiErr := id.GenID(id.AccountAddressIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Address](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Address
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *addressSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewAddressRepo()

			created, apiErr := txRepo.Create(txCtx, addressID, geolocationID, accountAddressID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeAddress,
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

	normalizedName, apiErr := normalizeOptionalAddressName(params.Name)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.Name = normalizedName

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
			txRepo := txSvc.repos.NewAddressRepo()

			// Verify address is in account
			inAccount, apiErr := txRepo.IsInAccount(txCtx, params.AccountID, params.AddressID)
			if apiErr != nil {
				return apiErr
			}
			if !inAccount {
				return apierror.NewResourceNotFoundError("Address not found.")
			}

			// Fetch existing address to detect core field changes
			existing, apiErr := txRepo.Get(txCtx, domain.GetAddressParams{
				AccountID: params.AccountID,
				AddressID: params.AddressID,
			})
			if apiErr != nil {
				return apiErr
			}

			streetLine2Field := params.StreetLine2

			// Check if core geo fields changed
			coreGeoChanged := false
			if params.StreetLine1 != nil && (existing.Geolocation.StreetLine1 == nil || *params.StreetLine1 != *existing.Geolocation.StreetLine1) {
				coreGeoChanged = true
			}
			if params.Locality != nil && (existing.Geolocation.Locality == nil || *params.Locality != *existing.Geolocation.Locality) {
				coreGeoChanged = true
			}
			if params.State != nil && (existing.Geolocation.State == nil || *params.State != *existing.Geolocation.State) {
				coreGeoChanged = true
			}
			if params.PostalCode != nil && (existing.Geolocation.PostalCode == nil || *params.PostalCode != *existing.Geolocation.PostalCode) {
				coreGeoChanged = true
			}
			if params.Country != nil && *params.Country != existing.Geolocation.Country {
				coreGeoChanged = true
			}

			if coreGeoChanged {
				// Get current geolocation ID
				geoID, apiErr := txRepo.GetGeolocationIDByAddressID(txCtx, params.AddressID)
				if apiErr != nil {
					return apiErr
				}

				// Check if geolocation is shared
				sharedCount, apiErr := txRepo.GetGeolocationSharedCount(txCtx, geoID)
				if apiErr != nil {
					return apiErr
				}

				// Clear google_place_id on geo change
				clearGeoParams := params
				clearGeoParams.StreetLine2 = streetLine2Field.BackfillUnsetPtr(existing.Geolocation.StreetLine2)
				// Build the update params with cleared google_place_id by ensuring we send all geo fields to the update

				if sharedCount > 1 {
					// Shared: create new geolocation and relink
					newGeoID, apiErr := id.GenID(id.GeolocationIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}

					// Build create params from existing + updates
					createParams := domain.CreateAddressParams{
						StreetLine1: coalesceStringPtr(params.StreetLine1, existing.Geolocation.StreetLine1),
						StreetLine2: streetLine2Field.StringPtrAfterBackfill(existing.Geolocation.StreetLine2),
						Locality:    coalesceStringPtr(params.Locality, existing.Geolocation.Locality),
						State:       coalesceStringPtr(params.State, existing.Geolocation.State),
						PostalCode:  coalesceStringPtr(params.PostalCode, existing.Geolocation.PostalCode),
						Country:     coalesceString(params.Country, &existing.Geolocation.Country),
					}

					if apiErr := txRepo.CreateGeolocation(txCtx, newGeoID, createParams); apiErr != nil {
						return apiErr
					}

					if apiErr := txRepo.RelinkGeolocation(txCtx, params.AddressID, newGeoID); apiErr != nil {
						return apiErr
					}
				} else {
					// Not shared: update in-place, clear google_place_id
					if apiErr := txRepo.UpdateGeolocation(txCtx, geoID, clearGeoParams); apiErr != nil {
						return apiErr
					}
				}
			} else {
				// Only metadata changed, but still update line2 on geolocation
				if streetLine2Field.WasProvided() {
					geoID, apiErr := txRepo.GetGeolocationIDByAddressID(txCtx, params.AddressID)
					if apiErr != nil {
						return apiErr
					}
					geoUpdateParams := domain.UpdateAddressParams{
						StreetLine2: streetLine2Field,
					}
					if apiErr := txRepo.UpdateGeolocation(txCtx, geoID, geoUpdateParams); apiErr != nil {
						return apiErr
					}
				}
			}

			params.Phone = params.Phone.BackfillUnsetPtr(existing.Phone)
			params.Email = params.Email.BackfillUnsetPtr(existing.Email)

			// Update address metadata
			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(existing, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeAddress,
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
		wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeAddress, params.AddressID)
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
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeAddress, address.ID, address); apiErr != nil {
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

// checkAddressReadPermission checks the appropriate read permission based on the identity context. For a cross-account target the check is precise (customers:read / suppliers:read). For the actor's own account it accepts any of the read permissions the read endpoints declare, so the downstream check never rejects a caller the coarse gateway gate admitted.
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
	// Own-account: accept any read permission the read endpoints declare, so the
	// downstream check never rejects a caller the coarse gateway gate admitted.
	return identity.CheckHasAnyPermission(
		types.Permission{Domain: types.PermissionDomainAddresses, Action: types.ActionRead},
		types.Permission{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
		types.Permission{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
	)
}

// checkAddressWritePermission checks the appropriate write permission based on the identity context. For a cross-account target the check is precise (customers:update / suppliers:update). For the actor's own account it accepts any of the write permissions the write endpoints declare, so the downstream check never rejects a caller the coarse gateway gate admitted.
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
	// Own-account: accept any write permission the write endpoints declare, so the
	// downstream check never rejects a caller the coarse gateway gate admitted. This
	// mirrors the legacy Dashboard, which gated own-account address writes on
	// customers:update rather than a dedicated addresses domain.
	return identity.CheckHasAnyPermission(
		types.Permission{Domain: types.PermissionDomainAddresses, Action: action},
		types.Permission{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate},
		types.Permission{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
	)
}

func coalesceStringPtr(update *string, existing *string) *string {
	if update != nil {
		return update
	}
	return existing
}

func coalesceString(update *string, existing *string) string {
	if update != nil {
		return *update
	}
	if existing != nil {
		return *existing
	}
	return ""
}

func normalizeAddressName(name string) (string, *apierror.APIError) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", apierror.NewValidationErrorWithParam("Address name is required.", "name")
	}
	return trimmed, nil
}

func normalizeOptionalAddressName(name *string) (*string, *apierror.APIError) {
	if name == nil {
		return nil, nil
	}
	normalized, apiErr := normalizeAddressName(*name)
	if apiErr != nil {
		return nil, apiErr
	}
	return &normalized, nil
}
