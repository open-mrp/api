package service

import (
	"context"
	"fmt"

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

var shipmentLineSvcTracer = tracing.GetTracer("core-service.shipment_line_service")

type shipmentLineSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ShipmentLineSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ShipmentLineSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("shipment line service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("shipment line service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("shipment line service: tx manager is required")
	}
	return nil
}

func NewShipmentLineSvc(config *ShipmentLineSvcConfig) domain.ShipmentLineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shipmentLineSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *shipmentLineSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *shipmentLineSvcImpl) withTx(ctx context.Context, fn func(context.Context, *shipmentLineSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &shipmentLineSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *shipmentLineSvcImpl) ListShipmentLines(ctx context.Context, params domain.ListShipmentLinesParams) (*domain.ListShipmentLinesResult, *apierror.APIError) {
	ctx, span := shipmentLineSvcTracer.Start(ctx, "service.shipment_line.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkShipmentLineReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		// Counterparty-aware: a customer-portal relation actor may read the shipment lines
		// of their own order. Data stays scoped to Target.AccountID; the owner-side
		// CheckReadAccess only allows the actor->target direction and wrongly rejects them.
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	// Verify shipment is in account
	shipmentRepo := s.repos.NewShipmentRepo()
	inAccount, apiErr := shipmentRepo.IsInAccount(ctx, params.AccountID, params.ShipmentID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Shipment not found."))
	}

	return s.repos.NewShipmentLineRepo().List(ctx, params)
}

func (s *shipmentLineSvcImpl) GetShipmentLine(ctx context.Context, accountID, shipmentID, shipmentLineID string) (*domain.ShipmentLine, *apierror.APIError) {
	ctx, span := shipmentLineSvcTracer.Start(ctx, "service.shipment_line.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkShipmentLineReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		// Counterparty-aware: a customer-portal relation actor may read the shipment lines
		// of their own order. Data stays scoped to Target.AccountID; the owner-side
		// CheckReadAccess only allows the actor->target direction and wrongly rejects them.
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	acctID := identity.Target.AccountID

	shipmentRepo := s.repos.NewShipmentRepo()
	inAccount, apiErr := shipmentRepo.IsInAccount(ctx, acctID, shipmentID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Shipment not found."))
	}

	lineRepo := s.repos.NewShipmentLineRepo()
	inShipment, apiErr := lineRepo.IsInShipment(ctx, shipmentLineID, shipmentID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inShipment {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Shipment line not found in this shipment."))
	}

	return lineRepo.Get(ctx, shipmentLineID)
}

func (s *shipmentLineSvcImpl) CreateShipmentLine(ctx context.Context, params domain.CreateShipmentLineEndpointParams) (*domain.ShipmentLine, *apierror.APIError) {
	ctx, span := shipmentLineSvcTracer.Start(ctx, "service.shipment_line.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionCreate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ShipmentLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		lineID, apiErr := id.GenID(id.ShipmentLineIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		var result *domain.ShipmentLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentLineSvcImpl) *apierror.APIError {
			txShipmentRepo := txSvc.repos.NewShipmentRepo()

			// Validate shipment is in account
			inAccount, apiErr := txShipmentRepo.IsInAccount(txCtx, params.AccountID, params.ShipmentID)
			if apiErr != nil {
				return apiErr
			}
			if !inAccount {
				return apierror.NewResourceNotFoundError("Shipment not found.")
			}

			// Resolve the parent sales order so the audit event can be scoped to the order's history tree.
			rootShipment, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{AccountID: params.AccountID, ShipmentID: params.ShipmentID})
			if apiErr != nil {
				return apiErr
			}

			txLineRepo := txSvc.repos.NewShipmentLineRepo()

			created, apiErr := txLineRepo.Create(txCtx, lineID, quantityID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionCreate,
				ResourceType:     constants.ObjectTypeShipmentLine,
				ResourceID:       created.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   rootShipment.SalesOrderID,
				Changes:          changes,
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

func (s *shipmentLineSvcImpl) UpdateShipmentLine(ctx context.Context, params domain.UpdateShipmentLineEndpointParams) (*domain.ShipmentLine, *apierror.APIError) {
	ctx, span := shipmentLineSvcTracer.Start(ctx, "service.shipment_line.update")
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ShipmentLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ShipmentLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentLineSvcImpl) *apierror.APIError {
			txShipmentRepo := txSvc.repos.NewShipmentRepo()

			// Validate shipment is in account
			inAccount, apiErr := txShipmentRepo.IsInAccount(txCtx, params.AccountID, params.ShipmentID)
			if apiErr != nil {
				return apiErr
			}
			if !inAccount {
				return apierror.NewResourceNotFoundError("Shipment not found.")
			}

			txLineRepo := txSvc.repos.NewShipmentLineRepo()

			// Validate line belongs to shipment
			inShipment, apiErr := txLineRepo.IsInShipment(txCtx, params.ShipmentLineID, params.ShipmentID)
			if apiErr != nil {
				return apiErr
			}
			if !inShipment {
				return apierror.NewResourceNotFoundError("Shipment line not found in this shipment.")
			}

			old, apiErr := txLineRepo.Get(txCtx, params.ShipmentLineID)
			if apiErr != nil {
				return apiErr
			}

			// Resolve the parent sales order so the audit event can be scoped to the order's history tree.
			rootShipment, apiErr := txShipmentRepo.Get(txCtx, domain.GetShipmentParams{AccountID: params.AccountID, ShipmentID: params.ShipmentID})
			if apiErr != nil {
				return apiErr
			}

			// Backfill unchanged nullable fields with existing values. Since the SQL uses direct assignment (no COALESCE) for these fields, we must provide the existing value when the field was not sent.
			if params.QuantityUnitID == nil {
				params.QuantityUnitID = &old.QuantityUnitID
			}

			updated, apiErr := txLineRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeShipmentLine,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   rootShipment.SalesOrderID,
				Changes:          changes,
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

func (s *shipmentLineSvcImpl) DeleteShipmentLine(ctx context.Context, params domain.DeleteShipmentLineEndpointParams) *apierror.APIError {
	ctx, span := shipmentLineSvcTracer.Start(ctx, "service.shipment_line.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	lineRepo := s.repos.NewShipmentLineRepo()

	// Validate line belongs to shipment
	inShipment, apiErr := lineRepo.IsInShipment(ctx, params.ShipmentLineID, params.ShipmentID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !inShipment {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Shipment line not found in this shipment."))
	}

	shipmentLine, apiErr := lineRepo.Get(ctx, params.ShipmentLineID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeShipmentLine, params.ShipmentLineID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This shipment line has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	// Resolve the parent sales order so the audit event can be scoped to the order's history tree.
	rootShipment, apiErr := s.repos.NewShipmentRepo().Get(ctx, domain.GetShipmentParams{AccountID: params.AccountID, ShipmentID: params.ShipmentID})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shipmentLineSvcImpl) *apierror.APIError {
		txLineRepo := txSvc.repos.NewShipmentLineRepo()

		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeShipmentLine, shipmentLine.ID, shipmentLine); apiErr != nil {
			return apiErr
		}

		if apiErr := txLineRepo.Delete(txCtx, params.ShipmentLineID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(shipmentLine, (*domain.ShipmentLine)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionDelete,
			ResourceType:     constants.ObjectTypeShipmentLine,
			ResourceID:       shipmentLine.ID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   rootShipment.SalesOrderID,
			Changes:          changes,
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

// checkShipmentLineReadPermission checks the appropriate read permission based on the identity context. Internal actors need shipments:read for their own account, or customers:read / suppliers:read for external accounts.
func checkShipmentLineReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainShipments, types.ActionRead)
}
