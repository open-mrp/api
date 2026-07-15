package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
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
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/textutil"
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

	// Populate the derived payment status and linked payment intent IDs for the whole page in batched queries (no per-order N+1), defaulting any order without payment activity to unpaid.
	if len(result.SalesOrders) > 0 {
		orderIDs := make([]string, len(result.SalesOrders))
		for i, order := range result.SalesOrders {
			orderIDs[i] = order.ID
		}
		statuses, apiErr := repo.GetPaymentStatuses(ctx, params.AccountID, orderIDs)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		paymentIntentIDs, apiErr := repo.GetPaymentIntentIDs(ctx, params.AccountID, orderIDs)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		progress, apiErr := repo.GetFulfillmentProgress(ctx, orderIDs)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, order := range result.SalesOrders {
			if status, ok := statuses[order.ID]; ok {
				order.PaymentStatus = status
			} else {
				order.PaymentStatus = constants.SalesOrderPaymentStatusUnpaid
			}
			order.PaymentIntentIDs = paymentIntentIDs[order.ID]
			p := progress[order.ID]
			order.PickedCompletion = p.PickedCompletion
			order.PackedCompletion = p.PackedCompletion
			order.InvoicedCompletion = p.InvoicedCompletion
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

	if includesSalesOrderInvoices(params.Includes) {
		for _, order := range result.SalesOrders {
			ids, apiErr := repo.GetInvoiceIDs(ctx, order.ID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			order.InvoiceIDs = ids
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

	// Populate the derived payment status and linked payment intent IDs (always present on the resource).
	statuses, apiErr := repo.GetPaymentStatuses(ctx, params.AccountID, []string{order.ID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if status, ok := statuses[order.ID]; ok {
		order.PaymentStatus = status
	} else {
		order.PaymentStatus = constants.SalesOrderPaymentStatusUnpaid
	}
	paymentIntentIDs, apiErr := repo.GetPaymentIntentIDs(ctx, params.AccountID, []string{order.ID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	order.PaymentIntentIDs = paymentIntentIDs[order.ID]

	progress, apiErr := repo.GetFulfillmentProgress(ctx, []string{order.ID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	p := progress[order.ID]
	order.PickedCompletion = p.PickedCompletion
	order.PackedCompletion = p.PackedCompletion
	order.InvoicedCompletion = p.InvoicedCompletion

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

	if includesSalesOrderInvoices(params.Includes) {
		ids, apiErr := repo.GetInvoiceIDs(ctx, params.SalesOrderID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		order.InvoiceIDs = ids
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

	// Authorization: internal users need sales_orders:create; a customer entering an
	// order on the portal may self-create only for their own account AND must hold
	// purchase_orders:create (to them the order is a purchase — see the Customer
	// role). Other actor types cannot create orders.
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
		// Customer-side capability: authorized by the customer's OWN-account
		// purchase_orders:create (their carried role), not an owner-account permission.
		if apiErr := identity.CheckHasRelationCapability(types.PermissionDomainPurchaseOrders, types.ActionCreate); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
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

		// Reject a duplicate customer-PO number FIRST — before the more expensive line/pricing resolution and the external shipping-rate lookup — so a duplicate fails fast without that wasted work. The order number itself is allocated inside the write transaction below (not here), so a later failure — including the external Shippo rate lookup — never burns a number and leaves a permanent gap in the per-account sequence.
		orderRepo := s.repos.NewSalesOrderRepo()

		if params.CustomerPONumber != nil && *params.CustomerPONumber != "" {
			isDup, apiErr := orderRepo.IsDuplicateCustomerPO(ctx, params.AccountID, params.BuyerAccountID, *params.CustomerPONumber, nil)
			if apiErr != nil {
				return nil, cacheErr(apiErr)
			}
			if isDup {
				return nil, cacheErr(apierror.NewConflictErrorWithParam("A sales order with this customer PO number already exists.", "customer_po_number"))
			}
		}

		// Resolve everything else that only requires reads (and the live Shippo call) BEFORE opening the write transaction, so the external rate lookup never holds a DB transaction open across network latency. The transaction below performs only the inserts.
		addressRepo := s.repos.NewAddressRepo()

		// Reference the existing bill-to / ship-to addresses by ID (matching Dashboard, which only accepts address IDs; addresses are persisted separately). Each must belong to the order's owner or buyer account. The resolved ship-to feeds the sales-rep territory + shipping-rate logic below; the bill-to supplies the third-party freight-billing address.
		billAddr, apiErr := s.resolveOrderAddress(ctx, addressRepo, params.AccountID, params.BuyerAccountID, params.BillToAddressID)
		if apiErr != nil {
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

		// Fill carrier, service level, shipping term, and payment term from the buyer's customer-relation defaults whenever the caller omits them, mirroring the Dashboard create form (which pre-fills these from the selected customer). Carrier, shipping term, and payment term are mandatory on a readable order — the Dashboard order adapter rejects any order missing one — so an API create that omits them without a customer default to fall back on is failed here rather than persisted as a record that 500s on every read.
		if apiErr := s.applyCustomerOrderDefaults(ctx, &params); apiErr != nil {
			return nil, cacheErr(apiErr)
		}

		// Reject caller-supplied foreign keys that don't exist. Vitess does not
		// enforce these FKs, so without these checks a garbage id would be
		// silently stored as a dangling reference (or, for email-contact account
		// users, silently dropped) instead of failing the create.
		if apiErr := s.validateSalesOrderReferences(ctx, params); apiErr != nil {
			return nil, cacheErr(apiErr)
		}

		// Auto-assign sales rep when caller didn't supply one, matching Dashboard behavior: commission-exempt customer/product-lines yield no rep; otherwise prefer the customer's default sales rep, then zipcode territory, then state territory.
		salesRepID := params.SalesRepID
		if salesRepID == nil {
			shipState, shipPostal := shipAddr.State, shipAddr.Zip
			salesRepID = s.resolveSalesRepID(ctx, params.AccountID, params.BuyerAccountID, params.Lines, &shipState, &shipPostal)
		}

		// Resolve every line against the product/pricing data: validate the quantity unit against the product's unit group, default SKU/description from the product, derive the item + unit cost, and compute the unit price (internal actors may override via the line's unit_price; customer submissions are ignored and the price is computed server-side).
		resolvedLines, apiErr := resolveSalesOrderCreateLines(ctx, s.repos, params.AccountID, params.BuyerAccountID, identity.IsInternalUser(), params.Lines)
		if apiErr != nil {
			return nil, cacheErr(apiErr)
		}

		// Order total (product lines only), computed from the resolved unit prices — used by the shipping-rate minimum-order check and the discount line.
		orderTotal := calculateResolvedLinesTotal(resolvedLines)

		// Estimate the shipping rate via the freight-exemption / flat-rate / minimum-order / live-Shippo cascade (matches Dashboard). This is the only external call in the create path; computing it here on the outer receiver keeps the live Shippo HTTP request out of the transaction and uses the real Shippo factory.
		shippingRate, apiErr := s.estimateOrderShippingRate(ctx, params, billAddr, shipAddr, orderTotal)
		if apiErr != nil {
			return nil, cacheErr(apiErr)
		}

		var result *domain.SalesOrder
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txCtx = event.WithRepos(txCtx, txSvc.repos)

			txOrderRepo := txSvc.repos.NewSalesOrderRepo()
			txLineRepo := txSvc.repos.NewSalesOrderLineRepo()

			// Allocate the order number inside the transaction (the eager sys_property counter write happens here too) so that any failure in this create rolls the number back instead of leaving a permanent gap in the per-account sequence. Anything that can fail before this point (line/pricing resolution, the external Shippo rate lookup) therefore never consumes a number.
			orderNumber, apiErr := txOrderRepo.GetNextOrderNumber(txCtx, params.AccountID)
			if apiErr != nil {
				return apiErr
			}
			isDup, apiErr := txOrderRepo.IsDuplicateOrderNumber(txCtx, params.AccountID, orderNumber, nil)
			if apiErr != nil {
				return apiErr
			}
			if isDup {
				return apierror.NewConflictErrorWithParam("A sales order with this number already exists.", "number")
			}

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
		// Freight is no longer re-estimated automatically on ship-to / carrier / service-level changes. Users refresh freight explicitly via the quote-freight endpoint (QuoteSalesOrderFreight) and approve the new price, so an order update never silently overwrites the shipping line.
		var result *domain.SalesOrder
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesOrderRepo()

			// Validate order exists
			existing, apiErr := txRepo.Get(txCtx, params.AccountID, params.SalesOrderID)
			if apiErr != nil {
				return apiErr
			}

			// sales_rep_id references account_user.id. Reject a caller-supplied value that
			// isn't a real account_user for this account — otherwise a user id (or any other
			// id) is stored unvalidated and no join resolves it, silently blanking the rep.
			// Only an explicitly-set value is checked; omit/clear skip the lookup.
			if repID, ok := params.SalesRepID.Value(); ok && repID != "" {
				if _, apiErr := txSvc.repos.NewAccountUserRepo().GetDetailByAccountAndID(txCtx, params.AccountID, repID, nil); apiErr != nil {
					return mapSalesOrderReferenceError(apiErr, "Sales rep not found.", "sales_rep_id")
				}
			}

			// Decide whether the caller changed carrier / service level / ship-to BEFORE the
			// backfill below rewrites omitted fields to the existing values.
			shippingChanged := salesOrderShippingChanged(existing, params)

			// Non-nullable optional FKs: when the caller omits a field (nil), preserve the
			// existing value. These cannot be cleared — an empty value would set an invalid
			// FK, not null the column.
			if params.CarrierID == nil {
				params.CarrierID = existing.CarrierID
			}
			if params.ShippingTermID == nil {
				params.ShippingTermID = existing.ShippingTermID
			}
			if params.PaymentTermID == nil {
				params.PaymentTermID = existing.PaymentTermID
			}
			if params.BuyerAccountID == nil {
				params.BuyerAccountID = &existing.BuyerAccountID
			}

			// Clearable nullable fields: backfill an unset (omitted) field from the existing
			// order so the repo's plain narg keeps it; an explicit clear resolves to NULL,
			// and a set value overwrites. This is how the three-state (set/clear/leave)
			// survives to the SQL — an empty string is a real value, never a clear.
			params.CustomerPONumber = params.CustomerPONumber.BackfillUnsetPtr(existing.CustomerPONumber)
			params.Note = params.Note.BackfillUnsetPtr(existing.Note)
			params.ServiceLevelID = params.ServiceLevelID.BackfillUnsetPtr(existing.ServiceLevelID)
			params.CarrierBillingType = params.CarrierBillingType.BackfillUnsetPtr(existing.CarrierBillingType)
			params.CarrierBillingAccount = params.CarrierBillingAccount.BackfillUnsetPtr(existing.CarrierBillingAccount)
			params.SalesRepID = params.SalesRepID.BackfillUnsetPtr(existing.SalesRepID)
			params.OrderDiscountID = params.OrderDiscountID.BackfillUnsetPtr(existing.OrderDiscountID)
			params.PromisedAt = params.PromisedAt.BackfillUnsetPtr(existing.PromisedAt)

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

			// When the order's carrier, service level, or ship-to changed, re-sync the
			// order's existing shipment records out-of-band via the outbox (matching
			// legacy updateCarrierByOrder / updateShipToByOrder, which cascaded these
			// edits to shipments synchronously). Emitting inside the tx keeps the event
			// atomic with the update.
			if shippingChanged && txSvc.salesOrderPublisher != nil {
				// The outbox publisher reads the RepoFactory from the context (as the create
				// flow does at the top of its tx); inject it before publishing.
				pubCtx := event.WithRepos(txCtx, txSvc.repos)
				if apiErr := txSvc.salesOrderPublisher.PublishSalesOrderShippingUpdated(pubCtx, messaging.SalesOrderShippingUpdatedData{
					SalesOrderID: updated.ID,
					AccountID:    params.AccountID,
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
	if order.CompletedAt != nil || order.SalesOrderStatusCode == string(constants.SalesOrderStatusCodeFulfilled) {
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

				// Pick lines start at 0 picked: a pick line's quantity is the amount picked so far (quantity_picked = SUM(pick_line.value)), filled in later by the pick action — matching legacy and v2's own pack-path placeholder. Seeding it with the ordered quantity would make the order read as fully picked the instant it is issued and break PickAllLines / remaining-quantity math.
				if apiErr := txLineRepo.CreateQuantity(txCtx, pickQtyID, "0", line.QuantityUnitID); apiErr != nil {
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
				SalesOrderStatusCode: "issued",
				IssuedAt:             &now,
			}, "SalesOrderStatusCode", "IssuedAt")

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
				SalesOrderStatusCode: "estimate",
			}, "SalesOrderStatusCode", "IssuedAt")

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

			// Closing the order also closes its pick: pack every still-open pick line and
			// mark the pick finished, so the pick reads as complete alongside the order.
			if order.PickID != nil {
				txPickRepo := txSvc.repos.NewPickRepo()
				if apiErr := txPickRepo.CloseOpenPickLines(txCtx, *order.PickID); apiErr != nil {
					return apiErr
				}
				if apiErr := txPickRepo.UpdateFinishedAt(txCtx, params.AccountID, *order.PickID, now); apiErr != nil {
					return apiErr
				}
			}

			changes := audit.ComputeChanges(order, &domain.SalesOrder{
				SalesOrderStatusCode: "fulfilled",
				CompletedAt:          &now,
			}, "SalesOrderStatusCode", "CompletedAt")

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

			// Reopening the order reopens its pick: clear the pick's finished flag and reopen
			// (unpack) every pick line that is not yet complete — its picked quantity is below
			// the ordered quantity — so outstanding lines can be worked again. Fully-picked
			// lines stay packed.
			if order.PickID != nil {
				txPickRepo := txSvc.repos.NewPickRepo()
				if apiErr := txPickRepo.ReopenIncompletePickLines(txCtx, *order.PickID); apiErr != nil {
					return apiErr
				}
				if apiErr := txPickRepo.ClearFinishedAt(txCtx, params.AccountID, *order.PickID); apiErr != nil {
					return apiErr
				}
			}

			changes := audit.ComputeChanges(order, &domain.SalesOrder{
				SalesOrderStatusCode: "issued",
			}, "SalesOrderStatusCode", "CompletedAt")

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
		if apiErr := s.sendOrderAcknowledgementEmail(ctx, params.AccountID, params.SalesOrderID); apiErr != nil {
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

// portalRegisterLink returns the customer-portal registration URL for the account, or "" when the account has no customer portal configured. A verified custom portal domain is targeted directly; otherwise the slug-prefixed frontend URL is used. Best-effort: lookup failures yield "".
func (s *salesOrderSvcImpl) portalRegisterLink(ctx context.Context, accountID string) string {
	// Path mirrors the frontend's FrontendPaths.register ("/auth/register").
	const registerPath = "/auth/register"
	if domain, _ := s.repos.NewPortalDomainRepo().GetByAccountID(ctx, accountID); domain != nil && domain.Status == constants.PortalDomainStatusVerified {
		return "https://" + domain.Domain + registerPath
	}
	slug, _ := s.repos.NewAccountRepo().GetPortalSlug(ctx, accountID)
	if slug != nil && strings.TrimSpace(*slug) != "" && s.frontendURL != "" {
		return fmt.Sprintf("%s/%s%s", s.frontendURL, *slug, registerPath)
	}
	return ""
}

// sendOrderAcknowledgementEmail publishes the order-acknowledgement email to the contacts configured for the order and marks the order as acknowledged. No-ops if there are no recipients or the notification publisher is not configured.
func (s *salesOrderSvcImpl) sendOrderAcknowledgementEmail(ctx context.Context, accountID, salesOrderID string) *apierror.APIError {
	if s.notificationPublisher == nil {
		return nil
	}

	repo := s.repos.NewSalesOrderRepo()
	recipients, apiErr := repo.GetAcknowledgementRecipients(ctx, salesOrderID)
	if apiErr != nil {
		return apiErr
	}
	if len(recipients) == 0 {
		return nil
	}

	order, apiErr := repo.Get(ctx, accountID, salesOrderID)
	if apiErr != nil {
		return apiErr
	}
	lines, apiErr := repo.GetLines(ctx, salesOrderID)
	if apiErr != nil {
		return apiErr
	}

	// The seller's branding (logo, contact) and origin address power the email
	// letterhead/footer and the PDF letterhead. Both are best-effort: the
	// acknowledgement still sends with a blank letterhead if either lookup fails.
	account, _ := s.repos.NewAccountRepo().GetByID(ctx, accountID)
	originAddr, _ := repo.GetAccountOriginAddress(ctx, accountID)

	data := buildOrderAcknowledgementData(order, lines, account, originAddr)
	// The acknowledgement recipients are shown as contact emails under Bill To.
	data.ContactEmails = recipients
	// Gate the "Order Online" CTA on the account having a customer portal.
	data.OrderOnlineLink = s.portalRegisterLink(ctx, accountID)
	// Fetch the logo bytes for the PDF letterhead (best-effort; the email uses the URL).
	if data.LogoURL != "" {
		data.LogoImageType, data.LogoImage = fetchAckLogoImage(ctx, data.LogoURL)
	}

	emailData := messaging.EmailSendData{
		To:         recipients,
		Subject:    fmt.Sprintf("Sales Order %s", data.OrderNumber),
		TemplateID: constants.EmailTemplateOrderAcknowledgement,
		Params:     data.emailParams(),
		AccountID:  &accountID,
	}

	// Attach the generated order-acknowledgement PDF, matching legacy (which attached a
	// rendered PDF of the order). A PDF failure degrades gracefully to an attachment-free
	// email rather than blocking the acknowledgement.
	if pdfBytes, err := buildOrderAcknowledgementPDF(data); err == nil {
		encoded := base64.StdEncoding.EncodeToString(pdfBytes)
		filename := "order-acknowledgement-" + data.OrderNumber + ".pdf"
		contentType := "application/pdf"
		emailData.AttachmentData = &encoded
		emailData.AttachmentFilename = &filename
		emailData.AttachmentContentType = &contentType
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
		// The outbox publisher reads the RepoFactory from the context; inject it before publishing.
		pubCtx := event.WithRepos(txCtx, txSvc.repos)
		if apiErr := s.notificationPublisher.PublishSendEmail(pubCtx, emailData); apiErr != nil {
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

	// Carry the shipping product's description onto the line (matches product lines, which default their description from the product). Falls back to the SKU so the freight line is never left blank when the system product has no description configured.
	description := shippingProduct.ProductDescription
	if description == nil || *description == "" {
		description = &shippingProduct.ProductSKU
	}

	_, apiErr = s.repos.NewSalesOrderLineRepo().Create(ctx, lineID, domain.CreateSalesOrderLineParams{
		SalesOrderID:               orderID,
		AccountID:                  params.AccountID,
		ProductID:                  shippingProduct.ProductID,
		ProductSKU:                 shippingProduct.ProductSKU,
		ProductDescription:         description,
		QuantityValue:              "1",
		QuantityUnitID:             shippingProduct.QuantityUnitID,
		UnitPriceValue:             rate,
		UnitPriceNumeratorUnitID:   currencyUnitID,
		UnitPriceDenominatorUnitID: shippingProduct.QuantityUnitID,
	})
	return apiErr
}

// estimateFreightForOrder re-estimates an existing order's freight (shipping) charge from its CURRENT persisted ship-to, carrier, service level, and lines, using the same freight-exemption / flat-rate / minimum-order / live-Shippo cascade as the create path. Returns the rate rounded to cents (matching Dashboard's update path). Read-only: it does not mutate the order — the caller (the quote-freight endpoint) returns it for the user to review and approve. Runs on the outer receiver so the live Shippo call stays out of any write transaction.
func (s *salesOrderSvcImpl) estimateFreightForOrder(ctx context.Context, accountID, salesOrderID string) (string, *apierror.APIError) {
	existing, apiErr := s.repos.NewSalesOrderRepo().Get(ctx, accountID, salesOrderID)
	if apiErr != nil {
		return "", apiErr
	}

	shipAddr, apiErr := s.resolveOrderAddress(ctx, s.repos.NewAddressRepo(), accountID, existing.BuyerAccountID, existing.ShippingAddressID)
	if apiErr != nil {
		return "", apiErr
	}

	// Identify the synthesized shipping / discount lines so they are excluded from the weight and minimum-order-total inputs (they are not product lines).
	var shippingProductID, creditProductID string
	if shippingProduct, apiErr := s.repos.NewProductRepo().GetSystemProduct(ctx, accountID, "shipping"); apiErr != nil {
		return "", apiErr
	} else if shippingProduct != nil {
		shippingProductID = shippingProduct.ProductID
	}
	if creditProduct, apiErr := s.repos.NewProductRepo().GetSystemProduct(ctx, accountID, "credit"); apiErr != nil {
		return "", apiErr
	} else if creditProduct != nil {
		creditProductID = creditProduct.ProductID
	}

	lines, apiErr := s.repos.NewSalesOrderRepo().GetLines(ctx, salesOrderID)
	if apiErr != nil {
		return "", apiErr
	}

	// Rebuild the create-path line inputs (product lines only) that drive the parcel weight and product-line freight exemption, and sum the minimum-order total.
	lineInputs := make([]domain.CreateSalesOrderLineInput, 0, len(lines))
	total := decimal.Zero
	for _, l := range lines {
		if l.ProductID == nil {
			continue
		}
		// The minimum-order total mirrors legacy update's calculateTotalOrdered: it sums EVERY line, including the order's existing shipping charge and the (negative) discount line — not just the product lines.
		if qty, err := decimal.NewFromString(l.QuantityValue); err == nil {
			if price, err := decimal.NewFromString(l.UnitPriceValue); err == nil {
				total = total.Add(qty.Mul(price))
			}
		}
		// Weight + product-line freight-exemption inputs are the product lines only — exclude the synthesized shipping / discount lines, matching the create path.
		if *l.ProductID == shippingProductID || *l.ProductID == creditProductID {
			continue
		}
		lineInputs = append(lineInputs, domain.CreateSalesOrderLineInput{
			ProductID:      *l.ProductID,
			QuantityValue:  l.QuantityValue,
			QuantityUnitID: l.QuantityUnitID,
		})
	}
	orderTotal, _ := total.Round(2).Float64()

	// Resolve the bill-to for third-party freight billing, mirroring the create path.
	billAddr, apiErr := s.resolveOrderAddress(ctx, s.repos.NewAddressRepo(), accountID, existing.BuyerAccountID, existing.BillingAddressID)
	if apiErr != nil {
		return "", apiErr
	}

	rateStr, apiErr := s.estimateOrderShippingRate(ctx, domain.CreateSalesOrderParams{
		AccountID:             accountID,
		BuyerAccountID:        existing.BuyerAccountID,
		CarrierID:             existing.CarrierID,
		ServiceLevelID:        existing.ServiceLevelID,
		CarrierBillingType:    existing.CarrierBillingType,
		CarrierBillingAccount: existing.CarrierBillingAccount,
		Lines:                 lineInputs,
	}, billAddr, shipAddr, orderTotal)
	if apiErr != nil {
		return "", apiErr
	}

	rate, err := decimal.NewFromString(rateStr)
	if err != nil {
		return "", apierror.NewInternalError(err, "Issue parsing the estimated shipping rate.")
	}
	return rate.Round(2).String(), nil
}

// ptrStringChanged reports whether two optional string values differ (one nil and the other not, or both set to different values).
func ptrStringChanged(a, b *string) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return *a != *b
}

// salesOrderShippingChanged reports whether the update request changes the order's
// carrier, service level, or ship-to address versus the existing order. It reads the
// raw request params (before backfill), so an omitted field never counts as a change.
func salesOrderShippingChanged(existing *domain.SalesOrder, params domain.UpdateSalesOrderParams) bool {
	if params.CarrierID != nil && ptrStringChanged(existing.CarrierID, params.CarrierID) {
		return true
	}
	if params.ShippingAddressID != nil && existing.ShippingAddressID != *params.ShippingAddressID {
		return true
	}
	if params.ServiceLevelID.WasProvided() {
		if params.ServiceLevelID.IsClear() {
			return existing.ServiceLevelID != nil
		}
		if v, ok := params.ServiceLevelID.Value(); ok {
			return existing.ServiceLevelID == nil || *existing.ServiceLevelID != v
		}
	}
	return false
}

// estimateOrderShippingRate computes the posted shipping rate for a new order using the shared freight-exemption / flat-rate / minimum-order / live-Shippo cascade, mirroring Dashboard's estimatePostedShippingRate. Returns the rate as a decimal string (not rounded to cents, matching Dashboard's create path). The live Shippo rate already includes the 10% markup applied by the Shippo client.
func (s *salesOrderSvcImpl) estimateOrderShippingRate(ctx context.Context, params domain.CreateSalesOrderParams, billTo, shipTo domain.ShippingAddress, orderTotal float64) (string, *apierror.APIError) {
	// Third-party-billed orders pass the third party's account + bill-to country/zip
	// through to the carrier, matching Dashboard's createShippingLine.
	var billing *domain.ShippingBilling
	if params.CarrierBillingType != nil && *params.CarrierBillingType == string(constants.CarrierBillingTypeThirdParty) {
		account := ""
		if params.CarrierBillingAccount != nil {
			account = *params.CarrierBillingAccount
		}
		billing = &domain.ShippingBilling{
			Type:    "THIRD_PARTY",
			Account: account,
			Country: billTo.Country,
			Zip:     billTo.Zip,
		}
	}
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

	// Origin (ship-from) = the seller account's default billing address. When absent
	// this stays the zero value; estimateShippingRate refuses to quote a live Shippo
	// rate from an empty origin (rather than silently returning a bad/zero rate) but
	// still short-circuits freight-exempt / no-carrier orders to 0 without it.
	var from domain.ShippingAddress
	if fromAddr, apiErr := s.repos.NewSalesOrderRepo().GetAccountOriginAddress(ctx, params.AccountID); apiErr != nil {
		return "", apiErr
	} else if fromAddr != nil {
		from = *fromAddr
	}

	rate, apiErr := estimateShippingRate(ctx, s.repos, s.shippoFactory, s.encryptionKey, domain.EstimateRateParams{
		AccountID:      params.AccountID,
		CarrierID:      ptrutil.Deref(params.CarrierID),
		ServiceLevelID: ptrutil.Deref(params.ServiceLevelID),
		ProductLineIDs: productLineIDs,
		CustomerID:     &params.BuyerAccountID,
		FromAddress:    from,
		ToAddress:      shipTo,
		OrderTotal:     &orderTotal,
		Billing:        billing,
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

	prices, apiErr := computeSalesOrderLinePrices(ctx, s.repos, accountID, params.BuyerAccountID, identity.IsInternalUser(), params.Lines)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.SalesOrderLineQuote, len(params.Lines))
	for i, l := range params.Lines {
		out[i] = domain.SalesOrderLineQuote{ProductID: l.ProductID, UnitPrice: domain.RateValue(prices[i])}
	}
	return out, nil
}

func (s *salesOrderSvcImpl) QuoteSalesOrderFreight(ctx context.Context, params domain.QuoteSalesOrderFreightParams) (*domain.SalesOrderFreightQuote, *apierror.APIError) {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.quote_freight")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	rate, apiErr := s.estimateFreightForOrder(ctx, accountID, params.SalesOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The freight rate is currency (numerator) per shipping unit (denominator). Source the units from the account's shipping system product / currency base unit, matching how the shipping line is stored (see synthesizeShippingLine). The units may be empty when the account has no shipping product configured; the caller applies the value onto the existing shipping line, whose units already match.
	currencyUnitID, apiErr := s.repos.NewUnitRepo().GetCurrencyBaseUnitID(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var denominatorUnitID string
	if shippingProduct, apiErr := s.repos.NewProductRepo().GetSystemProduct(ctx, accountID, "shipping"); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	} else if shippingProduct != nil {
		denominatorUnitID = shippingProduct.QuantityUnitID
	}

	return &domain.SalesOrderFreightQuote{
		UnitPrice: domain.RateValue{
			Value:             rate,
			NumeratorUnitID:   currencyUnitID,
			DenominatorUnitID: denominatorUnitID,
		},
	}, nil
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

// validateSalesOrderReferences rejects caller-supplied foreign keys that don't
// exist in the account. carrier_id and order_discount_id are validated
// separately (in the shipping-rate estimate and resolveOrderDiscountID
// respectively); this covers the remaining references that were previously
// accepted unchecked.
func (s *salesOrderSvcImpl) validateSalesOrderReferences(ctx context.Context, params domain.CreateSalesOrderParams) *apierror.APIError {
	if params.ServiceLevelID != nil && *params.ServiceLevelID != "" {
		if _, apiErr := s.repos.NewServiceLevelRepo().Get(ctx, params.AccountID, *params.ServiceLevelID); apiErr != nil {
			return mapSalesOrderReferenceError(apiErr, "Service level not found.", "service_level_id")
		}
	}
	if params.ShippingTermID != nil && *params.ShippingTermID != "" {
		if _, apiErr := s.repos.NewShippingTermRepo().Get(ctx, domain.GetShippingTermParams{AccountID: params.AccountID, ShippingTermID: *params.ShippingTermID}); apiErr != nil {
			return mapSalesOrderReferenceError(apiErr, "Shipping term not found.", "shipping_term_id")
		}
	}
	if params.PaymentTermID != nil && *params.PaymentTermID != "" {
		if _, apiErr := s.repos.NewPaymentTermRepo().Get(ctx, domain.GetPaymentTermParams{AccountID: params.AccountID, PaymentTermID: *params.PaymentTermID}); apiErr != nil {
			return mapSalesOrderReferenceError(apiErr, "Payment term not found.", "payment_term_id")
		}
	}
	if params.SalesRepID != nil && *params.SalesRepID != "" {
		if _, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, params.AccountID, *params.SalesRepID, nil); apiErr != nil {
			return mapSalesOrderReferenceError(apiErr, "Sales rep not found.", "sales_rep_id")
		}
	}
	// Order-acknowledgement and invoice email contacts are recipients on the buyer (customer) side, so their account_user_id must resolve within the buyer's account, not the acting seller account.
	if apiErr := s.validateEmailContactAccountUsers(ctx, params.BuyerAccountID, params.AcknowledgementEmailContacts, "acknowledgement_email_contacts"); apiErr != nil {
		return apiErr
	}
	if apiErr := s.validateEmailContactAccountUsers(ctx, params.BuyerAccountID, params.InvoiceEmailContacts, "invoice_email_contacts"); apiErr != nil {
		return apiErr
	}
	return nil
}

// validateEmailContactAccountUsers rejects an order email-contact whose account_user_id does not exist in the buyer's account, rather than silently dropping the reference. buyerAccountID scopes the lookup: these contacts are customer-side recipients, so an account_user of the seller (acting) account is not a valid contact.
func (s *salesOrderSvcImpl) validateEmailContactAccountUsers(ctx context.Context, buyerAccountID string, contacts []domain.SalesOrderEmailContactInput, param string) *apierror.APIError {
	for _, c := range contacts {
		if c.AccountUserID == "" {
			continue
		}
		if _, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, buyerAccountID, c.AccountUserID, nil); apiErr != nil {
			return mapSalesOrderReferenceError(apiErr, "Email contact account user not found.", param)
		}
	}
	return nil
}

// mapSalesOrderReferenceError turns a not-found lookup into a field-scoped 400
// validation error and passes any other error through unchanged.
func mapSalesOrderReferenceError(apiErr *apierror.APIError, notFoundMsg, param string) *apierror.APIError {
	if apierror.IsNotFound(apiErr) {
		return apierror.NewValidationErrorWithParam(notFoundMsg, param)
	}
	return apiErr
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

// applyCustomerOrderDefaults fills the carrier, service level, shipping term, and payment term the caller omitted from the buyer's customer-relation defaults, matching the Dashboard create form which pre-fills these from the selected customer. Carrier, shipping term, and payment term are mandatory on a readable order (the Dashboard order adapter throws when any is absent), so when neither the request nor a customer default supplies one, this fails with a 400 instead of persisting an order that cannot be read back. The default service level is only adopted alongside a defaulted carrier, so the two never end up referencing different carriers.
func (s *salesOrderSvcImpl) applyCustomerOrderDefaults(ctx context.Context, params *domain.CreateSalesOrderParams) *apierror.APIError {
	if params.CarrierID == nil || params.ServiceLevelID == nil || params.ShippingTermID == nil || params.PaymentTermID == nil {
		customer, apiErr := s.repos.NewCustomerRepo().Get(ctx, params.AccountID, params.BuyerAccountID, nil)
		if apiErr != nil {
			return apiErr
		}
		if customer != nil {
			if params.CarrierID == nil {
				params.CarrierID = customer.DefaultCarrierID
				if params.ServiceLevelID == nil {
					params.ServiceLevelID = customer.DefaultServiceLevelID
				}
			}
			if params.ShippingTermID == nil {
				params.ShippingTermID = customer.DefaultShippingTermID
			}
			if params.PaymentTermID == nil {
				params.PaymentTermID = customer.DefaultPaymentTermID
			}
		}
	}

	if params.CarrierID == nil {
		return apierror.NewValidationErrorWithParam("Carrier is required and the customer has no default carrier.", "carrier_id")
	}
	if params.ShippingTermID == nil {
		return apierror.NewValidationErrorWithParam("Shipping term is required and the customer has no default shipping term.", "shipping_term_id")
	}
	if params.PaymentTermID == nil {
		return apierror.NewValidationErrorWithParam("Payment term is required and the customer has no default payment term.", "payment_term_id")
	}
	return nil
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
		out.Street1 = ptrutil.Deref(a.Geolocation.StreetLine1)
		out.Street2 = a.Geolocation.StreetLine2
		out.City = ptrutil.Deref(a.Geolocation.Locality)
		out.State = ptrutil.Deref(a.Geolocation.State)
		out.Zip = ptrutil.Deref(a.Geolocation.PostalCode)
		out.Country = a.Geolocation.Country
	}
	return out
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

		// Decrypt Stripe credentials. The credential blob is sealed with the account ID as additional authenticated data (matching how both this service and the legacy dashboard API encrypt integration credentials), so the same account ID must be supplied here.
		decrypted, err := crypto.DecryptAESGCM(encryptedCreds, s.encryptionKey, []byte(params.AccountID))
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to decrypt Stripe credentials."))
		}

		var stripeCreds domain.StripeCredentials
		if err := json.Unmarshal(decrypted, &stripeCreds); err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Stripe credentials."))
		}

		// Resolve the buyer's Stripe customer. Legacy requires the counterparty to
		// already be a Stripe customer and bills the session to it
		// (order.svc.ts:298-343 → stripe.ts createOneTimeCheckoutSession), rather than
		// passing a bare customer_email — so the payment method can be saved onto the
		// customer and the charge is attributed correctly.
		stripeCustomerID, _, apiErr := s.repos.NewCustomerRepo().GetStripeCustomerID(ctx, params.AccountID, order.BuyerAccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if stripeCustomerID == nil || *stripeCustomerID == "" {
			return nil, tracing.Trace(span, apierror.NewValidationError("Customer is not a Stripe customer."))
		}

		// Charge a single aggregate line item for the order's net total — the sum of
		// every line's extended price (including the negative discount credit line and
		// the shipping line), rounded once to the nearest cent. This matches legacy
		// calculateTotalOrdered + stripe.ts. Emitting one Stripe line item per order
		// line would send the discount line's negative unit_amount, which Stripe
		// rejects, failing checkout for every discounted order.
		total := decimal.Zero
		for _, line := range lines {
			unitPrice, err1 := decimal.NewFromString(line.UnitPriceValue)
			qty, err2 := decimal.NewFromString(line.QuantityValue)
			if err1 != nil || err2 != nil {
				continue
			}
			total = total.Add(unitPrice.Mul(qty))
		}
		amountCents := total.Mul(decimal.NewFromInt(100)).Round(0).IntPart()

		description := ""
		if order.CustomerPONumber != nil && *order.CustomerPONumber != "" {
			description = fmt.Sprintf("PO #%s", *order.CustomerPONumber)
		}
		checkoutItems := []domain.CheckoutLineItem{{
			Name:        fmt.Sprintf("SO #%s", textutil.FormatRecordNumber(order.Number)),
			Description: description,
			AmountCents: amountCents,
			Quantity:    1,
		}}

		// Build the checkout success/cancel redirect URLs server-side from the
		// account's portal slug. These are never accepted from the caller: Stripe
		// redirects the customer to them verbatim after checkout, so a caller-supplied
		// URL would turn the emailed checkout link into an open-redirect/phishing
		// vector. Both land on the customer-portal order page (mirroring the embedded
		// checkout return URL) with a payment-result query param the page reads.
		slug, apiErr := s.repos.NewAccountRepo().GetPortalSlug(ctx, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if slug == nil || *slug == "" {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account portal slug not found."))
		}
		orderPageURL := fmt.Sprintf("%s/%s/dashboard/sales-orders/%s", s.frontendURL, *slug, order.ID)
		successURL := orderPageURL + "?payment=success"
		cancelURL := orderPageURL + "?payment=cancelled"

		// Create Stripe checkout session (foreign mutation)
		checkoutClient := s.checkoutClientFactory.Build(stripeCreds.PrivateKey)
		checkoutSession, apiErr := checkoutClient.CreateOneTimeCheckoutSession(ctx, domain.CreateCheckoutSessionParams{
			StripeCustomerID: *stripeCustomerID,
			LineItems:        checkoutItems,
			SuccessURL:       &successURL,
			CancelURL:        &cancelURL,
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
				// The outbox publisher reads the RepoFactory from the context; inject it before publishing.
				pubCtx := event.WithRepos(txCtx, txSvc.repos)
				if pubErr := s.notificationPublisher.PublishSendEmail(pubCtx, messaging.EmailSendData{
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

		// 3. Decrypt Stripe credentials. The credential blob is sealed with the account ID as additional authenticated data (matching how both this service and the legacy dashboard API encrypt integration credentials), so the same account ID must be supplied here.
		decrypted, err := crypto.DecryptAESGCM(encryptedCreds, s.encryptionKey, []byte(targetAccountID))
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
				// A customer account may not carry its own email; fall back to the acting portal
				// user's email, matching the legacy checkout flow (which used the current user's
				// email when the customer had none).
				if identity.Actor != nil && identity.Actor.ID != "" {
					actorUser, apiErr := s.repos.NewUserRepo().FindByID(ctx, identity.Actor.ID)
					if apiErr != nil {
						return nil, tracing.Trace(span, apiErr)
					}
					if actorUser != nil && actorUser.Email != nil && *actorUser.Email != "" {
						customerEmail = actorUser.Email
					}
				}
			}
			if customerEmail == nil || *customerEmail == "" {
				return nil, tracing.Trace(span, apierror.NewInternalError(nil, "No email found for customer or current user."))
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

func (s *salesOrderSvcImpl) ProcessAccountStripeWebhook(ctx context.Context, accountID string, rawPayload []byte, signature string) *apierror.APIError {
	ctx, span := salesOrderSvcTracer.Start(ctx, "service.sales_order.process_account_stripe_webhook")
	defer span.End()

	integrationRepo := s.repos.NewAccountIntegrationRepo()
	encryptedCreds, isActive, apiErr := integrationRepo.GetEncryptedCredentials(ctx, accountID, constants.IntegrationCodeStripe)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !isActive {
		return tracing.Trace(span, apierror.NewValidationError("Stripe integration is not active for this account."))
	}

	// Decrypt Stripe credentials. The credential blob is sealed with the account ID as additional authenticated data (matching how both this service and the legacy dashboard API encrypt integration credentials), so the same account ID must be supplied here.
	decrypted, err := crypto.DecryptAESGCM(encryptedCreds, s.encryptionKey, []byte(accountID))
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to decrypt Stripe credentials."))
	}
	var stripeCreds domain.StripeCredentials
	if err := json.Unmarshal(decrypted, &stripeCreds); err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse Stripe credentials."))
	}
	if stripeCreds.WebhookSecret == "" {
		return tracing.Trace(span, apierror.NewValidationError("Stripe integration has no webhook secret configured for this account."))
	}

	checkoutClient := s.checkoutClientFactory.Build(stripeCreds.PrivateKey)
	event, paymentIntent, apiErr := checkoutClient.ConstructWebhookEvent(rawPayload, signature, stripeCreds.WebhookSecret)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Event coverage mirrors the retired legacy webhook: succeeded links the payment and records the receivables transaction, failed/canceled roll that back, and payout.paid stamps when funds actually land. Anything else is acknowledged without action.
	switch event.Type {
	case "payment_intent.succeeded":
		if paymentIntent == nil || paymentIntent.ID == "" {
			return nil
		}
		return tracing.Trace(span, s.handleAccountPaymentIntentSucceeded(ctx, accountID, paymentIntent))
	case "payment_intent.payment_failed":
		if paymentIntent == nil || paymentIntent.ID == "" {
			return nil
		}
		return tracing.Trace(span, s.handleAccountPaymentIntentFailed(ctx, paymentIntent.ID))
	case "payment_intent.canceled":
		if paymentIntent == nil || paymentIntent.ID == "" {
			return nil
		}
		return tracing.Trace(span, s.handleAccountPaymentIntentCanceled(ctx, paymentIntent.ID))
	case "payout.paid":
		return tracing.Trace(span, s.handleAccountPayoutPaid(ctx, accountID, checkoutClient, event.RawJSON))
	default:
		return nil
	}
}

// handleAccountPaymentIntentSucceeded links the payment intent to its order and records the receivables payment transaction, mirroring the legacy webhook. Both steps are idempotent so Stripe retries and duplicate deliveries are safe.
func (s *salesOrderSvcImpl) handleAccountPaymentIntentSucceeded(ctx context.Context, accountID string, paymentIntent *domain.StripePaymentIntent) *apierror.APIError {
	orderID := paymentIntent.Metadata["orderID"]
	if orderID == "" {
		slog.InfoContext(ctx, "account stripe webhook payment intent has no orderID metadata, skipping",
			"account_id", accountID,
			"payment_intent_id", paymentIntent.ID,
		)
		return nil
	}

	// The metadata originates from the vendor's own Stripe account, so it cannot be trusted to reference the vendor's orders. Resolve the order scoped to this account and acknowledge (without linking) anything that doesn't match, rather than erroring, so Stripe doesn't retry.
	orderRepo := s.repos.NewSalesOrderRepo()
	order, apiErr := orderRepo.Get(ctx, accountID, orderID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			slog.WarnContext(ctx, "account stripe webhook references an unknown order for this account, skipping",
				"account_id", accountID,
				"order_id", orderID,
				"payment_intent_id", paymentIntent.ID,
			)
			return nil
		}
		return apiErr
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
		if apiErr := txSvc.RecordOrderPayment(txCtx, orderID, paymentIntent.ID); apiErr != nil {
			return apiErr
		}

		// The receivables transaction is only recorded when the metadata's customer matches the order's buyer, mirroring the legacy guards.
		customerID := paymentIntent.Metadata["customerID"]
		if customerID == "" || order.BuyerAccountID != customerID {
			return nil
		}

		txRepo := txSvc.repos.NewTransactionRepo()
		existing, apiErr := txRepo.FindByStripePaymentID(txCtx, paymentIntent.ID)
		if apiErr != nil {
			return apiErr
		}
		if existing != nil {
			return nil
		}

		txID, apiErr := id.GenID(id.TransactionIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		number, apiErr := txRepo.FetchAndIncrementTransactionNumber(txCtx, accountID)
		if apiErr != nil {
			return apiErr
		}
		dollarUnitID, apiErr := txRepo.GetDollarUnitID(txCtx)
		if apiErr != nil {
			return apiErr
		}

		amount := decimal.NewFromInt(paymentIntent.Amount).Div(decimal.NewFromInt(100)).String()
		note := "Payment captured by Stripe"
		return txRepo.Create(txCtx, txID, number, string(constants.TransactionTypePayment), accountID, customerID, &paymentIntent.ID, stripeTransactionMethodCode(paymentIntent.PaymentMethodTypes), nil, nil, &note, amount, dollarUnitID)
	})
}

// handleAccountPaymentIntentFailed marks the recorded transaction as failed and unwinds its allocations and the order link. Mirrors the legacy webhook: a payment intent with no recorded transaction is a no-op.
func (s *salesOrderSvcImpl) handleAccountPaymentIntentFailed(ctx context.Context, paymentIntentID string) *apierror.APIError {
	return s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewTransactionRepo()
		record, apiErr := txRepo.FindByStripePaymentID(txCtx, paymentIntentID)
		if apiErr != nil {
			return apiErr
		}
		if record == nil {
			return nil
		}

		if apiErr := txRepo.UpdateNote(txCtx, record.ID, "Payment failed in Stripe"); apiErr != nil {
			return apiErr
		}
		if apiErr := txRepo.DeleteAllocations(txCtx, record.ID); apiErr != nil {
			return apiErr
		}
		return txSvc.unlinkOrderPaymentIntent(txCtx, paymentIntentID)
	})
}

// handleAccountPaymentIntentCanceled deletes the recorded transaction entirely along with its allocations, amount, and the order link. Mirrors the legacy webhook: a payment intent with no recorded transaction is a no-op.
func (s *salesOrderSvcImpl) handleAccountPaymentIntentCanceled(ctx context.Context, paymentIntentID string) *apierror.APIError {
	return s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewTransactionRepo()
		record, apiErr := txRepo.FindByStripePaymentID(txCtx, paymentIntentID)
		if apiErr != nil {
			return apiErr
		}
		if record == nil {
			return nil
		}

		if apiErr := txRepo.DeleteAllocations(txCtx, record.ID); apiErr != nil {
			return apiErr
		}
		if apiErr := txRepo.Delete(txCtx, record.ID); apiErr != nil {
			return apiErr
		}
		if apiErr := txRepo.DeleteQuantity(txCtx, record.AmountID); apiErr != nil {
			return apiErr
		}
		return txSvc.unlinkOrderPaymentIntent(txCtx, paymentIntentID)
	})
}

// handleAccountPayoutPaid stamps funds_received_at on every transaction funded by the payout, resolved by walking the payout's balance transactions in Stripe.
func (s *salesOrderSvcImpl) handleAccountPayoutPaid(ctx context.Context, accountID string, checkoutClient domain.StripeCheckoutClient, rawObject []byte) *apierror.APIError {
	var payout struct {
		ID          string `json:"id"`
		ArrivalDate int64  `json:"arrival_date"`
	}
	if err := json.Unmarshal(rawObject, &payout); err != nil || payout.ID == "" {
		slog.WarnContext(ctx, "account stripe webhook payout event could not be parsed, skipping", "account_id", accountID)
		return nil
	}

	ids, apiErr := checkoutClient.ListPayoutPaymentIntentIDs(ctx, payout.ID)
	if apiErr != nil {
		return apiErr
	}
	if len(ids) == 0 {
		return nil
	}

	return s.repos.NewTransactionRepo().UpdateFundsReceivedByStripePaymentIDs(ctx, accountID, ids, time.Unix(payout.ArrivalDate, 0))
}

// unlinkOrderPaymentIntent removes the order↔payment-intent link if one exists.
func (s *salesOrderSvcImpl) unlinkOrderPaymentIntent(ctx context.Context, paymentIntentID string) *apierror.APIError {
	opiRepo := s.repos.NewOrderPaymentIntentRepo()
	link, apiErr := opiRepo.FindByPaymentIntentID(ctx, paymentIntentID)
	if apiErr != nil {
		return apiErr
	}
	if link == nil {
		return nil
	}
	return opiRepo.Delete(ctx, link.ID)
}

// stripeTransactionMethodCode maps Stripe payment method types to the internal transaction method code, mirroring the legacy webhook's mapping and precedence: card, then US bank account, then Link (which reads as credit card).
func stripeTransactionMethodCode(methods []string) *string {
	if slices.Contains(methods, "card") || (!slices.Contains(methods, "us_bank_account") && slices.Contains(methods, "link")) {
		code := string(domain.TransactionMethodCreditCard)
		return &code
	}
	if slices.Contains(methods, "us_bank_account") {
		code := string(domain.TransactionMethodACH)
		return &code
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

		// Calculate the aggregated material demand across all item-backed lines (one reservation per material).
		materialDemandRepo := s.repos.NewMaterialDemandRepo()
		demandLines := make([]domain.MaterialDemandLineInput, 0, len(bomLines))
		for _, line := range bomLines {
			demandLines = append(demandLines, domain.MaterialDemandLineInput{
				ItemID:  line.ItemID,
				Measure: line.QuantityValue,
				UnitID:  line.QuantityUnitID,
			})
		}
		allDemands, apiErr := materialDemandRepo.GetMaterialDemandForOrder(ctx, params.AccountID, demandLines)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Compute the batches to create by walking each line's production-flow graph and
		// finding the material-only production blocks (matches legacy getBaseBatches). This
		// replaces the naive one-batch-per-line approach. Done here (outside the tx) since it
		// reads the flow + unit graphs.
		batchLines := make([]domain.ProductionBatchLineInput, 0, len(bomLines))
		for _, line := range bomLines {
			batchLines = append(batchLines, domain.ProductionBatchLineInput{
				ProducedItemID: line.ItemID,
				OrderedMeasure: line.QuantityValue,
				OrderedUnitID:  line.QuantityUnitID,
			})
		}
		batchItems, apiErr := s.computeProductionBatchItems(ctx, params.AccountID, batchLines)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		result := &domain.CreateSalesOrderProductionRunResult{}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
			txRunRepo := txSvc.repos.NewProductionRunQueryRepo()
			txOrderRepo := txSvc.repos.NewSalesOrderRepo()
			txBatchRepo := txSvc.repos.NewBatchRepo()

			// Create the production run
			if apiErr := txRunRepo.Create(txCtx, runID, responsibleUserID, runNumber, params.AccountID); apiErr != nil {
				return apiErr
			}

			// Create one batch per material-only production block (see computeProductionBatchItems).
			for _, item := range batchItems {
				batchID, apiErr := id.GenID(id.BatchIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				_, apiErr = txBatchRepo.Create(txCtx, batchID, domain.CreateBatchParams{
					AccountID:       params.AccountID,
					ItemID:          item.ItemID,
					ProductionRunID: runID,
					Quantity: domain.CreateQuantityParams{
						Measure: item.Measure,
						UnitID:  item.UnitID,
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

			// Load the created run (with its batch count and responsible-user join) so the
			// response is the full production run resource, not just its id.
			createdRun, apiErr := txSvc.repos.NewProductionRunRepo().Get(txCtx, domain.GetProductionRunParams{
				ProductionRunID: runID,
				AccountID:       params.AccountID,
			})
			if apiErr != nil {
				return apiErr
			}
			result.ProductionRun = createdRun

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

func includesSalesOrderInvoices(includes []string) bool {
	for _, inc := range includes {
		if inc == "related.invoices" {
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

// attachSalesOrderContacts batch-loads email recipients for the given orders (one query for the whole set) and assigns them per order. No-op when none are passed.
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
