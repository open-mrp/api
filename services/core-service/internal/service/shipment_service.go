package service

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
)

var shipmentSvcTracer = tracing.GetTracer("core-service.shipment_service")

type shipmentSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
	shippoFactory   domain.ShippoClientFactory
	notificationPub domain.NotificationPublisher
}

type ShipmentSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
	ShippoFactory   domain.ShippoClientFactory
	NotificationPub domain.NotificationPublisher
}

func (c *ShipmentSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("shipment service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("shipment service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("shipment service: tx manager is required")
	}
	return nil
}

func NewShipmentSvc(config *ShipmentSvcConfig) domain.ShipmentSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shipmentSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
		shippoFactory:   config.ShippoFactory,
		notificationPub: config.NotificationPub,
	}
}

func (s *shipmentSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *shipmentSvcImpl) withTx(ctx context.Context, fn func(context.Context, *shipmentSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &shipmentSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
			shippoFactory:   s.shippoFactory,
			notificationPub: s.notificationPub,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *shipmentSvcImpl) ListShipments(ctx context.Context, params domain.ListShipmentsParams) (*domain.ListShipmentsResult, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewShipmentRepo().List(ctx, params)
}

func (s *shipmentSvcImpl) GetShipment(ctx context.Context, params domain.GetShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	shipment, apiErr := s.repos.NewShipmentRepo().Get(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Load includes
	for _, inc := range params.Includes {
		switch inc {
		case "lines":
			lines, apiErr := s.repos.NewShipmentLineRepo().ListByShipment(ctx, params.ShipmentID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			shipment.Lines = lines
		case "shipping_cases":
			cases, apiErr := s.repos.NewShippingCaseRepo().ListByShipment(ctx, params.ShipmentID)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			shipment.ShippingCases = cases
		}
	}

	return shipment, nil
}

func (s *shipmentSvcImpl) UpdateShipment(ctx context.Context, params domain.UpdateShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Shipment](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Shipment
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewShipmentRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}

			// Backfill unchanged nullable fields with existing values.
			// Since the SQL uses direct assignment (no COALESCE) for these fields,
			// we must provide the existing value when the field was not sent.
			if params.ServiceLevelID == nil {
				params.ServiceLevelID = old.ServiceLevelID
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if slices.Contains(params.Includes, "lines") {
				lines, apiErr := txSvc.repos.NewShipmentLineRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.Lines = lines
			}
			if slices.Contains(params.Includes, "shipping_cases") {
				cases, apiErr := txSvc.repos.NewShippingCaseRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.ShippingCases = cases
			}

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeShipment,
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

func (s *shipmentSvcImpl) DeleteShipment(ctx context.Context, params domain.DeleteShipmentParams) *apierror.APIError {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	shipmentRepo := s.repos.NewShipmentRepo()

	shipment, apiErr := shipmentRepo.Get(ctx, domain.GetShipmentParams{
		AccountID:  params.AccountID,
		ShipmentID: params.ShipmentID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeShipment, params.ShipmentID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(
					span,
					apierror.NewAlreadyDeletedError("This shipment has already been deleted and can no longer be modified."),
				)
			}
		}
		return tracing.Trace(span, apiErr)
	}

	return s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
		// Unpack pick lines associated with this shipment's lines (clear packed_at)
		if apiErr := txSvc.repos.NewPickLineRepo().UnpackByShipment(txCtx, params.ShipmentID); apiErr != nil {
			return apiErr
		}
		// Find the pick for this shipment's order and mark it as unpacked (clear finished_at)
		pickID, apiErr := txSvc.repos.NewPickRepo().FindIDByShipmentOrder(txCtx, params.AccountID, params.ShipmentID)
		if apiErr != nil {
			return apiErr
		}
		if pickID != "" {
			if apiErr := txSvc.repos.NewPickRepo().ClearFinishedAt(txCtx, params.AccountID, pickID); apiErr != nil {
				return apiErr
			}
		}

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeShipment, shipment.ID, shipment); apiErr != nil {
			return apiErr
		}

		// Delete shipping cases first
		if apiErr := txSvc.repos.NewShippingCaseRepo().DeleteByShipment(txCtx, params.ShipmentID); apiErr != nil {
			return apiErr
		}
		// Delete shipment lines
		if apiErr := txSvc.repos.NewShipmentLineRepo().DeleteByShipment(txCtx, params.ShipmentID); apiErr != nil {
			return apiErr
		}
		// Delete shipment
		if apiErr := txSvc.repos.NewShipmentRepo().Delete(txCtx, params.AccountID, params.ShipmentID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(shipment, (*domain.Shipment)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeShipment,
			ResourceID:   shipment.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})
}

func (s *shipmentSvcImpl) ShipShipment(ctx context.Context, params domain.ShipShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.ship")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Shipment](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Phase 1: Validate shipment and gather data
		shipmentRepo := s.repos.NewShipmentRepo()
		shipment, apiErr := shipmentRepo.Get(ctx, domain.GetShipmentParams{
			AccountID:  params.AccountID,
			ShipmentID: params.ShipmentID,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if shipment.StatusCode == "shipped" {
			return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("Shipment has already been shipped.", "id"))
		}

		// TODO: Phase 2 - Create shipping labels via Shippo (foreign mutation)
		// This would involve:
		// 1. Getting the account's Shippo API key from account_integration
		// 2. Creating instant labels via Shippo
		// 3. Uploading labels to S3
		// 4. Updating shipping cases with tracking info
		// After labels are created, advance recovery point to RecoveryPointShipLabelsCreated

		// Phase 3: Atomic transaction - mark shipped, create invoice, add SSCC
		var result *domain.Shipment
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
			txShipmentRepo := txSvc.repos.NewShipmentRepo()
			txCaseRepo := txSvc.repos.NewShippingCaseRepo()

			// Mark shipping cases as shipped
			if apiErr := txCaseRepo.MarkShippedByShipment(txCtx, params.ShipmentID); apiErr != nil {
				return apiErr
			}

			// Add SSCC to shipping cases
			cases, apiErr := txCaseRepo.ListByShipment(txCtx, params.ShipmentID)
			if apiErr != nil {
				return apiErr
			}
			for _, sc := range cases {
				if sc.SSCC == nil {
					counter, apiErr := txCaseRepo.FindAndIncrementSsccCounter(txCtx, params.AccountID)
					if apiErr != nil {
						return apiErr
					}
					sscc := domain.GenerateSSCC(counter)
					if apiErr := txCaseRepo.AddSscc(txCtx, sc.ID, sscc); apiErr != nil {
						return apiErr
					}
				}
			}

			// Mark shipment as shipped
			shippedByID := ""
			if identity.Actor != nil {
				shippedByID = identity.Actor.ID
			}
			if apiErr := txShipmentRepo.MarkShipped(txCtx, params.AccountID, params.ShipmentID, shippedByID); apiErr != nil {
				return apiErr
			}

			// Re-fetch for response
			updated, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if slices.Contains(params.Includes, "lines") {
				lines, apiErr := txSvc.repos.NewShipmentLineRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.Lines = lines
			}
			if slices.Contains(params.Includes, "shipping_cases") {
				cases, apiErr := txSvc.repos.NewShippingCaseRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.ShippingCases = cases
			}

			changes := audit.ComputeChanges(shipment, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeShipment,
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

	case domain.RecoveryPointShipLabelsCreated:
		// Labels were already created in a prior attempt. Proceed with atomic phase.
		// Fetch old state for audit diff
		old, apiErr := s.repos.NewShipmentRepo().Get(ctx, domain.GetShipmentParams{
			AccountID:  params.AccountID,
			ShipmentID: params.ShipmentID,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.Shipment
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
			txShipmentRepo := txSvc.repos.NewShipmentRepo()
			txCaseRepo := txSvc.repos.NewShippingCaseRepo()

			if apiErr := txCaseRepo.MarkShippedByShipment(txCtx, params.ShipmentID); apiErr != nil {
				return apiErr
			}

			cases, apiErr := txCaseRepo.ListByShipment(txCtx, params.ShipmentID)
			if apiErr != nil {
				return apiErr
			}
			for _, sc := range cases {
				if sc.SSCC == nil {
					counter, apiErr := txCaseRepo.FindAndIncrementSsccCounter(txCtx, params.AccountID)
					if apiErr != nil {
						return apiErr
					}
					sscc := domain.GenerateSSCC(counter)
					if apiErr := txCaseRepo.AddSscc(txCtx, sc.ID, sscc); apiErr != nil {
						return apiErr
					}
				}
			}

			shippedByID := ""
			if identity.Actor != nil {
				shippedByID = identity.Actor.ID
			}
			if apiErr := txShipmentRepo.MarkShipped(txCtx, params.AccountID, params.ShipmentID, shippedByID); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = updated

			if slices.Contains(params.Includes, "lines") {
				lines, apiErr := txSvc.repos.NewShipmentLineRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.Lines = lines
			}
			if slices.Contains(params.Includes, "shipping_cases") {
				cases, apiErr := txSvc.repos.NewShippingCaseRepo().ListByShipment(txCtx, params.ShipmentID)
				if apiErr != nil {
					return apiErr
				}
				result.ShippingCases = cases
			}

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeShipment,
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

func (s *shipmentSvcImpl) VoidShipment(ctx context.Context, params domain.VoidShipmentParams) (*domain.Shipment, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.void")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Shipment](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		// Phase 1: Validate shipment
		shipmentRepo := s.repos.NewShipmentRepo()
		shipment, apiErr := shipmentRepo.Get(ctx, domain.GetShipmentParams{
			AccountID:  params.AccountID,
			ShipmentID: params.ShipmentID,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if shipment.StatusCode != "shipped" {
			return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("Shipment is not in shipped status.", "id"))
		}

		// TODO: Phase 2 - Refund Shippo transactions (foreign mutation)
		// This would involve:
		// 1. Getting the shipping cases with Shippo transaction IDs
		// 2. Refunding each transaction via Shippo (skip in sandbox mode)
		// 3. Deleting labels from S3
		// After refunds complete, advance recovery point to RecoveryPointVoidLabelsRefunded

		// Phase 3: Atomic transaction - void cases, delete invoice, mark order unfulfilled, mark shipment packed
		fallthrough

	case domain.RecoveryPointVoidLabelsRefunded:
		var result *domain.Shipment
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentSvcImpl) *apierror.APIError {
			txShipmentRepo := txSvc.repos.NewShipmentRepo()
			txCaseRepo := txSvc.repos.NewShippingCaseRepo()
			txInvoiceRepo := txSvc.repos.NewInvoiceRepo()
			txSalesOrderRepo := txSvc.repos.NewSalesOrderRepo()

			// Look up the shipment to get the sales order ID for unfulfillment
			shipment, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}

			// Delete invoice if one exists for this shipment
			invoiceID, apiErr := txShipmentRepo.FindInvoiceIDByShipment(txCtx, params.AccountID, params.ShipmentID)
			if apiErr != nil {
				return apiErr
			}
			if invoiceID != nil {
				// TODO: Reverse inventory allocations by invoice (LIFO reversal).
				// The Dashboard performs a complex LIFO reversal of inventory allocations
				// for each invoice line's item. This involves:
				// 1. For each sale-type invoice line, find allocations for the order+item (newest first)
				// 2. Delete or reduce allocations, updating receipt/issue statuses
				// 3. Create new reserved issues for any unallocated quantity
				// 4. Create inventory change log entries
				// 5. Reallocate remaining open issues per item using FIFO
				// This needs to be implemented as a dedicated mediator or repository method.

				// Delete invoice lines then invoice
				if apiErr := txInvoiceRepo.DeleteLinesByInvoice(txCtx, *invoiceID); apiErr != nil {
					return apiErr
				}
				if apiErr := txInvoiceRepo.Delete(txCtx, params.AccountID, *invoiceID); apiErr != nil {
					return apiErr
				}
			}

			// Mark the sales order as unfulfilled (reset to "issued" status, clear completed_at and first_ship_at)
			if apiErr := txSalesOrderRepo.MarkUnfulfilled(txCtx, params.AccountID, shipment.SalesOrderID); apiErr != nil {
				return apiErr
			}

			// Void shipping cases (clear tracking, labels, freight amount)
			if apiErr := txCaseRepo.VoidByShipment(txCtx, params.ShipmentID); apiErr != nil {
				return apiErr
			}

			// Mark shipment as voided (back to packed, clear tracking/invoice/shipped info)
			if apiErr := txShipmentRepo.MarkVoided(txCtx, params.AccountID, params.ShipmentID); apiErr != nil {
				return apiErr
			}

			// Re-fetch for response
			updated, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{
				AccountID:  params.AccountID,
				ShipmentID: params.ShipmentID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(shipment, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeShipment,
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

func (s *shipmentSvcImpl) EstimateRate(ctx context.Context, params domain.EstimateRateParams) (float64, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.estimate_rate")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return 0, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionRead); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
	} else if identity.IsCustomerUser() {
		if params.CustomerID == nil || identity.ActorAccountID() == nil || *identity.ActorAccountID() != *params.CustomerID {
			return 0, tracing.Trace(span, apierror.NewAuthorizationError("You are not authorized to access this resource."))
		}
	} else {
		return 0, tracing.Trace(span, apierror.NewValidationError("Invalid actor type."))
	}

	if !identity.IsTargetAccountSet() {
		return 0, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	// Check product line freight exemption: if any product line is freight exempt, rate is 0.
	if len(params.ProductLineIDs) > 0 {
		productLineRepo := s.repos.NewProductLineRepo()
		for _, plID := range params.ProductLineIDs {
			pl, apiErr := productLineRepo.Get(ctx, domain.GetProductLineParams{
				AccountID:     params.AccountID,
				ProductLineID: plID,
			})
			if apiErr != nil {
				continue
			}
			if pl.FreightPolicy == constants.FreightPolicyFree {
				return 0, nil
			}
		}
	}

	// Check customer-level freight exemptions and shipping term logic.
	if params.CustomerID != nil {
		customerRepo := s.repos.NewCustomerRepo()
		customer, apiErr := customerRepo.Get(ctx, params.AccountID, *params.CustomerID, nil)
		if apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}

		// Customer or customer group is freight exempt.
		if customer.FreightPolicy == constants.FreightPolicyFree {
			return 0, nil
		}

		// Check the customer's default shipping term.
		if customer.DefaultShippingTermID != nil {
			shippingTermRepo := s.repos.NewShippingTermRepo()
			shippingTerm, apiErr := shippingTermRepo.Get(ctx, domain.GetShippingTermParams{
				AccountID:      params.AccountID,
				ShippingTermID: *customer.DefaultShippingTermID,
			})
			if apiErr != nil {
				return 0, tracing.Trace(span, apiErr)
			}

			// Shipping term is free freight.
			if shippingTerm.Type == constants.ShippingTermTypeFreeFreight {
				return 0, nil
			}

			// Shipping term has a flat rate.
			if shippingTerm.Type == constants.ShippingTermTypeFlatRateFreight && shippingTerm.FlatRate != nil {
				flatRate, err := strconv.ParseFloat(shippingTerm.FlatRate.Value, 64)
				if err != nil {
					return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse flat rate value."))
				}
				return flatRate, nil
			}

			// Shipping term has a minimum order value: free shipping if order exceeds threshold.
			if shippingTerm.MinimumOrderValue != nil && params.OrderTotal != nil {
				minValue, err := strconv.ParseFloat(shippingTerm.MinimumOrderValue.Value, 64)
				if err != nil {
					return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to parse minimum order value."))
				}
				if *params.OrderTotal > minValue {
					return 0, nil
				}
			}
		}
	}

	// Get carrier to find Shippo carrier account object ID.
	carrierRepo := s.repos.NewCarrierRepo()
	carrier, apiErr := carrierRepo.Get(ctx, domain.GetCarrierParams{AccountID: params.AccountID, CarrierID: params.CarrierID})
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	// If carrier doesn't have Shippo configured, return 0 (no rate available).
	if carrier.ShippoCarrierAccountID == nil || *carrier.ShippoCarrierAccountID == "" {
		return 0, nil
	}

	// Get service level token (optional).
	var serviceLevelToken string
	if params.ServiceLevelID != "" {
		serviceLevelRepo := s.repos.NewServiceLevelRepo()
		serviceLevel, apiErr := serviceLevelRepo.Get(ctx, params.AccountID, params.ServiceLevelID)
		if apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
		if serviceLevel.ServiceLevelToken != nil {
			serviceLevelToken = *serviceLevel.ServiceLevelToken
		}
	}

	// Check if account has Shippo integration enabled.
	integrationRepo := s.repos.NewAccountIntegrationRepo()
	hasIntegration, apiErr := integrationRepo.HasIntegration(ctx, params.AccountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	if !hasIntegration {
		return 0, nil
	}

	// Get account Shippo API key.
	encryptedCreds, _, apiErr := integrationRepo.GetEncryptedCredentials(ctx, params.AccountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	shippoClient := s.shippoFactory.Build(encryptedCreds)

	rate, apiErr := shippoClient.FetchShippingRate(ctx, domain.FetchShippingRateParams{
		CarrierAccountObjectID: *carrier.ShippoCarrierAccountID,
		ServiceLevelToken:      serviceLevelToken,
		FromAddress:            params.FromAddress,
		ToAddress:              params.ToAddress,
		Parcels:                params.Parcels,
	})
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return rate, nil
}

func (s *shipmentSvcImpl) RateShop(ctx context.Context, params domain.RateShopParams) (*domain.RateShopResult, *apierror.APIError) {
	ctx, span := shipmentSvcTracer.Start(ctx, "service.shipment.rate_shop")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.IsInternalActor() {
		if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionRead); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if identity.IsCustomerUser() {
		if params.CustomerID == nil || identity.ActorAccountID() == nil || *identity.ActorAccountID() != *params.CustomerID {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("You are not authorized to access this resource."))
		}
	} else {
		return nil, tracing.Trace(span, apierror.NewValidationError("Invalid actor type."))
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID
	freightExemptResult := &domain.RateShopResult{
		Options:       []*domain.RateShopOption{},
		ExemptionType: new("freight_exempt"),
	}

	// 1. Check product line freight exemption: if any product line is freight exempt, return empty.
	if len(params.ProductLineIDs) > 0 {
		productLineRepo := s.repos.NewProductLineRepo()
		for _, plID := range params.ProductLineIDs {
			pl, apiErr := productLineRepo.Get(ctx, domain.GetProductLineParams{
				AccountID:     params.AccountID,
				ProductLineID: plID,
			})
			if apiErr != nil {
				continue
			}
			if pl.FreightPolicy == constants.FreightPolicyFree {
				return freightExemptResult, nil
			}
		}
	}

	// 2. Fetch customer and check customer/group freight exemption.
	var customer *domain.Customer
	var shippingTerm *domain.ShippingTerm
	if params.CustomerID != nil {
		customerRepo := s.repos.NewCustomerRepo()
		var apiErr *apierror.APIError
		customer, apiErr = customerRepo.Get(ctx, params.AccountID, *params.CustomerID, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Customer or customer group is freight exempt.
		if customer.FreightPolicy == constants.FreightPolicyFree {
			return freightExemptResult, nil
		}

		// 3. Check shipping term freight exemption.
		if customer.DefaultShippingTermID != nil {
			shippingTermRepo := s.repos.NewShippingTermRepo()
			shippingTerm, apiErr = shippingTermRepo.Get(ctx, domain.GetShippingTermParams{
				AccountID:      params.AccountID,
				ShippingTermID: *customer.DefaultShippingTermID,
			})
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			if shippingTerm.Type == constants.ShippingTermTypeFreeFreight {
				return freightExemptResult, nil
			}
		}
	}

	// Extract shipping term configuration.
	var flatRateValue *float64
	var minimumOrderValue *float64
	freeShippingOptionIDs := make(map[string]bool)

	if shippingTerm != nil {
		if shippingTerm.FlatRate != nil {
			v, err := strconv.ParseFloat(shippingTerm.FlatRate.Value, 64)
			if err == nil {
				flatRateValue = &v
			}
		}
		if shippingTerm.MinimumOrderValue != nil {
			v, err := strconv.ParseFloat(shippingTerm.MinimumOrderValue.Value, 64)
			if err == nil {
				minimumOrderValue = &v
			}
		}
		for _, optID := range shippingTerm.FreeShippingServiceLevelIDs {
			freeShippingOptionIDs[optID] = true
		}
	}

	hasFlatRate := flatRateValue != nil
	hasMinimumOrder := minimumOrderValue != nil
	isMinimumOrderMet := hasMinimumOrder && params.OrderTotal != nil && *params.OrderTotal > *minimumOrderValue
	hasFreeShippingRules := len(freeShippingOptionIDs) > 0

	// 4. List all carriers for the account.
	carrierRepo := s.repos.NewCarrierRepo()
	carriersResult, apiErr := carrierRepo.List(ctx, domain.ListCarriersParams{
		AccountID: params.AccountID,
		Limit:     1000,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Load options for each carrier.
	for _, carrier := range carriersResult.Carriers {
		options, apiErr := carrierRepo.ListOptionsByCarrierID(ctx, params.AccountID, carrier.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		carrier.ServiceLevels = options
	}

	// Filter for portal-enabled carriers/options when called by customer actor.
	carriers := carriersResult.Carriers
	if identity.IsCustomerUser() {
		var filtered []*domain.Carrier
		for _, c := range carriers {
			if !c.IsPortalEnabled {
				continue
			}
			var portalOptions []*domain.ServiceLevel
			for _, o := range c.ServiceLevels {
				if o.IsPortalEnabled {
					portalOptions = append(portalOptions, o)
				}
			}
			carrierCopy := *c
			carrierCopy.ServiceLevels = portalOptions
			filtered = append(filtered, &carrierCopy)
		}
		carriers = filtered
	}

	// 5. Check if account has Shippo integration.
	integrationRepo := s.repos.NewAccountIntegrationRepo()
	hasShippoIntegration, apiErr := integrationRepo.HasIntegration(ctx, params.AccountID, constants.IntegrationCodeShippo)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var shippoClient domain.ShippoClient
	if hasShippoIntegration {
		encryptedCreds, _, apiErr := integrationRepo.GetEncryptedCredentials(ctx, params.AccountID, constants.IntegrationCodeShippo)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		shippoClient = s.shippoFactory.Build(encryptedCreds)
	}

	// 6. For each carrier, fetch rates.
	var allOptions []*domain.RateShopOption
	for _, carrier := range carriers {
		if carrier.ShippoCarrierAccountID == nil || *carrier.ShippoCarrierAccountID == "" || shippoClient == nil {
			// Non-Shippo carrier: include each option with rate 0.
			for _, opt := range carrier.ServiceLevels {
				allOptions = append(allOptions, &domain.RateShopOption{
					CarrierID:        carrier.ID,
					CarrierName:      carrier.Name,
					ServiceLevelID:   opt.ID,
					ServiceLevelName: opt.Name,
					Rate:             0,
				})
			}
			continue
		}

		// Fetch all rates from Shippo for this carrier.
		shippoRates, apiErr := shippoClient.FetchAllShippingRates(ctx, domain.FetchAllShippingRatesParams{
			CarrierAccountObjectID: *carrier.ShippoCarrierAccountID,
			FromAddress:            params.FromAddress,
			ToAddress:              params.ToAddress,
			Parcels:                params.Parcels,
		})
		if apiErr != nil {
			// Skip carriers that fail to fetch rates.
			continue
		}

		// Map Shippo rates to carrier options by matching service level token.
		for _, shippoRate := range shippoRates {
			for _, opt := range carrier.ServiceLevels {
				if opt.ServiceLevelToken != nil && *opt.ServiceLevelToken == shippoRate.ServiceLevelToken {
					allOptions = append(allOptions, &domain.RateShopOption{
						CarrierID:        carrier.ID,
						CarrierName:      carrier.Name,
						ServiceLevelID:   opt.ID,
						ServiceLevelName: opt.Name,
						Rate:             shippoRate.Amount,
						EstimatedDays:    shippoRate.EstimatedDays,
					})
					break
				}
			}
		}
	}

	// 7. Post-process rates: apply flat rate, minimum order, and free shipping rules.
	for _, opt := range allOptions {
		isEligibleForFreeShipping := true
		if hasFreeShippingRules {
			isEligibleForFreeShipping = freeShippingOptionIDs[opt.ServiceLevelID]
		}

		if isMinimumOrderMet && isEligibleForFreeShipping {
			opt.Rate = 0
		} else if hasFlatRate {
			opt.Rate = *flatRateValue
		}
	}

	// 8. Sort by rate ascending.
	sort.Slice(allOptions, func(i, j int) bool {
		return allOptions[i].Rate < allOptions[j].Rate
	})

	// 9. Determine exemption type.
	var exemptionType *string
	if isMinimumOrderMet {
		exemptionType = new("minimum_order_met")
	} else if hasFlatRate {
		exemptionType = new("flat_rate")
	} else {
		exemptionType = new("none")
	}

	result := &domain.RateShopResult{
		Options:       allOptions,
		ExemptionType: exemptionType,
	}
	if hasFlatRate {
		result.FlatRate = flatRateValue
	}

	return result, nil
}
