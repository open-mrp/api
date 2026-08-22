package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var customerProductLineAccessRepoTracer = tracing.GetTracer("core-service.customer_product_line_access_repository")

type customerProductLineAccessRepoImpl struct {
	queries *sqlc.Queries
}

func NewCustomerProductLineAccessRepo(queries *sqlc.Queries) domain.CustomerProductLineAccessRepo {
	return &customerProductLineAccessRepoImpl{queries: queries}
}

func customerAccessCreatedAt(a *domain.CustomerProductLineAccess) time.Time { return a.CreatedAt }
func customerAccessCustomerID(a *domain.CustomerProductLineAccess) string   { return a.CustomerID }

func buildCustomerAccessSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
}

// groupCustomerForwardRows groups flat forward rows into domain objects, preserving order.
func groupCustomerForwardRows(rows []sqlc.ListCustomerProductLineAccessForwardRow) []*domain.CustomerProductLineAccess {
	var result []*domain.CustomerProductLineAccess
	seen := map[string]int{} // customerID -> index in result

	for _, row := range rows {
		idx, ok := seen[row.CustomerID]
		if !ok {
			idx = len(result)
			seen[row.CustomerID] = idx
			result = append(result, &domain.CustomerProductLineAccess{
				CustomerID:     row.CustomerID,
				CustomerName:   row.CustomerName,
				CustomerNumber: row.CustomerNumber,
				CreatedAt:      row.CreatedAt,
				UpdatedAt:      row.UpdatedAt,
			})
		}
		result[idx].ProductLines = append(result[idx].ProductLines, domain.ProductLineInfo{
			ID:   row.ProductLineID,
			Name: row.ProductLineName,
		})
	}

	return result
}

// groupCustomerBackwardRows groups flat backward rows into domain objects, preserving order.
func groupCustomerBackwardRows(rows []sqlc.ListCustomerProductLineAccessBackwardRow) []*domain.CustomerProductLineAccess {
	var result []*domain.CustomerProductLineAccess
	seen := map[string]int{}

	for _, row := range rows {
		idx, ok := seen[row.CustomerID]
		if !ok {
			idx = len(result)
			seen[row.CustomerID] = idx
			result = append(result, &domain.CustomerProductLineAccess{
				CustomerID:     row.CustomerID,
				CustomerName:   row.CustomerName,
				CustomerNumber: row.CustomerNumber,
				CreatedAt:      row.CreatedAt,
				UpdatedAt:      row.UpdatedAt,
			})
		}
		result[idx].ProductLines = append(result[idx].ProductLines, domain.ProductLineInfo{
			ID:   row.ProductLineID,
			Name: row.ProductLineName,
		})
	}

	return result
}

// customerRowLimitMultiplier is used to over-fetch rows so we get enough unique customers after grouping. Since each customer may have multiple product lines, we fetch more rows than the requested limit.
const customerRowLimitMultiplier int32 = 20

func (r *customerProductLineAccessRepoImpl) List(ctx context.Context, params domain.ListCustomerProductLineAccessParams) (*domain.ListCustomerProductLineAccessResult, *apierror.APIError) {
	ctx, span := customerProductLineAccessRepoTracer.Start(ctx, "repository.customer_product_line_access.list")
	defer span.End()

	searchQuery := buildCustomerAccessSearchParams(params.Query)
	rowLimit := (params.Limit + 1) * customerRowLimitMultiplier

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListCustomerProductLineAccessBackward(ctx, sqlc.ListCustomerProductLineAccessBackwardParams{
				OwnerAccountID:  params.AccountID,
				SearchQuery:     searchQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           rowLimit,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			groups := groupCustomerBackwardRows(rows)
			result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, customerAccessCreatedAt, customerAccessCustomerID)
			return &domain.ListCustomerProductLineAccessResult{Items: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListCustomerProductLineAccessForward(ctx, sqlc.ListCustomerProductLineAccessForwardParams{
			OwnerAccountID:  params.AccountID,
			SearchQuery:     searchQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           rowLimit,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		groups := groupCustomerForwardRows(rows)
		result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, customerAccessCreatedAt, customerAccessCustomerID)
		return &domain.ListCustomerProductLineAccessResult{Items: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListCustomerProductLineAccessForward(ctx, sqlc.ListCustomerProductLineAccessForwardParams{
		OwnerAccountID: params.AccountID,
		SearchQuery:    searchQuery,
		Limit:          rowLimit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	groups := groupCustomerForwardRows(rows)
	result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, customerAccessCreatedAt, customerAccessCustomerID)
	return &domain.ListCustomerProductLineAccessResult{Items: result, PageInfo: pageInfo}, nil
}

func (r *customerProductLineAccessRepoImpl) Get(ctx context.Context, accountID, customerID string) (*domain.CustomerProductLineAccess, *apierror.APIError) {
	ctx, span := customerProductLineAccessRepoTracer.Start(ctx, "repository.customer_product_line_access.get")
	defer span.End()

	// Verify customer relation exists and belongs to this account.
	arRow, err := r.queries.GetAccountRelationForCustomer(ctx, sqlc.GetAccountRelationForCustomerParams{
		OwnerAccountID:        accountID,
		CounterpartyAccountID: customerID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Get the product lines for this account relation.
	plRows, err := r.queries.GetCustomerProductLineAccess(ctx, arRow.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if len(plRows) == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Customer product line access not found."))
	}

	productLines := make([]domain.ProductLineInfo, len(plRows))
	for i, row := range plRows {
		productLines[i] = domain.ProductLineInfo{
			ID:   row.ProductLineID,
			Name: row.ProductLineName,
		}
	}

	return &domain.CustomerProductLineAccess{
		CustomerID:     customerID,
		CustomerName:   arRow.Name,
		CustomerNumber: arRow.ExternalNumber,
		ProductLines:   productLines,
		CreatedAt:      arRow.CreatedAt,
		UpdatedAt:      arRow.UpdatedAt,
	}, nil
}

func (r *customerProductLineAccessRepoImpl) Create(ctx context.Context, params domain.CreateCustomerProductLineAccessParams) (*domain.CustomerProductLineAccess, *apierror.APIError) {
	ctx, span := customerProductLineAccessRepoTracer.Start(ctx, "repository.customer_product_line_access.create")
	defer span.End()

	// Verify customer relation exists.
	arRow, err := r.queries.GetAccountRelationForCustomer(ctx, sqlc.GetAccountRelationForCustomerParams{
		OwnerAccountID:        params.AccountID,
		CounterpartyAccountID: params.CustomerID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Verify no existing product line access mapping exists for this customer.
	count, err := r.queries.CountAccountRelationProductLinesByRelationID(ctx, arRow.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if count > 0 {
		return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam("Product line access already exists for this customer. Use update instead.", "customer_id"))
	}

	// Verify each product line exists and belongs to this account.
	for _, plID := range params.ProductLineIDs {
		plCount, err := r.queries.ProductLineExistsByIDAndAccount(ctx, sqlc.ProductLineExistsByIDAndAccountParams{
			ID:        plID,
			AccountID: gosql.NullString{String: params.AccountID, Valid: true},
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if plCount == 0 {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Product line not found: "+plID))
		}
	}

	// Insert each product line mapping.
	for _, plID := range params.ProductLineIDs {
		newID, apiErr := id.GenID(id.AccountRelationProductLineIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		err := r.queries.InsertAccountRelationProductLine(ctx, sqlc.InsertAccountRelationProductLineParams{
			ID:                newID,
			AccountRelationID: arRow.ID,
			ProductLineID:     plID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Re-fetch and return the created access.
	return r.Get(ctx, params.AccountID, params.CustomerID)
}

func (r *customerProductLineAccessRepoImpl) Update(ctx context.Context, params domain.UpdateCustomerProductLineAccessParams) (*domain.CustomerProductLineAccess, *apierror.APIError) {
	ctx, span := customerProductLineAccessRepoTracer.Start(ctx, "repository.customer_product_line_access.update")
	defer span.End()

	// Verify customer relation exists.
	arRow, err := r.queries.GetAccountRelationForCustomer(ctx, sqlc.GetAccountRelationForCustomerParams{
		OwnerAccountID:        params.AccountID,
		CounterpartyAccountID: params.CustomerID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Verify each product line exists and belongs to this account.
	for _, plID := range params.ProductLineIDs {
		plCount, err := r.queries.ProductLineExistsByIDAndAccount(ctx, sqlc.ProductLineExistsByIDAndAccountParams{
			ID:        plID,
			AccountID: gosql.NullString{String: params.AccountID, Valid: true},
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if plCount == 0 {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Product line not found: "+plID))
		}
	}

	// Delete old rows.
	_, err = r.queries.DeleteAccountRelationProductLinesByRelationID(ctx, arRow.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Insert new rows.
	for _, plID := range params.ProductLineIDs {
		newID, apiErr := id.GenID(id.AccountRelationProductLineIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		err := r.queries.InsertAccountRelationProductLine(ctx, sqlc.InsertAccountRelationProductLineParams{
			ID:                newID,
			AccountRelationID: arRow.ID,
			ProductLineID:     plID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Re-fetch and return the updated access.
	return r.Get(ctx, params.AccountID, params.CustomerID)
}

func (r *customerProductLineAccessRepoImpl) Delete(ctx context.Context, accountID, customerID string) *apierror.APIError {
	ctx, span := customerProductLineAccessRepoTracer.Start(ctx, "repository.customer_product_line_access.delete")
	defer span.End()

	// Verify customer relation exists.
	arRow, err := r.queries.GetAccountRelationForCustomer(ctx, sqlc.GetAccountRelationForCustomerParams{
		OwnerAccountID:        accountID,
		CounterpartyAccountID: customerID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Verify mapping exists.
	count, err := r.queries.CountAccountRelationProductLinesByRelationID(ctx, arRow.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if count == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("No product line access found for this customer."))
	}

	// Delete all rows.
	_, err = r.queries.DeleteAccountRelationProductLinesByRelationID(ctx, arRow.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *customerProductLineAccessRepoImpl) ExistsByCustomerID(ctx context.Context, accountID, customerID string) (bool, *apierror.APIError) {
	ctx, span := customerProductLineAccessRepoTracer.Start(ctx, "repository.customer_product_line_access.exists_by_customer_id")
	defer span.End()

	arRow, err := r.queries.GetAccountRelationForCustomer(ctx, sqlc.GetAccountRelationForCustomerParams{
		OwnerAccountID:        accountID,
		CounterpartyAccountID: customerID,
	})
	if err != nil {
		// If no relation found, customer doesn't exist
		return false, nil
	}

	count, err := r.queries.CountAccountRelationProductLinesByRelationID(ctx, arRow.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
