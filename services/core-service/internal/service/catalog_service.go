package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

// checkCatalogReadPermission checks the appropriate read permission based on the target context.
// Internal actors need products:read for their own account, or customers:read / suppliers:read for external accounts.
func checkCatalogReadPermission(identity *types.Identity) *apierror.APIError {
	if !identity.IsInternalActor() {
		return nil
	}
	if identity.IsTargetCustomerAccount() {
		return identity.CheckHasPermission(types.PermissionDomainCustomers, types.ActionRead)
	}
	if identity.IsTargetSupplierAccount() {
		return identity.CheckHasPermission(types.PermissionDomainSuppliers, types.ActionRead)
	}
	return identity.CheckHasPermission(types.PermissionDomainProducts, types.ActionRead)
}

var catalogSvcTracer = tracing.GetTracer("core-service.service.catalog")

type catalogSvcImpl struct {
	repos           domain.RepoFactory
	mediatorFactory domain.MediatorFactory
}

type CatalogSvcConfig struct {
	Repos           domain.RepoFactory
	MediatorFactory domain.MediatorFactory
}

func (c *CatalogSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("catalog service: Repos is required")
	}
	if c.MediatorFactory == nil {
		return fmt.Errorf("catalog service: MediatorFactory is required")
	}
	return nil
}

func NewCatalogSvc(config *CatalogSvcConfig) domain.CatalogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &catalogSvcImpl{
		repos:           config.Repos,
		mediatorFactory: config.MediatorFactory,
	}
}

func (s *catalogSvcImpl) mediators() domain.Mediators {
	return s.mediatorFactory.Build(s.repos)
}

const defaultCatalogLimit int32 = 100

// ListCatalogProductLines returns a paginated list of product lines available in the catalog.
// Supports both internal and customer actors via CheckIsAssignedActor.
func (s *catalogSvcImpl) ListCatalogProductLines(ctx context.Context, params domain.ListCatalogProductLinesParams) (*domain.ListCatalogProductLinesResult, *apierror.APIError) {
	ctx, span := catalogSvcTracer.Start(ctx, "service.catalog.list_product_lines")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkCatalogReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	accountID := identity.Target.AccountID

	var allProductLines []*domain.CatalogProductLine
	var apiErr *apierror.APIError

	if identity.IsCustomerUser() && identity.Actor.AccountID != nil {
		allProductLines, apiErr = s.repos.NewCatalogRepo().ListProductLinesForCustomer(ctx, accountID, *identity.Actor.AccountID)
	} else {
		allProductLines, apiErr = s.repos.NewCatalogRepo().ListProductLines(ctx, accountID)
	}
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Apply query filter.
	if params.Query != nil && *params.Query != "" {
		query := strings.ToLower(*params.Query)
		filtered := make([]*domain.CatalogProductLine, 0, len(allProductLines))
		for _, pl := range allProductLines {
			if strings.Contains(strings.ToLower(pl.Name), query) {
				filtered = append(filtered, pl)
			}
		}
		allProductLines = filtered
	}

	// Apply cursor-based pagination.
	limit := params.Limit
	if limit <= 0 {
		limit = defaultCatalogLimit
	}

	productLines, pageInfo := paginateByID(allProductLines, limit, params.Cursor, func(pl *domain.CatalogProductLine) string { return pl.ID })

	return &domain.ListCatalogProductLinesResult{
		ProductLines: productLines,
		PageInfo:     pageInfo,
	}, nil
}

// ListCatalogProducts returns a paginated list of products in a specific product line, grouped by item category.
// Supports both internal and customer actors via CheckIsAssignedActor.
func (s *catalogSvcImpl) ListCatalogProducts(ctx context.Context, params domain.ListCatalogProductsParams) (*domain.ListCatalogProductsResult, *apierror.APIError) {
	ctx, span := catalogSvcTracer.Start(ctx, "service.catalog.list_products")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}

	if apiErr := identity.CheckIsAssignedActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := checkCatalogReadPermission(identity); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsExternalTarget() {
		meds := s.mediators()
		if apiErr := meds.ReadAccess.CheckReadAccess(ctx, *identity.ActorAccountID(), identity.Target.AccountID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	accountID := identity.Target.AccountID

	var allCategories []*domain.CatalogCategory
	var apiErr *apierror.APIError

	if identity.IsCustomerUser() && identity.Actor.AccountID != nil {
		allCategories, apiErr = s.repos.NewCatalogRepo().ListProductsForCustomer(ctx, accountID, *identity.Actor.AccountID, params.ProductLineID)
	} else {
		allCategories, apiErr = s.repos.NewCatalogRepo().ListProducts(ctx, accountID, params.ProductLineID)
	}
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Apply query filter on category name.
	if params.Query != nil && *params.Query != "" {
		query := strings.ToLower(*params.Query)
		filtered := make([]*domain.CatalogCategory, 0, len(allCategories))
		for _, cat := range allCategories {
			if strings.Contains(strings.ToLower(cat.Name), query) {
				filtered = append(filtered, cat)
			}
		}
		allCategories = filtered
	}

	// Apply cursor-based pagination.
	limit := params.Limit
	if limit <= 0 {
		limit = defaultCatalogLimit
	}

	categories, pageInfo := paginateByID(allCategories, limit, params.Cursor, func(cat *domain.CatalogCategory) string { return cat.ID })

	return &domain.ListCatalogProductsResult{
		Categories: categories,
		PageInfo:   pageInfo,
	}, nil
}

// paginateByID applies simple cursor-based pagination to a slice of items using the item's ID as cursor.
// The cursor is the ID of the last item on the previous page. Items after that cursor are returned.
func paginateByID[T any](items []T, limit int32, cursor *string, getID func(T) string) ([]T, pagination.PageInfo) {
	if len(items) == 0 {
		return items, pagination.PageInfo{}
	}

	// If a cursor is provided, skip items up to and including the cursor.
	startIdx := 0
	if cursor != nil && *cursor != "" {
		for i, item := range items {
			if getID(item) == *cursor {
				startIdx = i + 1
				break
			}
		}
	}

	// Slice from startIdx.
	remaining := items[startIdx:]

	hasNextPage := len(remaining) > int(limit)
	hasPrevPage := startIdx > 0

	if hasNextPage {
		remaining = remaining[:limit]
	}

	var pi pagination.PageInfo
	pi.HasNextPage = hasNextPage
	pi.HasPrevPage = hasPrevPage

	if hasNextPage && len(remaining) > 0 {
		lastID := getID(remaining[len(remaining)-1])
		pi.NextCursor = &lastID
	}
	if hasPrevPage && len(remaining) > 0 {
		firstID := getID(remaining[0])
		pi.PrevCursor = &firstID
	}

	return remaining, pi
}
