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

var territorySvcTracer = tracing.GetTracer("core-service.territory_service")

type territorySvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type TerritorySvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *TerritorySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("territory service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("territory service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("territory service: tx manager is required")
	}
	return nil
}

func NewTerritorySvc(config *TerritorySvcConfig) domain.TerritorySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &territorySvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *territorySvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *territorySvcImpl) withTx(ctx context.Context, fn func(context.Context, *territorySvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &territorySvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// territoryAuditIncludes are what every write reads its territory with, so the audit diff sees the rep and product line.
var territoryAuditIncludes = []string{"sales_rep", "product_line"}

func (s *territorySvcImpl) BatchGetTerritoriesByIDs(ctx context.Context, ids []string) ([]*domain.Territory, *apierror.APIError) {
	ctx, span := territorySvcTracer.Start(ctx, "service.territory.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTerritories, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(ids) == 0 {
		return nil, nil
	}

	return s.repos.NewTerritoryRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

func (s *territorySvcImpl) ListTerritories(ctx context.Context, params domain.ListTerritoriesParams) (*domain.ListTerritoriesResult, *apierror.APIError) {
	ctx, span := territorySvcTracer.Start(ctx, "service.territory.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTerritories, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := checkTerritoryAccount(identity, params.AccountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewTerritoryRepo().List(ctx, params)
}

func (s *territorySvcImpl) GetTerritory(ctx context.Context, params domain.GetTerritoryParams) (*domain.Territory, *apierror.APIError) {
	ctx, span := territorySvcTracer.Start(ctx, "service.territory.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTerritories, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := checkTerritoryAccount(identity, params.AccountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewTerritoryRepo().Get(ctx, params)
}

func (s *territorySvcImpl) CreateTerritory(ctx context.Context, params domain.CreateTerritoryParams) (*domain.Territory, *apierror.APIError) {
	ctx, span := territorySvcTracer.Start(ctx, "service.territory.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTerritories, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := checkTerritoryAccount(identity, params.AccountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := validateTerritoryZipcodes(params.StartZipcode, params.EndZipcode); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// An end ZIP code with no start matches nothing, so it is dropped.
	if params.StartZipcode == nil {
		params.EndZipcode = nil
	}
	if params.EndZipcode != nil && *params.EndZipcode < *params.StartZipcode {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("End zipcode must not be before the start zipcode.", "end_zipcode"))
	}
	params.Includes = territoryAuditIncludes

	territoryID, apiErr := id.GenID(id.TerritoryIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Territory](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Territory
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *territorySvcImpl) *apierror.APIError {
			if apiErr := txSvc.validateTerritoryReferences(txCtx, params.AccountID, &params.SalesRepID, params.ProductLineID); apiErr != nil {
				return apiErr
			}

			created, apiErr := txSvc.repos.NewTerritoryRepo().Create(txCtx, territoryID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeTerritory,
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

func (s *territorySvcImpl) UpdateTerritory(ctx context.Context, params domain.UpdateTerritoryParams) (*domain.Territory, *apierror.APIError) {
	ctx, span := territorySvcTracer.Start(ctx, "service.territory.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTerritories, types.ActionUpdate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := checkTerritoryAccount(identity, params.AccountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := validateTerritoryZipcodes(params.StartZipcode, params.EndZipcode); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkTerritoryClears(params); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.Includes = territoryAuditIncludes

	meds := s.mediators()

	idempotencyKey, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}

	switch domain.RecoveryPoint(idempotencyKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Territory](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Territory
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *territorySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewTerritoryRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetTerritoryParams{
				AccountID:   params.AccountID,
				TerritoryID: params.TerritoryID,
				Includes:    territoryAuditIncludes,
			})
			if apiErr != nil {
				return apiErr
			}

			// Only a reference the update changes is checked, so a territory whose rep has since been removed can still be edited.
			salesRepID := params.SalesRepID
			if salesRepID != nil && *salesRepID == old.SalesRepID {
				salesRepID = nil
			}
			productLineID := params.ProductLineID
			if productLineID != nil && old.ProductLine != nil && *productLineID == old.ProductLine.ID {
				productLineID = nil
			}
			if apiErr := txSvc.validateTerritoryReferences(txCtx, params.AccountID, salesRepID, productLineID); apiErr != nil {
				return apiErr
			}

			if apiErr := resolveUpdatedZipcodes(old, &params); apiErr != nil {
				return apiErr
			}

			// Backfill unchanged nullable fields with existing values. Since the SQL uses direct assignment (no COALESCE) for these fields, we must provide the existing value when the field was not sent.
			if params.SalesRepID == nil && old.SalesRepID != "" {
				params.SalesRepID = &old.SalesRepID
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
				ResourceType: constants.ObjectTypeTerritory,
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

func (s *territorySvcImpl) DeleteTerritory(ctx context.Context, params domain.DeleteTerritoryParams) *apierror.APIError {
	ctx, span := territorySvcTracer.Start(ctx, "service.territory.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainTerritories, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if apiErr := checkTerritoryAccount(identity, params.AccountID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	territory, apiErr := s.repos.NewTerritoryRepo().Get(ctx, domain.GetTerritoryParams{
		AccountID:   params.AccountID,
		TerritoryID: params.TerritoryID,
		Includes:    territoryAuditIncludes,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeTerritory, params.TerritoryID, params.AccountID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This territory has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *territorySvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeTerritory, territory.ID, params.AccountID, territory); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewTerritoryRepo().Delete(txCtx, params); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(territory, (*domain.Territory)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeTerritory,
			ResourceID:   territory.ID,
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

// checkTerritoryAccount refuses a path naming an account other than the one acted for as not found, like another tenant's territory.
func checkTerritoryAccount(identity *types.Identity, accountID string) *apierror.APIError {
	if accountID != identity.Target.AccountID {
		return apierror.NewResourceNotFoundError("Resource not found.")
	}
	return nil
}

// validateTerritoryReferences refuses a sales rep or product line the account does not have, or a rep it has removed; the territory table has no foreign keys.
func (s *territorySvcImpl) validateTerritoryReferences(ctx context.Context, accountID string, salesRepID, productLineID *string) *apierror.APIError {
	if salesRepID != nil {
		rep, apiErr := s.repos.NewAccountUserRepo().GetDetailByAccountAndID(ctx, accountID, *salesRepID, nil)
		if apiErr != nil {
			return territoryReferenceError(apiErr, "Sales rep not found.", "sales_rep_id")
		}
		if rep.StatusCode == constants.AccountUserStatusRemoved {
			return apierror.NewValidationErrorWithParam("Sales rep has been removed from the account.", "sales_rep_id")
		}
	}
	if productLineID != nil {
		if _, apiErr := s.repos.NewProductLineRepo().Get(ctx, domain.GetProductLineParams{AccountID: accountID, ProductLineID: *productLineID}); apiErr != nil {
			return territoryReferenceError(apiErr, "Product line not found.", "product_line_id")
		}
	}
	return nil
}

func territoryReferenceError(apiErr *apierror.APIError, notFoundMsg, param string) *apierror.APIError {
	if apierror.IsNotFound(apiErr) {
		return apierror.NewValidationErrorWithParam(notFoundMsg, param)
	}
	return apiErr
}

func validateTerritoryZipcodes(start, end *int32) *apierror.APIError {
	if start != nil && (*start < 501 || *start > 99999) {
		return apierror.NewValidationErrorWithParam("Start zipcode must be between 501 and 99999.", "start_zipcode")
	}
	if end != nil && (*end < 501 || *end > 99999) {
		return apierror.NewValidationErrorWithParam("End zipcode must be between 501 and 99999.", "end_zipcode")
	}
	return nil
}

// checkTerritoryClears refuses an update that both clears a field and sets it, rather than let either silently win.
func checkTerritoryClears(params domain.UpdateTerritoryParams) *apierror.APIError {
	switch {
	case params.ClearStartZipcode && params.StartZipcode != nil:
		return apierror.NewValidationErrorWithParam("Send either start_zipcode or clear_start_zipcode, not both.", "clear_start_zipcode")
	case params.ClearEndZipcode && params.EndZipcode != nil:
		return apierror.NewValidationErrorWithParam("Send either end_zipcode or clear_end_zipcode, not both.", "clear_end_zipcode")
	case params.ClearProductLine && params.ProductLineID != nil:
		return apierror.NewValidationErrorWithParam("Send either product_line_id or clear_product_line, not both.", "clear_product_line")
	}
	return nil
}

// resolveUpdatedZipcodes checks the range an update leaves, dropping an end ZIP code left without a start as create does. A reversed range names the end if the update sent one.
func resolveUpdatedZipcodes(old *domain.Territory, params *domain.UpdateTerritoryParams) *apierror.APIError {
	start := old.StartZipcode
	if params.ClearStartZipcode {
		start = nil
	} else if params.StartZipcode != nil {
		start = params.StartZipcode
	}
	if start == nil {
		params.EndZipcode = nil
		params.ClearEndZipcode = true
		return nil
	}

	end := old.EndZipcode
	if params.ClearEndZipcode {
		end = nil
	} else if params.EndZipcode != nil {
		end = params.EndZipcode
	}
	if end == nil || *end >= *start {
		return nil
	}
	if params.EndZipcode != nil {
		return apierror.NewValidationErrorWithParam("End zipcode must not be before the start zipcode.", "end_zipcode")
	}
	return apierror.NewValidationErrorWithParam("Start zipcode must not be after the end zipcode.", "start_zipcode")
}
