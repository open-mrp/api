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
	"github.com/augno/api/services/core-service/internal/event"
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
	salesOrderPublisher   domain.SalesOrderEventPublisher
	shippoFactory         domain.ShippoClientFactory
	encryptionKey         []byte
	frontendURL           string
}

type SalesOrderSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// CheckoutClientFactory (optional; default: nil) builds Stripe checkout clients. It is not validated at construction; checkout code paths panic at runtime if it is unset.
	CheckoutClientFactory domain.StripeCheckoutClientFactory

	// NotificationPublisher (optional; default: nil) publishes notification messages to the outbox. It is not validated at construction.
	NotificationPublisher domain.NotificationPublisher

	// SalesOrderPublisher (optional; default: nil) publishes sales-order domain events to the outbox. When nil, the sales-order-created event is skipped. It is not validated at construction.
	SalesOrderPublisher domain.SalesOrderEventPublisher

	// ShippoFactory (optional; default: nil) builds Shippo clients for live shipping-rate estimation on create. When nil, the synthesized shipping line falls back to rate 0 (after honoring all freight-exemption / flat-rate / minimum-order rules).
	ShippoFactory domain.ShippoClientFactory

	// EncryptionKey (optional; default: nil) encrypts sensitive fields at rest. It is not validated at construction.
	EncryptionKey []byte

	// FrontendURL (optional; default: "") is the dashboard base URL used in links. It is not validated at construction.
	FrontendURL string
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
		salesOrderPublisher:   config.SalesOrderPublisher,
		shippoFactory:         config.ShippoFactory,
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
			repos:               f,
			mediatorFactory:     s.mediatorFactory,
			txManager:           s.txManager,
			salesOrderPublisher: s.salesOrderPublisher,
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

	// Populate the derived payment status for the whole page in one batched query (no per-order N+1), defaulting any order without payment activity to unpaid.
	if len(result.SalesOrders) > 0 {
		orderIDs := make([]string, len(result.SalesOrders))
		for i, order := range result.SalesOrders {
			orderIDs[i] = order.ID
		}
		statuses, apiErr := repo.GetPaymentStatuses(ctx, params.AccountID, orderIDs)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, order := range result.SalesOrders {
			if status, ok := statuses[order.ID]; ok {
				order.PaymentStatus = status
			} else {
				order.PaymentStatus = constants.SalesOrderPaymentStatusUnpaid
			}
		}
	}

	// The list now returns the full sales-order shape; expand lines per order only when requested (inline-joined fields are always present).
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

	if includesSalesOrderContacts(params.Includes) {
		if apiErr := attachSalesOrderContacts(ctx, repo, result.SalesOrders); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
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

	// Populate the derived payment status (always present on the resource).
	statuses, apiErr := repo.GetPaymentStatuses(ctx, params.AccountID, []string{order.ID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if status, ok := statuses[order.ID]; ok {
		order.PaymentStatus = status
	} else {
		order.PaymentStatus = constants.SalesOrderPaymentStatusUnpaid
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

	if includesSalesOrderContacts(params.Includes) {
		if apiErr := attachSalesOrderContacts(ctx, repo, []*domain.SalesOrder{order}); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
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

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Authorization (matches Dashboard): internal users need the create permission; customer users may self-create only for their own account; other actor types cannot create orders.
	switch {
	case identity.IsInternalUser():
		if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionCreate); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	case identity.IsCustomerUser():
		actorAccountID := identity.ActorAccountID()
		if actorAccountID == nil || params.BuyerAccountID != *actorAccountID {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("You are not authorized to create an order for this customer."))
		}
	default:
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You are not authorized to create sales orders."))
	}

	params.AccountID = identity.Target.AccountID

	// For customer-portal creates, fold the customer's saved note into the order note (matches Dashboard: [customer.note, data.note] joined).
	if identity.IsCustomerUser() {
		if customer, apiErr := s.repos.NewCustomerRepo().Get(ctx, params.AccountID, params.BuyerAccountID, nil); apiErr == nil && customer != nil && customer.Note != nil && *customer.Note != "" {
			folded := *customer.Note
			if params.Note != nil && *params.Note != "" {
				folded = *customer.Note + "\n\n" + *params.Note
			}
			params.Note = &folded
		}
	}

	// Enforce the account's invoice-count plan limit before creating the order (matches Dashboard's canCreateInvoice guard). Sandboxes are exempt.
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

		// Cache any failure from here on under the idempotency key so a retried request replays the same error (matches the in-transaction behavior this create flow previously relied on).
		cacheErr := func(apiErr *apierror.APIError) *apierror.APIError {
			return meds.Idempotency.CacheErrorResponse(ctx, idempotencyKey.TypeID, apiErr)
		}

		// Reserve the order number and reject duplicate order / customer-PO numbers FIRST — before the more expensive line/pricing resolution and the external shipping-rate lookup — so a duplicate fails fast without that wasted work.
		orderRepo := s.repos.NewSalesOrderRepo()

		orderNumber, apiErr := orderRepo.GetNextOrderNumber(ctx, params.AccountID)
		if apiErr != nil {
			return nil, cacheErr(apiErr)
		}

		isDup, apiErr := orderRepo.IsDuplicateOrderNumber(ctx, params.AccountID, orderNumber, nil)
		if apiErr != nil {
			return nil, cacheErr(apiErr)
		}
		if isDup {
			return nil, cacheErr(apierror.NewConflictErrorWithParam("A sales order with this number already exists.", "number"))
		}

		if params.CustomerPONumber != nil && *params.CustomerPONumber != "" {
			isDup, apiErr = orderRepo.IsDuplicateCustomerPO(ctx, params.AccountID, params.BuyerAccountID, *params.CustomerPONumber, nil)
			if apiErr != nil {
				return nil, cacheErr(apiErr)
			}
			if isDup {
				return nil, cacheErr(apierror.NewConflictErrorWithParam("A sales order with this customer PO number already exists.", "customer_po_number"))
			}
		}

		// Resolve everything else that only requires reads (and the live Shippo call) BEFORE opening the write transaction, so the external rate lookup never holds a DB transaction open across network latency. The transaction below performs only the inserts.
		addressRepo := s.repos.NewAddressRepo()

		// Reference the existing bill-to / ship-to addresses by ID (matching Dashboard, which only accepts address IDs; addresses are persisted separately). Each must belong to the order's owner or buyer account. The resolved ship-to feeds the sales-rep territory + shipping-rate logic below.
		if _, apiErr := s.resolveOrderAddress(ctx, addressRepo, params.AccountID, params.BuyerAccountID, params.BillToAddressID); apiErr != nil {
			return nil, cacheErr(apiErr)
		}
		shipAddr, apiErr := s.resolveOrderAddress(ctx, addressRepo, params.AccountID, params.BuyerAccountID, params.ShipToAddressID)
		if apiErr != nil {
			return nil, cacheErr(apiErr)
		}

		// Resolve the order discount, which the caller may pass as either its ID or its unique code; store the resolved ID on the order.
		if params.OrderDiscountID != nil && *params.OrderDiscountID != "" {
			resolvedDiscountID, apiErr := s.resolveOrderDiscountID(ctx, params.AccountID, *params.OrderDiscountID)
			if apiErr != nil {
				return nil, cacheErr(apiErr)
			}
			params.OrderDiscountID = &resolvedDiscountID
		}

		// Auto-assign sales rep when caller didn't supply one, matching Dashboard behavior: commission-exempt customer/product-lines yield no rep; otherwise prefer the customer's default sales rep, then zipcode territory, then state territory.
		salesRepID := params.SalesRepID
		if salesRepID == nil {
			shipState, shipPostal := shipAddr.State, shipAddr.Zip
			salesRepID = s.resolveSalesRepID(ctx, params.AccountID, params.BuyerAccountID, params.Lines, &shipState, &shipPostal)
		}

		// Resolve every line against the product/pricing data: validate the quantity unit against the product's unit group, default SKU/description from the product, derive the item + unit cost, and compute the unit price (internal actors may override via the line's unit_price; customer submissions are ignored and the price is computed server-side).
		resolvedLines, apiErr := s.resolveSalesOrderCreateLines(ctx, params.AccountID, params.BuyerAccountID, identity.IsInternalUser(), params.Lines)
		if apiErr != nil {
			return nil, cacheErr(apiErr)
		}

		// Order total (product lines only), computed from the resolved unit prices — used by the shipping-rate minimum-order check and the discount line.
		orderTotal := calculateResolvedLinesTotal(resolvedLines)

		// Estimate the shipping rate via the freight-exemption / flat-rate / minimum-order / live-Shippo cascade (matches Dashboard). This is the only external call in the create path; computing it here on the outer receiver keeps the live Shippo HTTP request out of the transaction and uses the real Shippo factory.
		shippingRate, apiErr := s.estimateOrderShippingRate(ctx, params, shipAddr, orderTotal)
		if apiErr != nil {
			return nil, cacheErr(apiErr)
		}

		var result *domain.SalesOrder
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txCtx = event.WithRepos(txCtx, txSvc.repos)

			txOrderRepo := txSvc.repos.NewSalesOrderRepo()
			txLineRepo := txSvc.repos.NewSalesOrderLineRepo()

			// Create the order. SellerAccountID and OwnerAccountID default to the target account (the account creating the order), matching Dashboard behavior.
			createParams := domain.CreateSalesOrderParams{
				AccountID:             params.AccountID,
				BuyerAccountID:        params.BuyerAccountID,
				SellerAccountID:       params.AccountID,
				OwnerAccountID:        params.AccountID,
				Number:                orderNumber,
				SalesOrderStatusCode:  params.SalesOrderStatusCode,
				BillingAddressID:      params.BillToAddressID,
				ShippingAddressID:     params.ShipToAddressID,
				CustomerPONumber:      params.CustomerPONumber,
				Note:                  params.Note,
				CarrierID:             params.CarrierID,
				ServiceLevelID:        params.ServiceLevelID,
				CarrierBillingType:    params.CarrierBillingType,
				CarrierBillingAccount: params.CarrierBillingAccount,
				PriorityCode:          params.PriorityCode,
				SalesRepID:            salesRepID,
				ShippingTermID:        params.ShippingTermID,
				PaymentTermID:         params.PaymentTermID,
				OrderDiscountID:       params.OrderDiscountID,
				PromisedAt:            params.PromisedAt,
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

			// Create order lines (resolved above, before the transaction).
			for _, rl := range resolvedLines {
				lineID, apiErr := id.GenID(id.OrderLineIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				itemID := rl.ItemID
				costValue := rl.UnitCost.Value
				costNum := rl.UnitCost.NumeratorUnitID
				costDen := rl.UnitCost.DenominatorUnitID
				_, apiErr = txLineRepo.Create(txCtx, lineID, domain.CreateSalesOrderLineParams{
					SalesOrderID:               orderID,
					AccountID:                  params.AccountID,
					ProductID:                  rl.ProductID,
					ItemID:                     &itemID,
					ProductSKU:                 rl.ProductSKU,
					ProductDescription:         rl.ProductDescription,
					QuantityValue:              rl.QuantityValue,
					QuantityUnitID:             rl.QuantityUnitID,
					UnitPriceValue:             rl.UnitPrice.Value,
					UnitPriceNumeratorUnitID:   rl.UnitPrice.NumeratorUnitID,
					UnitPriceDenominatorUnitID: rl.UnitPrice.DenominatorUnitID,
					UnitCostValue:              &costValue,
					UnitCostNumeratorUnitID:    &costNum,
					UnitCostDenominatorUnitID:  &costDen,
					EdiLineItemID:              rl.EdiLineItemID,
				})
				if apiErr != nil {
					return apiErr
				}
			}

			// Synthesize a shipping line (matches Dashboard, which always attaches one) using the rate estimated before the transaction.
			if apiErr := txSvc.synthesizeShippingLine(txCtx, orderID, params, shippingRate); apiErr != nil {
				return apiErr
			}

			// Synthesize a discount line when an order-level discount was supplied (matches Dashboard: emits a negative-price line against the account's credit product).
			if params.OrderDiscountID != nil {
				if apiErr := txSvc.synthesizeDiscountLine(txCtx, orderID, params, orderTotal); apiErr != nil {
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

			if includesSalesOrderContacts(params.Includes) {
				if apiErr := attachSalesOrderContacts(txCtx, txOrderRepo, []*domain.SalesOrder{order}); apiErr != nil {
					return apiErr
				}
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

			// Publish the sales-order-created event to the outbox (same transaction) so out-of-band consumers (e.g. CRM sync) can react after the order commits.
			if txSvc.salesOrderPublisher != nil {
				if apiErr := txSvc.salesOrderPublisher.PublishSalesOrderCreated(txCtx, messaging.SalesOrderCreatedData{
					SalesOrderID:   result.ID,
					AccountID:      result.OwnerAccountID,
					BuyerAccountID: result.BuyerAccountID,
					Number:         result.Number,
					StatusCode:     result.SalesOrderStatusCode,
				}); apiErr != nil {
					return apiErr
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

			// Backfill nullable fields that use direct assignment (not COALESCE) in SQL. When the caller omits a field (nil), we preserve the existing value. When the caller sends ptr("") the gateway maps it to SQL NULL to clear the field.
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

			// Address changes re-point the order to an existing address by ID (params.BillingAddressID / params.ShippingAddressID, applied via the order update below). To edit an address's contents, callers use the update-address endpoint directly.

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

			if includesSalesOrderContacts(params.Includes) {
				if apiErr := attachSalesOrderContacts(txCtx, txRepo, []*domain.SalesOrder{result}); apiErr != nil {
					return apiErr
				}
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

	// On successful issue transition, optionally send acknowledgement email to the contacts on the order (matching Dashboard behavior).
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

// sendOrderAcknowledgementEmail publishes the order-acknowledgement email to the contacts configured for the order and marks the order as acknowledged. No-ops if there are no recipients or the notification publisher is not configured.
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

// checkInvoicePlanLimit enforces the account's per-billing-period invoice plan limit before allowing a new sales order (which will typically generate an invoice). Sandbox accounts and accounts with no configured limit are exempt. Returns a validation error when the current count meets or exceeds the limit.
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

	// Derive the billing-period start the same way billing-service does: (period end - 1 month) when subscribed, else start of the current calendar month UTC.
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

// synthesizeShippingLine inserts the order's shipping line using the account's "shipping" system product and a rate already estimated by the caller (see estimateOrderShippingRate), matching Dashboard behavior where every sales order carries a dedicated shipping line. The rate is computed before the transaction so the live Shippo call does not run inside it. No-ops cleanly if the account has no shipping system product configured.
func (s *salesOrderSvcImpl) synthesizeShippingLine(ctx context.Context, orderID string, params domain.CreateSalesOrderParams, rate string) *apierror.APIError {
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
		UnitPriceValue:             rate,
		UnitPriceNumeratorUnitID:   currencyUnitID,
		UnitPriceDenominatorUnitID: shippingProduct.QuantityUnitID,
	})
	return apiErr
}

// estimateOrderShippingRate computes the posted shipping rate for a new order using the shared freight-exemption / flat-rate / minimum-order / live-Shippo cascade, mirroring Dashboard's estimatePostedShippingRate. Returns the rate as a decimal string (not rounded to cents, matching Dashboard's create path). The live Shippo rate already includes the 10% markup applied by the Shippo client.
func (s *salesOrderSvcImpl) estimateOrderShippingRate(ctx context.Context, params domain.CreateSalesOrderParams, shipTo domain.ShippingAddress, orderTotal float64) (string, *apierror.APIError) {
	productIDs := make([]string, 0, len(params.Lines))
	for _, l := range params.Lines {
		if l.ProductID != "" {
			productIDs = append(productIDs, l.ProductID)
		}
	}

	productInfo, apiErr := s.repos.NewSalesOrderRepo().GetProductTypesAndLines(ctx, productIDs)
	if apiErr != nil {
		return "", apiErr
	}

	typeByProduct := make(map[string]string, len(productInfo))
	seenProductLine := make(map[string]struct{})
	var productLineIDs []string
	for _, p := range productInfo {
		typeByProduct[p.ProductID] = p.ProductTypeCode
		if p.ProductLineID != nil {
			if _, ok := seenProductLine[*p.ProductLineID]; !ok {
				seenProductLine[*p.ProductLineID] = struct{}{}
				productLineIDs = append(productLineIDs, *p.ProductLineID)
			}
		}
	}

	weight, apiErr := s.estimateOrderWeight(ctx, params.AccountID, params.Lines, typeByProduct)
	if apiErr != nil {
		return "", apiErr
	}

	// Origin (ship-from) = the seller account's default billing address.
	var from domain.ShippingAddress
	if fromAddr, apiErr := s.repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, params.AccountID); apiErr != nil {
		return "", apiErr
	} else if fromAddr != nil {
		from = *fromAddr
	}

	rate, apiErr := estimateShippingRate(ctx, s.repos, s.shippoFactory, domain.EstimateRateParams{
		AccountID:      params.AccountID,
		CarrierID:      derefString(params.CarrierID),
		ServiceLevelID: derefString(params.ServiceLevelID),
		ProductLineIDs: productLineIDs,
		CustomerID:     &params.BuyerAccountID,
		FromAddress:    from,
		ToAddress:      shipTo,
		OrderTotal:     &orderTotal,
		Parcels: []domain.Parcel{{
			Weight: strconv.FormatFloat(weight, 'f', -1, 64),
			Length: "23.5",
			Width:  "13",
			Height: "9.5",
		}},
	})
	if apiErr != nil {
		return "", apiErr
	}

	return strconv.FormatFloat(rate, 'f', -1, 64), nil
}

// calculateResolvedLinesTotal sums quantity × (computed) unit price over the resolved product lines, rounded to cents (matches Dashboard's calculateTotalOrdered used for the shipping min-order threshold and the order discount).
func calculateResolvedLinesTotal(lines []domain.ResolvedSalesOrderLine) float64 {
	total := decimal.Zero
	for _, l := range lines {
		qty, err1 := decimal.NewFromString(l.QuantityValue)
		price, err2 := decimal.NewFromString(l.UnitPrice.Value)
		if err1 != nil || err2 != nil {
			continue
		}
		total = total.Add(qty.Mul(price))
	}
	f, _ := total.Round(2).Float64()
	return f
}

// shippingWeightMultipliers maps a quantity-unit abbreviation to its per-base-unit weight multiplier, mirroring Dashboard's estimateOrderWeight (0.15 lb base per ea).
var shippingWeightMultipliers = map[string]float64{
	"ea":     1,
	"pr":     2,
	"ct10ea": 10,
	"ct5pr":  10,
	"ct10pr": 20,
	"ct20ea": 20,
	"ct12ea": 12,
	"ct6pr":  12,
	"ct12pr": 24,
	"ct8ea":  8,
	"cs40ea": 40,
	"cs50ea": 50,
	"ct16ea": 16,
}

// estimateOrderWeight estimates the parcel weight (lbs) for the order's sale-product lines, mirroring Dashboard's estimateOrderWeight. Non-sale products and unknown units contribute nothing.
func (s *salesOrderSvcImpl) estimateOrderWeight(ctx context.Context, accountID string, lines []domain.CreateSalesOrderLineInput, typeByProduct map[string]string) (float64, *apierror.APIError) {
	seenUnit := make(map[string]struct{})
	var unitIDs []string
	for _, l := range lines {
		if l.QuantityUnitID == "" {
			continue
		}
		if _, ok := seenUnit[l.QuantityUnitID]; !ok {
			seenUnit[l.QuantityUnitID] = struct{}{}
			unitIDs = append(unitIDs, l.QuantityUnitID)
		}
	}

	abbrevByUnit := make(map[string]string)
	if len(unitIDs) > 0 {
		units, apiErr := s.repos.NewUnitRepo().GetByIDs(ctx, accountID, unitIDs)
		if apiErr != nil {
			return 0, apiErr
		}
		for _, u := range units {
			abbrevByUnit[u.ID] = u.Abbreviation
		}
	}

	const baseWeight = 0.15
	weight := 0.0
	for _, l := range lines {
		// Only sale products contribute. A product with no known type still counts (matches Dashboard, which only skips when productType is present and not sale).
		if tc, ok := typeByProduct[l.ProductID]; ok && tc != string(constants.ProductTypeCodeSale) {
			continue
		}
		mult, ok := shippingWeightMultipliers[abbrevByUnit[l.QuantityUnitID]]
		if !ok {
			continue
		}
		qty, err := decimal.NewFromString(l.QuantityValue)
		if err != nil {
			continue
		}
		qf, _ := qty.Float64()
		weight += qf * baseWeight * mult
	}
	return weight, nil
}

// QuoteSalesOrderLinePrices computes the unit price for each requested line without creating an order, for displaying prices to users (including the customer portal). Internal users (with sales-order read) and customer actors (for their own account) may request quotes. The price is always computed server-side — there is no override.
func (s *salesOrderSvcImpl) QuoteSalesOrderLinePrices(ctx context.Context, params domain.QuoteSalesOrderLinePricesParams) ([]domain.SalesOrderLineQuote, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.quote_line_prices")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	switch {
	case identity.IsInternalUser():
		if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	case identity.IsCustomerUser():
		actorAccountID := identity.ActorAccountID()
		if actorAccountID == nil || params.BuyerAccountID != *actorAccountID {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("You are not authorized to quote prices for this customer."))
		}
	default:
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You are not authorized to quote sales order prices."))
	}

	accountID := identity.Target.AccountID

	prices, apiErr := s.computeSalesOrderLinePrices(ctx, accountID, params.BuyerAccountID, identity.IsInternalUser(), params.Lines)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.SalesOrderLineQuote, len(params.Lines))
	for i, l := range params.Lines {
		out[i] = domain.SalesOrderLineQuote{ProductID: l.ProductID, UnitPrice: domain.RateValue(prices[i])}
	}
	return out, nil
}

// resolveOrderDiscountID resolves a caller-supplied order-discount reference — which may be either the discount's ID or its unique code — to the discount's ID.
func (s *salesOrderSvcImpl) resolveOrderDiscountID(ctx context.Context, accountID, idOrCode string) (string, *apierror.APIError) {
	discountRepo := s.repos.NewOrderDiscountRepo()
	if d, apiErr := discountRepo.Get(ctx, domain.GetOrderDiscountParams{AccountID: accountID, OrderDiscountID: idOrCode}); apiErr == nil {
		return d.ID, nil
	} else if !apierror.IsNotFound(apiErr) {
		return "", apiErr
	}
	d, apiErr := discountRepo.FindByCode(ctx, accountID, idOrCode)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			return "", apierror.NewValidationErrorWithParam("Order discount not found.", "order_discount_id")
		}
		return "", apiErr
	}
	return d.ID, nil
}

// synthesizeDiscountLine emits a negative-price order line against the account's credit product to realize an order-level discount, matching Dashboard behavior. No-ops if the discount, credit product, or currency base unit cannot be resolved (a missing credit product should not fail the create; the discount amount will just not appear as a line item).
func (s *salesOrderSvcImpl) synthesizeDiscountLine(ctx context.Context, orderID string, params domain.CreateSalesOrderParams, orderTotal float64) *apierror.APIError {
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

	// The discount applies to the order total (product lines only), computed from the resolved unit prices.
	total := decimal.NewFromFloat(orderTotal)
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

// computeDiscountAmount returns the discount amount given an order-level discount and the pre-discount total. Caps at totalOrdered and rounds to nearest cent.
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

// resolveSalesRepID auto-assigns a sales rep when the caller omits one. Resolution order (matches Dashboard): commission-exempt customer/group → none; all line product-lines commission-exempt → none; customer default → zipcode territory → state territory → none. Returns nil when no match is found; any lookup error is swallowed (rep stays unset rather than failing the order).
func (s *salesOrderSvcImpl) resolveSalesRepID(ctx context.Context, accountID, buyerAccountID string, lines []domain.CreateSalesOrderLineInput, shipToState, shipToPostalCode *string) *string {
	customerRepo := s.repos.NewCustomerRepo()

	// 1. Commission-exempt customer / customer group → no rep.
	if exempt, apiErr := customerRepo.IsCommissionExempt(ctx, accountID, buyerAccountID); apiErr == nil && exempt {
		return nil
	}

	// 2. Every line's product line is commission-exempt → no rep.
	productIDs := make([]string, 0, len(lines))
	for _, l := range lines {
		if l.ProductID != "" {
			productIDs = append(productIDs, l.ProductID)
		}
	}
	if allExempt, apiErr := s.repos.NewSalesOrderRepo().AreAllLineProductLinesCommissionExempt(ctx, productIDs); apiErr == nil && allExempt {
		return nil
	}

	// 3. Customer default sales rep.
	if customer, apiErr := customerRepo.Get(ctx, accountID, buyerAccountID, nil); apiErr == nil && customer != nil && customer.DefaultSalesRepID != nil {
		return customer.DefaultSalesRepID
	}

	territoryRepo := s.repos.NewTerritoryRepo()

	// 4. Zipcode territory lookup.
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

	// 5. State-only territory lookup.
	if shipToState != nil && *shipToState != "" {
		if rep, apiErr := territoryRepo.FindSalesRepByState(ctx, accountID, *shipToState); apiErr == nil && rep != nil {
			return rep
		}
	}

	return nil
}

// resolveOrderAddress validates that an order's bill-to / ship-to address exists and belongs to the order's owner or buyer account (matching Dashboard, which only accepts existing address IDs), and returns it as a ShippingAddress for the sales-rep territory + shipping-rate logic.
func (s *salesOrderSvcImpl) resolveOrderAddress(ctx context.Context, addressRepo domain.AddressRepo, ownerAccountID, buyerAccountID, addressID string) (domain.ShippingAddress, *apierror.APIError) {
	// Prefer the buyer (customer) account — that is where order addresses live in the Dashboard flow — then fall back to the order's owner account.
	acct := ""
	for _, candidate := range []string{buyerAccountID, ownerAccountID} {
		inAccount, apiErr := addressRepo.IsInAccount(ctx, candidate, addressID)
		if apiErr != nil {
			return domain.ShippingAddress{}, apiErr
		}
		if inAccount {
			acct = candidate
			break
		}
	}
	if acct == "" {
		return domain.ShippingAddress{}, apierror.NewValidationError("Address does not belong to the order's owner or buyer account.")
	}

	existing, apiErr := addressRepo.Get(ctx, domain.GetAddressParams{AccountID: acct, AddressID: addressID})
	if apiErr != nil {
		return domain.ShippingAddress{}, apiErr
	}
	return shippingAddressFromDomain(existing), nil
}

// shippingAddressFromDomain projects a stored Address (+ geolocation) into the flat ShippingAddress used by the shipping-rate cascade.
func shippingAddressFromDomain(a *domain.Address) domain.ShippingAddress {
	out := domain.ShippingAddress{Name: a.Name, Phone: a.Phone, Email: a.Email}
	if a.Geolocation != nil {
		out.Street1 = derefString(a.Geolocation.StreetLine1)
		out.Street2 = a.Geolocation.StreetLine2
		out.City = derefString(a.Geolocation.Locality)
		out.State = derefString(a.Geolocation.State)
		out.Zip = derefString(a.Geolocation.PostalCode)
		out.Country = a.Geolocation.Country
	}
	return out
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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

// replaceOrderEmailContacts clears existing rows for the given notification type and inserts fresh rows for the supplied contacts. Matches Dashboard's delete-and-recreate behavior on update.
func replaceOrderEmailContacts(ctx context.Context, repo domain.SalesOrderRepo, salesOrderID string, contacts []domain.SalesOrderEmailContactInput, notificationTypeCode string) *apierror.APIError {
	if apiErr := repo.DeleteEmailContactsByOrderAndType(ctx, salesOrderID, notificationTypeCode); apiErr != nil {
		return apiErr
	}
	return createOrderEmailContacts(ctx, repo, salesOrderID, contacts, notificationTypeCode)
}

// checkSalesOrderReadPermission checks the appropriate read permission based on the target context. Non-internal actors are gated by access checks rather than permissions. Internal actors targeting a customer or supplier account use the relationship domain.
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

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
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

	// Counterparty-write exception: this mutation is gated by counterparty read access rather than CheckEditAccess, which is owner-direction only and rejects targets with an active billing plan (i.e. every paying merchant a customer checks out against). The customer-actor gate above plus the account-relation check here is the intended authorization for checkout.
	if identity.IsExternalTarget() {
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, customerAccountID, targetAccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

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

func (s *salesOrderSvcImpl) RecordOrderPayment(ctx context.Context, salesOrderID, paymentIntentID string) *apierror.APIError {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.record_order_payment")
	defer span.End()

	repo := s.repos.NewOrderPaymentIntentRepo()

	// Idempotent: a Stripe webhook can be retried, so skip if this payment intent is already linked to an order.
	existing, apiErr := repo.FindByPaymentIntentID(ctx, paymentIntentID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if existing != nil {
		return nil
	}

	opiID, apiErr := id.GenID(id.OrderPaymentIntentIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := repo.Create(ctx, opiID, paymentIntentID, salesOrderID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
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

func includesSalesOrderContacts(includes []string) bool {
	for _, inc := range includes {
		if inc == "contacts" {
			return true
		}
	}
	return false
}

// attachSalesOrderContacts batch-loads email recipients for the given orders (one
// query for the whole set) and assigns them per order. No-op when none are passed.
func attachSalesOrderContacts(ctx context.Context, repo domain.SalesOrderRepo, orders []*domain.SalesOrder) *apierror.APIError {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]string, len(orders))
	for i, o := range orders {
		ids[i] = o.ID
	}
	contacts, apiErr := repo.GetContactsByOrders(ctx, ids)
	if apiErr != nil {
		return apiErr
	}
	for _, o := range orders {
		if c, ok := contacts[o.ID]; ok {
			o.InvoiceEmails = c.InvoiceEmails
			o.AcknowledgementEmails = c.AcknowledgementEmails
		}
	}
	return nil
}
