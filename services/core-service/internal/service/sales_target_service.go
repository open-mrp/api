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

var salesTargetSvcTracer = tracing.GetTracer("core-service.sales_target_service")

type salesTargetSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type SalesTargetSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *SalesTargetSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("sales target service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("sales target service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("sales target service: tx manager is required")
	}
	return nil
}

func NewSalesTargetSvc(config *SalesTargetSvcConfig) domain.SalesTargetSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &salesTargetSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *salesTargetSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *salesTargetSvcImpl) withTx(ctx context.Context, fn func(context.Context, *salesTargetSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &salesTargetSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// ListSalesTargets returns a paginated list of sales targets for an account user.
func (s *salesTargetSvcImpl) ListSalesTargets(ctx context.Context, params domain.ListSalesTargetsParams) (*domain.ListSalesTargetsResult, *apierror.APIError) {
	ctx, span := salesTargetSvcTracer.Start(ctx, "service.sales_target.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesTargets, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewSalesTargetRepo()

	// Validate that the sales rep (account user) exists in the account.
	salesRepExists, apiErr := repo.SalesRepExistsInAccount(ctx, params.SalesRepID, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !salesRepExists {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account user not found."))
	}

	return repo.List(ctx, params)
}

// CreateSalesTarget creates a new sales target with idempotency support.
func (s *salesTargetSvcImpl) CreateSalesTarget(ctx context.Context, params domain.CreateSalesTargetParams) (*domain.SalesTarget, *apierror.APIError) {
	ctx, span := salesTargetSvcTracer.Start(ctx, "service.sales_target.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesTargets, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	// Validate that the sales rep (account user) exists in the account.
	repo := s.repos.NewSalesTargetRepo()
	salesRepExists, apiErr := repo.SalesRepExistsInAccount(ctx, params.SalesRepID, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !salesRepExists {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account user not found."))
	}

	targetID, apiErr := id.GenID(id.TargetIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.SalesTarget](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.SalesTarget
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesTargetSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesTargetRepo()

			if apiErr := txRepo.InsertQuantity(txCtx, quantityID, params.AmountValue, params.AmountUnitID); apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.Create(txCtx, targetID, params, quantityID); apiErr != nil {
				return apiErr
			}

			created, apiErr := txRepo.Get(txCtx, targetID)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeSalesTarget,
				ResourceID:   created.ID,
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

// UpsertSalesTarget creates or updates a sales target by ID. This is a PUT endpoint — idempotent by design, no idempotency key needed.
func (s *salesTargetSvcImpl) UpsertSalesTarget(ctx context.Context, params domain.UpsertSalesTargetParams) (*domain.SalesTarget, *apierror.APIError) {
	ctx, span := salesTargetSvcTracer.Start(ctx, "service.sales_target.upsert")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesTargets, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	repo := s.repos.NewSalesTargetRepo()

	// Validate that the sales rep (account user) exists in the account.
	salesRepExists, apiErr := repo.SalesRepExistsInAccount(ctx, params.SalesRepID, params.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !salesRepExists {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account user not found."))
	}

	exists, apiErr := repo.Exists(ctx, params.TargetID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if exists {
		// Update existing target.
		inAccount, apiErr := repo.IsInAccount(ctx, params.TargetID, params.AccountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if !inAccount {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Sales target not found."))
		}

		// Get the existing target to find its amount_id for quantity update.
		existing, apiErr := repo.Get(ctx, params.TargetID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		// Dashboard only updates the quantity measure (value) on existing targets. Dates and unit are not changed on update.
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesTargetSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesTargetRepo()

			if apiErr := txRepo.UpdateQuantity(txCtx, existing.AmountID, params.AmountValue, existing.AmountUnitID); apiErr != nil {
				return apiErr
			}

			updated, apiErr := txRepo.Get(txCtx, params.TargetID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(existing, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSalesTarget,
				ResourceID:   updated.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return nil
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else {
		// Create new target.
		quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *salesTargetSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSalesTargetRepo()

			if apiErr := txRepo.InsertQuantity(txCtx, quantityID, params.AmountValue, params.AmountUnitID); apiErr != nil {
				return apiErr
			}

			if apiErr := txRepo.Create(txCtx, params.TargetID, domain.CreateSalesTargetParams{
				AccountID:    params.AccountID,
				SalesRepID:   params.SalesRepID,
				StartDate:    params.StartDate,
				EndDate:      params.EndDate,
				AmountValue:  params.AmountValue,
				AmountUnitID: params.AmountUnitID,
			}, quantityID); apiErr != nil {
				return apiErr
			}

			created, apiErr := txRepo.Get(txCtx, params.TargetID)
			if apiErr != nil {
				return apiErr
			}

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeSalesTarget,
				ResourceID:   created.ID,
				Changes:      changes,
			}); apiErr != nil {
				return apiErr
			}

			return nil
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Return refreshed target.
	return repo.Get(ctx, params.TargetID)
}
