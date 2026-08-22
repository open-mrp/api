package service

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	s3client "github.com/open-mrp/api/shared/cloud/s3"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

var shippingCaseSvcTracer = tracing.GetTracer("core-service.shipping_case_service")

type shippingCaseSvcImpl struct {
	repos                domain.RepoFactory
	mediatorFactory      domain.MediatorFactory
	txManager            TransactionManager
	s3Client             s3client.ObjectStore
	shippingLabelsBucket string
}

type ShippingCaseSvcConfig struct {
	// RepoFactory (required) is the repository factory.
	RepoFactory domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager

	// S3Client (required) is the object store client used for file storage.
	S3Client s3client.ObjectStore

	// ShippingLabelsBucket (optional; default: "") is the S3 bucket for shipping labels. It is not validated at construction.
	ShippingLabelsBucket string
}

func (c *ShippingCaseSvcConfig) validate() error {
	if c.RepoFactory == nil {
		return fmt.Errorf("shipping case service: repo factory is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("shipping case service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("shipping case service: tx manager is required")
	}
	if c.S3Client == nil {
		return fmt.Errorf("shipping case service: s3 client is required")
	}
	return nil
}

func NewShippingCaseSvc(config *ShippingCaseSvcConfig) domain.ShippingCaseSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shippingCaseSvcImpl{
		repos:                config.RepoFactory,
		mediatorFactory:      config.MediatorFactory,
		txManager:            config.TxManager,
		s3Client:             config.S3Client,
		shippingLabelsBucket: config.ShippingLabelsBucket,
	}
}

func (s *shippingCaseSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *shippingCaseSvcImpl) withTx(ctx context.Context, fn func(context.Context, *shippingCaseSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &shippingCaseSvcImpl{
			repos:                f,
			mediatorFactory:      s.mediatorFactory,
			txManager:            s.txManager,
			s3Client:             s.s3Client,
			shippingLabelsBucket: s.shippingLabelsBucket,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *shippingCaseSvcImpl) GetShippingCase(ctx context.Context, accountID, shippingCaseID string) (*domain.ShippingCase, *apierror.APIError) {
	ctx, span := shippingCaseSvcTracer.Start(ctx, "service.shipping_case.get")
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
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	repo := s.repos.NewShippingCaseRepo()
	sc, apiErr := repo.Get(ctx, identity.Target.AccountID, shippingCaseID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return sc, nil
}

func (s *shippingCaseSvcImpl) UpdateShippingCase(ctx context.Context, params domain.UpdateShippingCaseParams) (*domain.ShippingCase, *apierror.APIError) {
	ctx, span := shippingCaseSvcTracer.Start(ctx, "service.shipping_case.update")
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
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	params.AccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.ShippingCase](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ShippingCase
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shippingCaseSvcImpl) *apierror.APIError {
			txShippingCaseRepo := txSvc.repos.NewShippingCaseRepo()
			txQuantityRepo := txSvc.repos.NewQuantityRepo()

			// Fetch old state before any mutations for audit diff
			old, apiErr := txShippingCaseRepo.Get(txCtx, params.AccountID, params.ShippingCaseID)
			if apiErr != nil {
				return apiErr
			}

			// Update tracking number on the shipping case itself
			if params.TrackingNumber != nil {
				if apiErr := txShippingCaseRepo.Update(txCtx, params); apiErr != nil {
					return apiErr
				}
			}

			// Update freight amount quantity if provided
			if params.FreightAmountValue != nil || params.FreightAmountUnitID != nil {
				scObjType := constants.ObjectTypeShippingCase
				_, apiErr = txQuantityRepo.Update(txCtx, domain.UpdateQuantityParams{
					QuantityID: old.FreightAmountID,
					Value:      params.FreightAmountValue,
					UnitID:     params.FreightAmountUnitID,
					ObjectID:   &params.ShippingCaseID,
					ObjectType: &scObjType,
				})
				if apiErr != nil {
					return apiErr
				}
			}

			// Update freight weight quantity if provided
			if params.FreightWeightValue != nil || params.FreightWeightUnitID != nil {
				scObjType2 := constants.ObjectTypeShippingCase
				_, apiErr = txQuantityRepo.Update(txCtx, domain.UpdateQuantityParams{
					QuantityID: old.FreightWeightID,
					Value:      params.FreightWeightValue,
					UnitID:     params.FreightWeightUnitID,
					ObjectID:   &params.ShippingCaseID,
					ObjectType: &scObjType2,
				})
				if apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch the full shipping case
			updated, apiErr := txShippingCaseRepo.Get(txCtx, params.AccountID, params.ShippingCaseID)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			// The case hangs off a sales order; stamping it as the root is what puts a case edit in
			// that order's history alongside its shipment and invoice.
			rootOrderID, apiErr := txShippingCaseRepo.GetSalesOrderID(txCtx, params.AccountID, params.ShippingCaseID)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeShippingCase,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   rootOrderID,
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

// Corrects the tracking number of a case that has already shipped, which the ordinary update path
// is not meant to reach. Reserved for admins recovering a mis-recorded dispatch.
func (s *shippingCaseSvcImpl) AdminUpdateShippingCaseTracking(ctx context.Context, params domain.AdminUpdateShippingCaseTrackingParams) (*domain.ShippingCase, *apierror.APIError) {
	ctx, span := shippingCaseSvcTracer.Start(ctx, "service.shipping_case.admin_update_tracking")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckIsAdmin(); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ShippingCase](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ShippingCase
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shippingCaseSvcImpl) *apierror.APIError {
			txShippingCaseRepo := txSvc.repos.NewShippingCaseRepo()

			old, apiErr := txShippingCaseRepo.Get(txCtx, params.AccountID, params.ShippingCaseID)
			if apiErr != nil {
				return apiErr
			}

			if old.ShippedAt == nil {
				return apierror.NewValidationError("Shipping case has not been shipped yet. Use the regular update endpoint.")
			}

			if params.TrackingNumber != nil {
				if apiErr := txShippingCaseRepo.Update(txCtx, domain.UpdateShippingCaseParams{
					AccountID:      params.AccountID,
					ShippingCaseID: params.ShippingCaseID,
					TrackingNumber: params.TrackingNumber,
				}); apiErr != nil {
					return apiErr
				}
			}

			updated, apiErr := txShippingCaseRepo.Get(txCtx, params.AccountID, params.ShippingCaseID)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			// The case hangs off a sales order; stamping it as the root is what puts a case edit in
			// that order's history alongside its shipment and invoice.
			rootOrderID, apiErr := txShippingCaseRepo.GetSalesOrderID(txCtx, params.AccountID, params.ShippingCaseID)
			if apiErr != nil {
				return apiErr
			}

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:      domain.ServiceName,
				Action:           constants.AuditActionUpdate,
				ResourceType:     constants.ObjectTypeShippingCase,
				ResourceID:       updated.ID,
				RootResourceType: constants.ObjectTypeSalesOrder,
				RootResourceID:   rootOrderID,
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

func (s *shippingCaseSvcImpl) DeleteShippingCase(ctx context.Context, accountID, shippingCaseID string) *apierror.APIError {
	ctx, span := shippingCaseSvcTracer.Start(ctx, "service.shipping_case.delete")
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
	if !identity.IsTargetAccountSet() {
		return tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	repo := s.repos.NewShippingCaseRepo()

	shippingCase, apiErr := repo.Get(ctx, identity.Target.AccountID, shippingCaseID)
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeShippingCase, shippingCaseID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This shipping case has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *shippingCaseSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeShippingCase, shippingCase.ID, shippingCase); apiErr != nil {
			return apiErr
		}

		// Resolved before the delete, since the row it walks is about to go.
		rootOrderID, apiErr := txSvc.repos.NewShippingCaseRepo().GetSalesOrderID(txCtx, identity.Target.AccountID, shippingCaseID)
		if apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewShippingCaseRepo().Delete(txCtx, identity.Target.AccountID, shippingCaseID); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(shippingCase, (*domain.ShippingCase)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:      domain.ServiceName,
			Action:           constants.AuditActionDelete,
			ResourceType:     constants.ObjectTypeShippingCase,
			ResourceID:       shippingCase.ID,
			RootResourceType: constants.ObjectTypeSalesOrder,
			RootResourceID:   rootOrderID,
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

func (s *shippingCaseSvcImpl) GetShippingCaseLabel(ctx context.Context, accountID, shippingCaseID string) (*string, *apierror.APIError) {
	ctx, span := shippingCaseSvcTracer.Start(ctx, "service.shipping_case.get_label")
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
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}

	repo := s.repos.NewShippingCaseRepo()
	number, apiErr := repo.GetNumber(ctx, identity.Target.AccountID, shippingCaseID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	s3Key := shippingLabelS3Key(identity.Target.AccountID, number)

	// Reads a label core-service itself uploads on ship and deletes on void, so the AugnoProdCoreS3Access
	// IRSA policy (open-mrp/infra: production/terraform/core.tf) must keep Get/Put/Delete on the shipping-labels bucket.
	exists, apiErr := s.s3Client.FileExists(ctx, s.shippingLabelsBucket, s3Key)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !exists {
		return nil, nil
	}

	url, apiErr := s.s3Client.GetPresignedURL(ctx, s.shippingLabelsBucket, s3Key, time.Hour)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &url, nil
}

// Locates a shipping case's stored label. The layout is shared with the dashboard API, which uploads
// the labels core-service reads here and deletes on void.
func shippingLabelS3Key(accountID, caseNumber string) string {
	return fmt.Sprintf("shipping-labels/%s/%s.gif", accountID, caseNumber)
}
