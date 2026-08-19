package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var purchaseOrderSvcTracer = tracing.GetTracer("core-service.purchase_order_service")

type purchaseOrderSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	txManager             TransactionManager
	notificationPublisher domain.NotificationPublisher
}

type PurchaseOrderSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// NotificationPublisher (optional; default: nil) publishes notification messages to the outbox. When nil, status-change emails (e.g. purchase-order submission on issue) are skipped. It is not validated at construction.
	NotificationPublisher domain.NotificationPublisher
}

func (c *PurchaseOrderSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("purchase order service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("purchase order service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("purchase order service: tx manager is required")
	}
	return nil
}

func NewPurchaseOrderSvc(config *PurchaseOrderSvcConfig) domain.PurchaseOrderSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &purchaseOrderSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		txManager:             config.TxManager,
		notificationPublisher: config.NotificationPublisher,
	}
}

func (s *purchaseOrderSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *purchaseOrderSvcImpl) withTx(ctx context.Context, fn func(context.Context, *purchaseOrderSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &purchaseOrderSvcImpl{
			repos:                 f,
			mediatorFactory:       s.mediatorFactory,
			txManager:             s.txManager,
			notificationPublisher: s.notificationPublisher,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *purchaseOrderSvcImpl) ListPurchaseOrders(ctx context.Context, params domain.ListPurchaseOrdersParams) (*domain.ListPurchaseOrdersResult, *apierror.APIError) {
	ctx, span := purchaseOrderSvcTracer.Start(ctx, "service.purchase_order.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPurchaseOrders, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewPurchaseOrderRepo()
	result, apiErr := repo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Expand lines per order only when requested (so the list can serve the lines.item array filter and list rows that render line data).
	for _, include := range params.Includes {
		if include == "lines" {
			for _, order := range result.PurchaseOrders {
				lines, apiErr := repo.GetLines(ctx, order.ID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				order.Lines = lines
			}
			break
		}
	}

	return result, nil
}

func (s *purchaseOrderSvcImpl) GetPurchaseOrder(ctx context.Context, params domain.GetPurchaseOrderParams) (*domain.PurchaseOrder, *apierror.APIError) {
	ctx, span := purchaseOrderSvcTracer.Start(ctx, "service.purchase_order.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPurchaseOrders, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewPurchaseOrderRepo()

	order, apiErr := repo.Get(ctx, params.AccountID, params.PurchaseOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Conditionally fetch lines and contacts based on includes
	for _, include := range params.Includes {
		switch include {
		case "lines":
			lines, apiErr := repo.GetLines(ctx, params.PurchaseOrderID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			order.Lines = lines
		case "contacts":
			contacts, apiErr := repo.GetEmailContacts(ctx, params.PurchaseOrderID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			order.Contacts = contacts
		case "receiving_order":
			if order.ReceivingOrderID != nil {
				receivingOrder, apiErr := s.repos.NewReceivingOrderRepo().Get(ctx, params.AccountID, *order.ReceivingOrderID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				order.ReceivingOrder = receivingOrder
			}
		}
	}

	return order, nil
}

func (s *purchaseOrderSvcImpl) CreatePurchaseOrder(ctx context.Context, params domain.CreatePurchaseOrderParams) (*domain.PurchaseOrder, *apierror.APIError) {
	ctx, span := purchaseOrderSvcTracer.Start(ctx, "service.purchase_order.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPurchaseOrders, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// The supplier has to actually be one of this account's suppliers. Without the check any account ID is accepted, so an order can be placed against a customer — or an account in another tenancy — and the order then names a counterparty nobody agreed to buy from. Checked before the idempotency key is taken, so a rejected request is not remembered as an attempt.
	if _, apiErr := s.repos.NewSupplierRepo().Get(ctx, domain.GetSupplierParams{
		OwnerAccountID: params.AccountID,
		SupplierID:     params.SupplierAccountID,
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Checked here rather than inside the transaction so a bad unit is refused before an order number is taken and an idempotency key is spent.
	if apiErr := validatePurchaseOrderLineUnits(ctx, s.repos, params.AccountID, params.Lines, "quantity_unit_id"); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.PurchaseOrder](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		orderID, apiErr := id.GenID(id.OrderIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.PurchaseOrder
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderSvcImpl) *apierror.APIError {
			txOrderRepo := txSvc.repos.NewPurchaseOrderRepo()
			txAddressRepo := txSvc.repos.NewAddressRepo()
			txLineRepo := txSvc.repos.NewPurchaseOrderLineRepo()

			// Get next order number
			orderNumber, apiErr := txOrderRepo.GetNextOrderNumber(txCtx, params.AccountID)
			if apiErr != nil {
				return apiErr
			}

			// Check duplicate order number
			isDup, apiErr := txOrderRepo.IsDuplicateOrderNumber(txCtx, params.AccountID, orderNumber, nil)
			if apiErr != nil {
				return apiErr
			}
			if isDup {
				return apierror.NewConflictErrorWithParam("A purchase order with this number already exists.", "number")
			}

			// Create billing address
			billAddrID, apiErr := id.GenID(id.AddressIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			billGeoID, apiErr := id.GenID(id.GeolocationIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			billAcctAddrID, apiErr := id.GenID(id.AccountAddressIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}

			billName := ""
			if params.BillToName != nil {
				billName = *params.BillToName
			}
			billCountry := ""
			if params.BillToCountry != nil {
				billCountry = *params.BillToCountry
			}

			_, apiErr = txAddressRepo.Create(txCtx, billAddrID, billGeoID, billAcctAddrID, domain.CreateAddressParams{
				AccountID:   params.AccountID,
				Name:        billName,
				StreetLine1: params.BillToStreetLine1,
				StreetLine2: params.BillToStreetLine2,
				Locality:    params.BillToLocality,
				State:       params.BillToState,
				PostalCode:  params.BillToPostalCode,
				Country:     billCountry,
			})
			if apiErr != nil {
				return apiErr
			}

			// Create shipping address
			shipAddrID, apiErr := id.GenID(id.AddressIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			shipGeoID, apiErr := id.GenID(id.GeolocationIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			shipAcctAddrID, apiErr := id.GenID(id.AccountAddressIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}

			shipName := ""
			if params.ShipToName != nil {
				shipName = *params.ShipToName
			}
			shipCountry := ""
			if params.ShipToCountry != nil {
				shipCountry = *params.ShipToCountry
			}

			_, apiErr = txAddressRepo.Create(txCtx, shipAddrID, shipGeoID, shipAcctAddrID, domain.CreateAddressParams{
				AccountID:   params.AccountID,
				Name:        shipName,
				StreetLine1: params.ShipToStreetLine1,
				StreetLine2: params.ShipToStreetLine2,
				Locality:    params.ShipToLocality,
				State:       params.ShipToState,
				PostalCode:  params.ShipToPostalCode,
				Country:     shipCountry,
			})
			if apiErr != nil {
				return apiErr
			}

			// Create the order buyer_account_id = owner_account_id (our account), seller_account_id = supplier_account_id
			createParams := domain.CreatePurchaseOrderParams{
				AccountID:             params.AccountID,
				SupplierAccountID:     params.SupplierAccountID,
				Number:                orderNumber,
				SalesOrderStatusCode:  "estimate",
				BillingAddressID:      billAddrID,
				ShippingAddressID:     shipAddrID,
				Note:                  params.Note,
				CarrierID:             params.CarrierID,
				ServiceLevelID:        params.ServiceLevelID,
				CarrierBillingType:    params.CarrierBillingType,
				CarrierBillingAccount: params.CarrierBillingAccount,
				PriorityCode:          params.PriorityCode,
				ShippingTermID:        params.ShippingTermID,
				PaymentTermID:         params.PaymentTermID,
				PromisedAt:            params.PromisedAt,
			}

			_, apiErr = txOrderRepo.Create(txCtx, orderID, createParams)
			if apiErr != nil {
				return apiErr
			}

			// Create order lines
			for _, lineInput := range params.Lines {
				lineID, apiErr := id.GenID(id.OrderLineIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				lineParams := domain.CreatePurchaseOrderLineParams{
					SalesOrderID:               orderID,
					AccountID:                  params.AccountID,
					ProductID:                  lineInput.ProductID,
					ItemID:                     lineInput.ItemID,
					ProductSKU:                 lineInput.ProductSKU,
					ProductDescription:         lineInput.ProductDescription,
					QuantityValue:              lineInput.QuantityValue,
					QuantityUnitID:             lineInput.QuantityUnitID,
					UnitPriceValue:             lineInput.UnitPriceValue,
					UnitPriceNumeratorUnitID:   lineInput.UnitPriceNumeratorUnitID,
					UnitPriceDenominatorUnitID: lineInput.UnitPriceDenominatorUnitID,
					UnitCostValue:              lineInput.UnitCostValue,
					UnitCostNumeratorUnitID:    lineInput.UnitCostNumeratorUnitID,
					UnitCostDenominatorUnitID:  lineInput.UnitCostDenominatorUnitID,
				}

				_, apiErr = txLineRepo.Create(txCtx, lineID, lineParams)
				if apiErr != nil {
					return apiErr
				}

				// Ensure supplier material link for lines with an item_id
				if lineInput.ItemID != nil {
					if apiErr := ensureSupplierMaterialLink(txCtx, txSvc.repos, params.AccountID, orderID, *lineInput.ItemID, lineInput.ProductSKU); apiErr != nil {
						return apiErr
					}
				}
			}

			// Create email contacts
			for _, accountUserID := range params.ContactAccountUserIDs {
				contactID, apiErr := id.GenID(id.OrderEmailIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				if apiErr := txOrderRepo.CreateEmailContact(txCtx, contactID, orderID, accountUserID, "purchaseOrderSubmission"); apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch the complete order
			order, apiErr := txOrderRepo.Get(txCtx, params.AccountID, orderID)
			if apiErr != nil {
				return apiErr
			}

			if slices.Contains(params.Includes, "lines") {
				lines, apiErr := txOrderRepo.GetLines(txCtx, orderID)
				if apiErr != nil {
					return apiErr
				}
				order.Lines = lines
			}
			if slices.Contains(params.Includes, "contacts") {
				contacts, apiErr := txOrderRepo.GetEmailContacts(txCtx, orderID)
				if apiErr != nil {
					return apiErr
				}
				order.Contacts = contacts
			}

			result = order

			changes := audit.ComputeChanges(nil, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypePurchaseOrder,
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

func (s *purchaseOrderSvcImpl) UpdatePurchaseOrder(ctx context.Context, params domain.UpdatePurchaseOrderParams) (*domain.PurchaseOrder, *apierror.APIError) {
	ctx, span := purchaseOrderSvcTracer.Start(ctx, "service.purchase_order.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPurchaseOrders, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.PurchaseOrder](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.PurchaseOrder
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPurchaseOrderRepo()

			// Fetch old state for audit diff
			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.PurchaseOrderID)
			if apiErr != nil {
				return apiErr
			}

			// Backfill unchanged nullable fields with existing values. Since the SQL uses direct assignment (no COALESCE) for these fields, we must provide the existing value when the field was not sent.
			if params.BillingAddressID == nil {
				params.BillingAddressID = &old.BillingAddressID
			}
			if params.ShippingAddressID == nil {
				params.ShippingAddressID = &old.ShippingAddressID
			}

			// Check duplicate order number if provided
			if params.Number != nil && *params.Number != "" {
				isDup, apiErr := txRepo.IsDuplicateOrderNumber(txCtx, params.AccountID, *params.Number, &params.PurchaseOrderID)
				if apiErr != nil {
					return apiErr
				}
				if isDup {
					return apierror.NewConflictErrorWithParam("A purchase order with this number already exists.", "number")
				}
			}

			// Update the order
			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}

			// Handle contact replacement if contacts are provided
			if params.ContactAccountUserIDs != nil {
				// Delete old contacts
				if apiErr := txRepo.DeleteEmailContactsByOrder(txCtx, params.PurchaseOrderID); apiErr != nil {
					return apiErr
				}

				// Create new contacts
				for _, accountUserID := range params.ContactAccountUserIDs {
					contactID, apiErr := id.GenID(id.OrderEmailIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}

					if apiErr := txRepo.CreateEmailContact(txCtx, contactID, params.PurchaseOrderID, accountUserID, "purchaseOrderSubmission"); apiErr != nil {
						return apiErr
					}
				}
			}

			// Re-fetch the complete order
			refetched, apiErr := txRepo.Get(txCtx, params.AccountID, params.PurchaseOrderID)
			if apiErr != nil {
				return apiErr
			}

			if slices.Contains(params.Includes, "lines") {
				lines, apiErr := txRepo.GetLines(txCtx, params.PurchaseOrderID)
				if apiErr != nil {
					return apiErr
				}
				refetched.Lines = lines
			}
			if slices.Contains(params.Includes, "contacts") {
				contacts, apiErr := txRepo.GetEmailContacts(txCtx, params.PurchaseOrderID)
				if apiErr != nil {
					return apiErr
				}
				refetched.Contacts = contacts
			}

			result = refetched
			_ = updated

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePurchaseOrder,
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

func (s *purchaseOrderSvcImpl) DeletePurchaseOrder(ctx context.Context, params domain.DeletePurchaseOrderParams) *apierror.APIError {
	ctx, span := purchaseOrderSvcTracer.Start(ctx, "service.purchase_order.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPurchaseOrders, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewPurchaseOrderRepo()

	// Validate order exists and is not fulfilled
	order, apiErr := repo.Get(ctx, params.AccountID, params.PurchaseOrderID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypePurchaseOrder, params.PurchaseOrderID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This purchase order has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if order.CompletedAt != nil {
		return tracing.Trace(span, apierror.NewValidationError("Cannot delete a fulfilled purchase order."))
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewPurchaseOrderRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypePurchaseOrder, order.ID, order); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.DeleteCascade(txCtx, params.AccountID, params.PurchaseOrderID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(order, (*domain.PurchaseOrder)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypePurchaseOrder,
			ResourceID:   order.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
}

func (s *purchaseOrderSvcImpl) BulkDeletePurchaseOrders(ctx context.Context, params domain.BulkDeletePurchaseOrdersParams) *apierror.APIError {
	ctx, span := purchaseOrderSvcTracer.Start(ctx, "service.purchase_order.bulk_delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPurchaseOrders, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewPurchaseOrderRepo()
		for _, orderID := range params.PurchaseOrderIDs {
			order, apiErr := txRepo.Get(txCtx, params.AccountID, orderID)
			if apiErr != nil {
				return apiErr
			}
			if order.CompletedAt != nil {
				return apierror.NewValidationError("Cannot delete a fulfilled purchase order: " + orderID)
			}
			if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypePurchaseOrder, order.ID, order); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.DeleteCascade(txCtx, params.AccountID, orderID); apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(order, (*domain.PurchaseOrder)(nil))

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionDelete,
				ResourceType: constants.ObjectTypePurchaseOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}
		}
		return nil
	})
}

func (s *purchaseOrderSvcImpl) ChangePurchaseOrderStatus(ctx context.Context, params domain.ChangePurchaseOrderStatusParams) (*domain.PurchaseOrder, *apierror.APIError) {
	ctx, span := purchaseOrderSvcTracer.Start(ctx, "service.purchase_order.change_status")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPurchaseOrders, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewPurchaseOrderRepo()

	// Get the current order
	order, apiErr := repo.Get(ctx, params.AccountID, params.PurchaseOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	now := time.Now()

	switch params.StatusChange {
	case domain.PurchaseOrderStatusChangeIssue:
		if order.SalesOrderStatusCode != "estimate" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Order must be in estimate status to issue."))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPurchaseOrderRepo()
			txLineRepo := txSvc.repos.NewPurchaseOrderLineRepo()
			txReceivingRepo := txSvc.repos.NewReceivingOrderRepo()

			// Update status to issued
			if apiErr := txRepo.UpdateStatus(txCtx, params.AccountID, params.PurchaseOrderID, "issued", &now, nil); apiErr != nil {
				return apiErr
			}

			// Create receiving order
			receivingOrderID, apiErr := id.GenID(id.ReceivingOrderIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := txReceivingRepo.Create(txCtx, receivingOrderID, order.Number, params.PurchaseOrderID, params.AccountID); apiErr != nil {
				return apiErr
			}

			// Create receiving order lines for each PO line
			lines, apiErr := txRepo.GetLines(txCtx, params.PurchaseOrderID)
			if apiErr != nil {
				return apiErr
			}
			for _, line := range lines {
				receivingLineID, apiErr := id.GenID(id.ReceivingOrderLineIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				qtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				// Create a quantity record for the receiving order line
				if apiErr := txLineRepo.CreateQuantity(txCtx, qtyID, line.QuantityValue, line.QuantityUnitID); apiErr != nil {
					return apiErr
				}

				if apiErr := txReceivingRepo.CreateLine(txCtx, receivingLineID, receivingOrderID, qtyID, line.ID); apiErr != nil {
					return apiErr
				}
			}

			// Send email notification if requested
			if params.SendEmail && s.notificationPublisher != nil {
				contacts, apiErr := txRepo.GetEmailContacts(txCtx, params.PurchaseOrderID)
				if apiErr != nil {
					return apiErr
				}
				if len(contacts) > 0 {
					if pubErr := s.notificationPublisher.PublishSendEmail(txCtx, messaging.EmailSendData{
						TemplateID: constants.EmailTemplatePurchaseOrderSubmission,
						Params: map[string]any{
							"order_id":     params.PurchaseOrderID,
							"order_number": order.Number,
						},
					}); pubErr != nil {
						return pubErr
					}
				}
				// Mark acknowledgment sent
				if apiErr := txRepo.UpdateAcknowledgmentSent(txCtx, params.AccountID, params.PurchaseOrderID); apiErr != nil {
					return apiErr
				}
			}

			updated, apiErr := txRepo.Get(txCtx, params.AccountID, params.PurchaseOrderID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(order, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePurchaseOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return nil
		})

	case domain.PurchaseOrderStatusChangeUnissue:
		if order.SalesOrderStatusCode != "issued" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Order must be in issued status to unissue."))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPurchaseOrderRepo()
			txReceivingRepo := txSvc.repos.NewReceivingOrderRepo()

			// Delete receiving order lines and receiving order
			if apiErr := txReceivingRepo.DeleteLinesByOrderID(txCtx, params.PurchaseOrderID); apiErr != nil {
				return apiErr
			}
			if apiErr := txReceivingRepo.DeleteByOrderID(txCtx, params.PurchaseOrderID); apiErr != nil {
				return apiErr
			}

			// Update status back to estimate
			if apiErr := txRepo.UpdateStatus(txCtx, params.AccountID, params.PurchaseOrderID, "estimate", nil, nil); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Get(txCtx, params.AccountID, params.PurchaseOrderID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(order, updated)

			return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePurchaseOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			})
		})

	case domain.PurchaseOrderStatusChangeClose:
		if order.SalesOrderStatusCode != "issued" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Order must be in issued status to close."))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPurchaseOrderRepo()
			txReceivingRepo := txSvc.repos.NewReceivingOrderRepo()

			if apiErr := txRepo.UpdateStatus(txCtx, params.AccountID, params.PurchaseOrderID, "fulfilled", order.IssuedAt, &now); apiErr != nil {
				return apiErr
			}

			if apiErr := txReceivingRepo.MarkComplete(txCtx, params.PurchaseOrderID); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Get(txCtx, params.AccountID, params.PurchaseOrderID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(order, updated)

			return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePurchaseOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			})
		})

	case domain.PurchaseOrderStatusChangeOpen:
		if order.SalesOrderStatusCode != "fulfilled" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Order must be in fulfilled status to re-open."))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *purchaseOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPurchaseOrderRepo()
			txReceivingRepo := txSvc.repos.NewReceivingOrderRepo()

			if apiErr := txRepo.UpdateStatus(txCtx, params.AccountID, params.PurchaseOrderID, "issued", order.IssuedAt, nil); apiErr != nil {
				return apiErr
			}

			if apiErr := txReceivingRepo.MarkIncomplete(txCtx, params.PurchaseOrderID); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Get(txCtx, params.AccountID, params.PurchaseOrderID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(order, updated)

			return audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePurchaseOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			})
		})

	default:
		return nil, tracing.Trace(span, apierror.NewValidationError("Invalid status change action: "+params.StatusChange))
	}

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Re-fetch the updated order
	updatedOrder, apiErr := repo.Get(ctx, params.AccountID, params.PurchaseOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if slices.Contains(params.Includes, "lines") {
		lines, apiErr := repo.GetLines(ctx, params.PurchaseOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		updatedOrder.Lines = lines
	}
	if slices.Contains(params.Includes, "contacts") {
		contacts, apiErr := repo.GetEmailContacts(ctx, params.PurchaseOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		updatedOrder.Contacts = contacts
	}

	return updatedOrder, nil
}

// ensureSupplierMaterialLink checks if a material is linked to the supplier on the purchase order. If not, it finds the material by item ID and creates the link.
func ensureSupplierMaterialLink(ctx context.Context, repos domain.RepoFactory, accountID, purchaseOrderID, itemID, itemSKU string) *apierror.APIError {
	poRepo := repos.NewPurchaseOrderRepo()
	supplierMaterialRepo := repos.NewSupplierMaterialRepo()
	materialRepo := repos.NewMaterialRepo()

	// Get the supplier account ID from the purchase order
	supplierAccountID, apiErr := poRepo.GetSupplierID(ctx, accountID, purchaseOrderID)
	if apiErr != nil {
		return apiErr
	}

	// Find the material by item ID
	material, apiErr := materialRepo.GetByItemID(ctx, accountID, itemID)
	if apiErr != nil {
		// If material doesn't exist for this item, skip linking
		return nil
	}

	// Check if already linked
	exists, apiErr := supplierMaterialRepo.ExistsByMaterialAndSupplier(ctx, accountID, material.ID, supplierAccountID)
	if apiErr != nil {
		return apiErr
	}
	if exists {
		return nil
	}

	// Create the supplier material link
	smID, apiErr := id.GenID(id.SupplierMaterialIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	_, _ = supplierMaterialRepo.Create(ctx, smID, domain.CreateSupplierMaterialParams{
		OwnerAccountID:     accountID,
		MaterialID:         material.ID,
		SupplierAccountID:  supplierAccountID,
		SupplierPartNumber: itemSKU,
		IsActive:           true,
	})
	// Ignore errors from creation - a conflict means the link already exists, and any other error should not block the purchase order operation.

	return nil
}
