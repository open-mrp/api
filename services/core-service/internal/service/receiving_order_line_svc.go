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
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/tracing"
	"github.com/shopspring/decimal"
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
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

			old, apiErr := txRepo.GetLine(txCtx, params.LineID)
			if apiErr != nil {
				return apiErr
			}

			if params.QuantityValue != nil {
				if apiErr := txRepo.UpdateLineQuantity(txCtx, params.LineID, *params.QuantityValue); apiErr != nil {
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
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

			old, apiErr := txRepo.GetLine(txCtx, lineID)
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
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
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

	// Calculate quantity yet to be received
	remainingValue, _, apiErr := repo.CalculateQuantityYetToBeReceived(ctx, lineID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Parse remaining as decimal, if > 0 update line quantity
	remaining, err := decimal.NewFromString(remainingValue)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue parsing remaining quantity."))
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

			old, apiErr := txRepo.GetLine(txCtx, lineID)
			if apiErr != nil {
				return apiErr
			}

			if remaining.GreaterThan(decimal.Zero) {
				if apiErr := txRepo.UpdateLineQuantity(txCtx, lineID, remainingValue); apiErr != nil {
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
