package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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

var pickSvcTracer = tracing.GetTracer("core-service.pick_service")

type pickSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type PickSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
	TxManager       TransactionManager
}

func (c *PickSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("pick service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("pick service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("pick service: tx manager is required")
	}
	return nil
}

func NewPickSvc(config *PickSvcConfig) domain.PickSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &pickSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *pickSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *pickSvcImpl) withTx(ctx context.Context, fn func(context.Context, *pickSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &pickSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *pickSvcImpl) ListPicks(ctx context.Context, params domain.ListPicksParams) (*domain.ListPicksResult, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkPickReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewPickRepo()
	result, apiErr := repo.List(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Expand departments per pick only when requested (so the list can serve the
	// department_ids array filter and rows that render department pills).
	for _, include := range params.Includes {
		if include == "departments" {
			for _, pick := range result.Picks {
				depts, apiErr := repo.GetDepartments(ctx, pick.ID)
				if apiErr != nil {
					return nil, tracing.Trace(span, apiErr)
				}
				pick.Departments = depts
			}
			break
		}
	}

	return result, nil
}

func (s *pickSvcImpl) GetPick(ctx context.Context, pickID string, includes []string) (*domain.Pick, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	repo := s.repos.NewPickRepo()

	pick, apiErr := repo.Get(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Departments are always returned (matches Dashboard behavior).
	departments, apiErr := repo.GetDepartments(ctx, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	pick.Departments = departments

	if includesPickLines(includes) {
		lines, apiErr := repo.GetLines(ctx, pickID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		pick.Lines = lines
	}

	return pick, nil
}

func (s *pickSvcImpl) UpdatePick(ctx context.Context, params domain.UpdatePickParams) (*domain.Pick, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Pick](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Pick
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *pickSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPickRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.PickID)
			if apiErr != nil {
				return apiErr
			}

			if params.Number != nil {
				if apiErr := txRepo.UpdateNumber(txCtx, params.AccountID, params.PickID, *params.Number); apiErr != nil {
					return apiErr
				}
			}

			if params.FinishedAt != nil {
				if *params.FinishedAt == nil {
					if apiErr := txRepo.ClearFinishedAt(txCtx, params.AccountID, params.PickID); apiErr != nil {
						return apiErr
					}
				} else {
					if apiErr := txRepo.UpdateFinishedAt(txCtx, params.AccountID, params.PickID, **params.FinishedAt); apiErr != nil {
						return apiErr
					}
				}
			}

			pick, apiErr := txRepo.Get(txCtx, params.AccountID, params.PickID)
			if apiErr != nil {
				return apiErr
			}
			result = pick

			if includesPickLines(params.Includes) {
				lines, apiErr := txRepo.GetLines(txCtx, params.PickID)
				if apiErr != nil {
					return apiErr
				}
				result.Lines = lines
			}

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePick,
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

func (s *pickSvcImpl) PickAllLines(ctx context.Context, pickID string) (*domain.Pick, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.pick_all_lines")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	var result *domain.Pick
	apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *pickSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewPickRepo()

		old, apiErr := txRepo.Get(txCtx, accountID, pickID)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.PickAllLines(txCtx, pickID); apiErr != nil {
			return apiErr
		}

		pick, apiErr := txRepo.Get(txCtx, accountID, pickID)
		if apiErr != nil {
			return apiErr
		}

		lines, apiErr := txRepo.GetLines(txCtx, pickID)
		if apiErr != nil {
			return apiErr
		}
		pick.Lines = lines

		departments, apiErr := txRepo.GetDepartments(txCtx, pickID)
		if apiErr != nil {
			return apiErr
		}
		pick.Departments = departments

		result = pick

		changes := audit.ComputeChanges(old, result)

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypePick,
			ResourceID:   result.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *pickSvcImpl) VoidPick(ctx context.Context, pickID string) (*domain.Pick, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.void")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	repo := s.repos.NewPickRepo()

	hasShipped, apiErr := repo.HasShippedItems(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if hasShipped {
		return nil, tracing.Trace(span, apierror.NewValidationError("Cannot void a pick with shipped items."))
	}

	old, apiErr := repo.Get(ctx, accountID, pickID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var result *domain.Pick
	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *pickSvcImpl) *apierror.APIError {
		txRepo := txSvc.repos.NewPickRepo()

		if apiErr := txRepo.VoidAllLines(txCtx, pickID); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.DeleteDuplicatePickLines(txCtx, accountID, pickID); apiErr != nil {
			return apiErr
		}

		if apiErr := txRepo.ClearFinishedAt(txCtx, accountID, pickID); apiErr != nil {
			return apiErr
		}

		pick, apiErr := txRepo.Get(txCtx, accountID, pickID)
		if apiErr != nil {
			return apiErr
		}

		lines, apiErr := txRepo.GetLines(txCtx, pickID)
		if apiErr != nil {
			return apiErr
		}
		pick.Lines = lines

		result = pick

		changes := audit.ComputeChanges(old, result)

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypePick,
			ResourceID:   result.ID,
			Changes:      changes,
		}); apiErr != nil {
			return apiErr
		}

		return nil
	})

	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return result, nil
}

func (s *pickSvcImpl) PackPick(ctx context.Context, pickID string, shipmentCaseCount int32) (*domain.PackPickResult, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.pack")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.PackPickResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.PackPickResult
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *pickSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPickRepo()
			pickLineRepo := txSvc.repos.NewPickLineRepo()

			// Find lines eligible for packing
			linesToPack, apiErr := txRepo.FindLinesToPack(txCtx, pickID)
			if apiErr != nil {
				return apiErr
			}
			if len(linesToPack) == 0 {
				return apierror.NewValidationError("No lines to pack.")
			}

			// Mark lines as packed
			if apiErr := txRepo.PackLines(txCtx, pickID); apiErr != nil {
				return apiErr
			}

			// For each packed line's order line (deduplicated), calculate remaining and create new pick lines if needed
			processedOrderLines := make(map[string]bool)
			for _, line := range linesToPack {
				if processedOrderLines[line.SalesOrderLineID] {
					continue
				}
				processedOrderLines[line.SalesOrderLineID] = true

				remainingValue, unitID, apiErr := pickLineRepo.CalculateRemainingForOrderLine(txCtx, line.SalesOrderLineID)
				if apiErr != nil {
					return apiErr
				}

				remainingFloat, _ := strconv.ParseFloat(remainingValue, 64)
				if remainingFloat > 0 {
					// Only create a remaining pick line if there isn't already an unpacked one for this order line
					hasUnpacked, apiErr := pickLineRepo.HasUnpackedPickLineForOrderLine(txCtx, line.SalesOrderLineID)
					if apiErr != nil {
						return apiErr
					}
					if hasUnpacked {
						continue
					}

					newPickLineID, apiErr := id.GenID(id.PickLineIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					newQuantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					// Create with quantity 0 — remaining lines are placeholders that need explicit quantity assignment
					if apiErr := txRepo.CreateQuantity(txCtx, newQuantityID, "0", unitID); apiErr != nil {
						return apiErr
					}
					if apiErr := pickLineRepo.CreateForRemaining(txCtx, newPickLineID, newQuantityID, pickID, line.SalesOrderLineID); apiErr != nil {
						return apiErr
					}
				}
			}

			// Get sales order info for shipment creation
			salesOrder, apiErr := txRepo.GetSalesOrderForPick(txCtx, accountID, pickID)
			if apiErr != nil {
				return apiErr
			}

			// Count existing shipments to determine shipment number
			count, apiErr := txRepo.CountShipmentsByOrder(txCtx, salesOrder.ID)
			if apiErr != nil {
				return apiErr
			}

			var shipmentNumber string
			if count == 0 {
				shipmentNumber = salesOrder.Number
			} else {
				shipmentNumber = fmt.Sprintf("%s-%d", salesOrder.Number, count+1)
			}

			// Generate shipment ID
			shipmentID, apiErr := id.GenID(id.ShipmentIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}

			// Create shipment
			if apiErr := txRepo.CreateShipment(txCtx, domain.CreateShipmentFromPickParams{
				ID:                shipmentID,
				Number:            shipmentNumber,
				SalesOrderID:      salesOrder.ID,
				CarrierID:         salesOrder.CarrierID,
				ServiceLevelID:    salesOrder.ServiceLevelID,
				ShippingAddressID: salesOrder.ShippingAddressID,
				StatusCode:        "packed",
				AccountID:         accountID,
			}); apiErr != nil {
				return apiErr
			}

			// Create shipment lines for each packed line
			for _, line := range linesToPack {
				shipmentLineID, apiErr := id.GenID(id.ShipmentLineIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				shipmentLineQtyID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := txRepo.CreateQuantity(txCtx, shipmentLineQtyID, line.QuantityValue, line.QuantityUnitID); apiErr != nil {
					return apiErr
				}
				if apiErr := txRepo.CreateShipmentLine(txCtx, domain.CreateShipmentLineParams{
					ID:               shipmentLineID,
					ShipmentID:       shipmentID,
					SalesOrderLineID: line.SalesOrderLineID,
					QuantityID:       shipmentLineQtyID,
				}); apiErr != nil {
					return apiErr
				}
			}

			// Create shipping cases
			for i := 0; i < int(shipmentCaseCount); i++ {
				caseID, apiErr := id.GenID(id.ShippingCaseIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				caseNumber := fmt.Sprintf("%s-%d", shipmentNumber, i+1)

				freightAmountID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				freightWeightID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}

				if apiErr := txRepo.CreateQuantity(txCtx, freightAmountID, "0", "dollar"); apiErr != nil {
					return apiErr
				}
				if apiErr := txRepo.CreateQuantity(txCtx, freightWeightID, "0", "lb"); apiErr != nil {
					return apiErr
				}

				if apiErr := txRepo.CreateShippingCase(txCtx, domain.CreateShippingCaseParams{
					ID:              caseID,
					Number:          caseNumber,
					FreightAmountID: freightAmountID,
					FreightWeightID: freightWeightID,
					ShipmentID:      shipmentID,
					CarrierID:       salesOrder.CarrierID,
					AccountID:       accountID,
				}); apiErr != nil {
					return apiErr
				}
			}

			// Mark pick as finished if all lines are now packed
			if apiErr := txRepo.MarkFinishedIfAllPacked(txCtx, pickID); apiErr != nil {
				return apiErr
			}

			// Get final pick state
			pick, apiErr := txRepo.Get(txCtx, accountID, pickID)
			if apiErr != nil {
				return apiErr
			}

			lines, apiErr := txRepo.GetLines(txCtx, pickID)
			if apiErr != nil {
				return apiErr
			}
			pick.Lines = lines

			result = &domain.PackPickResult{
				Pick:           pick,
				ShipmentNumber: shipmentNumber,
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePick,
				ResourceID:   pick.ID,
				Changes:      nil,
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

// checkPickReadPermission checks the appropriate read permission based on the identity context.
// Internal actors need picks:read for their own account, or customers:read / suppliers:read for external accounts.
func checkPickReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionRead)
}

func (s *pickSvcImpl) GetPickShipments(ctx context.Context, params domain.GetPickShipmentsParams) (*domain.PickShipmentsResult, *apierror.APIError) {
	ctx, span := pickSvcTracer.Start(ctx, "service.pick.get_shipments")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPicks, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewPickRepo().GetShipmentNumbers(ctx, params)
}

func includesPickLines(includes []string) bool {
	for _, inc := range includes {
		if inc == "lines" || strings.HasPrefix(inc, "lines.") {
			return true
		}
	}
	return false
}
