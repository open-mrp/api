package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idemresponse"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/tracing"
)

var manufacturingMethodSvcTracer = tracing.GetTracer("core-service.manufacturing_method_service")

// maxMethodOperations and maxMethodMaterials bound one method, so every read and activation of it stays small.
const (
	maxMethodOperations = 100
	maxMethodMaterials  = 500
)

type manufacturingMethodSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type ManufacturingMethodSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *ManufacturingMethodSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("manufacturing method service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("manufacturing method service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("manufacturing method service: tx manager is required")
	}
	return nil
}

func NewManufacturingMethodSvc(config *ManufacturingMethodSvcConfig) domain.ManufacturingMethodSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &manufacturingMethodSvcImpl{repos: config.Repos, mediatorFactory: config.MediatorFactory, txManager: config.TxManager}
}

func (s *manufacturingMethodSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *manufacturingMethodSvcImpl) withTx(ctx context.Context, fn func(context.Context, *manufacturingMethodSvcImpl) *apierror.APIError) *apierror.APIError {
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		return fn(txCtx, &manufacturingMethodSvcImpl{repos: f, mediatorFactory: s.mediatorFactory, txManager: s.txManager})
	})
}

// methodIdentity is the caller, checked for an action on manufacturing methods. Reads allow include reads.
func methodIdentity(ctx context.Context, action types.Action) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	check := identity.CheckIsInternalActor
	if action == types.ActionRead {
		check = identity.CheckIsInternalActorForRead
	}
	if apiErr := check(); apiErr != nil {
		return nil, apiErr
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainManufacturingMethods, action); apiErr != nil {
		return nil, apiErr
	}
	return identity, nil
}

// idempotentWrite runs a write once per Idempotency-Key: fn runs in a transaction and its result (or error) is cached
// with it, so a retry gets the same response.
func idempotentWrite[T any](ctx context.Context, s *manufacturingMethodSvcImpl, identity *types.Identity,
	fn func(context.Context, *manufacturingMethodSvcImpl) (*T, *apierror.APIError)) (*T, *apierror.APIError) {
	meds := s.mediators()
	key, apiErr := meds.Idempotency.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, apiErr
	}
	switch domain.RecoveryPoint(key.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idemresponse.UnmarshalCachedResponse[T](ctx, key.ResponseCode, key.ResponseBody)
		if err != nil {
			return nil, apierror.NewInternalError(err, "Issue unmarshalling cached response.")
		}
		return cached.Data, cached.Error
	case domain.RecoveryPointStarted:
		var result *T
		apiErr := s.withTx(ctx, func(txCtx context.Context, txSvc *manufacturingMethodSvcImpl) *apierror.APIError {
			out, apiErr := fn(txCtx, txSvc)
			if apiErr != nil {
				return apiErr
			}
			result = out
			return txSvc.mediators().Idempotency.CacheSuccessResponse(txCtx, key.TypeID, result)
		})
		if apiErr != nil {
			return nil, meds.Idempotency.CacheErrorResponse(ctx, key.TypeID, apiErr)
		}
		return result, nil
	default:
		return nil, apierror.NewInvariantViolationError("Unexpected recovery point: " + key.RecoveryPoint)
	}
}

func (s *manufacturingMethodSvcImpl) publishAudit(ctx context.Context, action constants.AuditAction, objectType constants.ObjectType, resourceID string, before, after any) *apierror.APIError {
	return audit.NewPublisher().Publish(ctx, s.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       action,
		ResourceType: objectType,
		ResourceID:   resourceID,
		Changes:      audit.ComputeChanges(before, after),
	})
}

// getMethod reads a method, answering 410 for one that was deleted.
func (s *manufacturingMethodSvcImpl) getMethod(ctx context.Context, accountID, methodID string) (*domain.ManufacturingMethod, *apierror.APIError) {
	m, apiErr := s.repos.NewManufacturingMethodRepo().Get(ctx, accountID, methodID)
	if apiErr != nil && apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
		gone, err := s.repos.NewDeletedRecordRepo().ExistsInAccount(ctx, constants.DeletedRecordResourceTypeManufacturingMethod, methodID, accountID)
		if err != nil {
			return nil, err
		}
		if gone {
			return nil, apierror.NewGoneError("This manufacturing method has been deleted.")
		}
	}
	return m, apiErr
}

// lockDraft holds a method for an edit and refuses one that isn't a draft.
func (s *manufacturingMethodSvcImpl) lockDraft(ctx context.Context, accountID, methodID string) (string, *apierror.APIError) {
	if _, apiErr := s.getMethod(ctx, accountID, methodID); apiErr != nil {
		return "", apiErr
	}
	status, itemID, apiErr := s.repos.NewManufacturingMethodRepo().Lock(ctx, accountID, methodID)
	if apiErr != nil {
		return "", apiErr
	}
	if status != constants.ManufacturingMethodStatusDraft {
		return "", apperr.NewInvalidStateTransitionError(fmt.Sprintf(
			"This manufacturing method is %s, and only drafts can be changed. Create a new version to edit it.", status))
	}
	return itemID, nil
}

// --- methods ---

func (s *manufacturingMethodSvcImpl) ListManufacturingMethods(ctx context.Context, params domain.ListManufacturingMethodsParams) (*domain.ListManufacturingMethodsResult, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.list")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, st := range params.Statuses {
		if !st.IsValid() {
			return nil, tracing.Trace(span, apierror.NewParameterInvalidError("statuses", fmt.Sprintf("Unknown status %q.", st)))
		}
	}
	params.AccountID = identity.Target.AccountID
	result, apiErr := s.repos.NewManufacturingMethodRepo().List(ctx, params)
	return result, tracing.Trace(span, apiErr)
}

func (s *manufacturingMethodSvcImpl) BatchGetManufacturingMethodsByIDs(ctx context.Context, ids []string) ([]*domain.ManufacturingMethod, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.batch_get_by_ids")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	methods, apiErr := s.repos.NewManufacturingMethodRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
	return methods, tracing.Trace(span, apiErr)
}

func (s *manufacturingMethodSvcImpl) DefaultManufacturingMethodIDs(ctx context.Context, itemIDs []string) (map[string]string, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.default_ids")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	ids, apiErr := s.repos.NewManufacturingMethodRepo().DefaultMethodIDs(ctx, identity.Target.AccountID, itemIDs)
	return ids, tracing.Trace(span, apiErr)
}

// requireMethodItem checks the item a method makes or consumes is the account's and active.
func (s *manufacturingMethodSvcImpl) requireMethodItem(ctx context.Context, accountID, param, itemID string) (*domain.Item, *apierror.APIError) {
	items, apiErr := s.repos.NewItemRepo().GetByIDs(ctx, accountID, []string{itemID})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(items) == 0 {
		return nil, apierror.NewFieldError(param, apierror.CodeInvalidFormat, "Item not found.")
	}
	if apiErr := domain.RequireAssignable(param, "item", items[0].Status); apiErr != nil {
		return nil, apiErr
	}
	return items[0], nil
}

// requireQuantity checks a quantity's value is a non-negative decimal (positive when positive is set) in a unit that
// exists, and of the wanted dimension when one is given.
func (s *manufacturingMethodSvcImpl) requireQuantity(ctx context.Context, param string, q *domain.QuantityValue, positive bool, dimension constants.UnitDimension) *apierror.APIError {
	v, err := decimal.NewFromString(q.Value)
	if err != nil {
		return apierror.NewFieldError(param+".value", apierror.CodeInvalidFormat, "Must be a decimal number.")
	}
	if v.IsNegative() || (positive && v.IsZero()) {
		msg := "Must not be negative."
		if positive {
			msg = "Must be greater than zero."
		}
		return apierror.NewFieldError(param+".value", apierror.CodeInvalidFormat, msg)
	}
	dims, apiErr := s.repos.NewUnitRepo().GetDimensionCodes(ctx, []string{q.UnitID})
	if apiErr != nil {
		return apiErr
	}
	dim, ok := dims[q.UnitID]
	if !ok {
		return apierror.NewFieldError(param+".unit_id", apierror.CodeInvalidFormat, "Unit not found.")
	}
	if dimension != "" && dim != string(dimension) {
		return apierror.NewFieldError(param+".unit_id", apierror.CodeInvalidFormat, fmt.Sprintf("Must be a %s unit.", dimension))
	}
	q.UnitType = dim
	return nil
}

// requireTimeRate checks a time-per-output rate: a non-negative value, a time unit, and a per-unit that isn't money.
func (s *manufacturingMethodSvcImpl) requireTimeRate(ctx context.Context, param string, r *domain.RateValue) *apierror.APIError {
	v, err := decimal.NewFromString(r.Value)
	if err != nil {
		return apierror.NewFieldError(param+".value", apierror.CodeInvalidFormat, "Must be a decimal number.")
	}
	if v.IsNegative() {
		return apierror.NewFieldError(param+".value", apierror.CodeInvalidFormat, "Must not be negative.")
	}
	dims, apiErr := s.repos.NewUnitRepo().GetDimensionCodes(ctx, []string{r.NumeratorUnitID, r.DenominatorUnitID})
	if apiErr != nil {
		return apiErr
	}
	unit, ok := dims[r.NumeratorUnitID]
	if !ok {
		return apierror.NewFieldError(param+".unit_id", apierror.CodeInvalidFormat, "Unit not found.")
	}
	if unit != string(constants.UnitDimensionTime) {
		return apierror.NewFieldError(param+".unit_id", apierror.CodeInvalidFormat, "Must be a time unit, such as hours.")
	}
	per, ok := dims[r.DenominatorUnitID]
	if !ok {
		return apierror.NewFieldError(param+".per_unit_id", apierror.CodeInvalidFormat, "Unit not found.")
	}
	if per == unitDimensionCodeCurrency {
		return apierror.NewFieldError(param+".per_unit_id", apierror.CodeInvalidFormat, "Must be a unit of output, not a currency.")
	}
	return nil
}

// requirePercent checks a whole-number percent string: a decimal above -100.
func requirePercent(param, v string) *apierror.APIError {
	d, err := decimal.NewFromString(v)
	if err != nil {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "Must be a decimal number of percent, such as \"15\".")
	}
	if d.LessThanOrEqual(decimal.NewFromInt(-100)) {
		return apierror.NewFieldError(param, apierror.CodeInvalidFormat, "Must be greater than -100.")
	}
	return nil
}

func (s *manufacturingMethodSvcImpl) CreateManufacturingMethod(ctx context.Context, params domain.CreateManufacturingMethodParams) (*domain.ManufacturingMethod, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.create")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	methodID, apiErr := id.GenID(id.ManufacturingMethodIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := idempotentWrite(ctx, s, identity, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) (*domain.ManufacturingMethod, *apierror.APIError) {
		if _, apiErr := tx.requireMethodItem(txCtx, accountID, "item_id", params.ItemID); apiErr != nil {
			return nil, apiErr
		}
		output, apiErr := tx.outputQuantity(txCtx, accountID, params.ItemID, params.OutputQuantity)
		if apiErr != nil {
			return nil, apiErr
		}
		repo := tx.repos.NewManufacturingMethodRepo()
		version, apiErr := repo.NextVersion(txCtx, params.ItemID)
		if apiErr != nil {
			return nil, apiErr
		}
		m := &domain.ManufacturingMethod{
			ID: methodID, AccountID: accountID, ItemID: params.ItemID, Version: version,
			Status: constants.ManufacturingMethodStatusDraft, OutputQuantity: output,
		}
		if apiErr := repo.Insert(txCtx, m); apiErr != nil {
			return nil, apiErr
		}
		created, apiErr := repo.Get(txCtx, accountID, methodID)
		if apiErr != nil {
			return nil, apiErr
		}
		return created, tx.publishAudit(txCtx, constants.AuditActionCreate, constants.ObjectTypeManufacturingMethod, methodID, nil, created)
	})
	return result, tracing.Trace(span, apiErr)
}

// outputQuantity is a method's output: the one given, validated, or one of the item's stocking unit.
func (s *manufacturingMethodSvcImpl) outputQuantity(ctx context.Context, accountID, itemID string, given *domain.QuantityValue) (domain.QuantityValue, *apierror.APIError) {
	if given != nil {
		q := *given
		return q, s.requireQuantity(ctx, "output_quantity", &q, true, "")
	}
	unit, apiErr := s.repos.NewItemRepo().GetStockingUnit(ctx, accountID, itemID)
	if apiErr != nil {
		return domain.QuantityValue{}, apiErr
	}
	q := domain.QuantityValue{Value: "1", UnitID: unit.BaseUnitID}
	return q, s.requireQuantity(ctx, "output_quantity", &q, true, "")
}

func (s *manufacturingMethodSvcImpl) UpdateManufacturingMethod(ctx context.Context, params domain.UpdateManufacturingMethodParams) (*domain.ManufacturingMethod, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.update")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	result, apiErr := idempotentWrite(ctx, s, identity, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) (*domain.ManufacturingMethod, *apierror.APIError) {
		if _, apiErr := tx.lockDraft(txCtx, accountID, params.MethodID); apiErr != nil {
			return nil, apiErr
		}
		repo := tx.repos.NewManufacturingMethodRepo()
		before, apiErr := repo.Get(txCtx, accountID, params.MethodID)
		if apiErr != nil {
			return nil, apiErr
		}
		if params.OutputQuantity != nil {
			q := *params.OutputQuantity
			if apiErr := tx.requireQuantity(txCtx, "output_quantity", &q, true, ""); apiErr != nil {
				return nil, apiErr
			}
			if apiErr := repo.UpdateOutput(txCtx, accountID, params.MethodID, q); apiErr != nil {
				return nil, apiErr
			}
		}
		after, apiErr := repo.Get(txCtx, accountID, params.MethodID)
		if apiErr != nil {
			return nil, apiErr
		}
		return after, tx.publishAudit(txCtx, constants.AuditActionUpdate, constants.ObjectTypeManufacturingMethod, after.ID, before, after)
	})
	return result, tracing.Trace(span, apiErr)
}

func (s *manufacturingMethodSvcImpl) DeleteManufacturingMethod(ctx context.Context, methodID string) *apierror.APIError {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.delete")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionDelete)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	apiErr = s.withTx(ctx, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) *apierror.APIError {
		repo := tx.repos.NewManufacturingMethodRepo()
		m, apiErr := tx.getMethod(txCtx, accountID, methodID)
		if apiErr != nil {
			return apiErr
		}
		status, _, apiErr := repo.Lock(txCtx, accountID, methodID)
		if apiErr != nil {
			return apiErr
		}
		if status != constants.ManufacturingMethodStatusDraft {
			return apperr.NewInvalidStateTransitionError(fmt.Sprintf(
				"This manufacturing method is %s, and only drafts can be deleted. Active and archived versions are kept for history.", status))
		}
		if apiErr := tx.deleteMembers(txCtx, accountID, methodID); apiErr != nil {
			return apiErr
		}
		if apiErr := tx.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeManufacturingMethod, methodID, accountID, m); apiErr != nil {
			return apiErr
		}
		if apiErr := repo.Delete(txCtx, accountID, methodID); apiErr != nil {
			return apiErr
		}
		return tx.publishAudit(txCtx, constants.AuditActionDelete, constants.ObjectTypeManufacturingMethod, methodID, m, (*domain.ManufacturingMethod)(nil))
	})
	return tracing.Trace(span, apiErr)
}

// deleteMembers removes a draft's materials and operations with the value rows they own.
func (s *manufacturingMethodSvcImpl) deleteMembers(ctx context.Context, accountID, methodID string) *apierror.APIError {
	repo := s.repos.NewManufacturingMethodRepo()
	materials, apiErr := s.allMaterials(ctx, accountID, methodID)
	if apiErr != nil {
		return apiErr
	}
	for _, mat := range materials {
		if apiErr := repo.DeleteMaterial(ctx, mat); apiErr != nil {
			return apiErr
		}
	}
	ops, apiErr := s.allOperations(ctx, accountID, methodID)
	if apiErr != nil {
		return apiErr
	}
	for _, op := range ops {
		if apiErr := repo.DeleteOperation(ctx, accountID, op); apiErr != nil {
			return apiErr
		}
	}
	return nil
}

func (s *manufacturingMethodSvcImpl) allOperations(ctx context.Context, accountID, methodID string) ([]*domain.MethodOperation, *apierror.APIError) {
	repo := s.repos.NewManufacturingMethodRepo()
	ids, apiErr := repo.ListOperationIDs(ctx, methodID)
	if apiErr != nil {
		return nil, apiErr
	}
	return repo.GetOperationsByIDs(ctx, accountID, ids)
}

func (s *manufacturingMethodSvcImpl) allMaterials(ctx context.Context, accountID, methodID string) ([]*domain.MethodMaterial, *apierror.APIError) {
	res, apiErr := s.repos.NewManufacturingMethodRepo().ListMaterials(ctx, domain.ListMethodMembersParams{
		AccountID: accountID, MethodID: methodID, Limit: maxMethodMaterials,
	})
	if apiErr != nil {
		return nil, apiErr
	}
	return res.Materials, nil
}

// ActivateManufacturingMethod makes a draft the item's method for new production. The item's other active standard
// methods are archived, and execution moves to the new version: its last operation produces the item, its operations
// take their machines, and its operations link to the methods that make what they consume and use what it makes.
func (s *manufacturingMethodSvcImpl) ActivateManufacturingMethod(ctx context.Context, methodID string) (*domain.ManufacturingMethod, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.activate")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	result, apiErr := idempotentWrite(ctx, s, identity, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) (*domain.ManufacturingMethod, *apierror.APIError) {
		repo := tx.repos.NewManufacturingMethodRepo()
		before, apiErr := tx.getMethod(txCtx, accountID, methodID)
		if apiErr != nil {
			return nil, apiErr
		}
		status, itemID, apiErr := repo.Lock(txCtx, accountID, methodID)
		if apiErr != nil {
			return nil, apiErr
		}
		if status != constants.ManufacturingMethodStatusDraft {
			return nil, apperr.NewInvalidStateTransitionError(fmt.Sprintf("This manufacturing method is %s; only a draft can be activated.", status))
		}
		ops, apiErr := tx.allOperations(txCtx, accountID, methodID)
		if apiErr != nil {
			return nil, apiErr
		}
		if len(ops) == 0 {
			return nil, apperr.NewInvalidStateTransitionError("A manufacturing method needs at least one operation before it can be activated.")
		}
		for _, op := range ops {
			if op.WorkCenterID == nil {
				return nil, apperr.NewInvalidStateTransitionError(fmt.Sprintf("Operation %s has no work center; set one before activating.", op.ID))
			}
		}
		if _, apiErr := tx.requireMethodItem(txCtx, accountID, "item", itemID); apiErr != nil {
			return nil, apiErr
		}

		previous, apiErr := repo.ActiveStandardIDsForItem(txCtx, accountID, itemID)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, prevID := range previous {
			if apiErr := tx.archive(txCtx, accountID, prevID); apiErr != nil {
				return nil, apiErr
			}
		}
		if apiErr := tx.project(txCtx, accountID, before, ops); apiErr != nil {
			return nil, apiErr
		}
		now := time.Now().UTC()
		if apiErr := repo.SetStatus(txCtx, accountID, methodID, constants.ManufacturingMethodStatusActive, &now); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := event.PublishItemCostBasisChanged(txCtx, tx.repos, accountID, itemID, event.CostBasisManufacturingMethodActivated); apiErr != nil {
			return nil, apiErr
		}
		after, apiErr := repo.Get(txCtx, accountID, methodID)
		if apiErr != nil {
			return nil, apiErr
		}
		return after, tx.publishAudit(txCtx, constants.AuditActionUpdate, constants.ObjectTypeManufacturingMethod, methodID, before, after)
	})
	return result, tracing.Trace(span, apiErr)
}

// archive retires an active method: it stops producing its item for execution and keeps its history.
func (s *manufacturingMethodSvcImpl) archive(ctx context.Context, accountID, methodID string) *apierror.APIError {
	repo := s.repos.NewManufacturingMethodRepo()
	before, apiErr := repo.Get(ctx, accountID, methodID)
	if apiErr != nil {
		return apiErr
	}
	opIDs, apiErr := repo.ListOperationIDs(ctx, methodID)
	if apiErr != nil {
		return apiErr
	}
	for _, opID := range opIDs {
		if apiErr := repo.RemoveProjection(ctx, opID); apiErr != nil {
			return apiErr
		}
	}
	if apiErr := repo.SetStatus(ctx, accountID, methodID, constants.ManufacturingMethodStatusArchived, nil); apiErr != nil {
		return apiErr
	}
	after, apiErr := repo.Get(ctx, accountID, methodID)
	if apiErr != nil {
		return apiErr
	}
	return s.publishAudit(ctx, constants.AuditActionUpdate, constants.ObjectTypeManufacturingMethod, methodID, before, after)
}

// operationGroups splits operations, in order, into the groups that run together: an operation that runs alongside the
// previous one joins its group.
func operationGroups(ops []*domain.MethodOperation) [][]*domain.MethodOperation {
	var groups [][]*domain.MethodOperation
	for _, op := range ops {
		if len(groups) > 0 && op.Sequence == constants.OperationSequenceWithPrevious {
			groups[len(groups)-1] = append(groups[len(groups)-1], op)
			continue
		}
		groups = append(groups, []*domain.MethodOperation{op})
	}
	return groups
}

// project hands a method's operations to execution: the last operation produces the item, each operation takes its
// machines, operations link to the methods that make their materials and to whatever consumes the item, and each group
// of operations feeds the next.
func (s *manufacturingMethodSvcImpl) project(ctx context.Context, accountID string, m *domain.ManufacturingMethod, ops []*domain.MethodOperation) *apierror.APIError {
	repo := s.repos.NewManufacturingMethodRepo()
	last := ops[len(ops)-1]
	quantityID, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	productionID, apiErr := id.GenID(id.ProductionIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	if apiErr := repo.InsertQuantity(ctx, quantityID, m.OutputQuantity); apiErr != nil {
		return apiErr
	}
	if apiErr := repo.InsertProduction(ctx, productionID, m.ItemID, quantityID, last.ID); apiErr != nil {
		return apiErr
	}
	flow := s.mediators().ProductionFlow
	for _, op := range ops {
		if apiErr := repo.AssignMachinesToStep(ctx, accountID, op.ID, op.MachineIDs); apiErr != nil {
			return apiErr
		}
		if apiErr := flow.LinkFlow(ctx, op.ID, accountID); apiErr != nil {
			return apiErr
		}
	}
	groups := operationGroups(ops)
	for i := 1; i < len(groups); i++ {
		for _, up := range groups[i-1] {
			for _, down := range groups[i] {
				if apiErr := repo.ConnectSteps(ctx, up.ID, down.ID); apiErr != nil {
					return apiErr
				}
			}
		}
	}
	return nil
}

// NewManufacturingMethodVersion copies a method into a new draft of the same item, operations and materials included.
func (s *manufacturingMethodSvcImpl) NewManufacturingMethodVersion(ctx context.Context, methodID string) (*domain.ManufacturingMethod, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.new_version")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	newID, apiErr := id.GenID(id.ManufacturingMethodIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := idempotentWrite(ctx, s, identity, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) (*domain.ManufacturingMethod, *apierror.APIError) {
		repo := tx.repos.NewManufacturingMethodRepo()
		src, apiErr := tx.getMethod(txCtx, accountID, methodID)
		if apiErr != nil {
			return nil, apiErr
		}
		version, apiErr := repo.NextVersion(txCtx, src.ItemID)
		if apiErr != nil {
			return nil, apiErr
		}
		draft := &domain.ManufacturingMethod{
			ID: newID, AccountID: accountID, ItemID: src.ItemID, Version: version,
			Status: constants.ManufacturingMethodStatusDraft, OutputQuantity: src.OutputQuantity,
		}
		if apiErr := repo.Insert(txCtx, draft); apiErr != nil {
			return nil, apiErr
		}
		ops, apiErr := tx.allOperations(txCtx, accountID, methodID)
		if apiErr != nil {
			return nil, apiErr
		}
		opMap := make(map[string]string, len(ops))
		for _, op := range ops {
			copied := *op
			copied.MethodID = newID
			if copied.Type == "" {
				copied.Type = constants.OperationTypeInside
			}
			newOpID, apiErr := tx.insertOperation(txCtx, accountID, draft, &copied)
			if apiErr != nil {
				return nil, apiErr
			}
			opMap[op.ID] = newOpID
		}
		materials, apiErr := tx.allMaterials(txCtx, accountID, methodID)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, mat := range materials {
			copied := *mat
			copied.MethodID = newID
			if mat.OperationID != nil {
				if mapped, ok := opMap[*mat.OperationID]; ok {
					copied.OperationID = &mapped
				}
			}
			if apiErr := tx.insertMaterial(txCtx, &copied); apiErr != nil {
				return nil, apiErr
			}
		}
		created, apiErr := repo.Get(txCtx, accountID, newID)
		if apiErr != nil {
			return nil, apiErr
		}
		return created, tx.publishAudit(txCtx, constants.AuditActionCreate, constants.ObjectTypeManufacturingMethod, newID, nil, created)
	})
	return result, tracing.Trace(span, apiErr)
}

// GetProductionFlow walks an item's default method down through the methods of the materials it makes.
func (s *manufacturingMethodSvcImpl) GetProductionFlow(ctx context.Context, itemID string) (*domain.ProductionFlow, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.production_flow")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	items, apiErr := s.repos.NewItemRepo().GetByIDs(ctx, accountID, []string{itemID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(items) == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Item not found."))
	}

	repo := s.repos.NewManufacturingMethodRepo()
	flow := &domain.ProductionFlow{ItemID: itemID, MethodIDs: []string{}}
	seenItems := map[string]bool{itemID: true}
	frontier := []string{itemID}
	// A bill of materials deeper than this is a cycle someone wrote by hand.
	for depth := 0; depth < 25 && len(frontier) > 0; depth++ {
		defaults, apiErr := repo.DefaultMethodIDs(ctx, accountID, frontier)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		var methodIDs []string
		for _, item := range frontier {
			if mid, ok := defaults[item]; ok {
				methodIDs = append(methodIDs, mid)
			}
		}
		flow.MethodIDs = append(flow.MethodIDs, methodIDs...)
		var next []string
		for _, mid := range methodIDs {
			materials, apiErr := s.allMaterials(ctx, accountID, mid)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			for _, mat := range materials {
				if mat.Supply == constants.SupplyMake && !seenItems[mat.ItemID] {
					seenItems[mat.ItemID] = true
					next = append(next, mat.ItemID)
				}
			}
		}
		frontier = next
	}
	return flow, nil
}

// --- operations ---

func (s *manufacturingMethodSvcImpl) ListMethodOperations(ctx context.Context, params domain.ListMethodMembersParams) (*domain.ListMethodOperationsResult, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.list_operations")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID
	if _, apiErr := s.getMethod(ctx, params.AccountID, params.MethodID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, apiErr := s.repos.NewManufacturingMethodRepo().ListOperations(ctx, params)
	return result, tracing.Trace(span, apiErr)
}

func (s *manufacturingMethodSvcImpl) BatchGetMethodOperationsByIDs(ctx context.Context, ids []string) ([]*domain.MethodOperation, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.batch_get_operations")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	ops, apiErr := s.repos.NewManufacturingMethodRepo().GetOperationsByIDs(ctx, identity.Target.AccountID, ids)
	return ops, tracing.Trace(span, apiErr)
}

// operationName is the name an operation's production step row carries, which only the preview.9 production step
// routes show. Names are unique per account.
func (s *manufacturingMethodSvcImpl) operationName(ctx context.Context, accountID string, m *domain.ManufacturingMethod, opID string, position int32) (string, *apierror.APIError) {
	sku := m.ItemID
	if items, apiErr := s.repos.NewItemRepo().GetByIDs(ctx, accountID, []string{m.ItemID}); apiErr != nil {
		return "", apiErr
	} else if len(items) > 0 {
		sku = items[0].SKU
	}
	if len(sku) > 180 {
		sku = sku[:180]
	}
	name := fmt.Sprintf("%s v%d op %d", sku, m.Version, position)
	taken, apiErr := s.repos.NewManufacturingMethodRepo().OperationNameTaken(ctx, accountID, name)
	if apiErr != nil {
		return "", apiErr
	}
	if taken {
		name = fmt.Sprintf("%s v%d %s", sku, m.Version, opID)
	}
	return name, nil
}

// insertOperation writes a complete operation into a method, with fresh value rows, and returns its ID.
func (s *manufacturingMethodSvcImpl) insertOperation(ctx context.Context, accountID string, m *domain.ManufacturingMethod, op *domain.MethodOperation) (string, *apierror.APIError) {
	repo := s.repos.NewManufacturingMethodRepo()
	opID, apiErr := id.GenID(id.MethodOperationIDPrefix, nil)
	if apiErr != nil {
		return "", apiErr
	}
	op.ID = opID
	if apiErr := s.writeValueRows(ctx, op); apiErr != nil {
		return "", apiErr
	}
	name, apiErr := s.operationName(ctx, accountID, m, opID, op.Position)
	if apiErr != nil {
		return "", apiErr
	}
	if apiErr := repo.InsertOperation(ctx, accountID, name, op); apiErr != nil {
		return "", apiErr
	}
	return opID, repo.SetOperationMachines(ctx, accountID, opID, op.MachineIDs)
}

// writeValueRows stores an operation's rates and setup time as fresh rows and points the operation at them. The money
// rates keep a legacy rate row beside their columns for code that still reads it.
func (s *manufacturingMethodSvcImpl) writeValueRows(ctx context.Context, op *domain.MethodOperation) *apierror.APIError {
	repo := s.repos.NewManufacturingMethodRepo()
	units := s.repos.NewUnitRepo()
	newRate := func(v domain.RateValue) (string, *apierror.APIError) {
		rid, apiErr := id.GenID(id.RateIDPrefix, nil)
		if apiErr != nil {
			return "", apiErr
		}
		return rid, repo.InsertRate(ctx, rid, v)
	}
	money := func(p *domain.UnitPrice, param string) (string, *apierror.APIError) {
		legacy, apiErr := validatedLegacyCurrencyPrice(ctx, units, *p, param)
		if apiErr != nil {
			return "", apiErr
		}
		return newRate(domain.RateValue{Value: legacy.Value, NumeratorUnitID: legacy.NumeratorUnitID, DenominatorUnitID: legacy.DenominatorUnitID})
	}
	var apiErr *apierror.APIError
	if op.LaborRateID, apiErr = money(op.LaborRate, "labor_rate"); apiErr != nil {
		return apiErr
	}
	if op.OverheadRateID, apiErr = money(op.OverheadRate, "overhead_rate"); apiErr != nil {
		return apiErr
	}
	if op.LaborTimeID, apiErr = newRate(*op.LaborTime); apiErr != nil {
		return apiErr
	}
	op.MachineTimeID = nil
	if op.MachineTime != nil {
		rid, apiErr := newRate(*op.MachineTime)
		if apiErr != nil {
			return apiErr
		}
		op.MachineTimeID = &rid
	}
	op.SetupTimeQuantityID = nil
	if op.SetupTime != nil {
		qid, apiErr := id.GenID(id.QuantityIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := repo.InsertQuantity(ctx, qid, *op.SetupTime); apiErr != nil {
			return apiErr
		}
		op.SetupTimeQuantityID = &qid
	}
	return nil
}

// applyOperationInput merges an operation write into op and validates the result. creating fills defaults.
func (s *manufacturingMethodSvcImpl) applyOperationInput(ctx context.Context, accountID string, op *domain.MethodOperation, in domain.MethodOperationInput, creating bool) *apierror.APIError {
	if in.Sequence != nil {
		if !in.Sequence.IsValid() {
			return apierror.NewFieldError("sequence", apierror.CodeInvalidFormat, "Must be after_previous or with_previous.")
		}
		op.Sequence = *in.Sequence
	}
	var workCenterLabor *domain.UnitPrice
	if in.WorkCenterID != nil {
		departments, apiErr := s.repos.NewDepartmentRepo().GetByIDs(ctx, accountID, []string{*in.WorkCenterID})
		if apiErr != nil {
			return apiErr
		}
		if len(departments) == 0 {
			return apierror.NewFieldError("work_center_id", apierror.CodeInvalidFormat, "Work center not found.")
		}
		wc := *in.WorkCenterID
		op.WorkCenterID = &wc
		workCenterLabor = departments[0].LaborRate
	} else if creating {
		return apierror.NewMissingFieldError("work_center_id", "work_center_id is required for an inside operation.")
	}
	if in.MachineIDs != nil {
		ids := slices.Compact(slices.Clone(*in.MachineIDs))
		added := make([]string, 0, len(ids))
		for _, m := range ids {
			if !slices.Contains(op.MachineIDs, m) {
				added = append(added, m)
			}
		}
		if apiErr := requireAssignableMachines(ctx, s.repos, accountID, "machine_ids", added); apiErr != nil {
			return apiErr
		}
		op.MachineIDs = ids
	}
	if v, ok := in.ScanningStationID.Value(); ok {
		if apiErr := requireKnownScanningStations(ctx, s.repos, accountID, "scanning_station_id", []string{v}); apiErr != nil {
			return apiErr
		}
		op.ScanningStationID = &v
	} else if in.ScanningStationID.IsClear() {
		op.ScanningStationID = nil
	}
	if v, ok := in.SetupTime.Value(); ok {
		if apiErr := s.requireQuantity(ctx, "setup_time", &v, false, constants.UnitDimensionTime); apiErr != nil {
			return apiErr
		}
		op.SetupTime = &v
	} else if in.SetupTime.IsClear() {
		op.SetupTime = nil
	}
	if in.LaborTime != nil {
		v := *in.LaborTime
		if apiErr := s.requireTimeRate(ctx, "labor_time", &v); apiErr != nil {
			return apiErr
		}
		op.LaborTime = &v
	} else if creating {
		return apierror.NewMissingFieldError("labor_time", "labor_time is required.")
	}
	if v, ok := in.MachineTime.Value(); ok {
		if apiErr := s.requireTimeRate(ctx, "machine_time", &v); apiErr != nil {
			return apiErr
		}
		op.MachineTime = &v
	} else if in.MachineTime.IsClear() {
		op.MachineTime = nil
	}
	switch {
	case in.LaborRate != nil:
		v := *in.LaborRate
		op.LaborRate = &v
	case creating && workCenterLabor != nil:
		v := *workCenterLabor
		op.LaborRate = &v
	case creating:
		return apierror.NewMissingFieldError("labor_rate", "labor_rate is required when the work center has no labor rate.")
	}
	switch {
	case in.OverheadRate != nil:
		v := *in.OverheadRate
		op.OverheadRate = &v
	case creating:
		op.OverheadRate = &domain.UnitPrice{Amount: "0", Currency: op.LaborRate.Currency, PerUnitID: op.LaborRate.PerUnitID}
	}
	if in.LevelingPercent != nil {
		if apiErr := requirePercent("leveling_percent", *in.LevelingPercent); apiErr != nil {
			return apiErr
		}
		op.LevelingPercent = *in.LevelingPercent
	}
	if in.AllowancePercent != nil {
		if apiErr := requirePercent("allowance_percent", *in.AllowancePercent); apiErr != nil {
			return apiErr
		}
		op.AllowancePercent = *in.AllowancePercent
	}
	if v, ok := in.WorkInstructions.Value(); ok {
		op.WorkInstructions = &v
	} else if in.WorkInstructions.IsClear() {
		op.WorkInstructions = nil
	}
	return nil
}

// placeOperation moves an operation to position among ops (the method's operations, in order, without it), shifting
// the others, and returns where it landed: clamped to the end, and appended when position is nil.
func (s *manufacturingMethodSvcImpl) placeOperation(ctx context.Context, accountID string, ops []*domain.MethodOperation, opID string, position *int32) (int32, *apierror.APIError) {
	n := int32(len(ops)) // #nosec G115 -- a method has at most maxMethodOperations
	target := n + 1
	if position != nil {
		if *position < 1 {
			return 0, apierror.NewFieldError("position", apierror.CodeInvalidFormat, "Must be 1 or more.")
		}
		target = min(*position, n+1)
	}
	repo := s.repos.NewManufacturingMethodRepo()
	for i, other := range ops {
		want := int32(i) + 1 // #nosec G115 -- bounded as above
		if want >= target {
			want++
		}
		if other.Position != want {
			if apiErr := repo.SetOperationPosition(ctx, accountID, other.ID, want); apiErr != nil {
				return 0, apiErr
			}
		}
	}
	if opID != "" {
		if apiErr := repo.SetOperationPosition(ctx, accountID, opID, target); apiErr != nil {
			return 0, apiErr
		}
	}
	return target, nil
}

func (s *manufacturingMethodSvcImpl) CreateMethodOperation(ctx context.Context, params domain.CreateMethodOperationParams) (*domain.MethodOperation, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.create_operation")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	result, apiErr := idempotentWrite(ctx, s, identity, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) (*domain.MethodOperation, *apierror.APIError) {
		if _, apiErr := tx.lockDraft(txCtx, accountID, params.MethodID); apiErr != nil {
			return nil, apiErr
		}
		repo := tx.repos.NewManufacturingMethodRepo()
		m, apiErr := repo.Get(txCtx, accountID, params.MethodID)
		if apiErr != nil {
			return nil, apiErr
		}
		ops, apiErr := tx.allOperations(txCtx, accountID, params.MethodID)
		if apiErr != nil {
			return nil, apiErr
		}
		if len(ops) >= maxMethodOperations {
			return nil, apierror.NewConflictError(fmt.Sprintf("A manufacturing method can have at most %d operations.", maxMethodOperations))
		}
		op := &domain.MethodOperation{
			MethodID:         params.MethodID,
			Sequence:         constants.OperationSequenceAfterPrevious,
			Type:             constants.OperationTypeInside,
			MachineIDs:       []string{},
			LevelingPercent:  "0",
			AllowancePercent: "0",
		}
		if apiErr := tx.applyOperationInput(txCtx, accountID, op, params.MethodOperationInput, true); apiErr != nil {
			return nil, apiErr
		}
		position, apiErr := tx.placeOperation(txCtx, accountID, ops, "", params.Position)
		if apiErr != nil {
			return nil, apiErr
		}
		op.Position = position
		opID, apiErr := tx.insertOperation(txCtx, accountID, m, op)
		if apiErr != nil {
			return nil, apiErr
		}
		created, apiErr := tx.getOperation(txCtx, accountID, params.MethodID, opID)
		if apiErr != nil {
			return nil, apiErr
		}
		return created, tx.publishAudit(txCtx, constants.AuditActionCreate, constants.ObjectTypeMethodOperation, opID, nil, created)
	})
	return result, tracing.Trace(span, apiErr)
}

// getOperation reads one operation of a method.
func (s *manufacturingMethodSvcImpl) getOperation(ctx context.Context, accountID, methodID, opID string) (*domain.MethodOperation, *apierror.APIError) {
	ops, apiErr := s.repos.NewManufacturingMethodRepo().GetOperationsByIDs(ctx, accountID, []string{opID})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(ops) == 0 || ops[0].MethodID != methodID {
		return nil, apierror.NewNotFoundError("Operation not found.")
	}
	return ops[0], nil
}

func (s *manufacturingMethodSvcImpl) UpdateMethodOperation(ctx context.Context, params domain.UpdateMethodOperationParams) (*domain.MethodOperation, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.update_operation")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	result, apiErr := idempotentWrite(ctx, s, identity, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) (*domain.MethodOperation, *apierror.APIError) {
		if _, apiErr := tx.lockDraft(txCtx, accountID, params.MethodID); apiErr != nil {
			return nil, apiErr
		}
		repo := tx.repos.NewManufacturingMethodRepo()
		before, apiErr := tx.getOperation(txCtx, accountID, params.MethodID, params.OperationID)
		if apiErr != nil {
			return nil, apiErr
		}
		op := *before
		op.MachineIDs = slices.Clone(before.MachineIDs)
		if apiErr := tx.applyOperationInput(txCtx, accountID, &op, params.MethodOperationInput, false); apiErr != nil {
			return nil, apiErr
		}
		oldValueRates := []string{before.LaborRateID, before.LaborTimeID, before.OverheadRateID}
		if before.MachineTimeID != nil {
			oldValueRates = append(oldValueRates, *before.MachineTimeID)
		}
		var oldQuantities []string
		if before.SetupTimeQuantityID != nil {
			oldQuantities = append(oldQuantities, *before.SetupTimeQuantityID)
		}
		if apiErr := tx.writeValueRows(txCtx, &op); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := repo.UpdateOperation(txCtx, accountID, &op); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := repo.DeleteRates(txCtx, oldValueRates); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := repo.DeleteQuantities(txCtx, oldQuantities); apiErr != nil {
			return nil, apiErr
		}
		if params.MachineIDs != nil {
			if apiErr := repo.SetOperationMachines(txCtx, accountID, op.ID, op.MachineIDs); apiErr != nil {
				return nil, apiErr
			}
		}
		if params.Position != nil {
			ops, apiErr := tx.allOperations(txCtx, accountID, params.MethodID)
			if apiErr != nil {
				return nil, apiErr
			}
			others := slices.DeleteFunc(ops, func(o *domain.MethodOperation) bool { return o.ID == op.ID })
			if _, apiErr := tx.placeOperation(txCtx, accountID, others, op.ID, params.Position); apiErr != nil {
				return nil, apiErr
			}
		}
		after, apiErr := tx.getOperation(txCtx, accountID, params.MethodID, op.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		return after, tx.publishAudit(txCtx, constants.AuditActionUpdate, constants.ObjectTypeMethodOperation, op.ID, before, after)
	})
	return result, tracing.Trace(span, apiErr)
}

func (s *manufacturingMethodSvcImpl) DeleteMethodOperation(ctx context.Context, methodID, operationID string) *apierror.APIError {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.delete_operation")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	apiErr = s.withTx(ctx, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) *apierror.APIError {
		repo := tx.repos.NewManufacturingMethodRepo()
		op, apiErr := tx.getOperation(txCtx, accountID, methodID, operationID)
		if apiErr != nil {
			if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
				gone, err := tx.repos.NewDeletedRecordRepo().ExistsInAccount(txCtx, constants.DeletedRecordResourceTypeMethodOperation, operationID, accountID)
				if err != nil {
					return err
				}
				if gone {
					return apierror.NewGoneError("This operation has been deleted.")
				}
			}
			return apiErr
		}
		if _, apiErr := tx.lockDraft(txCtx, accountID, methodID); apiErr != nil {
			return apiErr
		}
		n, apiErr := repo.CountOperationMaterials(txCtx, operationID)
		if apiErr != nil {
			return apiErr
		}
		if n > 0 {
			return apierror.NewInUseError("Materials are consumed at this operation. Move them to another operation or delete them first.")
		}
		if apiErr := repo.DeleteOperation(txCtx, accountID, op); apiErr != nil {
			return apiErr
		}
		ops, apiErr := tx.allOperations(txCtx, accountID, methodID)
		if apiErr != nil {
			return apiErr
		}
		for i, other := range ops {
			if want := int32(i) + 1; other.Position != want { // #nosec G115 -- bounded by maxMethodOperations
				if apiErr := repo.SetOperationPosition(txCtx, accountID, other.ID, want); apiErr != nil {
					return apiErr
				}
			}
		}
		if apiErr := tx.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeMethodOperation, operationID, accountID, op); apiErr != nil {
			return apiErr
		}
		return tx.publishAudit(txCtx, constants.AuditActionDelete, constants.ObjectTypeMethodOperation, operationID, op, (*domain.MethodOperation)(nil))
	})
	return tracing.Trace(span, apiErr)
}

// --- materials ---

func (s *manufacturingMethodSvcImpl) ListMethodMaterials(ctx context.Context, params domain.ListMethodMembersParams) (*domain.ListMethodMaterialsResult, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.list_materials")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.AccountID = identity.Target.AccountID
	if _, apiErr := s.getMethod(ctx, params.AccountID, params.MethodID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, apiErr := s.repos.NewManufacturingMethodRepo().ListMaterials(ctx, params)
	return result, tracing.Trace(span, apiErr)
}

func (s *manufacturingMethodSvcImpl) BatchGetMethodMaterialsByIDs(ctx context.Context, ids []string) ([]*domain.MethodMaterial, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.batch_get_materials")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	materials, apiErr := s.repos.NewManufacturingMethodRepo().GetMaterialsByIDs(ctx, identity.Target.AccountID, ids)
	return materials, tracing.Trace(span, apiErr)
}

// insertMaterial writes a complete material, with fresh quantity rows.
func (s *manufacturingMethodSvcImpl) insertMaterial(ctx context.Context, mat *domain.MethodMaterial) *apierror.APIError {
	repo := s.repos.NewManufacturingMethodRepo()
	materialID, apiErr := id.GenID(id.MethodMaterialIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	mat.ID = materialID
	if apiErr := s.writeMaterialQuantities(ctx, mat); apiErr != nil {
		return apiErr
	}
	return repo.InsertMaterial(ctx, mat)
}

func (s *manufacturingMethodSvcImpl) writeMaterialQuantities(ctx context.Context, mat *domain.MethodMaterial) *apierror.APIError {
	repo := s.repos.NewManufacturingMethodRepo()
	qid, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	sid, apiErr := id.GenID(id.QuantityIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	if apiErr := repo.InsertQuantity(ctx, qid, mat.Quantity); apiErr != nil {
		return apiErr
	}
	if apiErr := repo.InsertQuantity(ctx, sid, mat.ScrapQuantity); apiErr != nil {
		return apiErr
	}
	mat.QuantityID, mat.ScrapQuantityID = qid, sid
	return nil
}

// applyMaterialInput merges a material write into mat and validates it against the method. creating fills defaults.
func (s *manufacturingMethodSvcImpl) applyMaterialInput(ctx context.Context, accountID string, m *domain.ManufacturingMethod, mat *domain.MethodMaterial, in domain.MethodMaterialInput, creating bool) *apierror.APIError {
	if in.ItemID != nil {
		if *in.ItemID == m.ItemID {
			return apierror.NewFieldError("item_id", apierror.CodeInvalidFormat, "A method can't consume the item it makes.")
		}
		if _, apiErr := s.requireMethodItem(ctx, accountID, "item_id", *in.ItemID); apiErr != nil {
			return apiErr
		}
		mat.ItemID = *in.ItemID
	} else if creating {
		return apierror.NewMissingFieldError("item_id", "item_id is required.")
	}
	if in.Quantity != nil {
		q := *in.Quantity
		if apiErr := s.requireQuantity(ctx, "quantity", &q, true, ""); apiErr != nil {
			return apiErr
		}
		mat.Quantity = q
	} else if creating {
		return apierror.NewMissingFieldError("quantity", "quantity is required.")
	}
	if in.ScrapQuantity != nil {
		q := *in.ScrapQuantity
		if apiErr := s.requireQuantity(ctx, "scrap_quantity", &q, false, ""); apiErr != nil {
			return apiErr
		}
		mat.ScrapQuantity = q
	} else if creating {
		mat.ScrapQuantity = domain.QuantityValue{Value: "0", UnitID: mat.Quantity.UnitID, UnitType: mat.Quantity.UnitType}
	}
	if in.OperationID != nil || creating {
		ops, apiErr := s.repos.NewManufacturingMethodRepo().ListOperationIDs(ctx, m.ID)
		if apiErr != nil {
			return apiErr
		}
		switch {
		case in.OperationID != nil:
			if !slices.Contains(ops, *in.OperationID) {
				return apierror.NewFieldError("operation_id", apierror.CodeInvalidFormat, "Operation not found in this manufacturing method.")
			}
			v := *in.OperationID
			mat.OperationID = &v
		case len(ops) == 0:
			return apierror.NewMissingFieldError("operation_id", "Add an operation to the method before its materials.")
		default:
			first := ops[0]
			mat.OperationID = &first
		}
	}
	if in.Supply != nil {
		if !in.Supply.IsValid() {
			return apierror.NewFieldError("supply", apierror.CodeInvalidFormat, "Must be make, stock or buy.")
		}
		mat.Supply = *in.Supply
	} else if creating {
		made, apiErr := s.repos.NewManufacturingMethodRepo().ItemsWithActiveMethods(ctx, accountID, []string{mat.ItemID})
		if apiErr != nil {
			return apiErr
		}
		mat.Supply = constants.SupplyStock
		if made[mat.ItemID] {
			mat.Supply = constants.SupplyMake
		}
	}
	if v, ok := in.WorkInstructions.Value(); ok {
		mat.WorkInstructions = &v
	} else if in.WorkInstructions.IsClear() {
		mat.WorkInstructions = nil
	}
	return nil
}

func (s *manufacturingMethodSvcImpl) CreateMethodMaterial(ctx context.Context, params domain.CreateMethodMaterialParams) (*domain.MethodMaterial, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.create_material")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	result, apiErr := idempotentWrite(ctx, s, identity, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) (*domain.MethodMaterial, *apierror.APIError) {
		if _, apiErr := tx.lockDraft(txCtx, accountID, params.MethodID); apiErr != nil {
			return nil, apiErr
		}
		repo := tx.repos.NewManufacturingMethodRepo()
		m, apiErr := repo.Get(txCtx, accountID, params.MethodID)
		if apiErr != nil {
			return nil, apiErr
		}
		existing, apiErr := tx.allMaterials(txCtx, accountID, params.MethodID)
		if apiErr != nil {
			return nil, apiErr
		}
		if len(existing) >= maxMethodMaterials {
			return nil, apierror.NewConflictError(fmt.Sprintf("A manufacturing method can have at most %d materials.", maxMethodMaterials))
		}
		mat := &domain.MethodMaterial{MethodID: params.MethodID}
		if apiErr := tx.applyMaterialInput(txCtx, accountID, m, mat, params.MethodMaterialInput, true); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := tx.insertMaterial(txCtx, mat); apiErr != nil {
			return nil, apiErr
		}
		created, apiErr := tx.getMaterial(txCtx, accountID, params.MethodID, mat.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		return created, tx.publishAudit(txCtx, constants.AuditActionCreate, constants.ObjectTypeMethodMaterial, mat.ID, nil, created)
	})
	return result, tracing.Trace(span, apiErr)
}

func (s *manufacturingMethodSvcImpl) getMaterial(ctx context.Context, accountID, methodID, materialID string) (*domain.MethodMaterial, *apierror.APIError) {
	materials, apiErr := s.repos.NewManufacturingMethodRepo().GetMaterialsByIDs(ctx, accountID, []string{materialID})
	if apiErr != nil {
		return nil, apiErr
	}
	if len(materials) == 0 || materials[0].MethodID != methodID {
		return nil, apierror.NewNotFoundError("Material not found.")
	}
	return materials[0], nil
}

func (s *manufacturingMethodSvcImpl) UpdateMethodMaterial(ctx context.Context, params domain.UpdateMethodMaterialParams) (*domain.MethodMaterial, *apierror.APIError) {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.update_material")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	result, apiErr := idempotentWrite(ctx, s, identity, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) (*domain.MethodMaterial, *apierror.APIError) {
		if _, apiErr := tx.lockDraft(txCtx, accountID, params.MethodID); apiErr != nil {
			return nil, apiErr
		}
		repo := tx.repos.NewManufacturingMethodRepo()
		m, apiErr := repo.Get(txCtx, accountID, params.MethodID)
		if apiErr != nil {
			return nil, apiErr
		}
		before, apiErr := tx.getMaterial(txCtx, accountID, params.MethodID, params.MaterialID)
		if apiErr != nil {
			return nil, apiErr
		}
		mat := *before
		if apiErr := tx.applyMaterialInput(txCtx, accountID, m, &mat, params.MethodMaterialInput, false); apiErr != nil {
			return nil, apiErr
		}
		oldQuantities := []string{before.QuantityID, before.ScrapQuantityID}
		if apiErr := tx.writeMaterialQuantities(txCtx, &mat); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := repo.UpdateMaterial(txCtx, &mat); apiErr != nil {
			return nil, apiErr
		}
		if apiErr := repo.DeleteQuantities(txCtx, oldQuantities); apiErr != nil {
			return nil, apiErr
		}
		after, apiErr := tx.getMaterial(txCtx, accountID, params.MethodID, mat.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		return after, tx.publishAudit(txCtx, constants.AuditActionUpdate, constants.ObjectTypeMethodMaterial, mat.ID, before, after)
	})
	return result, tracing.Trace(span, apiErr)
}

func (s *manufacturingMethodSvcImpl) DeleteMethodMaterial(ctx context.Context, methodID, materialID string) *apierror.APIError {
	ctx, span := manufacturingMethodSvcTracer.Start(ctx, "service.manufacturing_method.delete_material")
	defer span.End()

	identity, apiErr := methodIdentity(ctx, types.ActionUpdate)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	apiErr = s.withTx(ctx, func(txCtx context.Context, tx *manufacturingMethodSvcImpl) *apierror.APIError {
		mat, apiErr := tx.getMaterial(txCtx, accountID, methodID, materialID)
		if apiErr != nil {
			if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
				gone, err := tx.repos.NewDeletedRecordRepo().ExistsInAccount(txCtx, constants.DeletedRecordResourceTypeMethodMaterial, materialID, accountID)
				if err != nil {
					return err
				}
				if gone {
					return apierror.NewGoneError("This material has been deleted.")
				}
			}
			return apiErr
		}
		if _, apiErr := tx.lockDraft(txCtx, accountID, methodID); apiErr != nil {
			return apiErr
		}
		if apiErr := tx.repos.NewManufacturingMethodRepo().DeleteMaterial(txCtx, mat); apiErr != nil {
			return apiErr
		}
		if apiErr := tx.repos.NewDeletedRecordRepo().CreateInAccount(txCtx, constants.DeletedRecordResourceTypeMethodMaterial, materialID, accountID, mat); apiErr != nil {
			return apiErr
		}
		return tx.publishAudit(txCtx, constants.AuditActionDelete, constants.ObjectTypeMethodMaterial, materialID, mat, (*domain.MethodMaterial)(nil))
	})
	return tracing.Trace(span, apiErr)
}
