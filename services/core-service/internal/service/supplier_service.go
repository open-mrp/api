package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/tracing"
)

var supplierSvcTracer = tracing.GetTracer("core-service.supplier_service")

type supplierSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type SupplierSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *SupplierSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("supplier service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("supplier service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("supplier service: tx manager is required")
	}
	return nil
}

func NewSupplierSvc(config *SupplierSvcConfig) domain.SupplierSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &supplierSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *supplierSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *supplierSvcImpl) withTx(ctx context.Context, fn func(context.Context, *supplierSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &supplierSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *supplierSvcImpl) ListSuppliers(ctx context.Context, params domain.ListSuppliersParams) (*domain.ListSuppliersResult, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.list")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	return s.repos.NewSupplierRepo().List(ctx, params)
}

// BatchGetSuppliersByIDs serves the suppliers purchase and receiving orders name, so it admits whoever may read those orders.
func (s *supplierSvcImpl) BatchGetSuppliersByIDs(ctx context.Context, ids, includes []string) ([]*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActorForRead(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasAnyPermission(
		types.Permission{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		types.Permission{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionRead},
		types.Permission{Domain: types.PermissionDomainReceivingOrders, Action: types.ActionRead},
	); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewSupplierRepo().GetByIDs(ctx, identity.Target.AccountID, ids, includes)
}

func (s *supplierSvcImpl) GetSupplier(ctx context.Context, params domain.GetSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.get")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Internal actors need suppliers:read permission.
	// Supplier actors can view their own record.
	if identity.IsSupplierUser() {
		if identity.Actor.AccountID == nil || *identity.Actor.AccountID != params.SupplierID {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this supplier."))
		}
	} else {
		if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.OwnerAccountID = identity.Target.AccountID
	return s.repos.NewSupplierRepo().Get(ctx, params)
}

func (s *supplierSvcImpl) CreateSupplier(ctx context.Context, params domain.CreateSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.create")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if params.ArchiveStatus == "" {
		params.ArchiveStatus = constants.ArchiveStatusActive
	}
	if apiErr := checkSupplierEnums(&params.ArchiveStatus, params.CarrierBillingType); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID, apiErr := id.GenID(id.AccountIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	relationID, apiErr := id.GenID(id.AccountRelationIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Supplier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Supplier
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierSvcImpl) *apierror.APIError {
			txSupplierRepo := txSvc.repos.NewSupplierRepo()

			// Every create takes a number, typed in or allocated, so it queues behind the owner's other number
			// writers before anything else is read.
			if apiErr := txSupplierRepo.LockNumbers(txCtx, params.OwnerAccountID); apiErr != nil {
				return apiErr
			}

			if params.Number != "" {
				exists, apiErr := txSupplierRepo.ExistsByNumber(txCtx, params.OwnerAccountID, params.Number, nil)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("number", "A supplier with this number already exists.")
				}
			} else {
				sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				next, apiErr := txSupplierRepo.AllocateNextNumber(txCtx, sysPropertyID, params.OwnerAccountID)
				if apiErr != nil {
					return apiErr
				}
				params.Number = strconv.FormatInt(next, 10)
			}

			var refs customerRefs
			refs.add(customerRefCarrier, params.DefaultCarrierID, nil, "defaults.carrier_id")
			refs.add(customerRefServiceLevel, params.DefaultServiceLevelID, nil, "defaults.service_level_id")
			refs.add(customerRefPaymentTerm, params.PaymentTermID, nil, "defaults.payment_term_id")
			refs.add(customerRefShippingTerm, params.ShippingTermID, nil, "defaults.shipping_term_id")
			if apiErr := checkCustomerRefs(txCtx, txSvc.repos, params.OwnerAccountID, refs); apiErr != nil {
				return apiErr
			}
			if apiErr := checkServiceLevelOnCarrier(txCtx, txSvc.repos, params.DefaultCarrierID, params.DefaultServiceLevelID, "defaults.service_level_id"); apiErr != nil {
				return apiErr
			}

			var billToAddressID *string
			if params.BillToAddress != nil {
				addrID, geoID, acctAddrID, err := generateAddressIDs()
				if err != nil {
					return err
				}
				params.BillToAddress.AccountID = accountID
				if _, apiErr := txSvc.repos.NewAddressRepo().Create(txCtx, addrID, geoID, acctAddrID, *params.BillToAddress); apiErr != nil {
					return apiErr
				}
				billToAddressID = &addrID
			}

			// The ship-to reuses the bill-to when it is the same address, and defaults to it when omitted.
			var shipToAddressID *string
			if params.ShipToAddress != nil && params.BillToAddress != nil && addressParamsEqual(*params.BillToAddress, *params.ShipToAddress) {
				shipToAddressID = billToAddressID
			} else if params.ShipToAddress != nil {
				addrID, geoID, acctAddrID, err := generateAddressIDs()
				if err != nil {
					return err
				}
				params.ShipToAddress.AccountID = accountID
				if _, apiErr := txSvc.repos.NewAddressRepo().Create(txCtx, addrID, geoID, acctAddrID, *params.ShipToAddress); apiErr != nil {
					return apiErr
				}
				shipToAddressID = &addrID
			}
			if shipToAddressID == nil && billToAddressID != nil {
				shipToAddressID = billToAddressID
			}

			userIncludes := params.Includes
			params.Includes = nil
			if _, apiErr := txSupplierRepo.Create(txCtx, accountID, relationID, params, billToAddressID, shipToAddressID); apiErr != nil {
				return apiErr
			}
			if params.Email != nil || params.Phone != nil || params.URL != nil {
				if apiErr := txSvc.repos.NewCustomerRepo().UpdateBranding(txCtx, accountID, params.Email, params.Phone, params.URL); apiErr != nil {
					return apiErr
				}
			}
			created, apiErr := txSupplierRepo.Get(txCtx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: accountID, Includes: userIncludes})
			if apiErr != nil {
				return apiErr
			}
			result = created

			if _, apiErr := txSvc.mediators().Site.CreateDefaultSite(txCtx, accountID); apiErr != nil {
				return apiErr
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeSupplier,
				ResourceID:   created.ID,
				Changes:      audit.ComputeChanges(nil, created),
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

func (s *supplierSvcImpl) UpdateSupplier(ctx context.Context, params domain.UpdateSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.update")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkSupplierEnums(params.ArchiveStatus, params.CarrierBillingType); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkSupplierNotBlank(params); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[domain.Supplier](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Supplier
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierSvcImpl) *apierror.APIError {
			txSupplierRepo := txSvc.repos.NewSupplierRepo()

			// Before the first read, or the number check below reads a snapshot older than the lock.
			if params.Number != nil {
				if apiErr := txSupplierRepo.LockNumbers(txCtx, params.OwnerAccountID); apiErr != nil {
					return apiErr
				}
			}

			old, apiErr := txSupplierRepo.Get(txCtx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: params.SupplierID})
			if apiErr != nil {
				return apiErr
			}

			if params.Number != nil && *params.Number != old.Number {
				exists, apiErr := txSupplierRepo.ExistsByNumber(txCtx, params.OwnerAccountID, *params.Number, &params.SupplierID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("number", "A supplier with this number already exists.")
				}
			}

			var refs customerRefs
			refs.add(customerRefCarrier, params.DefaultCarrierID.ValuePtr(), old.DefaultCarrierID, "defaults.carrier_id")
			refs.add(customerRefServiceLevel, params.DefaultServiceLevelID.ValuePtr(), old.DefaultServiceLevelID, "defaults.service_level_id")
			refs.add(customerRefPaymentTerm, params.PaymentTermID.ValuePtr(), old.PaymentTermID, "defaults.payment_term_id")
			refs.add(customerRefShippingTerm, params.ShippingTermID.ValuePtr(), old.ShippingTermID, "defaults.shipping_term_id")
			if apiErr := checkCustomerRefs(txCtx, txSvc.repos, params.OwnerAccountID, refs); apiErr != nil {
				return apiErr
			}

			w := mergeSupplierWrite(params, old)
			if !equalStringPtr(w.DefaultCarrierID, old.DefaultCarrierID) || !equalStringPtr(w.DefaultServiceLevelID, old.DefaultServiceLevelID) {
				if apiErr := checkServiceLevelOnCarrier(txCtx, txSvc.repos, w.DefaultCarrierID, w.DefaultServiceLevelID, "defaults.service_level_id"); apiErr != nil {
					return apiErr
				}
			}
			if apiErr := txSvc.checkSupplierAddress(txCtx, params.SupplierID, params.BillToAddressID.ValuePtr(), old.BillToAddressID, "defaults.bill_to_address_id"); apiErr != nil {
				return apiErr
			}
			if apiErr := txSvc.checkSupplierAddress(txCtx, params.SupplierID, params.ShipToAddressID.ValuePtr(), old.ShipToAddressID, "defaults.ship_to_address_id"); apiErr != nil {
				return apiErr
			}

			if params.Email.WasProvided() || params.Phone.WasProvided() || params.URL.WasProvided() {
				if apiErr := txSvc.repos.NewCustomerRepo().UpdateBranding(txCtx, params.SupplierID,
					clearableOr(params.Email, old.Email), clearableOr(params.Phone, old.Phone), clearableOr(params.URL, old.URL)); apiErr != nil {
					return apiErr
				}
			}

			updated, apiErr := txSupplierRepo.Update(txCtx, w, nil)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSupplier,
				ResourceID:   updated.ID,
				Changes:      audit.ComputeChanges(old, updated),
			}); apiErr != nil {
				return apiErr
			}

			result, apiErr = txSupplierRepo.Get(txCtx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: params.SupplierID, Includes: params.Includes})
			if apiErr != nil {
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

// mergeSupplierWrite is the relation as an update leaves it: each field sent, and what the supplier held otherwise.
func mergeSupplierWrite(params domain.UpdateSupplierParams, old *domain.Supplier) domain.SupplierRelationWrite {
	w := domain.SupplierRelationWrite{
		OwnerAccountID:        params.OwnerAccountID,
		SupplierID:            params.SupplierID,
		Name:                  old.Name,
		Number:                old.Number,
		IsActive:              old.ArchiveStatus.IsActive(),
		Instructions:          clearableOr(params.Instructions, old.Instructions),
		CarrierBillingType:    old.CarrierBillingType,
		CarrierBillingAccount: clearableOr(params.CarrierBillingAccount, old.CarrierBillingAccount),
		DefaultCarrierID:      clearableOr(params.DefaultCarrierID, old.DefaultCarrierID),
		DefaultServiceLevelID: clearableOr(params.DefaultServiceLevelID, old.DefaultServiceLevelID),
		PaymentTermID:         clearableOr(params.PaymentTermID, old.PaymentTermID),
		ShippingTermID:        clearableOr(params.ShippingTermID, old.ShippingTermID),
		BillToAddressID:       clearableOr(params.BillToAddressID, old.BillToAddressID),
		ShipToAddressID:       clearableOr(params.ShipToAddressID, old.ShipToAddressID),
	}
	if params.Name != nil {
		w.Name = *params.Name
	}
	if params.Number != nil {
		w.Number = *params.Number
	}
	if params.ArchiveStatus != nil {
		w.IsActive = params.ArchiveStatus.IsActive()
	}
	if params.CarrierBillingType != nil {
		w.CarrierBillingType = params.CarrierBillingType
	}
	return w
}

// clearableOr is c's value when it was sent (nil when cleared), else held.
func clearableOr(c field.Clearable[string], held *string) *string {
	if !c.WasProvided() {
		return held
	}
	return c.ValuePtr()
}

// checkSupplierEnums refuses a status or carrier billing type outside its values.
func checkSupplierEnums(status *constants.ArchiveStatus, billing *constants.CarrierBillingType) *apierror.APIError {
	if status != nil && !status.IsValid() {
		return apierror.NewFieldError("status", apierror.CodeInvalidFormat, "status must be one of active or archived.")
	}
	if billing != nil && *billing != constants.CarrierBillingTypeSender && *billing != constants.CarrierBillingTypeThirdParty {
		return apierror.NewFieldError("carrier_billing.type", apierror.CodeInvalidFormat, "carrier_billing.type must be one of sender or third_party.")
	}
	return nil
}

// checkSupplierNotBlank refuses a blank value where null is how a field is cleared.
func checkSupplierNotBlank(params domain.UpdateSupplierParams) *apierror.APIError {
	for _, f := range []struct {
		value field.Clearable[string]
		param string
	}{
		{params.Instructions, "instructions"},
		{params.DefaultCarrierID, "defaults.carrier_id"},
		{params.DefaultServiceLevelID, "defaults.service_level_id"},
		{params.PaymentTermID, "defaults.payment_term_id"},
		{params.ShippingTermID, "defaults.shipping_term_id"},
		{params.BillToAddressID, "defaults.bill_to_address_id"},
		{params.ShipToAddressID, "defaults.ship_to_address_id"},
	} {
		if v, ok := f.value.Value(); ok && strings.TrimSpace(v) == "" {
			return apierror.NewFieldError(f.param, apierror.CodeInvalidFormat, "Field '"+f.param+"' must not be blank; send null to clear it.")
		}
	}
	return nil
}

func (s *supplierSvcImpl) DeleteSupplier(ctx context.Context, params domain.DeleteSupplierParams) (*domain.Supplier, *apierror.APIError) {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.delete")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionDelete); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	supplier, apiErr := s.repos.NewSupplierRepo().Get(ctx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: params.SupplierID})
	if apiErr != nil {
		if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeSupplier, params.SupplierID, params.OwnerAccountID)
			if deletedCheckErr != nil {
				return nil, tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return nil, tracing.Trace(span, apierror.NewGoneError("This supplier has already been deleted and can no longer be modified."))
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *supplierSvcImpl) *apierror.APIError {
		return deleteSupplierInTx(txCtx, txSvc.repos, params.OwnerAccountID, supplier)
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return supplier, nil
}

// deleteSupplierInTx refuses a supplier that purchase orders or supplier items still reference, then deletes it,
// leaving a tombstone and an audit event.
func deleteSupplierInTx(ctx context.Context, repos domain.RepoFactory, ownerAccountID string, supplier *domain.Supplier) *apierror.APIError {
	labels, apiErr := repos.NewSupplierRepo().InUseBy(ctx, ownerAccountID, supplier.ID)
	if apiErr != nil {
		return apiErr
	}
	if len(labels) > 0 {
		return domain.SupplierInUse.InUseError(labels)
	}
	if apiErr := repos.NewDeletedRecordRepo().CreateInAccount(ctx, constants.DeletedRecordResourceTypeSupplier, supplier.ID, ownerAccountID, supplier); apiErr != nil {
		return apiErr
	}
	if apiErr := repos.NewSupplierRepo().Delete(ctx, ownerAccountID, supplier.ID); apiErr != nil {
		return apiErr
	}
	return audit.NewPublisher().Publish(ctx, repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionDelete,
		ResourceType: constants.ObjectTypeSupplier,
		ResourceID:   supplier.ID,
		Changes:      audit.ComputeChanges(supplier, (*domain.Supplier)(nil)),
	})
}

func (s *supplierSvcImpl) BulkDeleteSuppliers(ctx context.Context, params domain.BulkDeleteSuppliersParams) *apierror.APIError {
	ctx, span := supplierSvcTracer.Start(ctx, "service.supplier.bulk_delete")
	defer span.End()

	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	supplierRepo := s.repos.NewSupplierRepo()
	suppliers := make([]*domain.Supplier, 0, len(params.SupplierIDs))
	for _, supplierID := range params.SupplierIDs {
		supplier, apiErr := supplierRepo.Get(ctx, domain.GetSupplierParams{OwnerAccountID: params.OwnerAccountID, SupplierID: supplierID})
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		suppliers = append(suppliers, supplier)
	}

	// All or none: one supplier still in use refuses the whole request.
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *supplierSvcImpl) *apierror.APIError {
		for _, supplier := range suppliers {
			if apiErr := deleteSupplierInTx(txCtx, txSvc.repos, params.OwnerAccountID, supplier); apiErr != nil {
				return apiErr
			}
		}
		return nil
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// checkSupplierAddress refuses a default address that is not one of the supplier's own. Another account's
// address would otherwise be read back through this supplier.
func (s *supplierSvcImpl) checkSupplierAddress(ctx context.Context, supplierID string, addressID, current *string, param string) *apierror.APIError {
	if addressID == nil || (current != nil && *current == *addressID) {
		return nil
	}
	inAccount, apiErr := s.repos.NewAddressRepo().IsInAccount(ctx, supplierID, *addressID)
	if apiErr != nil {
		return apiErr
	}
	if !inAccount {
		return apierror.NewNotFoundError("No address of this supplier has the provided ID.").WithParam(param)
	}
	return nil
}

// generateAddressIDs creates IDs for address, geolocation, and account_address records.
func generateAddressIDs() (addressID, geolocationID, accountAddressID string, apiErr *apierror.APIError) {
	addressID, apiErr = id.GenID(id.AddressIDPrefix, nil)
	if apiErr != nil {
		return "", "", "", apiErr
	}
	geolocationID, apiErr = id.GenID(id.GeolocationIDPrefix, nil)
	if apiErr != nil {
		return "", "", "", apiErr
	}
	accountAddressID, apiErr = id.GenID(id.AccountAddressIDPrefix, nil)
	if apiErr != nil {
		return "", "", "", apiErr
	}
	return addressID, geolocationID, accountAddressID, nil
}
