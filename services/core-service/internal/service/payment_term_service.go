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
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var paymentTermSvcTracer = tracing.GetTracer("core-service.payment_term_service")

type paymentTermSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type PaymentTermSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *PaymentTermSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("payment term service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("payment term service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("payment term service: tx manager is required")
	}
	return nil
}

func NewPaymentTermSvc(config *PaymentTermSvcConfig) domain.PaymentTermSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &paymentTermSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *paymentTermSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *paymentTermSvcImpl) withTx(ctx context.Context, fn func(context.Context, *paymentTermSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &paymentTermSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *paymentTermSvcImpl) ListPaymentTerms(ctx context.Context, params domain.ListPaymentTermsParams) (*domain.ListPaymentTermsResult, *apierror.APIError) {
	ctx, span := paymentTermSvcTracer.Start(ctx, "service.payment_term.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPaymentTerms, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewPaymentTermRepo().List(ctx, params)
}

func (s *paymentTermSvcImpl) GetPaymentTerm(ctx context.Context, paymentTermID string) (*domain.PaymentTerm, *apierror.APIError) {
	ctx, span := paymentTermSvcTracer.Start(ctx, "service.payment_term.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPaymentTerms, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewPaymentTermRepo().Get(ctx, domain.GetPaymentTermParams{
		AccountID:     identity.Target.AccountID,
		PaymentTermID: paymentTermID,
	})
}

// BatchGetPaymentTermsByIDs returns payment terms matching the input IDs that the caller's account is authorized to read (account-scoped plus system terms).
func (s *paymentTermSvcImpl) BatchGetPaymentTermsByIDs(ctx context.Context, ids []string) ([]*domain.PaymentTerm, *apierror.APIError) {
	ctx, span := paymentTermSvcTracer.Start(ctx, "service.payment_term.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPaymentTerms, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewPaymentTermRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

func (s *paymentTermSvcImpl) CreatePaymentTerm(ctx context.Context, params domain.CreatePaymentTermParams) (*domain.PaymentTerm, *apierror.APIError) {
	ctx, span := paymentTermSvcTracer.Start(ctx, "service.payment_term.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPaymentTerms, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	paymentTermID, apiErr := id.GenID(id.PaymentTermIDPrefix, nil)
	if apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.PaymentTerm](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.PaymentTerm
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *paymentTermSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPaymentTermRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A payment term with this name already exists.", "name")
			}

			created, apiErr := txRepo.Create(txCtx, paymentTermID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypePaymentTerm,
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

func (s *paymentTermSvcImpl) UpdatePaymentTerm(ctx context.Context, params domain.UpdatePaymentTermParams) (*domain.PaymentTerm, *apierror.APIError) {
	ctx, span := paymentTermSvcTracer.Start(ctx, "service.payment_term.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPaymentTerms, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.PaymentTerm](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.PaymentTerm
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *paymentTermSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewPaymentTermRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetPaymentTermParams{AccountID: params.AccountID, PaymentTermID: params.PaymentTermID})
			if apiErr != nil {
				return apiErr
			}
			if old.AccountID == nil {
				return apierror.NewValidationError("Default payment terms cannot be modified.")
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.PaymentTermID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A payment term with this name already exists.", "name")
				}
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypePaymentTerm,
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

func (s *paymentTermSvcImpl) DeletePaymentTerm(ctx context.Context, paymentTermID string) *apierror.APIError {
	ctx, span := paymentTermSvcTracer.Start(ctx, "service.payment_term.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainPaymentTerms, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	paymentTerm, apiErr := s.repos.NewPaymentTermRepo().Get(ctx, domain.GetPaymentTermParams{AccountID: accountID, PaymentTermID: paymentTermID})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypePaymentTerm, paymentTermID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This payment term has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if paymentTerm.AccountID == nil {
		return tracing.Trace(span, apierror.NewValidationError("Default payment terms cannot be deleted."))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *paymentTermSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypePaymentTerm, paymentTerm.ID, paymentTerm); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewPaymentTermRepo().Delete(txCtx, domain.DeletePaymentTermParams{
			AccountID:     accountID,
			PaymentTermID: paymentTermID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(paymentTerm, (*domain.PaymentTerm)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypePaymentTerm,
			ResourceID:   paymentTerm.ID,
			Changes:      changes,
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
