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

var ediSvcTracer = tracing.GetTracer("core-service.edi_service")

type ediSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type EDISvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *EDISvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("edi service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("edi service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("edi service: tx manager is required")
	}
	return nil
}

func NewEDISvc(config *EDISvcConfig) domain.EDISvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &ediSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *ediSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *ediSvcImpl) withTx(ctx context.Context, fn func(context.Context, *ediSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &ediSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// ---------------------------------------------------------------------------
// DC Location methods
// ---------------------------------------------------------------------------

func (s *ediSvcImpl) ListDCLocations(ctx context.Context, params domain.ListDCLocationsParams) (*domain.ListDCLocationsResult, *apierror.APIError) {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.list_dc_locations")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEdiRuns, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	return s.repos.NewEDIRepo().ListDCLocations(ctx, params)
}

func (s *ediSvcImpl) GetDCLocation(ctx context.Context, dcLocationID string) (*domain.DCLocation, *apierror.APIError) {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.get_dc_location")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEdiRuns, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewEDIRepo().GetDCLocation(ctx, domain.GetDCLocationParams{
		OwnerAccountID: identity.Target.AccountID,
		DCLocationID:   dcLocationID,
	})
}

func (s *ediSvcImpl) CreateDCLocation(ctx context.Context, params domain.CreateDCLocationParams) (*domain.DCLocation, *apierror.APIError) {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.create_dc_location")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEdiRuns, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	dcLocationID, apiErr := id.GenID(id.DCLocationIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.DCLocation](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.DCLocation
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *ediSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewEDIRepo()

			created, apiErr := txRepo.CreateDCLocation(txCtx, dcLocationID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeDCLocation,
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

func (s *ediSvcImpl) UpdateDCLocation(ctx context.Context, params domain.UpdateDCLocationParams) (*domain.DCLocation, *apierror.APIError) {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.update_dc_location")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEdiRuns, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.OwnerAccountID = identity.Target.AccountID

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.DCLocation](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.DCLocation
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *ediSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewEDIRepo()

			old, apiErr := txRepo.GetDCLocation(txCtx, domain.GetDCLocationParams{
				OwnerAccountID: params.OwnerAccountID,
				DCLocationID:   params.DCLocationID,
			})
			if apiErr != nil {
				return apiErr
			}

			// Backfill unchanged nullable fields with existing values.
			// Since the SQL uses direct assignment (no COALESCE) for these fields,
			// we must provide the existing value when the field was not sent.
			if params.Location == nil {
				params.Location = &old.Location
			}
			if params.AccountID == nil {
				params.AccountID = &old.AccountID
			}

			updated, apiErr := txRepo.UpdateDCLocation(txCtx, params)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeDCLocation,
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

func (s *ediSvcImpl) DeleteDCLocation(ctx context.Context, dcLocationID string) *apierror.APIError {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.delete_dc_location")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEdiRuns, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	ownerAccountID := identity.Target.AccountID

	dcLocation, apiErr := s.repos.NewEDIRepo().GetDCLocation(ctx, domain.GetDCLocationParams{
		OwnerAccountID: ownerAccountID,
		DCLocationID:   dcLocationID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeDCLocation, dcLocationID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This DC location has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *ediSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeDCLocation, dcLocation.ID, dcLocation); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewEDIRepo().DeleteDCLocation(txCtx, domain.DeleteDCLocationParams{
			OwnerAccountID: ownerAccountID,
			DCLocationID:   dcLocationID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(dcLocation, (*domain.DCLocation)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeDCLocation,
			ResourceID:   dcLocation.ID,
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

// ---------------------------------------------------------------------------
// EDI Run methods
// ---------------------------------------------------------------------------

func (s *ediSvcImpl) ListEDIRuns(ctx context.Context, params domain.ListEDIRunsParams) (*domain.ListEDIRunsResult, *apierror.APIError) {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.list_edi_runs")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEdiRuns, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewEDIRepo().ListEDIRuns(ctx, params)
}

func (s *ediSvcImpl) GetEDIRun(ctx context.Context, ediRunID string) (*domain.EDIRun, *apierror.APIError) {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.get_edi_run")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEdiRuns, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewEDIRepo().GetEDIRun(ctx, identity.Target.AccountID, ediRunID)
}

// BatchGetDCLocationsByIDs returns DC locations matching the input IDs that
// the caller's account is authorized to read.
func (s *ediSvcImpl) BatchGetDCLocationsByIDs(ctx context.Context, ids []string) ([]*domain.DCLocation, *apierror.APIError) {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.batch_get_dc_locations_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEdiRuns, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewEDIRepo().GetDCLocationsByIDs(ctx, identity.Target.AccountID, ids)
}

// BatchGetEDIRunsByIDs returns EDI runs matching the input IDs that the
// caller's account is authorized to read.
func (s *ediSvcImpl) BatchGetEDIRunsByIDs(ctx context.Context, ids []string) ([]*domain.EDIRun, *apierror.APIError) {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.batch_get_edi_runs_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainEdiRuns, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewEDIRepo().GetEDIRunsByIDs(ctx, identity.Target.AccountID, ids)
}

// ---------------------------------------------------------------------------
// EDI Action methods
// ---------------------------------------------------------------------------

// PullOrders processes EDI operations. Scaffolded - actual FTP/XML/Stedi integration will be wired later.
func (s *ediSvcImpl) PullOrders(ctx context.Context) *apierror.APIError {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.pull_orders")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// TODO: Wire FTP/XML/Stedi integration for actual EDI processing.
	return nil
}

// ResubmitInvoice resubmits an invoice via EDI. Scaffolded - actual FTP upload will be wired later.
func (s *ediSvcImpl) ResubmitInvoice(ctx context.Context, invoiceID string) *apierror.APIError {
	ctx, span := ediSvcTracer.Start(ctx, "service.edi.resubmit_invoice")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInvoices, types.ActionUpdate); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if invoiceID == "" {
		return tracing.Trace(span, apierror.NewValidationError("Invoice ID is required."))
	}

	// TODO: Wire actual invoice resubmission via FTP/EDI.
	return nil
}
