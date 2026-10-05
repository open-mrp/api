package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

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

var sysPropertySvcTracer = tracing.GetTracer("core-service.sys_property_service")

type sysPropertySvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
	txManager       TransactionManager
}

type SysPropertySvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory

	// MediatorFactory (required) builds the mediators used by this service.
	MediatorFactory domain.MediatorFactory

	// TxManager (required) wraps multi-step operations in database transactions.
	TxManager TransactionManager
}

func (c *SysPropertySvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("sys property service: repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("sys property service: mediator factory is required")
	}
	if c.TxManager == nil {
		return fmt.Errorf("sys property service: tx manager is required")
	}
	return nil
}

func NewSysPropertySvc(config *SysPropertySvcConfig) domain.SysPropertySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &sysPropertySvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
		txManager:       config.TxManager,
	}
}

func (s *sysPropertySvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

func (s *sysPropertySvcImpl) withTx(ctx context.Context, fn func(context.Context, *sysPropertySvcImpl) *apierror.APIError) *apierror.APIError {
	if s.txManager == nil {
		panic("txManager is nil")
	}
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txSvc := &sysPropertySvcImpl{
			repos:           f,
			mediatorFactory: s.mediatorFactory,
			txManager:       s.txManager,
		}
		return fn(txCtx, txSvc)
	})
}

func (s *sysPropertySvcImpl) ListSysProperties(ctx context.Context, params domain.ListSysPropertiesParams) (*domain.ListSysPropertiesResult, *apierror.APIError) {
	ctx, span := sysPropertySvcTracer.Start(ctx, "service.sys_property.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSystemProperties, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewSysPropertyRepo().List(ctx, params)
}

func (s *sysPropertySvcImpl) GetSysProperty(ctx context.Context, sysPropertyID string) (*domain.SysProperty, *apierror.APIError) {
	ctx, span := sysPropertySvcTracer.Start(ctx, "service.sys_property.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSystemProperties, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewSysPropertyRepo().Get(ctx, identity.Target.AccountID, sysPropertyID)
}

// BatchGetSysPropertiesByIDs returns sys properties matching the input IDs. Account-scoped via the caller's identity.
func (s *sysPropertySvcImpl) BatchGetSysPropertiesByIDs(ctx context.Context, ids []string) ([]*domain.SysProperty, *apierror.APIError) {
	ctx, span := sysPropertySvcTracer.Start(ctx, "service.sys_property.batch_get_by_ids")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSystemProperties, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.repos.NewSysPropertyRepo().GetByIDs(ctx, identity.Target.AccountID, ids)
}

func (s *sysPropertySvcImpl) UpdateSysProperty(ctx context.Context, params domain.UpdateSysPropertyParams) (*domain.SysProperty, *apierror.APIError) {
	ctx, span := sysPropertySvcTracer.Start(ctx, "service.sys_property.update")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSystemProperties, types.ActionUpdate); apiErr != nil {
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.SysProperty](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.SysProperty
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *sysPropertySvcImpl) *apierror.APIError {
			txRepo := txSvc.repos.NewSysPropertyRepo()

			old, apiErr := txRepo.Get(txCtx, params.AccountID, params.ID)
			if apiErr != nil {
				return apiErr
			}

			// An omitted value leaves the counter where it is; zeroing it would reissue every number.
			value := old.Value
			if params.Value != nil {
				value = *params.Value
			}
			updated, apiErr := txRepo.UpdateValue(txCtx, params.AccountID, params.ID, value)
			if apiErr != nil {
				return apiErr
			}
			result = updated

			changes := audit.ComputeChanges(old, updated)

			if apiErr := audit.NewPublisher().Publish(txCtx, txSvc.repos.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionUpdate,
				ResourceType: constants.ObjectTypeSysProperty,
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

func (s *sysPropertySvcImpl) GetLatestSysPropertyValue(ctx context.Context, typeCode constants.SysPropertyTypeCode) (string, *apierror.APIError) {
	ctx, span := sysPropertySvcTracer.Start(ctx, "service.sys_property.get_latest_value")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return "", tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSystemProperties, types.ActionUpdate); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	// This read initializes a counter it does not find, so an unrecognized code would be
	// written straight through to a foreign key that has no such row — surfacing as a
	// "resource already exists" conflict, which tells the caller nothing about the typo.
	if !typeCode.IsValid() {
		return "", tracing.Trace(span, apierror.NewValidationErrorWithParam(
			"Unknown system property type. Supported values: "+strings.Join(constants.SysPropertyTypeCode("").EnumValues(), ", "),
			"type_code",
		))
	}

	accountID := identity.Target.AccountID
	repo := s.repos.NewSysPropertyRepo()

	sysProp, apiErr := repo.GetByTypeCode(ctx, accountID, typeCode)
	if apiErr != nil {
		if !apierror.IsNotFound(apiErr) {
			return "", tracing.Trace(span, apiErr)
		}
		first, apiErr := nextFreeNumber(ctx, repo, accountID, typeCode, 1)
		if apiErr != nil {
			return "", tracing.Trace(span, apiErr)
		}
		newID, genErr := id.GenID(id.SysPropertyIDPrefix, nil)
		if genErr != nil {
			return "", tracing.Trace(span, genErr)
		}
		created, createErr := repo.Create(ctx, newID, accountID, typeCode, first)
		if createErr != nil {
			return "", tracing.Trace(span, createErr)
		}
		return strconv.Itoa(int(created.Value)), nil
	}

	// SSCC count: always return current value without duplicate check (matches dashboard behavior)
	if typeCode == constants.SysPropertyTypeCodeSsccCount {
		return strconv.Itoa(int(sysProp.Value)), nil
	}

	next, apiErr := nextFreeNumber(ctx, repo, accountID, typeCode, sysProp.Value)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if next == sysProp.Value {
		return strconv.Itoa(int(next)), nil
	}

	// The counter rests on the number handed out without taking it, so reading again returns it again.
	moved, apiErr := repo.UpdateValue(ctx, accountID, sysProp.ID, next)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	return strconv.Itoa(int(moved.Value)), nil
}

// The search for a free number reads candidates in batches, one query each, growing so the usual
// short run of taken numbers costs one small query and a long one only a handful.
const (
	freeNumberFirstBatch  = 16
	freeNumberMaxBatch    = 1024
	freeNumberSearchLimit = 10_000
)

// nextFreeNumber returns the first number at or after from that no record in typeCode's series carries.
func nextFreeNumber(ctx context.Context, repo domain.SysPropertyRepo, accountID string, typeCode constants.SysPropertyTypeCode, from int32) (int32, *apierror.APIError) {
	start, end := int64(from), min(int64(from)+freeNumberSearchLimit, math.MaxInt32+1)
	for batch := int64(freeNumberFirstBatch); start < end; batch = min(batch*4, freeNumberMaxBatch) {
		candidates := make([]string, min(batch, end-start))
		for i := range candidates {
			candidates[i] = strconv.FormatInt(start+int64(i), 10)
		}
		taken, apiErr := repo.TakenNumbers(ctx, accountID, typeCode, candidates)
		if apiErr != nil {
			return 0, apiErr
		}
		inUse := make(map[string]bool, len(taken))
		for _, n := range taken {
			inUse[n] = true
		}
		for i, c := range candidates {
			if !inUse[c] {
				return int32(start + int64(i)), nil // #nosec G115 -- below end, which is at most MaxInt32+1
			}
		}
		start += int64(len(candidates))
	}
	return 0, apierror.NewResourceConflictError(fmt.Sprintf(
		"Every number from %d to %d is already in use. Move the counter past the numbers already issued.", from, start-1,
	))
}

func (s *sysPropertySvcImpl) GetSysPropertyValue(ctx context.Context, code string) (*domain.SysPropertyValue, *apierror.APIError) {
	ctx, span := sysPropertySvcTracer.Start(ctx, "service.sys_property.get_value")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSystemProperties, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	accountID := identity.Target.AccountID
	repo := s.repos.NewSysPropertyRepo()

	sysProp, apiErr := repo.GetByTypeCode(ctx, accountID, constants.SysPropertyTypeCode(code))
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.SysPropertyValue{
		Value: strconv.Itoa(int(sysProp.Value)),
	}, nil
}
