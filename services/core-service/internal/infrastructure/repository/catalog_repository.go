package repository

import (
	"context"
	gosql "database/sql"

	"go.opentelemetry.io/otel/trace"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var catalogRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.catalog")

type catalogRepoImpl struct {
	queries *sqlc.Queries
}

func NewCatalogRepo(queries *sqlc.Queries) domain.CatalogRepo {
	return &catalogRepoImpl{queries: queries}
}

func (r *catalogRepoImpl) ListProductLines(ctx context.Context, accountID string) ([]*domain.CatalogProductLine, *apierror.APIError) {
	ctx, span := catalogRepoTracer.Start(ctx, "repository.catalog.list_product_lines")
	defer span.End()

	rows, err := r.queries.ListCatalogProductLines(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productLines := make([]*domain.CatalogProductLine, len(rows))
	for i, row := range rows {
		productLines[i] = &domain.CatalogProductLine{
			ID:   row.ID,
			Name: row.Name,
		}
	}

	return productLines, nil
}

func (r *catalogRepoImpl) ListProductLinesForCustomer(ctx context.Context, accountID, customerAccountID string) ([]*domain.CatalogProductLine, *apierror.APIError) {
	ctx, span := catalogRepoTracer.Start(ctx, "repository.catalog.list_product_lines_for_customer")
	defer span.End()

	rows, err := r.queries.ListCatalogProductLinesForCustomer(ctx, sqlc.ListCatalogProductLinesForCustomerParams{
		AccountID:         accountID,
		CustomerAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productLines := make([]*domain.CatalogProductLine, len(rows))
	for i, row := range rows {
		productLines[i] = &domain.CatalogProductLine{
			ID:   row.ID,
			Name: row.Name,
		}
	}

	return productLines, nil
}

type catalogProductRow struct {
	CategoryID   string
	CategoryName string
	ItemID       string
	SKU          string
	Description  gosql.NullString
}

func (r *catalogRepoImpl) ListProducts(ctx context.Context, accountID, productLineID string) ([]*domain.CatalogCategory, *apierror.APIError) {
	ctx, span := catalogRepoTracer.Start(ctx, "repository.catalog.list_products")
	defer span.End()

	rows, err := r.queries.ListCatalogProducts(ctx, sqlc.ListCatalogProductsParams{
		ProductLineID:     gosql.NullString{String: productLineID, Valid: true},
		AccountID:         accountID,
		CategoryAccountID: gosql.NullString{String: accountID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productRows := make([]catalogProductRow, len(rows))
	for i, row := range rows {
		productRows[i] = catalogProductRow{
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			ItemID:       row.ItemID,
			SKU:          row.Sku,
			Description:  row.Description,
		}
	}

	return r.buildCatalogCategories(ctx, span, productRows)
}

func (r *catalogRepoImpl) ListProductsForCustomer(ctx context.Context, accountID, customerAccountID, productLineID string) ([]*domain.CatalogCategory, *apierror.APIError) {
	ctx, span := catalogRepoTracer.Start(ctx, "repository.catalog.list_products_for_customer")
	defer span.End()

	rows, err := r.queries.ListCatalogProductsForCustomer(ctx, sqlc.ListCatalogProductsForCustomerParams{
		ProductLineID:     gosql.NullString{String: productLineID, Valid: true},
		AccountID:         accountID,
		CategoryAccountID: gosql.NullString{String: accountID, Valid: true},
		CustomerAccountID: customerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	productRows := make([]catalogProductRow, len(rows))
	for i, row := range rows {
		productRows[i] = catalogProductRow{
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			ItemID:       row.ItemID,
			SKU:          row.Sku,
			Description:  row.Description,
		}
	}

	return r.buildCatalogCategories(ctx, span, productRows)
}

func (r *catalogRepoImpl) buildCatalogCategories(ctx context.Context, span trace.Span, rows []catalogProductRow) ([]*domain.CatalogCategory, *apierror.APIError) {
	if len(rows) == 0 {
		return []*domain.CatalogCategory{}, nil
	}

	// Group rows by category and collect IDs.
	categoryMap := make(map[string]*domain.CatalogCategory)
	var categoryOrder []string
	var categoryIDs []string
	var itemIDs []string
	for _, row := range rows {
		cat, exists := categoryMap[row.CategoryID]
		if !exists {
			cat = &domain.CatalogCategory{
				ID:         row.CategoryID,
				Name:       row.CategoryName,
				Properties: make([]*domain.CatalogProperty, 0),
				Products:   make([]*domain.CatalogProduct, 0),
			}
			categoryMap[row.CategoryID] = cat
			categoryOrder = append(categoryOrder, row.CategoryID)
			categoryIDs = append(categoryIDs, row.CategoryID)
		}

		description := ""
		if row.Description.Valid {
			description = row.Description.String
		}

		cat.Products = append(cat.Products, &domain.CatalogProduct{
			ItemID:      row.ItemID,
			SKU:         row.SKU,
			Description: description,
			Attributes:  make([]*domain.CatalogAttribute, 0),
		})
		itemIDs = append(itemIDs, row.ItemID)
	}

	// Fetch properties for all categories.
	propRows, err := r.queries.ListCatalogCategoryProperties(ctx, categoryIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for _, pr := range propRows {
		cat, exists := categoryMap[pr.ItemCategoryID]
		if !exists {
			continue
		}
		cat.Properties = append(cat.Properties, &domain.CatalogProperty{
			ID:   pr.PropertyID,
			Name: pr.PropertyName,
		})
	}

	// Fetch attributes for all items.
	attrRows, err := r.queries.ListCatalogProductAttributes(ctx, itemIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Build a map of item ID -> attributes for efficient lookup.
	itemAttrMap := make(map[string][]*domain.CatalogAttribute)
	for _, ar := range attrRows {
		itemAttrMap[ar.ItemID] = append(itemAttrMap[ar.ItemID], &domain.CatalogAttribute{
			ID:           ar.AttributeID,
			Name:         ar.AttributeName,
			PropertyID:   ar.PropertyID,
			PropertyName: ar.PropertyName,
		})
	}

	// Assign attributes to products.
	for _, cat := range categoryMap {
		for _, product := range cat.Products {
			if attrs, exists := itemAttrMap[product.ItemID]; exists {
				product.Attributes = attrs
			}
		}
	}

	categories := make([]*domain.CatalogCategory, len(categoryOrder))
	for i, id := range categoryOrder {
		categories[i] = categoryMap[id]
	}

	return categories, nil
}
