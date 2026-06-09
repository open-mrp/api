package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/crypto"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var salesOrderSvcTracer = tracing.GetTracer("core-service.sales_order_service")

type salesOrderSvcImpl struct {
	repos                 domain.RepoFactory
	mediatorFactory       domain.MediatorFactory
	txManager             TransactionManager
	checkoutClientFactory domain.StripeCheckoutClientFactory
	notificationPublisher domain.NotificationPublisher
	encryptionKey         []byte
	frontendURL           string
}

type SalesOrderSvcConfig struct {
	Repos                 domain.RepoFactory
	MediatorFactory       domain.MediatorFactory
	TxManager             TransactionManager
	CheckoutClientFactory domain.StripeCheckoutClientFactory
	NotificationPublisher domain.NotificationPublisher
	EncryptionKey         []byte
	FrontendURL           string
}

func (c *SalesOrderSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("sales order service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("sales order service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("sales order service: tx manager is required")
	}
	return nil
}

func NewSalesOrderSvc(config *SalesOrderSvcConfig) domain.SalesOrderSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &salesOrderSvcImpl{
		repos:                 config.Repos,
		mediatorFactory:       config.MediatorFactory,
		txManager:             config.TxManager,
		checkoutClientFactory: config.CheckoutClientFactory,
		notificationPublisher: config.NotificationPublisher,
		encryptionKey:         config.EncryptionKey,
		frontendURL:           config.FrontendURL,
	}
}

func (s *salesOrderSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *salesOrderSvcImpl) withTx(ctx context.Context, fn func(context.Context, *salesOrderSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &salesOrderSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *salesOrderSvcImpl) ListSalesOrders(ctx context.Context, params domain.ListSalesOrdersParams) (*domain.ListSalesOrdersResult, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkSalesOrderReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	// Customer actors can only see their own orders
	if identity.IsCustomerUser() {
		actorAccountID := identity.ActorAccountID()
		params.BuyerAccountID = actorAccountID
	}

	repo := s.repos.NewSalesOrderRepo()
	result, apiErr := repo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The list now returns the full sales-order shape; expand lines per order
	// only when requested (inline-joined fields are always present).
	if includesSalesOrderLines(params.Includes) {
		for _, order := range result.SalesOrders {
			lines, apiErr := repo.GetLines(ctx, order.ID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			order.Lines = lines
		}
	}

	if includesSalesOrderShipments(params.Includes) {
		for _, order := range result.SalesOrders {
			ids, apiErr := repo.GetShipmentIDs(ctx, order.ID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			order.ShipmentIDs = ids
		}
	}

	return result, nil
}

func (s *salesOrderSvcImpl) GetSalesOrder(ctx context.Context, params domain.GetSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkSalesOrderReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewSalesOrderRepo()

	var order *domain.SalesOrder
	var apiErr *apierror.APIError

	if identity.IsCustomerUser() {
		actorAccountID := *identity.ActorAccountID()
		order, apiErr = repo.GetForCustomer(ctx, params.AccountID, actorAccountID, params.SalesOrderID)
	} else {
		order, apiErr = repo.Get(ctx, params.AccountID, params.SalesOrderID)
	}

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if includesSalesOrderLines(params.Includes) {
		lines, apiErr := repo.GetLines(ctx, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		order.Lines = lines
	}

	if includesSalesOrderShipments(params.Includes) {
		ids, apiErr := repo.GetShipmentIDs(ctx, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		order.ShipmentIDs = ids
	}

	return order, nil
}

func (s *salesOrderSvcImpl) CreateSalesOrder(ctx context.Context, params domain.CreateSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Enforce the account's invoice-count plan limit before creating the order
	// (matches Dashboard's canCreateInvoice guard). Sandboxes are exempt.
	if apiErr := s.checkInvoicePlanLimit(ctx, params.AccountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.SalesOrder](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		if params.SalesOrderStatusCode == "" {
			params.SalesOrderStatusCode = string(constants.SalesOrderStatusCodeEstimate)
		}

		orderID, apiErr := id.GenID(id.OrderIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.SalesOrder
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txOrderRepo := txSvc.repos.NewSalesOrderRepo()
			txAddressRepo := txSvc.repos.NewAddressRepo()
			txLineRepo := txSvc.repos.NewSalesOrderLineRepo()

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
				return apierror.NewConflictErrorWithParam("A sales order with this number already exists.", "number")
			}

			// Check duplicate customer PO if provided
			if params.CustomerPONumber != nil && *params.CustomerPONumber != "" {
				isDup, apiErr = txOrderRepo.IsDuplicateCustomerPO(txCtx, params.AccountID, params.BuyerAccountID, *params.CustomerPONumber, nil)
				if apiErr != nil {
					return apiErr
				}
				if isDup {
					return apierror.NewConflictErrorWithParam("A sales order with this customer PO number already exists.", "customer_po_number")
				}
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

			// Auto-assign sales rep when caller didn't supply one, matching Dashboard behavior:
			// prefer the customer's default sales rep, then zipcode territory, then state territory.
			salesRepID := params.SalesRepID
			if salesRepID == nil {
				salesRepID = txSvc.resolveSalesRepID(txCtx, params.AccountID, params.BuyerAccountID, params.ShipToState, params.ShipToPostalCode)
			}

			// Create the order
			// SellerAccountID and OwnerAccountID default to the target account
			// (the account creating the order), matching Dashboard behavior.
			createParams := domain.CreateSalesOrderParams{
				AccountID:             params.AccountID,
				BuyerAccountID:        params.BuyerAccountID,
				SellerAccountID:       params.AccountID,
				OwnerAccountID:        params.AccountID,
				Number:                orderNumber,
				SalesOrderStatusCode:  params.SalesOrderStatusCode,
				BillingAddressID:      billAddrID,
				ShippingAddressID:     shipAddrID,
				CustomerPONumber:      params.CustomerPONumber,
				Note:                  params.Note,
				CarrierID:             params.CarrierID,
				ServiceLevelID:        params.ServiceLevelID,
				CarrierBillingType:    params.CarrierBillingType,
				CarrierBillingAccount: params.CarrierBillingAccount,
				PriorityCode:          params.PriorityCode,
				SalesRepID:            salesRepID,
				ShippingTermID:        params.ShippingTermID,
				SalesOrderTypeCode:    params.SalesOrderTypeCode,
				PaymentTermID:         params.PaymentTermID,
				OrderDiscountID:       params.OrderDiscountID,
			}

			_, apiErr = txOrderRepo.Create(txCtx, orderID, createParams)
			if apiErr != nil {
				return apiErr
			}

			// Create email contacts (matches Dashboard behavior)
			if apiErr := createOrderEmailContacts(txCtx, txOrderRepo, orderID, params.AcknowledgementEmailContacts, string(constants.AccountRelationNotificationTypeOrderAcknowledgement)); apiErr != nil {
				return apiErr
			}
			if apiErr := createOrderEmailContacts(txCtx, txOrderRepo, orderID, params.InvoiceEmailContacts, string(constants.AccountRelationNotificationTypeInvoice)); apiErr != nil {
				return apiErr
			}

			// Create order lines
			for _, lineInput := range params.Lines {
				lineID, apiErr := id.GenID(id.OrderLineIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				lineParams := domain.CreateSalesOrderLineParams{
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
					EdiLineItemID:              lineInput.EdiLineItemID,
				}

				_, apiErr = txLineRepo.Create(txCtx, lineID, lineParams)
				if apiErr != nil {
					return apiErr
				}
			}

			// Synthesize a shipping line (matches Dashboard, which always attaches one).
			// NOTE: rate is currently fixed at 0 — real rate estimation via the carrier's
			// Shippo connection is a separate enhancement; the line is still emitted so
			// downstream consumers that expect a shipping row keep working.
			if apiErr := txSvc.synthesizeShippingLine(txCtx, orderID, params); apiErr != nil {
				return apiErr
			}

			// Synthesize a discount line when an order-level discount was supplied
			// (matches Dashboard: emits a negative-price line against the account's credit product).
			if params.OrderDiscountID != nil {
				if apiErr := txSvc.synthesizeDiscountLine(txCtx, orderID, params); apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch the complete order
			order, apiErr := txOrderRepo.Get(txCtx, params.AccountID, orderID)
			if apiErr != nil {
				return apiErr
			}

			if includesSalesOrderLines(params.Includes) {
				lines, apiErr := txOrderRepo.GetLines(txCtx, orderID)
				if apiErr != nil {
					return apiErr
				}
				order.Lines = lines
			}

			result = order

			changes := audit.ComputeChanges(nil, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeSalesOrder,
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

func (s *salesOrderSvcImpl) UpdateSalesOrder(ctx context.Context, params domain.UpdateSalesOrderParams) (*domain.SalesOrder, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.SalesOrder](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.SalesOrder
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesOrderRepo()

			// Validate order exists
			existing, apiErr := txRepo.Get(txCtx, params.AccountID, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}

			// Backfill nullable fields that use direct assignment (not COALESCE) in SQL.
			// When the caller omits a field (nil), we preserve the existing value.
			// When the caller sends ptr("") the gateway maps it to SQL NULL to clear the field.
			if params.CarrierID == nil {
				params.CarrierID = existing.CarrierID
			}
			if params.ServiceLevelID == nil {
				params.ServiceLevelID = existing.ServiceLevelID
			}
			if params.SalesRepID == nil {
				params.SalesRepID = existing.SalesRepID
			}
			if params.ShippingTermID == nil {
				params.ShippingTermID = existing.ShippingTermID
			}
			if params.PaymentTermID == nil {
				params.PaymentTermID = existing.PaymentTermID
			}
			if params.OrderDiscountID == nil {
				params.OrderDiscountID = existing.OrderDiscountID
			}
			if params.BuyerAccountID == nil {
				params.BuyerAccountID = &existing.BuyerAccountID
			}

			// Validate order number uniqueness if being updated
			if params.Number != nil {
				isDuplicate, apiErr := txRepo.IsDuplicateOrderNumber(txCtx, params.AccountID, *params.Number, &params.SalesOrderID)
				if apiErr != nil {
					return apiErr
				}
				if isDuplicate {
					return apierror.NewConflictErrorWithParam("This order number is already taken.", "number")
				}
			}

			// Address changes re-point the order to an existing address by ID
			// (params.BillingAddressID / params.ShippingAddressID, applied via the
			// order update below). To edit an address's contents, callers use the
			// update-address endpoint directly.

			// Update the order
			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			// Replace email contacts when the caller supplied lists (nil = leave alone)
			if params.AcknowledgementEmailContacts != nil {
				if apiErr := replaceOrderEmailContacts(txCtx, txRepo, params.SalesOrderID, *params.AcknowledgementEmailContacts, string(constants.AccountRelationNotificationTypeOrderAcknowledgement)); apiErr != nil {
					return apiErr
				}
			}
			if params.InvoiceEmailContacts != nil {
				if apiErr := replaceOrderEmailContacts(txCtx, txRepo, params.SalesOrderID, *params.InvoiceEmailContacts, string(constants.AccountRelationNotificationTypeInvoice)); apiErr != nil {
					return apiErr
				}
			}

			if includesSalesOrderLines(params.Includes) {
				lines, apiErr := txRepo.GetLines(txCtx, params.SalesOrderID)
				if apiErr != nil {
					return apiErr
				}
				result.Lines = lines
			}

			changes := audit.ComputeChanges(existing, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSalesOrder,
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

func (s *salesOrderSvcImpl) DeleteSalesOrder(ctx context.Context, params domain.DeleteSalesOrderParams) *apierror.APIError {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewSalesOrderRepo()

	// Validate order exists and is not fulfilled
	order, apiErr := repo.Get(ctx, params.AccountID, params.SalesOrderID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeSalesOrder, params.SalesOrderID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This sales order has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if order.CompletedAt != nil {
		return tracing.Trace(span, apierror.NewValidationError("Cannot delete a fulfilled sales order."))
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewSalesOrderRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeSalesOrder, order.ID, order); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.DeleteCascade(txCtx, params.AccountID, params.SalesOrderID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(order, (*domain.SalesOrder)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeSalesOrder,
			ResourceID:   order.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
}

func (s *salesOrderSvcImpl) BulkDeleteSalesOrders(ctx context.Context, params domain.BulkDeleteSalesOrdersParams) *apierror.APIError {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.bulk_delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Error

	case domain.RecoveryPointStarted:
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesOrderRepo()
			for _, orderID := range params.SalesOrderIDs {
				order, apiErr := txRepo.Get(txCtx, params.AccountID, orderID)
				if apiErr != nil {
					return apiErr
				}
				if order.CompletedAt != nil || order.SalesOrderStatusCode == string(constants.SalesOrderStatusCodeFulfilled) {
					return apierror.NewValidationError("Cannot delete a fulfilled sales order: " + orderID)
				}
				if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeSalesOrder, order.ID, order); apiErr != nil {
					return apiErr
				}
				if apiErr := txRepo.DeleteCascade(txCtx, params.AccountID, orderID); apiErr != nil {
					return apiErr
				}

				changes := audit.ComputeChanges(order, (*domain.SalesOrder)(nil))

				if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
					ServiceName:  domain.ServiceName,
					Action:       constants.AuditActionDelete,
					ResourceType: constants.ObjectTypeSalesOrder,
					ResourceID:   order.ID,
					Changes:      changes,
				}); apiErr != nil {
					return apiErr
				}
			}
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, idempotencyKey.TypeID, (*struct{})(nil))
		})

		if apiErr != nil {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		return nil

	default:
		return tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idempotencyKey.RecoveryPoint))
	}
}

func (s *salesOrderSvcImpl) ChangeSalesOrderStatus(ctx context.Context, params domain.ChangeSalesOrderStatusParams) (*domain.SalesOrder, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.change_status")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewSalesOrderRepo()

	// Get the current order
	order, apiErr := repo.Get(ctx, params.AccountID, params.SalesOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	now := time.Now()

	switch params.StatusChange {
	case "issue":
		if order.SalesOrderStatusCode != "estimate" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Order must be in estimate status to issue."))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesOrderRepo()
			txLineRepo := txSvc.repos.NewSalesOrderLineRepo()

			// Update status
			if apiErr := txRepo.UpdateStatus(txCtx, params.AccountID, params.SalesOrderID, "issued", &now, nil); apiErr != nil {
				return apiErr
			}

			// Create pick
			pickID, apiErr := id.GenID(id.PickIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.CreatePick(txCtx, pickID, order.Number, params.SalesOrderID, params.AccountID); apiErr != nil {
				return apiErr
			}

			// Get only sale-type lines for pick creation and inventory reservation
			saleLines, apiErr := txRepo.GetSaleLinesForIssue(txCtx, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}

			// Delete any existing reserved inventory issues for this order
			if apiErr := txRepo.DeleteReservedInventoryIssues(txCtx, params.AccountID, params.SalesOrderID); apiErr != nil {
				return apiErr
			}

			for _, line := range saleLines {
				// Create pick line
				pickLineID, apiErr := id.GenID(id.PickLineIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				pickQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				if apiErr := txLineRepo.CreateQuantity(txCtx, pickQtyID, line.QuantityValue, line.QuantityUnitID); apiErr != nil {
					return apiErr
				}

				if apiErr := txRepo.CreatePickLine(txCtx, pickLineID, pickID, pickQtyID, line.ID); apiErr != nil {
					return apiErr
				}

				// Create reserved inventory issue for this line (if it has an item)
				if line.ItemID != nil {
					issueQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}

					if apiErr := txLineRepo.CreateQuantity(txCtx, issueQtyID, line.QuantityValue, line.QuantityUnitID); apiErr != nil {
						return apiErr
					}

					issueID, apiErr := id.GenID(id.InventoryIssueIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}

					if apiErr := txRepo.CreateReservedInventoryIssue(txCtx, issueID, params.AccountID, *line.ItemID, issueQtyID, params.SalesOrderID); apiErr != nil {
						return apiErr
					}
				}
			}

			changes := audit.ComputeChanges(order, &domain.SalesOrder{
				ID:                   order.ID,
				Number:               order.Number,
				SalesOrderStatusCode: "issued",
				IssuedAt:             &now,
			})

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSalesOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return nil
		})

	case "unissue":
		if order.SalesOrderStatusCode != "issued" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Order must be in issued status to unissue."))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesOrderRepo()

			// Delete pick lines and pick (quantities cleaned up too)
			if apiErr := txRepo.DeleteQuantitiesByPickLines(txCtx, params.SalesOrderID); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.DeletePickLinesBySalesOrder(txCtx, params.SalesOrderID); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.DeletePickBySalesOrder(txCtx, params.SalesOrderID); apiErr != nil {
				return apiErr
			}

			// Release reserved inventory: delete allocations first, then issues
			if apiErr := txRepo.DeleteInventoryAllocationsByReservedIssues(txCtx, params.AccountID, params.SalesOrderID); apiErr != nil {
				return apiErr
			}
			if apiErr := txRepo.DeleteReservedInventoryIssues(txCtx, params.AccountID, params.SalesOrderID); apiErr != nil {
				return apiErr
			}

			// Update status and clear issuedAt
			if apiErr := txRepo.UpdateStatus(txCtx, params.AccountID, params.SalesOrderID, "estimate", nil, nil); apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(order, &domain.SalesOrder{
				ID:                   order.ID,
				Number:               order.Number,
				SalesOrderStatusCode: "estimate",
			})

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSalesOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return nil
		})

	case "close":
		if order.SalesOrderStatusCode != "issued" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Order must be in issued status to close."))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesOrderRepo()

			if apiErr := txRepo.UpdateStatus(txCtx, params.AccountID, params.SalesOrderID, "fulfilled", order.IssuedAt, &now); apiErr != nil {
				return apiErr
			}

			// Mark pick as packed if one exists
			if order.PickID != nil {
				if apiErr := txSvc.repos.NewPickRepo().UpdateFinishedAt(txCtx, params.AccountID, *order.PickID, now); apiErr != nil {
					return apiErr
				}
			}

			changes := audit.ComputeChanges(order, &domain.SalesOrder{
				ID:                   order.ID,
				Number:               order.Number,
				SalesOrderStatusCode: "fulfilled",
				IssuedAt:             order.IssuedAt,
				CompletedAt:          &now,
			})

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSalesOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return nil
		})

	case "open":
		if order.SalesOrderStatusCode != "fulfilled" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Order must be in fulfilled status to re-open."))
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesOrderRepo()

			// Preserve issuedAt, clear completedAt
			if apiErr := txRepo.UpdateStatus(txCtx, params.AccountID, params.SalesOrderID, "issued", order.IssuedAt, nil); apiErr != nil {
				return apiErr
			}

			// Mark pick as unpacked if one exists
			if order.PickID != nil {
				if apiErr := txSvc.repos.NewPickRepo().ClearFinishedAt(txCtx, params.AccountID, *order.PickID); apiErr != nil {
					return apiErr
				}
			}

			changes := audit.ComputeChanges(order, &domain.SalesOrder{
				ID:                   order.ID,
				Number:               order.Number,
				SalesOrderStatusCode: "issued",
				IssuedAt:             order.IssuedAt,
			})

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSalesOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return nil
		})

	default:
		return nil, tracing.Trace(span, apierror.NewValidationError("Invalid status change action: "+params.StatusChange))
	}

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// On successful issue transition, optionally send acknowledgement email
	// to the contacts on the order (matching Dashboard behavior).
	if params.StatusChange == "issue" && params.SendEmail {
		if apiErr := s.sendOrderAcknowledgementEmail(ctx, params.AccountID, params.SalesOrderID, order.Number); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Re-fetch the updated order
	updatedOrder, apiErr := repo.Get(ctx, params.AccountID, params.SalesOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if includesSalesOrderLines(params.Includes) {
		lines, apiErr := repo.GetLines(ctx, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		updatedOrder.Lines = lines
	}

	return updatedOrder, nil
}

// sendOrderAcknowledgementEmail publishes the order-acknowledgement email to the
// contacts configured for the order and marks the order as acknowledged. No-ops
// if there are no recipients or the notification publisher is not configured.
func (s *salesOrderSvcImpl) sendOrderAcknowledgementEmail(ctx context.Context, accountID, salesOrderID, orderNumber string) *apierror.APIError {
	if s.notificationPublisher == nil {
		return nil
	}

	recipients, apiErr := s.repos.NewSalesOrderRepo().GetAcknowledgementRecipients(ctx, salesOrderID)
	if apiErr != nil {
		return apiErr
	}
	if len(recipients) == 0 {
		return nil
	}

	accountName, apiErr := s.repos.NewAccountRepo().GetName(ctx, accountID)
	if apiErr != nil {
		return apiErr
	}

	emailData := messaging.EmailSendData{
		To:         recipients,
		Subject:    fmt.Sprintf("Sales Order %s", orderNumber),
		TemplateID: constants.EmailTemplateOrderAcknowledgement,
		Params: map[string]any{
			"order_number": orderNumber,
			"account_name": accountName,
		},
		AccountID: &accountID,
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
		if apiErr := s.notificationPublisher.PublishSendEmail(txCtx, emailData); apiErr != nil {
			return apiErr
		}
		return txSvc.repos.NewSalesOrderRepo().MarkAcknowledgementSent(txCtx, accountID, salesOrderID)
	})
}

// checkInvoicePlanLimit enforces the account's per-billing-period invoice plan
// limit before allowing a new sales order (which will typically generate an invoice).
// Sandbox accounts and accounts with no configured limit are exempt.
// Returns a validation error when the current count meets or exceeds the limit.
func (s *salesOrderSvcImpl) checkInvoicePlanLimit(ctx context.Context, accountID string) *apierror.APIError {
	accountRepo := s.repos.NewAccountRepo()

	// Sandbox accounts are exempt.
	accountCtx, apiErr := accountRepo.GetAccountContext(ctx, accountID)
	if apiErr != nil {
		return apiErr
	}
	if accountCtx != nil && accountCtx.IsSandbox {
		return nil
	}

	planID, periodEnd, apiErr := accountRepo.GetPlanIDAndPeriodEnd(ctx, accountID)
	if apiErr != nil {
		return apiErr
	}
	if planID == nil {
		return nil
	}

	limits, apiErr := accountRepo.ListPlanLimits(ctx, *planID)
	if apiErr != nil {
		return apiErr
	}
	maxInvoices, ok := limits["invoices_maximum"]
	if !ok || maxInvoices == nil {
		// Unlimited / not configured → nothing to enforce.
		return nil
	}

	// Derive the billing-period start the same way billing-service does:
	// (period end - 1 month) when subscribed, else start of the current calendar month UTC.
	var periodStart time.Time
	if periodEnd != nil {
		periodStart = periodEnd.AddDate(0, -1, 0)
	} else {
		now := time.Now().UTC()
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	count, apiErr := s.repos.NewInvoiceRepo().CountSince(ctx, accountID, periodStart)
	if apiErr != nil {
		return apiErr
	}

	if count >= int64(*maxInvoices) {
		return apierror.NewValidationError(fmt.Sprintf("Your plan allows a maximum of %d invoices per billing period.", *maxInvoices))
	}

	return nil
}

// synthesizeShippingLine emits a shipping order line using the account's
// "shipping" system product, matching Dashboard behavior where every sales
// order carries a dedicated shipping line. The unit price is left at 0 here;
// rate estimation via the carrier API is a separate enhancement.
// No-ops cleanly if the account has no shipping system product configured.
func (s *salesOrderSvcImpl) synthesizeShippingLine(ctx context.Context, orderID string, params domain.CreateSalesOrderParams) *apierror.APIError {
	shippingProduct, apiErr := s.repos.NewProductRepo().GetSystemProduct(ctx, params.AccountID, "shipping")
	if apiErr != nil {
		return apiErr
	}
	if shippingProduct == nil {
		return nil
	}

	currencyUnitID, apiErr := s.repos.NewUnitRepo().GetCurrencyBaseUnitID(ctx)
	if apiErr != nil {
		return apiErr
	}

	lineID, apiErr := id.GenID(id.OrderLineIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	_, apiErr = s.repos.NewSalesOrderLineRepo().Create(ctx, lineID, domain.CreateSalesOrderLineParams{
		SalesOrderID:               orderID,
		AccountID:                  params.AccountID,
		ProductID:                  shippingProduct.ProductID,
		ProductSKU:                 shippingProduct.ProductSKU,
		QuantityValue:              "1",
		QuantityUnitID:             shippingProduct.QuantityUnitID,
		UnitPriceValue:             "0",
		UnitPriceNumeratorUnitID:   currencyUnitID,
		UnitPriceDenominatorUnitID: shippingProduct.QuantityUnitID,
	})
	return apiErr
}

// synthesizeDiscountLine emits a negative-price order line against the account's
// credit product to realize an order-level discount, matching Dashboard behavior.
// No-ops if the discount, credit product, or currency base unit cannot be resolved
// (a missing credit product should not fail the create; the discount amount will just
// not appear as a line item).
func (s *salesOrderSvcImpl) synthesizeDiscountLine(ctx context.Context, orderID string, params domain.CreateSalesOrderParams) *apierror.APIError {
	if params.OrderDiscountID == nil {
		return nil
	}

	discount, apiErr := s.repos.NewOrderDiscountRepo().Get(ctx, domain.GetOrderDiscountParams{
		AccountID:       params.AccountID,
		OrderDiscountID: *params.OrderDiscountID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return nil
		}
		return apiErr
	}

	// Compute total ordered from the input lines (qty * unit_price) and the discount amount.
	total := decimal.Zero
	for _, l := range params.Lines {
		qty, err := decimal.NewFromString(l.QuantityValue)
		if err != nil {
			continue
		}
		price, err := decimal.NewFromString(l.UnitPriceValue)
		if err != nil {
			continue
		}
		total = total.Add(qty.Mul(price))
	}

	discountAmount := computeDiscountAmount(discount, total)
	if discountAmount.IsZero() {
		return nil
	}

	creditProduct, apiErr := s.repos.NewProductRepo().GetSystemProduct(ctx, params.AccountID, "credit")
	if apiErr != nil {
		return apiErr
	}
	if creditProduct == nil {
		// No credit product configured for this account; skip rather than fail.
		return nil
	}

	currencyUnitID, apiErr := s.repos.NewUnitRepo().GetCurrencyBaseUnitID(ctx)
	if apiErr != nil {
		return apiErr
	}

	lineID, apiErr := id.GenID(id.OrderLineIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	negAmount := discountAmount.Neg().String()

	_, apiErr = s.repos.NewSalesOrderLineRepo().Create(ctx, lineID, domain.CreateSalesOrderLineParams{
		SalesOrderID:               orderID,
		AccountID:                  params.AccountID,
		ProductID:                  creditProduct.ProductID,
		ProductSKU:                 creditProduct.ProductSKU,
		QuantityValue:              "1",
		QuantityUnitID:             creditProduct.QuantityUnitID,
		UnitPriceValue:             negAmount,
		UnitPriceNumeratorUnitID:   currencyUnitID,
		UnitPriceDenominatorUnitID: creditProduct.QuantityUnitID,
	})
	return apiErr
}

// computeDiscountAmount returns the discount amount given an order-level discount
// and the pre-discount total. Caps at totalOrdered and rounds to nearest cent.
func computeDiscountAmount(discount *domain.OrderDiscount, total decimal.Decimal) decimal.Decimal {
	if discount == nil {
		return decimal.Zero
	}
	var amount decimal.Decimal
	switch discount.DiscountTypeCode {
	case string(constants.OrderDiscountTypePercentage):
		pct, err := decimal.NewFromString(discount.Percentage)
		if err != nil {
			return decimal.Zero
		}
		amount = pct.Mul(total)
	case string(constants.OrderDiscountTypeAmount):
		val, err := decimal.NewFromString(discount.Amount)
		if err != nil {
			return decimal.Zero
		}
		amount = val
	default:
		return decimal.Zero
	}

	if amount.GreaterThan(total) {
		amount = total
	}
	if amount.IsNegative() {
		return decimal.Zero
	}
	return amount.Round(2)
}

// resolveSalesRepID auto-assigns a sales rep when the caller omits one.
// Resolution order (matches Dashboard): customer default → zipcode territory → state territory → none.
// Returns nil when no match is found; any lookup error is swallowed (rep stays unset rather than failing the order).
func (s *salesOrderSvcImpl) resolveSalesRepID(ctx context.Context, accountID, buyerAccountID string, shipToState, shipToPostalCode *string) *string {
	// 1. Customer default sales rep.
	if customer, apiErr := s.repos.NewCustomerRepo().Get(ctx, accountID, buyerAccountID, nil); apiErr == nil && customer != nil && customer.DefaultSalesRepID != nil {
		return customer.DefaultSalesRepID
	}

	territoryRepo := s.repos.NewTerritoryRepo()

	// 2. Zipcode territory lookup.
	if shipToPostalCode != nil && *shipToPostalCode != "" {
		base := *shipToPostalCode
		if idx := strings.Index(base, "-"); idx >= 0 {
			base = base[:idx]
		}
		if zip, err := strconv.ParseInt(base, 10, 32); err == nil && zip >= 0 {
			if rep, apiErr := territoryRepo.FindSalesRepByZipcode(ctx, accountID, int32(zip)); apiErr == nil && rep != nil {
				return rep
			}
		}
	}

	// 3. State-only territory lookup.
	if shipToState != nil && *shipToState != "" {
		if rep, apiErr := territoryRepo.FindSalesRepByState(ctx, accountID, *shipToState); apiErr == nil && rep != nil {
			return rep
		}
	}

	return nil
}

// createOrderEmailContacts writes order_email_contact rows for the given contacts + notification type.
func createOrderEmailContacts(ctx context.Context, repo domain.SalesOrderRepo, salesOrderID string, contacts []domain.SalesOrderEmailContactInput, notificationTypeCode string) *apierror.APIError {
	for _, c := range contacts {
		contactID, apiErr := id.GenID(id.OrderEmailIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := repo.CreateEmailContact(ctx, contactID, salesOrderID, c.AccountUserID, notificationTypeCode); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

// replaceOrderEmailContacts clears existing rows for the given notification type
// and inserts fresh rows for the supplied contacts. Matches Dashboard's delete-and-recreate behavior on update.
func replaceOrderEmailContacts(ctx context.Context, repo domain.SalesOrderRepo, salesOrderID string, contacts []domain.SalesOrderEmailContactInput, notificationTypeCode string) *apierror.APIError {
	if apiErr := repo.DeleteEmailContactsByOrderAndType(ctx, salesOrderID, notificationTypeCode); apiErr != nil {
		return apiErr
	}
	return createOrderEmailContacts(ctx, repo, salesOrderID, contacts, notificationTypeCode)
}

// checkSalesOrderReadPermission checks the appropriate read permission based on the target context.
// Non-internal actors are gated by access checks rather than permissions.
// Internal actors targeting a customer or supplier account use the relationship domain.
func checkSalesOrderReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead)
}

func (s *salesOrderSvcImpl) CheckoutSalesOrder(ctx context.Context, params domain.CheckoutSalesOrderParams) (*domain.CheckoutSalesOrderResult, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.checkout")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.CheckoutSalesOrderResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Fetch Stripe integration credentials
		integrationRepo := s.repos.NewAccountIntegrationRepo()
		encryptedCreds, isActive, apiErr := integrationRepo.GetEncryptedCredentials(ctx, params.AccountID, constants.IntegrationCodeStripe)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if !isActive {
			return nil, tracing.Trace(span, apierror.NewValidationError("Stripe integration is not active for this account."))
		}

		// Check if the order already has a payment
		orderRepo := s.repos.NewSalesOrderRepo()
		hasPaid, apiErr := orderRepo.CheckPaymentStatus(ctx, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if hasPaid {
			return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("This order already has a payment intent.", "id"))
		}

		// Fetch the order
		order, apiErr := orderRepo.Get(ctx, params.AccountID, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Fetch order lines for pricing
		lines, apiErr := orderRepo.GetLines(ctx, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Decrypt Stripe credentials
		decrypted, err := crypto.DecryptAESGCM(encryptedCreds, s.encryptionKey, nil)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to decrypt Stripe credentials."))
		}

		var stripeCreds domain.StripeCredentials
		if err := json.Unmarshal(decrypted, &stripeCreds); err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Stripe credentials."))
		}

		// Build checkout line items from order lines
		checkoutItems := make([]domain.CheckoutLineItem, 0, len(lines))
		for _, line := range lines {
			unitPrice, parseErr := decimal.NewFromString(line.UnitPriceValue)
			if parseErr != nil {
				continue
			}
			qty, parseErr := decimal.NewFromString(line.QuantityValue)
			if parseErr != nil {
				continue
			}
			// Convert unit price to cents
			amountCents := unitPrice.Mul(decimal.NewFromInt(100)).IntPart()
			qtyInt := qty.IntPart()
			if qtyInt <= 0 {
				qtyInt = 1
			}

			name := line.ProductSKU
			desc := ""
			if line.ProductDescription != nil {
				desc = *line.ProductDescription
			}

			checkoutItems = append(checkoutItems, domain.CheckoutLineItem{
				Name:        name,
				Description: desc,
				AmountCents: amountCents,
				Quantity:    qtyInt,
			})
		}

		// Create Stripe checkout session (foreign mutation)
		checkoutClient := s.checkoutClientFactory.Build(stripeCreds.PrivateKey)
		checkoutSession, apiErr := checkoutClient.CreateOneTimeCheckoutSession(ctx, domain.CreateCheckoutSessionParams{
			CustomerEmail: params.Email,
			LineItems:     checkoutItems,
			SuccessURL:    params.SuccessURL,
			CancelURL:     params.CancelURL,
			PaymentIntentMetadata: map[string]string{
				"orderID":    order.ID,
				"customerID": order.BuyerAccountID,
			},
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		result := &domain.CheckoutSalesOrderResult{
			CheckoutURL: checkoutSession.URL,
		}

		// In transaction: send email + cache response
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			if s.notificationPublisher != nil {
				if pubErr := s.notificationPublisher.PublishSendEmail(txCtx, messaging.EmailSendData{
					To:         []string{params.Email},
					Subject:    "Your Order Checkout - " + order.Number,
					TemplateID: constants.EmailTemplateOrderCheckout,
					Params: map[string]any{
						"checkout_url": checkoutSession.URL,
						"order_number": order.Number,
						"account_name": order.CustomerName,
					},
				}); pubErr != nil {
					return pubErr
				}
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

func (s *salesOrderSvcImpl) CreateCustomerCheckoutSession(ctx context.Context, params domain.CreateCustomerCheckoutSessionParams) (*domain.CreateCustomerCheckoutSessionResult, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.create_customer_checkout_session")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsCustomerUser() {
		return nil, tracing.Trace(span, apierror.NewValidationError("Invalid actor type."))
	}

	actorAccountID := identity.ActorAccountID()
	if actorAccountID == nil {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("Actor account ID is required."))
	}
	customerAccountID := *actorAccountID
	targetAccountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.CreateCustomerCheckoutSessionResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// 1. Check Stripe integration exists
		integrationRepo := s.repos.NewAccountIntegrationRepo()
		hasIntegration, apiErr := integrationRepo.HasIntegration(ctx, targetAccountID, constants.IntegrationCodeStripe)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if !hasIntegration {
			return nil, tracing.Trace(span, apierror.NewValidationError("Stripe integration not found."))
		}

		// 2. Get encrypted Stripe credentials
		encryptedCreds, isActive, apiErr := integrationRepo.GetEncryptedCredentials(ctx, targetAccountID, constants.IntegrationCodeStripe)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if !isActive {
			return nil, tracing.Trace(span, apierror.NewValidationError("Stripe integration is not active for this account."))
		}

		// 3. Decrypt Stripe credentials
		decrypted, err := crypto.DecryptAESGCM(encryptedCreds, s.encryptionKey, nil)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to decrypt Stripe credentials."))
		}
		var stripeCreds domain.StripeCredentials
		if err := json.Unmarshal(decrypted, &stripeCreds); err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Stripe credentials."))
		}

		checkoutClient := s.checkoutClientFactory.Build(stripeCreds.PrivateKey)

		// 4. Resolve or create Stripe customer
		customerRepo := s.repos.NewCustomerRepo()
		stripeCustomerID, _, apiErr := customerRepo.GetStripeCustomerID(ctx, targetAccountID, customerAccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if stripeCustomerID == nil || *stripeCustomerID == "" {
			// Get customer email from branding
			customerEmail, apiErr := customerRepo.GetCustomerEmail(ctx, customerAccountID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			if customerEmail == nil || *customerEmail == "" {
				return nil, tracing.Trace(span, apierror.NewInternalError(nil, "Customer email not found."))
			}

			// Get customer details
			customer, apiErr := customerRepo.Get(ctx, targetAccountID, customerAccountID, nil)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			// Create Stripe customer (foreign mutation)
			stripeCust, apiErr := checkoutClient.CreateStripeCustomer(ctx, domain.CreateStripeCustomerParams{
				Email:      *customerEmail,
				Name:       customer.Name,
				Number:     customer.Number,
				CustomerID: customerAccountID,
			})
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			// Save Stripe customer ID to DB (in transaction)
			apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
				txCustomerRepo := txSvc.repos.NewCustomerRepo()
				return txCustomerRepo.SetStripeCustomerID(txCtx, targetAccountID, customerAccountID, stripeCust.ID, *customerEmail)
			})
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			stripeCustomerID = &stripeCust.ID
		}

		if stripeCustomerID == nil || *stripeCustomerID == "" {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Stripe customer not found."))
		}

		// 5. Get account slug for return URL
		accountRepo := s.repos.NewAccountRepo()
		slug, apiErr := accountRepo.GetPortalSlug(ctx, targetAccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if slug == nil {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account not found."))
		}

		returnURL := fmt.Sprintf("%s/%s/dashboard/sales-orders/%s", s.frontendURL, *slug, params.OrderID)

		// 6. Create embedded checkout session (foreign mutation)
		session, apiErr := checkoutClient.CreateEmbeddedCheckoutSession(ctx, domain.CreateEmbeddedCheckoutSessionParams{
			StripeCustomerID: *stripeCustomerID,
			AccountSlug:      *slug,
			CustomerID:       customerAccountID,
			OrderNumber:      params.OrderNumber,
			CustomerPO:       params.CustomerPO,
			OrderTotalCents:  params.OrderTotalCents,
			OrderID:          params.OrderID,
			ReturnURL:        returnURL,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		result := &domain.CreateCustomerCheckoutSessionResult{
			ClientSecret: session.ClientSecret,
		}

		// Cache response
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
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

func (s *salesOrderSvcImpl) CreateSalesOrderProductionRun(ctx context.Context, params domain.CreateSalesOrderProductionRunParams) (*domain.CreateSalesOrderProductionRunResult, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.create_production_run")
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

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.CreateSalesOrderProductionRunResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Get the account user for the current actor (responsible user for the run)
		var responsibleUserID string
		if identity.Actor != nil {
			accountUserRepo := s.repos.NewAccountUserRepo()
			accountUser, apiErr := accountUserRepo.FindByAccountAndUserID(ctx, identity.Actor.ID, params.AccountID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			responsibleUserID = accountUser.ID
		}

		// Fetch the order
		orderRepo := s.repos.NewSalesOrderRepo()
		order, apiErr := orderRepo.Get(ctx, params.AccountID, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Validate that the order doesn't already have a production run
		if order.ProductionRunID != nil {
			return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("This order already has a production run.", "id"))
		}

		// Get order lines for BOM
		bomLines, apiErr := orderRepo.GetLinesForBOM(ctx, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Generate production run ID and number
		runID, apiErr := id.GenID(id.ProductionRunIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		runRepo := s.repos.NewProductionRunQueryRepo()
		runNumber, apiErr := runRepo.GetNextNumber(ctx, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Calculate material demand for each line (used for inventory reservation below)
		materialDemandRepo := s.repos.NewMaterialDemandRepo()
		allDemands := make([]domain.MaterialDemandItem, 0)
		for _, line := range bomLines {
			demands, apiErr := materialDemandRepo.GetMaterialDemand(ctx, params.AccountID, line.ItemID, line.QuantityValue, line.QuantityUnitID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			allDemands = append(allDemands, demands...)
		}

		result := &domain.CreateSalesOrderProductionRunResult{
			ProductionRunID: runID,
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txRunRepo := txSvc.repos.NewProductionRunQueryRepo()
			txOrderRepo := txSvc.repos.NewSalesOrderRepo()
			txBatchRepo := txSvc.repos.NewBatchRepo()

			// Create the production run
			if apiErr := txRunRepo.Create(txCtx, runID, responsibleUserID, runNumber, params.AccountID); apiErr != nil {
				return apiErr
			}

			// Create batch records for each order line that has items
			for _, line := range bomLines {
				batchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				_, apiErr = txBatchRepo.Create(txCtx, batchID, domain.CreateBatchParams{
					AccountID:       params.AccountID,
					ItemID:          line.ItemID,
					ProductionRunID: runID,
					Quantity: domain.CreateQuantityParams{
						Measure: line.QuantityValue,
						UnitID:  line.QuantityUnitID,
					},
				})
				if apiErr != nil {
					return apiErr
				}
			}

			// Link order to production run
			if apiErr := txOrderRepo.SetProductionRunID(txCtx, params.AccountID, params.SalesOrderID, runID); apiErr != nil {
				return apiErr
			}

			// Audit the sales order update (production run link)
			updatedOrder, apiErr := txOrderRepo.Get(txCtx, params.AccountID, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(order, updatedOrder)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSalesOrder,
				ResourceID:   order.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			// Create reserved inventory issue records for material demand, linked to the order
			if len(allDemands) > 0 {
				txReservationRepo := txSvc.repos.NewInventoryReservationRepo()
				for _, demand := range allDemands {
					if apiErr := txReservationRepo.CreateMaterialReservation(txCtx, domain.CreateMaterialReservationParams{
						AccountID: params.AccountID,
						ItemID:    demand.ItemID,
						Measure:   demand.Measure,
						UnitID:    demand.UnitID,
						OrderID:   params.SalesOrderID,
					}); apiErr != nil {
						return apiErr
					}
				}
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

func includesSalesOrderLines(includes []string) bool {
	for _, inc := range includes {
		if inc == "lines" || strings.HasPrefix(inc, "lines.") {
			return true
		}
	}
	return false
}

func includesSalesOrderShipments(includes []string) bool {
	for _, inc := range includes {
		if inc == "related.shipments" {
			return true
		}
	}
	return false
}
