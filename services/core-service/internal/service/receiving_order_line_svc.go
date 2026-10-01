package service

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var receivingOrderLineSvcTracer = tracing.GetTracer("core-service.receiving_order_line_service")

type receivingOrderLineSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ReceivingOrderLineSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ReceivingOrderLineSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("receiving order line service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("receiving order line service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("receiving order line service: tx manager is required")
	}
	return nil
}

func NewReceivingOrderLineSvc(config *ReceivingOrderLineSvcConfig) domain.ReceivingOrderLineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &receivingOrderLineSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *receivingOrderLineSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *receivingOrderLineSvcImpl) withTx(ctx context.Context, fn func(context.Context, *receivingOrderLineSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &receivingOrderLineSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *receivingOrderLineSvcImpl) UpdateReceivingOrderLine(ctx context.Context, params domain.UpdateReceivingOrderLineParams) (*domain.ReceivingOrderLine, *apierror.APIError) {
	ctx, span := receivingOrderLineSvcTracer.Start(ctx, "service.receiving_order_line.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainReceivingOrders, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewReceivingOrderRepo()

	// Verify receiving order is in account
	inAccount, apiErr := repo.IsInAccount(ctx, params.AccountID, params.ReceivingOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Receiving order not found."))
	}

	// Verify line is in receiving order
	inOrder, apiErr := repo.IsLineInReceivingOrder(ctx, params.LineID, params.ReceivingOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inOrder {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Receiving order line not found."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ReceivingOrderLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ReceivingOrderLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *receivingOrderLineSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewReceivingOrderRepo()

			old, apiErr := changeableLine(txCtx, txRepo, params.AccountID, params.ReceivingOrderID, params.LineID)
			if apiErr != nil {
				return apiErr
			}

			if q := params.Quantity; q != nil {
				if q.Value.IsNegative() {
					return apierror.NewValidationErrorWithParam("The quantity must not be negative.", "quantity.value")
				}
				if apiErr := validateReceivedQuantityUnit(txCtx, txSvc.repos, params.AccountID, old, *q, "quantity.unit_id"); apiErr != nil {
					return apiErr
				}
				if apiErr := txRepo.UpdateLineQuantity(txCtx, params.LineID, q.Value.String(), q.UnitID); apiErr != nil {
					return apiErr
				}
			}

			line, apiErr := txRepo.GetLine(txCtx, params.LineID)
			if apiErr != nil {
				return apiErr
			}
			result = line

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeReceivingOrderLine,
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

func (s *receivingOrderLineSvcImpl) VoidReceivingOrderLine(ctx context.Context, receivingOrderID, lineID string) (*domain.ReceivingOrderLine, *apierror.APIError) {
	ctx, span := receivingOrderLineSvcTracer.Start(ctx, "service.receiving_order_line.void")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainReceivingOrders, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	repo := s.repos.NewReceivingOrderRepo()

	// Verify receiving order is in account
	inAccount, apiErr := repo.IsInAccount(ctx, accountID, receivingOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Receiving order not found."))
	}

	// Verify line is in receiving order
	inOrder, apiErr := repo.IsLineInReceivingOrder(ctx, lineID, receivingOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inOrder {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Receiving order line not found."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ReceivingOrderLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ReceivingOrderLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *receivingOrderLineSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewReceivingOrderRepo()

			old, apiErr := changeableLine(txCtx, txRepo, accountID, receivingOrderID, lineID)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.VoidLine(txCtx, lineID, accountID); apiErr != nil {
				return apiErr
			}

			line, apiErr := txRepo.GetLine(txCtx, lineID)
			if apiErr != nil {
				return apiErr
			}
			result = line

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeReceivingOrderLine,
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

func (s *receivingOrderLineSvcImpl) ReceiveReceivingOrderLine(ctx context.Context, receivingOrderID, lineID string) (*domain.ReceivingOrderLine, *apierror.APIError) {
	ctx, span := receivingOrderLineSvcTracer.Start(ctx, "service.receiving_order_line.receive")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainReceivingOrders, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	accountID := identity.Target.AccountID

	repo := s.repos.NewReceivingOrderRepo()

	// Verify receiving order is in account
	inAccount, apiErr := repo.IsInAccount(ctx, accountID, receivingOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Receiving order not found."))
	}

	// Verify line is in receiving order
	inOrder, apiErr := repo.IsLineInReceivingOrder(ctx, lineID, receivingOrderID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inOrder {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Receiving order line not found."))
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ReceivingOrderLine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ReceivingOrderLine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *receivingOrderLineSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewReceivingOrderRepo()

			old, apiErr := changeableLine(txCtx, txRepo, accountID, receivingOrderID, lineID)
			if apiErr != nil {
				return apiErr
			}

			// Finish the line: it takes whatever of its order line the other receiving lines do not already hold, in the unit the order line was ordered in.
			progress, apiErr := txRepo.ListReceivingProgress(txCtx, accountID, []string{old.OrderLineID})
			if apiErr != nil {
				return apiErr
			}
			if value, ok := receiveTarget(progressByOrderLine(progress)[old.OrderLineID], lineID); ok {
				if apiErr := txRepo.UpdateLineQuantity(txCtx, lineID, value.String(), old.OrderLineUnitID); apiErr != nil {
					return apiErr
				}
			}

			line, apiErr := txRepo.GetLine(txCtx, lineID)
			if apiErr != nil {
				return apiErr
			}
			result = line

			changes := audit.ComputeChanges(old, result)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeReceivingOrderLine,
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

// changeableLine reads a line that is about to be changed, refusing one that can no longer be.
//
// A stocked line's quantity is what went into inventory, and a completed order's lines are all stocked; editing, voiding or re-receiving either would leave the receiving order disagreeing with the stock it booked, so the change is refused. Void the receiving order first to reopen it.
func changeableLine(ctx context.Context, repo domain.ReceivingOrderRepo, accountID, receivingOrderID, lineID string) (*domain.ReceivingOrderLine, *apierror.APIError) {
	order, apiErr := repo.Get(ctx, accountID, receivingOrderID)
	if apiErr != nil {
		return nil, apiErr
	}
	if order.CompletedAt != nil {
		return nil, apierror.NewValidationError("The receiving order is complete; void it to change its lines.")
	}
	for _, line := range order.Lines {
		if line.ID != lineID {
			continue
		}
		if line.StockedAt != nil {
			return nil, apierror.NewValidationError("The receiving order line has already been stocked.")
		}
		return line, nil
	}
	return nil, apierror.NewResourceNotFoundError("Receiving order line not found.")
}
