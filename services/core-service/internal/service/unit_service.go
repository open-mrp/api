package service

import (
	"context"
	"fmt"
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

var unitSvcTracer = tracing.GetTracer("core-service.unit_service")

type unitSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type UnitSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *UnitSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("unit service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("unit service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("unit service: tx manager is required")
	}
	return nil
}

func NewUnitSvc(config *UnitSvcConfig) domain.UnitSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &unitSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *unitSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *unitSvcImpl) withTx(ctx context.Context, fn func(context.Context, *unitSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &unitSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

// ListUnits returns a paginated list of units for the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and units:read permission.
// 2. Require the Augno-Account header to scope the query.
// 3. Query the unit repository with the account ID and pagination params.
func (s *unitSvcImpl) ListUnits(ctx context.Context, params domain.ListUnitsParams) (*domain.ListUnitsResult, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnits, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewUnitRepo().List(ctx, params)
}

// GetUnit retrieves a single unit by ID, scoped to the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and units:read permission.
// 2. Require the Augno-Account header.
// 3. Fetch the unit from the repository by account ID and unit ID.
func (s *unitSvcImpl) GetUnit(ctx context.Context, unitID string) (*domain.Unit, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnits, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewUnitRepo().Get(ctx, domain.GetUnitParams{
		AccountID: identity.Target.AccountID,
		UnitID:    unitID,
	})
}

// CreateUnit creates a new unit of measure for the caller's account, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type, and units:create permission.
// 2. Generate a unique unit ID.
// 3. Upsert an idempotency key; if already finished, return the cached response.
// 4. Within a transaction, check for duplicate name and abbreviation.
// 5. Insert the unit record and cache the success response.
// 6. On error, cache the error response for idempotent replay.
func (s *unitSvcImpl) CreateUnit(ctx context.Context, params domain.CreateUnitParams) (*domain.Unit, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnits, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	unitID, apiErr := id.GenID(id.UnitIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Unit](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Unit
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewUnitRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A unit with this name already exists.", "name")
			}

			exists, apiErr = txRepo.ExistsByAbbreviation(txCtx, params.AccountID, params.Abbreviation, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A unit with this abbreviation already exists.", "abbreviation")
			}

			created, apiErr := txRepo.Create(txCtx, unitID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeUnit,
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

// UpdateUnit modifies an existing custom unit, with idempotency support.
//
// 1. Extract and validate the caller's identity, actor type, and units:update permission.
// 2. Upsert an idempotency key; if already finished, return the cached response.
// 3. Within a transaction, verify the unit exists and is not a system unit.
// 4. Check for duplicate name and abbreviation (excluding the current unit).
// 5. Apply the updates and cache the success response.
// 6. On error, cache the error response for idempotent replay.
func (s *unitSvcImpl) UpdateUnit(ctx context.Context, params domain.UpdateUnitParams) (*domain.Unit, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnits, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Unit](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Unit
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewUnitRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetUnitParams{AccountID: params.AccountID, UnitID: params.UnitID})
			if apiErr != nil {
				return apiErr
			}
			if old.AccountID == nil {
				return apierror.NewValidationError("System units cannot be modified.")
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.UnitID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A unit with this name already exists.", "name")
				}
			}

			if params.Abbreviation != nil {
				exists, apiErr := txRepo.ExistsByAbbreviation(txCtx, params.AccountID, *params.Abbreviation, &params.UnitID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A unit with this abbreviation already exists.", "abbreviation")
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
				ResourceType: constants.ObjectTypeUnit,
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

// DeleteUnit removes a custom unit by ID, scoped to the caller's account.
//
// 1. Extract and validate the caller's identity, actor type, and units:delete permission.
// 2. Fetch the unit to verify it exists and is not a system unit.
// 3. Within a transaction, delete the unit from the repository.
func (s *unitSvcImpl) DeleteUnit(ctx context.Context, unitID string) *apierror.APIError {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainUnits, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	unit, apiErr := s.repos.NewUnitRepo().Get(ctx, domain.GetUnitParams{AccountID: accountID, UnitID: unitID})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeUnit, unitID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This unit has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}
	if unit.AccountID == nil {
		return tracing.Trace(span, apierror.NewValidationError("System units cannot be deleted."))
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *unitSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeUnit, unit.ID, unit); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewUnitRepo().Delete(txCtx, domain.DeleteUnitParams{
			AccountID: accountID,
			UnitID:    unitID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(unit, (*domain.Unit)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeUnit,
			ResourceID:   unit.ID,
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

// ValidateUnits validates unit abbreviations and returns matching units.
//
// 1. Extract and validate the caller's identity (assigned actor — internal, customer, or supplier).
// 2. For internal actors, check the routed read permission; verify counterparty read access for external targets.
// 3. Extract abbreviations from the unit map and query the repository (case-insensitive).
// 4. Build a result map matching original keys to found units.
func (s *unitSvcImpl) ValidateUnits(ctx context.Context, params domain.ValidateUnitsParams) (*domain.ValidateUnitsResult, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.validate")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if apiErr := checkUnitReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	params.AccountID = identity.Target.AccountID

	// Extract unique lowercase abbreviations
	abbrevSet := make(map[string]struct{}, len(params.UnitMap))
	for _, abbrev := range params.UnitMap {
		abbrevSet[strings.ToLower(abbrev)] = struct{}{}
	}
	abbreviations := make([]string, 0, len(abbrevSet))
	for a := range abbrevSet {
		abbreviations = append(abbreviations, a)
	}

	if len(abbreviations) == 0 {
		return &domain.ValidateUnitsResult{Units: map[string]*domain.Unit{}}, nil
	}

	units, apiErr := s.repos.NewUnitRepo().FindByAbbreviations(ctx, params.AccountID, abbreviations)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build abbreviation -> unit map for quick lookup (case-insensitive)
	abbrevToUnit := make(map[string]*domain.Unit, len(units))
	for _, u := range units {
		abbrevToUnit[strings.ToLower(u.Abbreviation)] = u
	}

	// Match results back to original keys
	result := make(map[string]*domain.Unit, len(params.UnitMap))
	for key, abbrev := range params.UnitMap {
		if u, ok := abbrevToUnit[strings.ToLower(abbrev)]; ok {
			result[key] = u
		}
	}

	return &domain.ValidateUnitsResult{Units: result}, nil
}

func (s *unitSvcImpl) BatchGetUnitsByIDs(ctx context.Context, ids []string) ([]*domain.Unit, *apierror.APIError) {
	ctx, span := unitSvcTracer.Start(ctx, "service.unit.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsInternalActor() {
		if apiErr := checkUnitReadPermission(identity); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	} else if !identity.IsCustomerUser() && !identity.IsSupplierUser() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You do not have access to this resource."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckCounterpartyReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewUnitRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

// checkUnitReadPermission checks the appropriate read permission based on the identity context. Internal actors need units:read for their own account, or customers:read / suppliers:read for external accounts.
func checkUnitReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainUnits, types.ActionRead)
}
