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

var departmentSvcTracer = tracing.GetTracer("core-service.department_service")

type departmentSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type DepartmentSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *DepartmentSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("department service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("department service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("department service: tx manager is required")
	}
	return nil
}

func NewDepartmentSvc(config *DepartmentSvcConfig) domain.DepartmentSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &departmentSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *departmentSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *departmentSvcImpl) withTx(ctx context.Context, fn func(context.Context, *departmentSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &departmentSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *departmentSvcImpl) ListDepartments(ctx context.Context, params domain.ListDepartmentsParams) (*domain.ListDepartmentsResult, *apierror.APIError) {
	ctx, span := departmentSvcTracer.Start(ctx, "service.department.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDepartments, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewDepartmentRepo().List(ctx, params)
}

func (s *departmentSvcImpl) GetDepartment(ctx context.Context, departmentID string) (*domain.Department, *apierror.APIError) {
	ctx, span := departmentSvcTracer.Start(ctx, "service.department.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDepartments, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewDepartmentRepo().Get(ctx, domain.GetDepartmentParams{
		AccountID:    identity.Target.AccountID,
		DepartmentID: departmentID,
	})
}

func (s *departmentSvcImpl) CreateDepartment(ctx context.Context, params domain.CreateDepartmentParams) (*domain.Department, *apierror.APIError) {
	ctx, span := departmentSvcTracer.Start(ctx, "service.department.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDepartments, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	departmentID, apiErr := id.GenID(id.DepartmentIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Department](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Department
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *departmentSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewDepartmentRepo()

			exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A department with this name already exists.", "name")
			}

			if params.LaborRate != nil {
				if apiErr := ValidateCostRateUnits(txCtx, txSvc.repos.NewUnitRepo(), params.LaborRate.NumeratorUnitID, params.LaborRate.DenominatorUnitID, "labor_rate"); apiErr != nil {
					return apiErr
				}
				rateID, apiErr := id.GenID(id.RateIDPrefix, nil)
				if apiErr != nil {
					return apiErr
				}
				if apiErr := txRepo.InsertLaborRate(txCtx, rateID, *params.LaborRate); apiErr != nil {
					return apiErr
				}
				params.LaborRateID = &rateID
			}

			if _, apiErr := txRepo.Create(txCtx, departmentID, params); apiErr != nil {
				return apiErr
			}

			if len(params.ScanningStationIDs) > 0 {
				if apiErr := txRepo.SetScanningStationsDepartmentID(txCtx, departmentID, params.AccountID, params.ScanningStationIDs); apiErr != nil {
					return apiErr
				}
			}

			if len(params.MachineIDs) > 0 {
				if apiErr := txRepo.SetMachinesDepartmentID(txCtx, departmentID, params.MachineIDs); apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch to include connected sub-resources
			created, apiErr := txRepo.Get(txCtx, domain.GetDepartmentParams{
				AccountID:    params.AccountID,
				DepartmentID: departmentID,
			})
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeDepartment,
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

func (s *departmentSvcImpl) UpdateDepartment(ctx context.Context, params domain.UpdateDepartmentParams) (*domain.Department, *apierror.APIError) {
	ctx, span := departmentSvcTracer.Start(ctx, "service.department.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDepartments, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Department](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Department
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *departmentSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewDepartmentRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetDepartmentParams{
				AccountID:    params.AccountID,
				DepartmentID: params.DepartmentID,
			})
			if apiErr != nil {
				return apiErr
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.DepartmentID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A department with this name already exists.", "name")
				}
			}

			// Backfill unchanged nullable fields with existing values. Since the SQL uses direct assignment (no COALESCE) for notes, we must provide the existing value when the field was not sent.
			if params.Notes == nil {
				params.Notes = old.Notes
			}

			if params.LaborRate != nil {
				if apiErr := ValidateCostRateUnits(txCtx, txSvc.repos.NewUnitRepo(), params.LaborRate.NumeratorUnitID, params.LaborRate.DenominatorUnitID, "labor_rate"); apiErr != nil {
					return apiErr
				}
				if old.LaborRate != nil {
					// The department already owns a rate row; rewrite it in place so everything pointing at it sees the new value.
					if apiErr := txRepo.UpdateLaborRate(txCtx, old.LaborRate.ID, *params.LaborRate); apiErr != nil {
						return apiErr
					}
				} else {
					rateID, apiErr := id.GenID(id.RateIDPrefix, nil)
					if apiErr != nil {
						return apiErr
					}
					if apiErr := txRepo.InsertLaborRate(txCtx, rateID, *params.LaborRate); apiErr != nil {
						return apiErr
					}
					params.LaborRateID = &rateID
				}
			}

			updated, apiErr := txRepo.Update(txCtx, params)
			if apiErr != nil {
				return apiErr
			}

			if len(params.ScanningStationIDs) > 0 {
				if apiErr := txRepo.SetScanningStationsDepartmentID(txCtx, params.DepartmentID, params.AccountID, params.ScanningStationIDs); apiErr != nil {
					return apiErr
				}
			}

			if len(params.MachineIDs) > 0 {
				if apiErr := txRepo.SetMachinesDepartmentID(txCtx, params.DepartmentID, params.MachineIDs); apiErr != nil {
					return apiErr
				}
			}

			// Re-fetch if we connected sub-resources
			if len(params.ScanningStationIDs) > 0 || len(params.MachineIDs) > 0 {
				updated, apiErr = txRepo.Get(txCtx, domain.GetDepartmentParams{
					AccountID:    params.AccountID,
					DepartmentID: params.DepartmentID,
				})
				if apiErr != nil {
					return apiErr
				}
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeDepartment,
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

func (s *departmentSvcImpl) DeleteDepartment(ctx context.Context, departmentID string) *apierror.APIError {
	ctx, span := departmentSvcTracer.Start(ctx, "service.department.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDepartments, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	department, apiErr := s.repos.NewDepartmentRepo().Get(ctx, domain.GetDepartmentParams{
		AccountID:    accountID,
		DepartmentID: departmentID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeDepartment, departmentID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(
					span,
					apierror.NewAlreadyDeletedError("This department has already been deleted and can no longer be modified."),
				)
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *departmentSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeDepartment, department.ID, department); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewDepartmentRepo().Delete(txCtx, domain.DeleteDepartmentParams{
			AccountID:    accountID,
			DepartmentID: departmentID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(department, (*domain.Department)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeDepartment,
			ResourceID:   department.ID,
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

func (s *departmentSvcImpl) BatchGetDepartmentsByIDs(ctx context.Context, ids []string) ([]*domain.Department, *apierror.APIError) {
	ctx, span := departmentSvcTracer.Start(ctx, "service.department.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDepartments, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewDepartmentRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}
