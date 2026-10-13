package service

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/idempotency"
	"github.com/open-mrp/api/shared/tracing"
)

const defaultSuggestWindowMonths int32 = 24

func (s *machineSvcImpl) ListMachineItems(ctx context.Context, params domain.ListMachineItemsParams) (*domain.ListMachineItemsResult, *apierror.APIError) {
	ctx, span := machineSvcTracer.Start(ctx, "service.machine.list_items")
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

	if _, apiErr := s.repos.NewMachineRepo().Get(ctx, domain.GetMachineParams{
		AccountID: params.AccountID,
		MachineID: params.MachineID,
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	result, apiErr := s.repos.NewMachineRepo().ListItems(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return result, nil
}

func (s *machineSvcImpl) ReplaceMachineItems(ctx context.Context, params domain.ReplaceMachineItemsParams) (*domain.ListMachineItemsResult, *apierror.APIError) {
	ctx, span := machineSvcTracer.Start(ctx, "service.machine.replace_items")
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
		cached, err := idempotency.UnmarshalCachedResponse[domain.ListMachineItemsResult](ctx, idempotencyKey.ResponseCode, idempotencyKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error

	case domain.RecoveryPointStarted:
		var result *domain.ListMachineItemsResult
		apiErr = s.withTx(ctx, func(txCtx context.Context, txSvc *machineSvcImpl) *apierror.APIError {
			if _, apiErr := txSvc.repos.NewMachineRepo().Get(txCtx, domain.GetMachineParams{
				AccountID: params.AccountID,
				MachineID: params.MachineID,
			}); apiErr != nil {
				return apiErr
			}

			uniqueIDs := uniqueStrings(params.ItemIDs)
			if len(uniqueIDs) > 0 {
				items, apiErr := txSvc.repos.NewItemRepo().GetByIDsWithIncludes(txCtx, params.AccountID, uniqueIDs, nil)
				if apiErr != nil {
					return apiErr
				}
				if len(items) != len(uniqueIDs) {
					return apierror.NewValidationErrorWithParam("One or more items were not found.", "item_ids")
				}
			}

			if apiErr := txSvc.repos.NewMachineRepo().ReplaceItems(txCtx, params.AccountID, params.MachineID, uniqueIDs); apiErr != nil {
				return apiErr
			}

			listed, apiErr := txSvc.repos.NewMachineRepo().ListItems(txCtx, domain.ListMachineItemsParams{
				AccountID: params.AccountID,
				MachineID: params.MachineID,
				Limit:     100,
			})
			if apiErr != nil {
				return apiErr
			}
			result = listed

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

func (s *machineSvcImpl) SuggestMachineItemsFromScans(ctx context.Context, machineID string, windowMonths *int32) ([]*domain.MachineItemScanSuggestion, *apierror.APIError) {
	ctx, span := machineSvcTracer.Start(ctx, "service.machine.suggest_items_from_scans")
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

	accountID := identity.Target.AccountID

	if _, apiErr := s.repos.NewMachineRepo().Get(ctx, domain.GetMachineParams{
		AccountID: accountID,
		MachineID: machineID,
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	months := defaultSuggestWindowMonths
	if windowMonths != nil && *windowMonths > 0 {
		months = *windowMonths
	}

	windowEnd := time.Now().UTC()
	windowStart := windowEnd.AddDate(0, -int(months), 0)

	suggestions, apiErr := s.repos.NewMachineRepo().SuggestItemsFromScans(ctx, domain.SuggestMachineItemsFromScansParams{
		AccountID:   accountID,
		MachineID:   machineID,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return suggestions, nil
}

func uniqueStrings(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
