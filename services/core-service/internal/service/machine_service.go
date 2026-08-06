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

var machineSvcTracer = tracing.GetTracer("core-service.machine_service")

type machineSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	jobSvcFactory   domain.JobSvcFactory
	txManager       TransactionManager
}

type MachineSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// JobSvcFactory (required) builds the job service the async bulk upsert records on.
	JobSvcFactory domain.JobSvcFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *MachineSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("machine service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("machine service: mediator factory is required")
	}
	if c.JobSvcFactory == nil {
		return fmt.Errorf("machine service: job service factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("machine service: tx manager is required")
	}
	return nil
}

func NewMachineSvc(config *MachineSvcConfig) domain.MachineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &machineSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		jobSvcFactory:   config.JobSvcFactory,
		txManager:       config.TxManager,
	}
}

func (s *machineSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *machineSvcImpl) withTx(ctx context.Context, fn func(context.Context, *machineSvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &machineSvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			jobSvcFactory:   s.jobSvcFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *machineSvcImpl) BatchGetMachinesByIDs(ctx context.Context, ids []string) ([]*domain.Machine, *apierror.APIError) {
	ctx, span := machineSvcTracer.Start(ctx, "service.machine.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachines, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewMachineRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

func (s *machineSvcImpl) ListMachines(ctx context.Context, params domain.ListMachinesParams) (*domain.ListMachinesResult, *apierror.APIError) {
	ctx, span := machineSvcTracer.Start(ctx, "service.machine.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachines, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewMachineRepo().List(ctx, params)
}

func (s *machineSvcImpl) GetMachine(ctx context.Context, machineID string) (*domain.Machine, *apierror.APIError) {
	ctx, span := machineSvcTracer.Start(ctx, "service.machine.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachines, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewMachineRepo().Get(ctx, domain.GetMachineParams{
		AccountID: identity.Target.AccountID,
		MachineID: machineID,
	})
}

func (s *machineSvcImpl) CreateMachine(ctx context.Context, params domain.CreateMachineParams) (*domain.Machine, *apierror.APIError) {
	ctx, span := machineSvcTracer.Start(ctx, "service.machine.create")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachines, types.ActionCreate); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machineID, apiErr := id.GenID(id.MachineIDPrefix, nil)
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Machine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Machine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *machineSvcImpl) *apierror.APIError {
			txMachineRepo := txSvc.repos.NewMachineRepo()

			exists, apiErr := txMachineRepo.ExistsByName(txCtx, params.AccountID, params.Name, nil)
			if apiErr != nil {
				return apiErr
			}
			if exists {
				return apierror.NewConflictErrorWithParam("A machine with this name already exists.", "name")
			}

			serialExists, apiErr := txMachineRepo.ExistsBySerialNumber(txCtx, params.AccountID, params.SerialNumber, nil)
			if apiErr != nil {
				return apiErr
			}
			if serialExists {
				return apierror.NewConflictErrorWithParam("A machine with this serial number already exists.", "serial_number")
			}

			// Verify department belongs to account
			_, apiErr = txSvc.repos.NewDepartmentRepo().Get(txCtx, domain.GetDepartmentParams{
				AccountID:    params.AccountID,
				DepartmentID: params.DepartmentID,
			})
			if apiErr != nil {
				return apiErr
			}

			created, apiErr := txMachineRepo.Create(txCtx, machineID, params)
			if apiErr != nil {
				return apiErr
			}
			result = created

			changes := audit.ComputeChanges(nil, created)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeMachine,
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

func (s *machineSvcImpl) UpdateMachine(ctx context.Context, params domain.UpdateMachineParams) (*domain.Machine, *apierror.APIError) {
	ctx, span := machineSvcTracer.Start(ctx, "service.machine.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachines, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.Machine](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.Machine
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *machineSvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewMachineRepo()

			old, apiErr := txRepo.Get(txCtx, domain.GetMachineParams{
				AccountID: params.AccountID,
				MachineID: params.MachineID,
			})
			if apiErr != nil {
				return apiErr
			}

			if params.Name != nil {
				exists, apiErr := txRepo.ExistsByName(txCtx, params.AccountID, *params.Name, &params.MachineID)
				if apiErr != nil {
					return apiErr
				}
				if exists {
					return apierror.NewConflictErrorWithParam("A machine with this name already exists.", "name")
				}
			}

			if params.SerialNumber != nil {
				serialExists, apiErr := txRepo.ExistsBySerialNumber(txCtx, params.AccountID, *params.SerialNumber, &params.MachineID)
				if apiErr != nil {
					return apiErr
				}
				if serialExists {
					return apierror.NewConflictErrorWithParam("A machine with this serial number already exists.", "serial_number")
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
				ResourceType: constants.ObjectTypeMachine,
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

func (s *machineSvcImpl) DeleteMachine(ctx context.Context, machineID string) *apierror.APIError {
	ctx, span := machineSvcTracer.Start(ctx, "service.machine.delete")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMachines, types.ActionDelete); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID

	machine, apiErr := s.repos.NewMachineRepo().Get(ctx, domain.GetMachineParams{
		AccountID: accountID,
		MachineID: machineID,
	})
	if apiErr != nil {
		if apierror.IsNotFound(apiErr) {
			wasDeleted, deletedCheckErr := s.repos.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeMachine, machineID)
			if deletedCheckErr != nil {
				return tracing.Trace(span, deletedCheckErr)
			}
			if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This machine has already been deleted and can no longer be modified."))
			}
		}
		return tracing.Trace(span, apiErr)
	}

	apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *machineSvcImpl) *apierror.APIError {
		if apiErr := txSvc.repos.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeMachine, machine.ID, machine); apiErr != nil {
			return apiErr
		}

		if apiErr := txSvc.repos.NewMachineRepo().Delete(txCtx, domain.DeleteMachineParams{
			AccountID: accountID,
			MachineID: machineID,
		}); apiErr != nil {
			return apiErr
		}

		changes := audit.ComputeChanges(machine, (*domain.Machine)(nil))

		if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeMachine,
			ResourceID:   machine.ID,
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
