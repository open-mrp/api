package service

import (
	"context"
	"fmt"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var inventoryChangeLogSvcTracer = tracing.GetTracer("core-service.inventory_change_log_service")

type inventoryChangeLogSvcImpl struct {
	repos domain.RepoFactory
}

type InventoryChangeLogSvcConfig struct {
	// Repos (required) is the repository factory.
	Repos domain.RepoFactory
}

func (c *InventoryChangeLogSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("inventory change log service: repos is required")
	}
	return nil
}

func NewInventoryChangeLogSvc(config *InventoryChangeLogSvcConfig) domain.InventoryChangeLogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &inventoryChangeLogSvcImpl{
		repos: config.Repos,
	}
}

// ListInventoryChangeLogs returns a paginated list of inventory change logs for the caller's account.
func (s *inventoryChangeLogSvcImpl) ListInventoryChangeLogs(ctx context.Context, params domain.ListInventoryChangeLogsParams) (*domain.ListInventoryChangeLogsResult, *apierror.APIError) {
	ctx, span := inventoryChangeLogSvcTracer.Start(ctx, "service.inventory_change_log.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInventoryLogs, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID
	withDefaultInventoryChangeLogWindow(&params, time.Now().UTC())

	return s.repos.NewInventoryChangeLogRepo().List(ctx, params)
}

// defaultInventoryChangeLogWindow is how far back a list reaches when the caller gives no start. Stock
// moves are looked back on for weeks, not hours, but a busy account logs tens of thousands a month, and
// a combination of filters the indexes cannot order walks every one of them.
const defaultInventoryChangeLogWindow = 90 * 24 * time.Hour

// withDefaultInventoryChangeLogWindow bounds a list that is not about particular items. An item's
// history is wanted whole and is already served by the item index.
func withDefaultInventoryChangeLogWindow(params *domain.ListInventoryChangeLogsParams, now time.Time) {
	if params.StartDate != nil || len(params.ItemIDs) > 0 {
		return
	}
	end := now
	if params.EndDate != nil {
		end = *params.EndDate
	}
	start := end.Add(-defaultInventoryChangeLogWindow)
	params.StartDate = &start
}

// GetInventoryChangeLog returns a single inventory change log by ID.
func (s *inventoryChangeLogSvcImpl) GetInventoryChangeLog(ctx context.Context, id string) (*domain.InventoryChangeLog, *apierror.APIError) {
	ctx, span := inventoryChangeLogSvcTracer.Start(ctx, "service.inventory_change_log.get")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInventoryLogs, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return s.repos.NewInventoryChangeLogRepo().Get(ctx, identity.Target.AccountID, id)
}

// ExportInventoryChangeLogs returns all inventory change logs matching the provided filters.
func (s *inventoryChangeLogSvcImpl) ExportInventoryChangeLogs(ctx context.Context, params domain.ExportInventoryChangeLogsParams) ([]*domain.InventoryChangeLog, *apierror.APIError) {
	ctx, span := inventoryChangeLogSvcTracer.Start(ctx, "service.inventory_change_log.export")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainInventoryLogs, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	params.AccountID = identity.Target.AccountID

	return s.repos.NewInventoryChangeLogRepo().ListAll(ctx, params)
}
