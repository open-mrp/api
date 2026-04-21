package service

import (
	"context"
	"encoding/json"
	"fmt"
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
	if identity.IsInternalUser() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	// Customer actors can only see their own orders
	if identity.IsCustomerUser() {
		actorAccountID := identity.ActorAccountID()
		params.BuyerAccountID = actorAccountID
	}

	return s.repos.NewSalesOrderRepo().List(ctx, params)
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
	if identity.IsInternalUser() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead); apiErr != nil {
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

	// Conditionally fetch lines based on includes
	for _, include := range params.Includes {
		if include == "lines" {
			lines, apiErr := repo.GetLines(ctx, params.SalesOrderID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			order.Lines = lines
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

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionCreate); apiErr != nil {
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
				SalesRepID:            params.SalesRepID,
				ShippingTermID:        params.ShippingTermID,
				SalesOrderTypeCode:    params.SalesOrderTypeCode,
				PaymentTermID:         params.PaymentTermID,
				OrderDiscountID:       params.OrderDiscountID,
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

			// Re-fetch the complete order
			order, apiErr := txOrderRepo.Get(txCtx, params.AccountID, orderID)
			if apiErr != nil {
				return apiErr
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

			// Update bill-to address record if any bill-to fields are provided
			if params.BillToName != nil || params.BillToStreetLine1 != nil || params.BillToStreetLine2 != nil ||
				params.BillToLocality != nil || params.BillToState != nil || params.BillToPostalCode != nil || params.BillToCountry != nil {
				txAddrRepo := txSvc.repos.NewAddressRepo()

				_, apiErr = txAddrRepo.Update(txCtx, domain.UpdateAddressParams{
					AccountID: params.AccountID,
					AddressID: existing.BillingAddressID,
					Name:      params.BillToName,
				})
				if apiErr != nil {
					return apiErr
				}

				geoID, apiErr := txAddrRepo.GetGeolocationIDByAddressID(txCtx, existing.BillingAddressID)
				if apiErr != nil {
					return apiErr
				}

				apiErr = txAddrRepo.UpdateGeolocation(txCtx, geoID, domain.UpdateAddressParams{
					AccountID:   params.AccountID,
					AddressID:   existing.BillingAddressID,
					StreetLine1: params.BillToStreetLine1,
					StreetLine2: params.BillToStreetLine2,
					Locality:    params.BillToLocality,
					State:       params.BillToState,
					PostalCode:  params.BillToPostalCode,
					Country:     params.BillToCountry,
				})
				if apiErr != nil {
					return apiErr
				}
			}

			// Update ship-to address record if any ship-to fields are provided
			if params.ShipToName != nil || params.ShipToStreetLine1 != nil || params.ShipToStreetLine2 != nil ||
				params.ShipToLocality != nil || params.ShipToState != nil || params.ShipToPostalCode != nil || params.ShipToCountry != nil {
				txAddrRepo := txSvc.repos.NewAddressRepo()

				_, apiErr = txAddrRepo.Update(txCtx, domain.UpdateAddressParams{
					AccountID: params.AccountID,
					AddressID: existing.ShippingAddressID,
					Name:      params.ShipToName,
				})
				if apiErr != nil {
					return apiErr
				}

				geoID, apiErr := txAddrRepo.GetGeolocationIDByAddressID(txCtx, existing.ShippingAddressID)
				if apiErr != nil {
					return apiErr
				}

				apiErr = txAddrRepo.UpdateGeolocation(txCtx, geoID, domain.UpdateAddressParams{
					AccountID:   params.AccountID,
					AddressID:   existing.ShippingAddressID,
					StreetLine1: params.ShipToStreetLine1,
					StreetLine2: params.ShipToStreetLine2,
					Locality:    params.ShipToLocality,
					State:       params.ShipToState,
					PostalCode:  params.ShipToPostalCode,
					Country:     params.ShipToCountry,
				})
				if apiErr != nil {
					return apiErr
				}
			}

			// Update the order
			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

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

	return s.withTx(ctx, func(txCtx context.Context, txSvc *salesOrderSvcImpl) *apierror.APIError {
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
		return nil
	})
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

	// Re-fetch the updated order
	updatedOrder, apiErr := repo.Get(ctx, params.AccountID, params.SalesOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return updatedOrder, nil
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
